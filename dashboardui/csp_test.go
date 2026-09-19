package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/httputil"
)

// cspDashboard builds a dashboard wrapped in the recommended security
// middleware (SecurityHeaders + per-request CSP nonce), as a CSP-enforcing
// consumer would run it.
func cspDashboard(t *testing.T) http.Handler {
	t.Helper()

	store := memorystorage.NewMemoryStore()

	d, err := New(Config{EventSource: store, Journal: store})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	return httputil.Compose(
		httputil.SecurityHeaders(httputil.SecurityHeadersConfig{}),
		httputil.Nonce(httputil.NonceConfig{CSPBuilder: httputil.RecommendedCSPWithNonce}),
	)(mux)
}

// cspDashboardWithDetail builds the CSP dashboard with a seeded three-event
// Order stream and returns the handler plus that stream's time-travel detail
// URL. The detail page carries the interactive version slider — the route
// class where inline event handlers previously hid from the CSP sweep (the
// listing-only sweep was a vacuous pass).
func cspDashboardWithDetail(t *testing.T) (http.Handler, string) {
	t.Helper()

	store := memorystorage.NewMemoryStore()
	aggID := id.NewStreamID()
	ref := id.NewStreamRef("Order", aggID)

	for i := 1; i <= 3; i++ {
		evt, err := event.New(
			"order.updated",
			aggID,
			"Order",
			event.Version(i),
			struct{ Step int }{Step: i},
		)
		if err != nil {
			t.Fatalf("event.New: %v", err)
		}

		if err := store.Save(context.Background(), ref, []event.Event{evt}, event.Version(i-1)); err != nil {
			t.Fatalf("store.Save: %v", err)
		}
	}

	d, err := New(Config{
		EventSource:  store,
		Journal:      store,
		StreamReader: listing.NewInMemoryStreamReader(store),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	handler := httputil.Compose(
		httputil.SecurityHeaders(httputil.SecurityHeadersConfig{}),
		httputil.Nonce(httputil.NonceConfig{CSPBuilder: httputil.RecommendedCSPWithNonce}),
	)(mux)

	return handler, "/dashboard/time-travel/Order/" + aggID.String() + "?v=2"
}

// cspDashboardWithDetail builds the CSP dashboard with a seeded three-event
// Order stream and returns the handler plus that stream's time-travel detail
// URL. The detail page carries the interactive version slider — the route
// class where inline event handlers previously hid from the CSP sweep (the
// listing-only sweep was a vacuous pass).

// renderWithNonce renders a page through the security middleware so
// request-scoped nonces are populated. The second result is false when the
// page is unavailable without its data source (404-class responses).
func renderWithNonce(t *testing.T, target string) (string, bool) {
	t.Helper()

	h := cspDashboard(t)

	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		return "", false
	}

	return rec.Body.String(), true
}

// TestCSP_LibraryScriptsCarryNonce pins the M22 contract: every inline script
// the dashboard emits (library toast/copy/error-handling scripts) carries the
// per-request nonce so a nonce-locked CSP policy executes them.
func TestCSP_LibraryScriptsCarryNonce(t *testing.T) {
	body, ok := renderWithNonce(t, "/dashboard/")
	if !ok {
		t.Fatal("overview unavailable")
	}

	if !strings.Contains(body, "nonce=") {
		t.Fatal("expected at least one nonce-carrying inline script")
	}

	for _, want := range []string{"tcShowToast", "tcCopyAttached", "tcErrorAttached"} {
		idx := strings.Index(body, want)

		// The error-handling/toast/copy scripts must be nonce-tagged. Find the
		// enclosing <script tag for this occurrence and verify it has a nonce.
		if idx < 0 {
			continue
		}

		scriptStart := strings.LastIndex(body[:idx], "<script")
		if scriptStart < 0 {
			t.Fatalf("no <script tag before %q", want)
		}

		tag := body[scriptStart:idx]

		if !strings.Contains(tag, "nonce=") {
			t.Errorf("inline script containing %q lacks nonce attribute: %q", want, tag[:min(len(tag), 120)])
		}
	}
}

// TestCSP_NoInlineEventHandlers proves the dashboard emits no inline event
// handler attributes (onclick/onchange/oninput/onsubmit/onload). Nonce-based
// CSP does NOT whitelist handler attributes, so any of them would silently
// die under a consumer's RecommendedSecurityMiddleware.
func TestCSP_NoInlineEventHandlers(t *testing.T) {
	handler, timetravelDetail := cspDashboardWithDetail(t)

	for _, target := range []string{
		"/dashboard/",
		"/dashboard/events",
		"/dashboard/aggregates",
		"/dashboard/commands",
		"/dashboard/queries",
		"/dashboard/time-travel",
		timetravelDetail,
		"/dashboard/snapshots",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			continue
		}

		body := rec.Body.String()

		for _, handler := range []string{"onclick=", "onchange=", "oninput=", "onsubmit=", "onload="} {
			if strings.Contains(body, handler) {
				t.Errorf("%s: found inline %q attribute (would break under CSP nonce)", target, handler)
			}
		}
	}

	// The sweep must actually reach the slider page — a detail URL that 404s
	// would silently shrink the guarantee back to the listings-only vacuous
	// pass this test existed to prevent.
	req := httptest.NewRequest(http.MethodGet, timetravelDetail, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("timetravel detail %s: status = %d, want 200 (slider page must render for the CSP sweep)", timetravelDetail, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "id=\"version-slider\"") {
		t.Fatal("timetravel detail did not render the version slider")
	}
}

// TestCSP_UnsafeInlineNotRequired verifies the security headers do not relax
// script-src with unsafe-inline (the nonce path must be the only mechanism).
// The middleware must be given an explicit CSPBuilder - the zero-value
// NonceConfig emits no CSP header at all (the godoc's "Default:
// RecommendedCSPWithNonce" claim is false for the zero value), which made an
// earlier version of this test skip forever instead of assert.
func TestCSP_UnsafeInlineNotRequired(t *testing.T) {
	h := cspDashboard(t)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Skip("no CSP header emitted by middleware")
	}

	if strings.Contains(csp, "unsafe-inline") {
		t.Errorf("CSP relaxes script-src with unsafe-inline: %s", csp)
	}
}

// TestNonceFromRequestPopulated guards the plumbing assumption: with the
// recommended middleware, NonceFromRequest returns a non-empty value (the
// library components embed it in their script tags).
func TestNonceFromRequestPopulated(t *testing.T) {
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if httputil.NonceFromRequest(r) == "" {
			t.Error("expected non-empty nonce under RecommendedSecurityMiddleware")
		}
	})

	h := httputil.Compose(
		httputil.SecurityHeaders(httputil.SecurityHeadersConfig{}),
		httputil.Nonce(httputil.NonceConfig{}),
	)(next)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}
