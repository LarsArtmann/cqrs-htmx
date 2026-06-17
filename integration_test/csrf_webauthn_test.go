package integration_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v2"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
)

// TestCSRF_WithAuthHandler demonstrates wiring CSRF protection around the
// usermgmt AuthHandler. The root module's CSRFMiddleware composes cleanly with
// usermgmt's SessionMiddleware via Chain. This is the documented recipe for
// protecting WebAuthn and other state-changing endpoints from CSRF.
func TestCSRF_WithAuthHandler(t *testing.T) {
	t.Parallel()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	secure := false
	handler := usermgmt.NewAuthHandler(svc, usermgmt.HandlerConfig{Secure: &secure})
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Compose CSRF (outermost) with session enrichment.
	wrapped := cqrshtmx.Chain(
		cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
		usermgmt.NewSessionMiddleware(svc, "session_token"),
	)(mux)

	// A POST without a CSRF token must be rejected (403 Forbidden) before
	// reaching the AuthHandler — proving CSRF protection is wired and active.
	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/register",
		strings.NewReader(`{"id":"csrf1","email":"csrf@test.com"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("CSRF should block tokenless POST, got %d", w.Code)
	}

	// Safe methods (GET) pass through CSRF and reach the handler — here returning
	// 401 (no session) rather than 403, confirming CSRF does not block them.
	getReq := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	getW := httptest.NewRecorder()
	wrapped.ServeHTTP(getW, getReq)

	if getW.Code == http.StatusForbidden {
		t.Errorf("CSRF should not block safe GET, got 403")
	}

	// A CSRF cookie is issued so the client can read the token and send it back
	// (as the X-CSRF-Token header) on subsequent state-changing requests.
	var hasTokenCookie bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "csrf_token" {
			hasTokenCookie = true
		}
	}
	if !hasTokenCookie {
		t.Error("expected a csrf_token cookie to be set for the client")
	}
}
