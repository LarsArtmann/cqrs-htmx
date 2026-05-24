package cqrshtmx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/csrf"
)

func TestAuthModeString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		mode authMode
		want string
	}{
		{authNone, "none"},
		{authRequired, "required"},
		{authAuthorized, "authorized"},
		{authMode(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.mode.String(); got != tc.want {
			t.Errorf("authMode(%d).String() = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

func TestCSRFTokenFromRequestWithGorillaToken(t *testing.T) {
	t.Parallel()
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}
	mw := csrf.Protect(secret, csrf.CookieName("csrf_test"))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := csrfTokenFromRequest(r)
		if token == "" {
			t.Error("csrfTokenFromRequest returned empty when gorilla token is set")
		}
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	mw(inner).ServeHTTP(w, r)
}

func TestCSRFTokenFromRequestFallback(t *testing.T) {
	t.Parallel()
	ctx := WithCSRFToken(context.Background(), "fallback-token")
	r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	token := csrfTokenFromRequest(r)
	if token != "fallback-token" {
		t.Errorf("csrfTokenFromRequest fallback = %q, want %q", token, "fallback-token")
	}
}

func TestSameSiteDefaultCase(t *testing.T) {
	t.Parallel()
	cfg := CSRFConfig{SameSite: http.SameSite(99)}
	result := cfg.sameSite()
	if result != csrf.SameSiteLaxMode {
		t.Errorf("sameSite(unknown) = %v, want Lax", result)
	}
}

func TestBuildGorillaOptionsWithDomain(t *testing.T) {
	t.Parallel()
	cfg := CSRFConfig{
		Secret:   make([]byte, 32),
		Domain:   "example.com",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	opts := buildGorillaOptions(cfg)
	if len(opts) < 9 {
		t.Errorf("buildGorillaOptions with domain returned %d opts, want >= 9", len(opts))
	}
}

func TestBuildGorillaOptionsWithTrustedOrigins(t *testing.T) {
	t.Parallel()
	cfg := CSRFConfig{
		Secret:         make([]byte, 32),
		Secure:         true,
		TrustedOrigins: []string{"https://example.com"},
	}
	opts := buildGorillaOptions(cfg)
	if len(opts) < 9 {
		t.Errorf("buildGorillaOptions with trusted origins returned %d opts, want >= 9", len(opts))
	}
}

func TestBuildGorillaOptionsWithErrorHandler(t *testing.T) {
	t.Parallel()
	cfg := CSRFConfig{
		Secret:       make([]byte, 32),
		ErrorHandler: func(http.ResponseWriter, *http.Request, error) {},
	}
	opts := buildGorillaOptions(cfg)
	if len(opts) < 9 {
		t.Errorf("buildGorillaOptions with error handler returned %d opts, want >= 9", len(opts))
	}
}

func TestEvictionHeapPushNonPtr(t *testing.T) {
	t.Parallel()
	h := &evictionHeap{}
	h.Push("not a pointer")
	if h.Len() != 0 {
		t.Errorf("Push(non-pointer) should be ignored, got len=%d", h.Len())
	}
}
