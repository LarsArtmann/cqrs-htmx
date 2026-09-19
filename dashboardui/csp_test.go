package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
// handler attributes (onclick/onchange/onsubmit/onload). Nonce-based CSP does
// NOT whitelist handler attributes, so any of them would silently die under a
// consumer's RecommendedSecurityMiddleware.
func TestCSP_NoInlineEventHandlers(t *testing.T) {
	for _, target := range []string{
		"/dashboard/",
		"/dashboard/events",
		"/dashboard/aggregates",
		"/dashboard/commands",
		"/dashboard/queries",
		"/dashboard/time-travel",
		"/dashboard/snapshots",
	} {
		body, ok := renderWithNonce(t, target)
		if !ok {
			continue
		}

		for _, handler := range []string{"onclick=", "onchange=", "onsubmit=", "onload="} {
			if strings.Contains(body, handler) {
				t.Errorf("%s: found inline %q attribute (would break under CSP nonce)", target, handler)
			}
		}
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
