package setup_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/httputil"
)

// Tests in this file cover the Config.NonceFunc seam: a consumer whose CSP
// nonce does not come from httputil.Nonce (a custom context key, an upstream
// header, a generator they own) must be able to hand it to the panels instead
// of rebuilding the panel configuration by hand.

func renderLoginPage(t *testing.T, cfg setup.Config) *httptest.ResponseRecorder {
	t.Helper()

	bundle, err := setup.New(cfg)
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	if bundle.Login == nil {
		t.Fatal("login page should be mounted")
	}

	mux := http.NewServeMux()
	mux.Handle("/login", bundle.Login)

	rec := httptest.NewRecorder()
	bundle.Middleware()(mux).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /login: status %d, want 200", rec.Code)
	}

	return rec
}

func TestNonceFunc_ReachesLoginPage(t *testing.T) {
	t.Parallel()

	const wantNonce = "custom-nonce-abc123"

	rec := renderLoginPage(t, setup.Config{
		Title: "Nonce Seam",
		// The bundle's security middleware installs its OWN nonce; the seam
		// must win, because that is the whole point of an override.
		NonceFunc: func(*http.Request) string { return wantNonce },
	})

	if !strings.Contains(rec.Body.String(), `nonce="`+wantNonce+`"`) {
		t.Fatalf("login page does not carry the configured nonce %q", wantNonce)
	}
}

// TestNonceFunc_DefaultUsesContextNonce pins the default: with no NonceFunc,
// the panel reads the value httputil.Nonce puts into the context.
func TestNonceFunc_DefaultUsesContextNonce(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{Title: "Default Nonce"})
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	const wantNonce = "context-nonce-xyz789"

	handler := bundle.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bundle.Login.ServeHTTP(w, r.WithContext(httputil.WithNonce(r.Context(), wantNonce)))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /login: status %d, want 200", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), `nonce="`+wantNonce+`"`) {
		t.Fatal("default nonce wiring no longer reads httputil's context value")
	}
}

// TestNonceFunc_OverridesContextNonce proves precedence: an explicit NonceFunc
// wins over a nonce already present in the context.
func TestNonceFunc_OverridesContextNonce(t *testing.T) {
	t.Parallel()

	const configured = "configured-nonce-000"

	bundle, err := setup.New(setup.Config{
		Title:     "Override Nonce",
		NonceFunc: func(*http.Request) string { return configured },
	})
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	handler := bundle.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bundle.Login.ServeHTTP(w, r.WithContext(httputil.WithNonce(r.Context(), "context-nonce-111")))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))

	body := rec.Body.String()
	if !strings.Contains(body, `nonce="`+configured+`"`) {
		t.Fatal("configured nonce did not win over the context nonce")
	}

	if strings.Contains(body, `nonce="context-nonce-111"`) {
		t.Fatal("context nonce leaked past an explicit NonceFunc")
	}
}
