package dashboardui

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// seedTypedEvents appends one event per type to the store's journal (per-step
// saves — a single bulk publish strands deliveries, the tenant-cascade lesson)
// and returns the dashboard mux over it.
func seedTypedEvents(t *testing.T, types ...event.Type) *http.ServeMux {
	t.Helper()

	store := memorystorage.NewMemoryStore()

	for _, evtType := range types {
		aggID := id.NewStreamID()
		evt, _ := event.New(evtType, aggID, "User", event.Version(1), struct{}{})

		if err := store.Save(
			context.Background(),
			id.NewStreamRef("User", aggID),
			[]event.Event{evt},
			event.Version(0),
		); err != nil {
			t.Fatalf("seed %s: %v", evtType, err)
		}
	}

	return mustTestDashboardMuxWithConfig(t, Config{EventSource: store, Journal: store})
}

// eventsSearchRung drives one rung of the events-search fallback ladder: a
// request with the given method, the urlencoded criteria body, and optional
// override headers. HX-Request is set so the response is the HTMX partial the
// filter form's swap expects.
func eventsSearchRung(
	t *testing.T,
	mux *http.ServeMux,
	method, body string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, "/dashboard/events/search", strings.NewReader(body))
	req.Header.Set("Hx-Request", "true")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	return rec
}

// TestEventsSearch_LadderParity pins the fallback ladder's core contract for
// the events filter: the native QUERY rung, the POST fallback twin, and the
// POST + X-HTTP-Method-Override rung (rewritten to QUERY by the dashboard's
// MethodOverrideMiddleware) answer ONE handler and therefore byte-identical
// fragments — docs/guides/http-query-method.md.
func TestEventsSearch_LadderParity(t *testing.T) {
	t.Parallel()

	mux := seedTypedEvents(t, "Alpha", "Beta")

	rungs := map[string]*httptest.ResponseRecorder{
		"native QUERY": eventsSearchRung(t, mux, cqrshtmx.MethodQuery, "type=Alpha", nil),
		"POST twin":    eventsSearchRung(t, mux, http.MethodPost, "type=Alpha", nil),
		"override POST (rung 2)": eventsSearchRung(
			t,
			mux,
			http.MethodPost,
			"type=Alpha",
			map[string]string{"X-HTTP-Method-Override": "QUERY"},
		),
	}

	golden := rungs["native QUERY"]
	if golden.Code != http.StatusOK {
		t.Fatalf("native QUERY status = %d, want 200; body=%s", golden.Code, golden.Body.String())
	}
	if !strings.Contains(golden.Body.String(), "Alpha") {
		t.Errorf("filtered page must keep the Alpha row; body=%s", golden.Body.String())
	}

	if strings.Contains(golden.Body.String(), "Beta") {
		t.Error("filtered page must exclude the Beta row")
	}

	if !strings.Contains(golden.Body.String(), `id="main-content"`) {
		t.Errorf("HTMX partial response must carry the main-content region; body=%s", golden.Body.String())
	}

	for name, rung := range rungs {
		if rung.Code != golden.Code {
			t.Errorf("%s: status = %d, want %d", name, rung.Code, golden.Code)
		}

		if !bytes.Equal(rung.Body.Bytes(), golden.Body.Bytes()) {
			t.Errorf("%s: fragment diverges from the native QUERY rung (byte-parity contract)", name)
		}
	}
}

// TestEventsSearch_InvalidBodyIs400 pins the search rungs' only client-fault
// path: a body over the cap or unparseable as a urlencoded form answers 400
// (the same refuse-don't-guess posture as the GET form's decoder).
func TestEventsSearch_InvalidBodyIs400(t *testing.T) {
	t.Parallel()

	mux := seedTypedEvents(t, "Alpha")

	oversized := strings.Repeat("x", maxEventFilterBody+1)
	if rec := eventsSearchRung(t, mux, cqrshtmx.MethodQuery, oversized, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("oversized body: status = %d, want 400", rec.Code)
	}

	// url.ParseQuery refuses a bare '%' escape; the rung must reject, not
	// silently filter on an empty value.
	if rec := eventsSearchRung(t, mux, cqrshtmx.MethodQuery, "%zz", nil); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed body: status = %d, want 400", rec.Code)
	}
}

// TestCQRSQueryExtAssetServed pins the asset row the filter form's hx-ext
// relies on: the extension script serves under the dashboard's asset prefix.
func TestCQRSQueryExtAssetServed(t *testing.T) {
	t.Parallel()

	mux := newTestDashboardMux(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/ext/cqrs-query.js", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("cqrs-query.js status = %d, want 200", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("cqrs-query.js Content-Type = %q, want a javascript type", ct)
	}

	if !strings.Contains(rec.Body.String(), "cqrs-query") {
		t.Error("served script is not the cqrs-query extension")
	}
}
