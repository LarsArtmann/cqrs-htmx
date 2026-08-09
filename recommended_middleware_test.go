package cqrshtmx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/httputil"
)

func TestRecommendedSecurityMiddleware_SetsAllHeaders(t *testing.T) {
	mw := RecommendedSecurityMiddleware()

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

	pp := rec.Header().Get("Permissions-Policy")
	if pp == "" {
		t.Fatal("expected Permissions-Policy header")
	}
	for _, want := range []string{"geolocation=()", "microphone=()", "camera=()", "payment=()", "usb=()"} {
		if !strings.Contains(pp, want) {
			t.Errorf("Permissions-Policy missing %q, got %q", want, pp)
		}
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "nonce-") {
		t.Errorf("expected CSP to contain a nonce, got %q", csp)
	}
	if !strings.Contains(csp, "'self'") {
		t.Errorf("expected CSP to allow 'self', got %q", csp)
	}
}

func TestRecommendedSecurityMiddleware_NonceAvailableInContext(t *testing.T) {
	mw := RecommendedSecurityMiddleware()

	var nonceFromCtx string
	handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nonceFromCtx = httputil.NonceFromRequest(r)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if nonceFromCtx == "" {
		t.Error("expected nonce to be available in request context via httputil.NonceFromRequest")
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, nonceFromCtx) {
		t.Errorf("CSP header should contain the nonce value %q, got %q", nonceFromCtx, csp)
	}
}

func TestRecommendedSecurityMiddleware_RecoversFromPanic(t *testing.T) {
	mw := RecommendedSecurityMiddleware()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	mw(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d (panic should be recovered)", rec.Code, http.StatusInternalServerError)
	}
}

func TestRegisterErrorClassifications_Idempotent(t *testing.T) {
	RegisterErrorClassifications()
	RegisterErrorClassifications()
	RegisterErrorClassifications()

	if status := errorfamily.Classify(http.ErrNotSupported).HTTPStatus(); status != http.StatusServiceUnavailable {
		t.Errorf("classification status = %d, want %d", status, http.StatusServiceUnavailable)
	}
}
