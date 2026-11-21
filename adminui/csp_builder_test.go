package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// Config.CSPBuilder feeds Handler.Middleware's CSP: a set builder changes only
// the Content-Security-Policy header, a nil builder keeps the recommended
// default (the img-src data: allowance is its fingerprint).
func TestMiddleware_CSPBuilderOverridesPolicy(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatal(err)
	}

	panel, err := New(Config{
		Service:    svc,
		Authorizer: RequireAuthenticated(),
		CSPBuilder: func(nonce string) string {
			return "default-src 'self'; script-src 'self' 'nonce-" + nonce + "'; style-src 'self'"
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	panel.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "style-src 'self'") {
		t.Errorf("custom builder's style-src missing from CSP: %q", csp)
	}

	if strings.Contains(csp, "img-src") {
		t.Errorf("custom CSP leaked the default's img-src allowance: %q", csp)
	}

	defaultPanel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}

	rec2 := httptest.NewRecorder()
	defaultPanel.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/", nil))

	defaultCSP := rec2.Header().Get("Content-Security-Policy")
	if !strings.Contains(defaultCSP, "img-src 'self' data:") {
		t.Errorf("nil builder lost the default CSP fingerprint: %q", defaultCSP)
	}
}
