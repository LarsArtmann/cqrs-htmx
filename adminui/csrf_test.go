package adminui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// TestCSRF_FullStackProvesProtection wires the production middleware chain
// (session → CSRF → panel.Middleware) and proves that:
//  1. A POST WITH a valid CSRF token succeeds (200 + HX-Redirect).
//  2. A POST WITHOUT a CSRF token is rejected (403).
//
// This is the integration test that was missing — all other flow tests bypass
// CSRF and therefore proved nothing about the CSRF wiring.
func TestCSRF_FullStackProvesProtection(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatal(err)
	}
	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}

	// Full stack: session → CSRF → recovery+security → panel.
	sessionMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(usermgmt.WithUser(r.Context(), user)))
		})
	}
	csrfMW := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{})
	stack := sessionMW(csrfMW(panel.Middleware()(http.StripPrefix("/admin", panel.Handler()))))

	// Step 1: GET the dashboard to obtain the CSRF cookie + token.
	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	stack.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET dashboard: status %d", getRec.Code)
	}

	// Extract the CSRF token from the <meta> tag in the rendered page.
	metaToken := extractCSRFToken(getRec.Body.String())
	if metaToken == "" {
		t.Fatal("CSRF meta tag not found in dashboard HTML — CSRFTokenHTMLMeta returned empty")
	}

	// Collect the CSRF cookie set by nosurf.
	var csrfCookie *http.Cookie
	for _, c := range getRec.Result().Cookies() {
		if c.Name == "csrf_token" {
			csrfCookie = c
		}
	}
	if csrfCookie == nil {
		t.Fatal("csrf_token cookie not set by middleware")
	}

	// Step 2: POST with valid token + cookie → should succeed.
	form := url.Values{"name": {"test-csrf"}, "display_name": {"Test CSRF"}}
	postOK := httptest.NewRecorder()
	postReq := httptest.NewRequest(http.MethodPost, "/admin/tenants", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("HX-Request", "true")
	postReq.Header.Set("Sec-Fetch-Site", "same-origin")
	postReq.Header.Set("X-CSRF-Token", metaToken)
	postReq.AddCookie(csrfCookie)
	stack.ServeHTTP(postOK, postReq)

	if postOK.Code == http.StatusForbidden {
		t.Fatal("POST with valid CSRF token was rejected (403) — CSRF wiring is broken")
	}
	if postOK.Header().Get("HX-Redirect") == "" {
		t.Errorf("POST with valid token: expected HX-Redirect, got status %d", postOK.Code)
	}

	// Step 3: POST WITHOUT token → should be rejected (403).
	postBad := httptest.NewRecorder()
	postReq2 := httptest.NewRequest(http.MethodPost, "/admin/tenants", strings.NewReader(form.Encode()))
	postReq2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq2.Header.Set("HX-Request", "true")
	postReq2.Header.Set("Sec-Fetch-Site", "same-origin")
	postReq2.AddCookie(csrfCookie) // cookie present, but no header token
	stack.ServeHTTP(postBad, postReq2)

	if postBad.Code != http.StatusForbidden {
		t.Errorf("POST without CSRF token: status %d, want 403", postBad.Code)
	}
}

// extractCSRFToken pulls the token from <meta name="csrf-token" content="VALUE">.
func extractCSRFToken(htmlBody string) string {
	idx := strings.Index(htmlBody, `name="csrf-token"`)
	if idx < 0 {
		return ""
	}
	contentStart := strings.Index(htmlBody[idx:], `content="`)
	if contentStart < 0 {
		return ""
	}
	contentStart += idx + len(`content="`)
	end := strings.Index(htmlBody[contentStart:], `"`)
	if end < 0 {
		return ""
	}
	return htmlBody[contentStart : contentStart+end]
}
