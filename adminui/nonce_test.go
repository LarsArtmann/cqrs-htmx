package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/httputil"
)

func TestNonce_FallbackToNonceFromRequest(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatal(err)
	}

	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}

	mw := panel.Middleware()

	var captured string
	handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = panel.nonce(r)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if captured == "" {
		t.Error("expected nonce to be non-empty when NonceFunc is nil and httputil.Nonce middleware is in the chain")
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if captured != "" && !strings.Contains(csp, captured) {
		t.Errorf("CSP header should contain nonce %q, got %q", captured, csp)
	}
}

func TestNonce_NonceFuncTakesPrecedence(t *testing.T) {
	custom := "custom-nonce-from-func"

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatal(err)
	}

	panel, err := New(Config{
		Service:    svc,
		Authorizer: RequireAuthenticated(),
		NonceFunc:  func(*http.Request) string { return custom },
	})
	if err != nil {
		t.Fatal(err)
	}

	mw := httputil.Nonce(httputil.DefaultNonceConfig())

	var captured string
	handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = panel.nonce(r)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if captured != custom {
		t.Errorf("nonce = %q, want %q (NonceFunc should take precedence over NonceFromRequest)", captured, custom)
	}
}

func TestNonce_EmptyWithoutMiddlewareOrFunc(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatal(err)
	}

	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := panel.nonce(req); got != "" {
		t.Errorf("nonce without middleware or NonceFunc = %q, want empty", got)
	}
}

func TestMiddleware_SetsSecurityHeaders(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatal(err)
	}

	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}

	mw := panel.Middleware()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	for _, tc := range []struct{ header, want string }{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	} {
		if got := rec.Header().Get(tc.header); got != tc.want {
			t.Errorf("%s = %q, want %q", tc.header, got, tc.want)
		}
	}
}

