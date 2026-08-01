package cqrshtmx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justinas/nosurf"
)

func okHandler200() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

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

func TestCSRFTokenFromRequestWithNosurfToken(t *testing.T) {
	t.Parallel()

	handler := nosurf.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := csrfTokenFromRequest(r)
		if token == "" {
			t.Error("csrfTokenFromRequest returned empty when nosurf token is set")
		}

		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)
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

func TestConfigureNosurfHandlerWithDomain(t *testing.T) {
	t.Parallel()

	handler := nosurf.New(okHandler200())
	cfg := CSRFConfig{
		Domain:   "example.com",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	configureNosurfHandler(handler, cfg)

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected a cookie to be set")
	}

	if cookies[0].Domain != "example.com" {
		t.Errorf("cookie domain = %q, want %q", cookies[0].Domain, "example.com")
	}
}

func TestConfigureNosurfHandlerWithTrustedOrigins(t *testing.T) {
	t.Parallel()

	handler := nosurf.New(okHandler200())
	cfg := CSRFConfig{
		Secure:         true,
		TrustedOrigins: []string{"https://example.com"},
	}
	configureNosurfHandler(handler, cfg)

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected a cookie to be set")
	}
}

func TestConfigureNosurfHandlerWithErrorHandler(t *testing.T) {
	t.Parallel()

	customCalled := false
	handler := nosurf.New(okHandler200())
	cfg := CSRFConfig{
		ErrorHandler: func(http.ResponseWriter, *http.Request, error) {
			customCalled = true
		},
	}
	configureNosurfHandler(handler, cfg)

	// POST without token to trigger failure handler
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	handler.ServeHTTP(w, r)

	if !customCalled {
		t.Error("expected custom error handler to be called")
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
