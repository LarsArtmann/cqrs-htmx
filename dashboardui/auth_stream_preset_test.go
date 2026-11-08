package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4/core"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// The Auth stream preset (2026-11-08 harvest): the events panel gets a
// one-click session-lifecycle filter riding comma-separated type matching.
// This test pins the three halves that could drift independently: the
// preset link on the page, the multi-type matching in core, and the
// end-to-end behavior through the handler.
func TestDashboard_AuthStreamPreset(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	aggID := id.NewStreamID()

	events := make([]event.Event, 0, 3)

	for i, evtType := range []string{"LoginSucceeded", "SessionEnded", "user.created"} {
		evt, err := event.New(event.Type(evtType), aggID, "User", event.Version(i+1), struct{ Name string }{Name: "Alice"})
		if err != nil {
			t.Fatalf("event.New(%s): %v", evtType, err)
		}

		events = append(events, evt)
	}

	if err := store.Save(context.Background(), id.NewStreamRef("User", aggID), events, event.Version(0)); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	mux := mustTestDashboardMuxWithConfig(t, Config{EventSource: store, Journal: store})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/events", nil)
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range strings.Split(core.AuthStreamTypes, ",") {
		if !strings.Contains(body, want) {
			t.Fatalf("unfiltered page should carry the Auth stream preset naming %q (preset href missing?)", want)
		}
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/dashboard/events?type="+core.AuthStreamTypes, nil)
	mux.ServeHTTP(rec2, req2)

	filtered := rec2.Body.String()
	if rec2.Code != http.StatusOK {
		t.Fatalf("filtered page status = %d; body=%s", rec2.Code, filtered)
	}

	if !strings.Contains(filtered, "LoginSucceeded") || !strings.Contains(filtered, "SessionEnded") {
		t.Error("auth-stream page should show both session lifecycle events")
	}

	if strings.Contains(filtered, "user.created") {
		t.Error("auth-stream page should NOT show non-auth events")
	}
}
