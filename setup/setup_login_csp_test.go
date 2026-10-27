package setup_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// Tests in this file cover the login page's Content-Security-Policy
// contract. The bundle's security middleware sets a nonce CSP on every
// response, so the login page's inline scripts must carry that same nonce
// (the browser blocks inline scripts without it) and img-src must allow
// the data: URI favicon the page ships. A consumer that disables the
// security middleware gets the no-CSP page back: scripts without a nonce
// attribute, no CSP header.

func TestLoginPage_InlineScriptsCarryCSPNonce(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title: "CSP Login",
		ServiceConfig: &usermgmt.ServiceConfig{
			WebAuthn: stubWebAuthn{},
		},
	})
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	mux := http.NewServeMux()
	mux.Handle("/login", bundle.Login)

	rec := httptest.NewRecorder()
	bundle.Middleware()(mux).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /login: status %d, want 200", rec.Code)
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "img-src 'self' data:") {
		t.Errorf("CSP must allow the login page's inline SVG favicon, got %q", csp)
	}

	const noncePrefix = "script-src 'self' 'nonce-"
	start := strings.Index(csp, noncePrefix)
	if start < 0 {
		t.Fatalf("CSP carries no script nonce, got %q", csp)
	}
	start += len(noncePrefix)
	nonce := csp[start : strings.IndexByte(csp[start:], '\'')+start]
	if nonce == "" {
		t.Fatal("extracted empty nonce from CSP")
	}

	body := rec.Body.String()
	if !strings.Contains(body, `<link rel="icon" href="data:image/svg+xml`) {
		t.Error("login page must ship its brand favicon as a data: URI")
	}
	if got := strings.Count(body, `nonce="`+nonce+`"`); got < 3 {
		t.Errorf(
			"expected the theme script, the config block, and the WebAuthn script to carry the CSP nonce, found %d nonce'd tags",
			got,
		)
	}

	for rest := body; ; {
		open := strings.Index(rest, "<script")
		if open < 0 {
			break
		}
		tag := rest[open : open+strings.IndexByte(rest[open:], '>')+1]
		if !strings.Contains(tag, "nonce=") && !strings.Contains(tag, " src=") {
			t.Errorf("inline script without CSP nonce would be browser-blocked: %q", tag)
		}
		rest = rest[open+len(tag):]
	}
}

func TestLoginPage_WithoutSecurityMiddleware_RendersUnnonceScripts(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title: "Bare Login",
		ServiceConfig: &usermgmt.ServiceConfig{
			WebAuthn: stubWebAuthn{},
		},
	})
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	rec := httptest.NewRecorder()
	bundle.Login.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /login: status %d, want 200", rec.Code)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); csp != "" {
		t.Errorf("bare mount must not set a CSP header, got %q", csp)
	}

	body := rec.Body.String()
	for rest := body; ; {
		open := strings.Index(rest, "<script")
		if open < 0 {
			break
		}
		tag := rest[open : open+strings.IndexByte(rest[open:], '>')+1]
		if strings.Contains(tag, "nonce=") {
			t.Errorf("no nonce middleware ran, yet a script carries a nonce: %q", tag)
		}
		rest = rest[open+len(tag):]
	}
}
