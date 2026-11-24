package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// --- F174: stats partial ETag / 304 ---

// TestStatsPartial_ETagAnd304 verifies the content-derived ETag round trip:
// the tag the server hands out on the first poll must answer a matching
// If-None-Match with 304 (the poller and the SSE-triggered refresh both hit
// this endpoint; unchanged stats should not be re-transferred), while a
// stale or absent tag gets the full 200.
func TestStatsPartial_ETagAnd304(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/admin/partials/stats", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first stats fetch: status %d", first.Code)
	}
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("stats partial served no ETag")
	}
	if !strings.HasPrefix(etag, `"`) {
		t.Errorf("ETag %q must be a quoted strong validator (RFC 7232)", etag)
	}

	// Exact tag -> 304, and a 304 carries no body.
	rec304 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/partials/stats", nil)
	req.Header.Set("If-None-Match", etag)
	h.ServeHTTP(rec304, req)
	if rec304.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match with served ETag: status %d, want 304", rec304.Code)
	}
	if rec304.Body.Len() != 0 {
		t.Errorf("304 carried a body of %d bytes", rec304.Body.Len())
	}
	if rec304.Header().Get("ETag") != etag {
		t.Errorf("304 ETag = %q, want the served tag %q", rec304.Header().Get("ETag"), etag)
	}

	// Weak-prefixed comparison (RFC 7232 allows W/) still matches.
	recWeak := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/partials/stats", nil)
	req.Header.Set("If-None-Match", "W/"+etag)
	h.ServeHTTP(recWeak, req)
	if recWeak.Code != http.StatusNotModified {
		t.Errorf("weak W/ tag: status %d, want 304", recWeak.Code)
	}

	// Wildcard matches any current representation.
	recStar := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/partials/stats", nil)
	req.Header.Set("If-None-Match", "*")
	h.ServeHTTP(recStar, req)
	if recStar.Code != http.StatusNotModified {
		t.Errorf("If-None-Match *: status %d, want 304", recStar.Code)
	}

	// A stale tag gets the full fragment.
	recStale := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/partials/stats", nil)
	req.Header.Set("If-None-Match", `"stale-tag"`)
	h.ServeHTTP(recStale, req)
	if recStale.Code != http.StatusOK {
		t.Errorf("stale tag: status %d, want 200", recStale.Code)
	}
	if !strings.Contains(recStale.Body.String(), `hx-trigger="every 30s"`) {
		t.Error("200 response lost the PolledRegion fragment")
	}
}

// TestStatsPartial_ETagIgnoresWallClock pins the honesty of the tag: two
// consecutive fetches must produce the SAME ETag even though the "Updated
// HH:MM:SS" footer re-renders with the current time. A wall-clock-derived
// tag would make the 304 path unreachable in practice.
func TestStatsPartial_ETagIgnoresWallClock(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/admin/partials/stats", nil))
	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/admin/partials/stats", nil))

	etag1 := first.Header().Get("ETag")
	etag2 := second.Header().Get("ETag")
	if etag1 == "" || etag2 == "" {
		t.Fatalf("missing ETags: %q / %q", etag1, etag2)
	}
	if etag1 != etag2 {
		t.Errorf("ETag unstable across renders: %q vs %q", etag1, etag2)
	}
}

// TestStripStatsStamp unit-pins the volatile footer replacement.
func TestStripStatsStamp(t *testing.T) {
	in := []byte(`<time class="tabular-nums" datetime="2026-11-24T13:00:01Z">Updated 13:00:01</time>`)
	out := string(stripStatsStamp(in))
	if strings.Contains(out, "13:00:01") {
		t.Errorf("wall clock survived stripping: %s", out)
	}
	if !strings.Contains(out, "<time>Updated</time>") {
		t.Errorf("stable placeholder missing: %s", out)
	}
	if !strings.Contains(string(in), "13:00:01") {
		t.Error("stripStatsStamp must not mutate its input")
	}
}

// TestIfNoneMatchSatisfied table-pins the RFC 7232 comparison shapes.
func TestIfNoneMatchSatisfied(t *testing.T) {
	const etag = `"adminui-stats-abc123"`
	cases := []struct {
		name   string
		header string
		want   bool
	}{
		{"empty header", "", false},
		{"exact", `"adminui-stats-abc123"`, true},
		{"weak prefix", `W/"adminui-stats-abc123"`, true},
		{"list match", `"other", "adminui-stats-abc123"`, true},
		{"list miss", `"other", "another"`, false},
		{"wildcard", "*", true},
		{"stale", `"stale"`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ifNoneMatchSatisfied(tc.header, etag); got != tc.want {
				t.Errorf("ifNoneMatchSatisfied(%q) = %v, want %v", tc.header, got, tc.want)
			}
		})
	}
}

// --- F175: SSE-trigger wiring pins ---

// jsLiveUpdatePins returns the admin.js substrings the SSE-triggered refresh
// depends on: the document-delegated listener for sync-client's
// "cqrshtmx:sync-events" event, the SSEURL seam gate, the stats-region
// selector, and the htmx.ajax refresh path (no inline handlers -> CSP-safe).
// A function, not a package var: the lint discipline keeps the package-level
// surface at zero.
func jsLiveUpdatePins() []string {
	return []string{
		"cqrshtmx:sync-events",
		"data-sse-url",
		`/partials/stats`,
		"htmx.ajax",
		"users-table",
		"tenants-table",
		"members-directory-table",
		"audit-panel",
	}
}

// TestLiveUpdates_JSWiringPinned pins the F175 admin.js block (same pin
// pattern as the slug-suggest JS): if the SSE-trigger wiring is renamed or
// dropped, the served asset fails here instead of silently losing live
// refresh.
func TestLiveUpdates_JSWiringPinned(t *testing.T) {
	js := panelAdminJS(t)
	for _, want := range jsLiveUpdatePins() {
		if !strings.Contains(js, want) {
			t.Errorf("admin.js missing live-update wiring %q", want)
		}
	}
}

// TestLiveUpdates_SSEURLSeam verifies the config seam end to end: SSEURL set
// -> the layout body carries data-sse-url (the seam admin.js gates on and
// the consumer's sync-client uses); unset -> the attribute is absent and the
// panel opens no SSE machinery of its own.
func TestLiveUpdates_SSEURLSeam(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	withSSE := httptest.NewRecorder()
	hSSE, _ := newTestPanel(t, user, Config{
		Service:    svc,
		SSEURL:     "/sync/events",
		Authorizer: RequireAuthenticated(),
	})
	hSSE.ServeHTTP(withSSE, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if !strings.Contains(withSSE.Body.String(), `data-sse-url="/sync/events"`) {
		t.Error("SSEURL set: layout body missing data-sse-url")
	}

	withoutSSE := httptest.NewRecorder()
	hPlain, _ := newTestPanel(t, user)
	hPlain.ServeHTTP(withoutSSE, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if strings.Contains(withoutSSE.Body.String(), "data-sse-url") {
		t.Error("SSEURL unset: data-sse-url must be absent (panel must not imply a live stream)")
	}
}
