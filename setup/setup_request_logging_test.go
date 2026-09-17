package setup_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
)

// M5 of the 2026-09-17 gap-bundle plan: Config.RequestLogging composes
// cqrshtmx.RequestLoggingSlog outermost in Bundle.Middleware — one access-log
// line per request when set, byte-identical silence when nil.

// TestMiddleware_RequestLoggingCapturesEachRequest drives the full
// Handler()-wrapped stack (Mount + Middleware) and asserts exactly one access
// line per request carrying method, path, and status.
func TestMiddleware_RequestLoggingCapturesEachRequest(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	bundle := setup.MustNew(setup.Config{Title: "Access Logs", RequestLogging: logger})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	server := httptest.NewServer(bundle.Handler(mux))
	defer server.Close()

	for range 3 {
		resp, err := server.Client().Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	logged := logBuf.String()

	if got := strings.Count(logged, "path=/health"); got != 3 {
		t.Errorf("expected exactly 3 access lines for path=/health, got %d in:\n%s", got, logged)
	}

	if !strings.Contains(logged, "method=GET") || !strings.Contains(logged, "status=200") {
		t.Errorf("access lines must carry method and status, got:\n%s", logged)
	}
}

// TestMiddleware_NilRequestLoggingIsSilent pins the zero value: no logger, no
// access lines — the middleware chain behaves exactly as before the field.
func TestMiddleware_NilRequestLoggingIsSilent(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer

	bundle := setup.MustNew(setup.Config{Title: "Silent"})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	server := httptest.NewServer(bundle.Handler(mux))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	if logBuf.Len() != 0 {
		t.Errorf("nil RequestLogging must not log anything, got:\n%s", logBuf.String())
	}
}
