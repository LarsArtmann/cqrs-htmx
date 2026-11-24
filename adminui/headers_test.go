package adminui

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// Header assertions for what the panel's own middleware chain
// ([Handler.Middleware] → cqrshtmx.RecommendedSecurityMiddlewareWithCSP)
// actually emits on every response class: an HTML page, an HTMX partial, and
// a static asset. The truth asserted here:
//
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: DENY
//   - Referrer-Policy: strict-origin-when-cross-origin
//   - Permissions-Policy: the recommended feature lockdown
//   - Content-Security-Policy: self + per-request nonce + img-src data:, with
//     the SAME nonce the page markup embeds in its inline <style>/<script>
//     tags
//   - Strict-Transport-Security: NOT set by the panel's own chain (the
//     recommended stack leaves TLS termination policy to the deployment) —
//     asserted absent so the docs stay true.
//
// The chain is the panel's own Middleware() wrapped around Handler() — the
// exact wiring the README recommends; a consumer adding HSTS at their edge
// is outside this test's contract.

// headersTestPanel builds a panel with a super-admin session injected and the
// panel's own security middleware outermost.
func headersTestPanel(t *testing.T) http.Handler {
	t.Helper()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	admin := identitymodel.NewUser(identitymodel.SyntheticUserID("01HADMINUSER"), "admin@example.com", "")
	if err := svc.GrantRoles(t.Context(),
		identitymodel.ActorIDFromUser(admin.ID),
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
		[]identitymodel.Role{identitymodel.RoleSuperAdmin},
	); err != nil {
		t.Fatalf("GrantRoles: %v", err)
	}

	panel, err := New(Config{Service: svc})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return panel.Middleware()(userInject(panel.Handler(), admin))
}

// assertSecurityHeaders asserts the header truth on one recorded response
// and returns the CSP nonce ("" when the response carries no nonce CSP).
func assertSecurityHeaders(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	headers := rec.Header()
	if got := headers.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := headers.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}
	if got := headers.Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Errorf("Referrer-Policy = %q, want strict-origin-when-cross-origin", got)
	}
	if got := headers.Get("Permissions-Policy"); !strings.Contains(got, "geolocation=()") {
		t.Errorf("Permissions-Policy = %q, want the recommended feature lockdown", got)
	}

	csp := headers.Get("Content-Security-Policy")
	nonce := cspNonce(csp)
	if nonce == "" {
		t.Fatalf("Content-Security-Policy = %q, want a per-request nonce", csp)
	}
	for _, fragment := range []string{"default-src 'self'", "img-src 'self' data:"} {
		if !strings.Contains(csp, fragment) {
			t.Errorf("CSP missing %q: %q", fragment, csp)
		}
	}

	if got := headers.Get("Strict-Transport-Security"); got != "" {
		t.Errorf(
			"Strict-Transport-Security = %q: the panel's own chain does not set HSTS (TLS policy belongs to the deployment edge); update this pin and the README if that changes",
			got,
		)
	}

	return nonce
}

// cspNonce extracts the nonce value from a nonce-bearing CSP header.
func cspNonce(csp string) string {
	match := regexp.MustCompile(`'nonce-([^']+)'`).FindStringSubmatch(csp)
	if len(match) < 2 {
		return ""
	}

	return match[1]
}

// TestHeaders_OnEveryResponseClass asserts the security header truth for the
// three response classes the panel serves: an HTML page, an HTMX partial,
// and a static asset.
func TestHeaders_OnEveryResponseClass(t *testing.T) {
	t.Parallel()

	handler := headersTestPanel(t)

	classes := []struct {
		name string
		path string
	}{
		{"page", "/"},
		{"partial", "/partials/stats"},
		{"asset", "/-/admin.js"},
	}

	for _, class := range classes {
		t.Run(class.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, class.path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("%s answered %d, want 200", class.path, rec.Code)
			}

			nonce := assertSecurityHeaders(t, rec)

			// The nonce is not just in the header: the PAGE markup consumes
			// it for its inline <style>/<script> tags. Partials are markup
			// fragments with no inline script of their own and assets have
			// no markup at all, so neither embeds it — that is the truth.
			if class.name == "page" && !strings.Contains(rec.Body.String(), `nonce="`+nonce+`"`) {
				t.Errorf("%s markup does not embed the CSP nonce %q", class.name, nonce)
			}
		})
	}
}

// TestHeaders_NoncePerRequest pins that each request gets a FRESH nonce —
// a reused nonce widens cache-based nonce reuse attacks.
func TestHeaders_NoncePerRequest(t *testing.T) {
	t.Parallel()

	handler := headersTestPanel(t)

	first := cspNonce(serveCSP(t, handler))
	second := cspNonce(serveCSP(t, handler))
	if first == "" || second == "" {
		t.Fatalf("nonce missing: %q / %q", first, second)
	}
	if first == second {
		t.Fatal("two requests carried the same CSP nonce")
	}
}

func serveCSP(t *testing.T, handler http.Handler) string {
	t.Helper()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard answered %d, want 200", rec.Code)
	}

	return rec.Header().Get("Content-Security-Policy")
}
