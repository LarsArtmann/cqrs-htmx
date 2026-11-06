package cqrshtmx

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// capturedRecord is one JSON-decoded slog record from the dispatch error path.
type capturedRecord struct {
	Level  string `json:"level"`
	Msg    string `json:"msg"`
	Method string `json:"method"`
	Path   string `json:"path"`

	Family    string `json:"family"`
	Code      string `json:"code"`
	Retryable bool   `json:"retryable"`
	// The fixture mirrors the handler's actual log wire format, which is
	// snake_case/dotted — not the linter's camelCase preference.
	ExitCode  int    `json:"exit_code"` //nolint:tagliatelle // mirrors the handler's snake_case log key

	ContextPoolSize string `json:"context.pool_size"` //nolint:tagliatelle // mirrors the handler's dotted log key
}

// dispatchFailureApp builds an App whose "Fail" command always dispatches err,
// then invokes the full HTTP pipeline so the dispatch error path runs.
func dispatchFailureApp(t *testing.T, err error) *httptest.ResponseRecorder {
	t.Helper()

	cmdDisp := command.NewDispatcher()
	cmdDisp.Register("Fail", func(context.Context, command.Command) error { return err })

	app := MustNew(Config{Commands: cmdDisp})
	mux := http.NewServeMux()
	mux.Handle("POST /fail", app.Command("Fail", DecodeJSON(func(_ struct{}) (command.Command, error) {
		core, cerr := command.New("Fail", id.NewStreamID())
		if cerr != nil {
			return nil, cerr
		}

		return core, nil
	})))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/fail", bytes.NewReader([]byte("{}"))))

	return rec
}

// The dispatch error log is CLASSIFIED: one record carries the errorfamily
// attrs (family/code/retryable/exit_code + context.* keys), and the LEVEL
// follows the family — Transient at WARN, every server-fault family at ERROR.
// The previous fixed-WARN record hid Corruption/Infrastructure failures at
// warning level and carried no classification at all.
func TestHandleErr_ClassifiedErrorLogging(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantLevel  string
		wantFamily string
	}{
		{
			name: "transient logs warn",
			err: errorfamily.Newf(errorfamily.Transient, "test.flap", "connection flapped").
				WithContext("attempt", "2"),
			wantLevel:  "WARN",
			wantFamily: "transient",
		},
		{
			name: "corruption logs error",
			err:  errorfamily.Newf(errorfamily.Corruption, "test.mangled", "payload checksum mismatch"),
			// Corruption is the data-loss family: it must NOT hide at WARN.
			wantLevel:  "ERROR",
			wantFamily: "corruption",
		},
		{
			name: "rejection logs error per fleet semantics",
			err:  errorfamily.Newf(errorfamily.Rejection, "test.bad_input", "name is empty"),
			// errorfamily's published mapping reserves WARN for retryable
			// families; client faults log ERROR with family="rejection" so
			// consumers can filter by the attr.
			wantLevel:  "ERROR",
			wantFamily: "rejection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			prev := slog.Default()

			slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
			t.Cleanup(func() { slog.SetDefault(prev) })

			rec := dispatchFailureApp(t, tt.err)
			if rec.Code == http.StatusInternalServerError && buf.Len() == 0 {
				t.Fatal("dispatch produced 500 but nothing was logged")
			}

			var record capturedRecord
			if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
				t.Fatalf("decode log record: %v\nraw: %s", err, buf.String())
			}

			if record.Level != tt.wantLevel {
				t.Errorf("level = %s, want %s (family %s)", record.Level, tt.wantLevel, tt.wantFamily)
			}

			if record.Family != tt.wantFamily {
				t.Errorf("family attr = %q, want %q", record.Family, tt.wantFamily)
			}

			if record.Msg == "" {
				t.Error("record message is empty — want the error text")
			}

			if record.Method != http.MethodPost || record.Path != "/fail" {
				t.Errorf("method/path attrs = %s %s, want POST /fail", record.Method, record.Path)
			}
		})
	}
}

// The retryable attr distinguishes self-healing families, and errorfamily
// context keys ride as context.* attrs on the same record.
func TestHandleErr_ClassifiedAttrsCarryContext(t *testing.T) {
	var buf bytes.Buffer

	prev := slog.Default()

	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	transient := errorfamily.Newf(errorfamily.Transient, "test.flap", "connection flapped").
		WithContext("pool_size", "7")
	dispatchFailureApp(t, transient)

	var record capturedRecord
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("decode log record: %v\nraw: %s", err, buf.String())
	}

	if !record.Retryable {
		t.Error("retryable attr = false, want true for Transient")
	}
	// The code attr carries the dispatch layer's wrapper code; the domain
	// code stays visible in the message chain.
	if record.Code != "cqrshtmx.dispatch.command_failed" {
		t.Errorf("code attr = %q, want the dispatch wrapper code", record.Code)
	}

	if !strings.Contains(record.Msg, "test.flap") {
		t.Errorf("message %q does not surface the inner domain code test.flap", record.Msg)
	}

	if record.ContextPoolSize != "7" {
		t.Errorf(
			"context.pool_size attr = %q, want 7 — the dispatch wrap must mirror inner context",
			record.ContextPoolSize,
		)
	}

	if record.ExitCode == 0 {
		t.Error("exit_code attr missing (0) — want errorfamily's mapping")
	}
}

// Non-errorfamily errors (plain wrapped errors from a consumer handler) still
// classify: errorfamily.Classify falls back to a family instead of skipping
// the record.
func TestHandleErr_PlainErrorsStillClassify(t *testing.T) {
	var buf bytes.Buffer

	prev := slog.Default()

	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	plain := errorfamily.Wrapf(errTestPlain, errorfamily.Infrastructure, "test.plain", "wrapped: %s", "boom")
	rec := dispatchFailureApp(t, plain)

	var record capturedRecord
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("decode log record: %v\nraw: %s", err, buf.String())
	}

	if rec.Code == http.StatusInternalServerError && record.Family == "" {
		t.Error("plain error produced no family attr")
	}
}

var errTestPlain = errorfamily.New(errorfamily.Rejection, "test.plain_inner", "inner")
