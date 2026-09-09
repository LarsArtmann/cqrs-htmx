package setup_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/httputil"
)

// Tests in this file cover the Config.CSRF knob:
//
//   - A tokenless mutation against a mounted admin route is rejected with 403.
//   - A custom CSRF config flows through to the issued cookie (Secure flag,
//     cookie name) — the escape hatch for HTTPS deployments.
//   - The default (CSRF nil) preserves the historical zero-config behavior:
//     a cookie without the Secure flag (fine for local HTTP dev).

func TestCSRF_TokenlessMutationRejected(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{Title: "CSRF Rejection Test"})
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)

	srv := httptest.NewServer(bundle.Middleware()(mux))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/admin/users", "application/x-www-form-urlencoded", strings.NewReader("name=x"))
	if err != nil {
		t.Fatalf("POST /admin/users: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("tokenless POST /admin/users: got status %d, want 403 Forbidden; body: %s", resp.StatusCode, body)
	}
}

func TestCSRFConfig_FlowsToMiddleware(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title: "CSRF Config Test",
		CSRF:  &httputil.CSRFConfig{Secure: true, CookieName: "csrf_test"},
	})
	defer func() { _ = bundle.Close() }()

	srv := httptest.NewServer(bundle.CSRFMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET through CSRF middleware: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var csrfCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "csrf_test" {
			csrfCookie = c

			break
		}
	}
	if csrfCookie == nil {
		t.Fatalf("custom CSRF cookie %q not issued; cookies: %v", "csrf_test", resp.Cookies())
	}
	if !csrfCookie.Secure {
		t.Error("custom CSRF config with Secure: true must issue a Secure-flagged cookie")
	}
}

func TestCSRFMiddleware_DefaultPreservesPlainCookie(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{Title: "CSRF Default Test"})
	defer func() { _ = bundle.Close() }()

	srv := httptest.NewServer(bundle.CSRFMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET through default CSRF middleware: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var csrfCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == httputil.DefaultCSRFCookieName {
			csrfCookie = c

			break
		}
	}
	if csrfCookie == nil {
		t.Fatalf("default CSRF cookie %q not issued; cookies: %v", httputil.DefaultCSRFCookieName, resp.Cookies())
	}
	if csrfCookie.Secure {
		t.Error("default CSRF cookie must stay un-secure (default-preserving knob; set Config.CSRF for HTTPS)")
	}
}
