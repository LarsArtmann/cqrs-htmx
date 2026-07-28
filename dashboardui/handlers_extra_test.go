package dashboardui

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestDefaultPayloadRenderer_Empty(t *testing.T) {
	out, err := DefaultPayloadRenderer{}.Render(nil, codec.EncodingJSON)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if string(out) != "{}" {
		t.Fatalf("expected {}, got %q", out)
	}
}

func TestDefaultPayloadRenderer_JSON(t *testing.T) {
	out, err := DefaultPayloadRenderer{}.Render([]byte(`{"b":2,"a":1}`), codec.EncodingJSON)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !bytes.Contains(out, []byte(`"a": 1`)) {
		t.Fatalf("expected pretty-printed JSON, got %q", out)
	}
}

func TestDefaultPayloadRenderer_UnknownEncoding(t *testing.T) {
	raw := []byte("raw-bytes")
	out, err := DefaultPayloadRenderer{}.Render(raw, codec.Encoding("custom"))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !bytes.Equal(out, raw) {
		t.Fatalf("expected passthrough, got %q", out)
	}
}

func TestDefaultPayloadRenderer_InvalidJSON(t *testing.T) {
	_, err := DefaultPayloadRenderer{}.Render([]byte("not-json"), codec.EncodingJSON)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRenderPayload_FallbackOnError(t *testing.T) {
	evt, err := event.New(
		event.Type("TestEvent"),
		id.NewStreamID(),
		"test",
		event.Version(1),
		[]byte("raw"),
		event.WithEncoding(codec.Encoding("bad")),
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}
	out := renderPayload(DefaultPayloadRenderer{}, evt)
	if !bytes.Equal(out, []byte("raw")) {
		t.Fatalf("expected fallback to raw payload, got %q", out)
	}
}

func TestCSRFToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("_csrf=abc123"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = r.ParseForm()
	r.Form.Set("_csrf", "abc123")
	if got := csrfToken(r); got != "abc123" {
		t.Fatalf("csrfToken: got %q", got)
	}
}

func TestDLOIndexHandler_Renders(t *testing.T) {
	d := mustTestDashboard(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/dead-letters", nil)
	d.dlqIndexHandler(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Dead-Letter Queue")) {
		t.Fatal("expected DLQ heading")
	}
}

func TestDLQDetailHandler_NoStore(t *testing.T) {
	d := mustTestDashboard(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/dead-letters/proj1", nil)
	r.SetPathValue("projection", "proj1")
	d.dlqDetailHandler(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestProjectionsIndexHandler_NoHost(t *testing.T) {
	d := mustTestDashboard(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/projections", nil)
	d.projectionsIndexHandler(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func mustTestDashboard(t *testing.T) *Dashboard {
	t.Helper()
	d, err := New(Config{
		Journal: &stubJournal{},
	})
	if err != nil {
		t.Fatalf("New Dashboard: %v", err)
	}
	return d
}

type stubJournal struct{}

func (stubJournal) ReadAll(ctx context.Context) ([]event.Event, error) { return nil, nil }

var _ event.Journal = stubJournal{}
var _ id.EventID = id.NewEventID()
