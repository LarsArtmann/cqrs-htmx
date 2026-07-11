package loginpage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func newTestService(t *testing.T) *usermgmt.Service {
	t.Helper()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func TestNew_NilService(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error for nil Service")
	}
}

func TestNew_Defaults(t *testing.T) {
	h, err := New(Config{Service: newTestService(t)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if h.cfg.Title != "Sign in" {
		t.Errorf("Title = %q, want %q", h.cfg.Title, "Sign in")
	}
	if h.cfg.Redirect != "/" {
		t.Errorf("Redirect = %q, want %q", h.cfg.Redirect, "/")
	}
	if h.cfg.AccentColor != DefaultAccentColor {
		t.Errorf("AccentColor = %q, want %q", h.cfg.AccentColor, DefaultAccentColor)
	}
}

func TestServeHTTP_RendersPage(t *testing.T) {
	h, err := New(Config{
		Service: newTestService(t),
		Title:   "Test App",
		Brand:   "Acme",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}

	body := w.Body.String()
	// Title and brand rendered
	if !strings.Contains(body, "<title>Test App</title>") {
		t.Error("page missing title")
	}
	if !strings.Contains(body, `class="lp-brand">Acme<`) {
		t.Error("page missing brand")
	}
	// Login form present
	if !strings.Contains(body, `id="lp-login-form"`) {
		t.Error("page missing login form")
	}
	if !strings.Contains(body, `id="lp-email"`) {
		t.Error("page missing email input")
	}
	if !strings.Contains(body, "Sign in with passkey") {
		t.Error("page missing passkey button")
	}
	// Config JSON present
	if !strings.Contains(body, "loginpage-config") {
		t.Error("page missing config script tag")
	}
	// Inline CSS and JS present
	if !strings.Contains(body, "--lp-accent") {
		t.Error("page missing inline CSS")
	}
	if !strings.Contains(body, "navigator.credentials") {
		t.Error("page missing inline JS")
	}
}

func TestServeHTTP_PostMethodNotAllowed(t *testing.T) {
	h, _ := New(Config{Service: newTestService(t)})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/login", nil)
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestServeHTTP_RegistrationHiddenWithoutWebAuthn(t *testing.T) {
	// Service has no WebAuthn provider, so there's nothing to register a passkey with.
	h, _ := New(Config{
		Service: newTestService(t),
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	body := w.Body.String()
	if strings.Contains(body, `id="lp-register-section"`) {
		t.Error("registration section should be hidden when WebAuthn is not configured")
	}
	if strings.Contains(body, "Create one") {
		t.Error("registration toggle link should be hidden")
	}
}

func TestServeHTTP_RegistrationHiddenByConfig(t *testing.T) {
	h, _ := New(Config{
		Service:        newTestService(t),
		NoRegistration: true,
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	body := w.Body.String()
	if strings.Contains(body, `id="lp-register-section"`) {
		t.Error("registration section should be hidden by NoRegistration")
	}
}

func TestPageTemplate_RegistrationSection(t *testing.T) {
	// Test the templ component directly with ShowReg=true to verify the
	// registration HTML is rendered. In production, ShowReg is only true when
	// the Service has WebAuthn configured.
	w := httptest.NewRecorder()
	data := PageData{
		Title:      "Test",
		Brand:      "Test",
		Subtitle:   "Sign in to your account",
		Accent:     "#4f46e5",
		ShowReg:    true,
		ConfigJSON: `{"redirect":"/"}`,
	}
	if err := Page(data).Render(context.Background(), w); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := w.Body.String()
	if !strings.Contains(body, `id="lp-register-section"`) {
		t.Error("registration section missing when ShowReg is true")
	}
	if !strings.Contains(body, "Create one") {
		t.Error("registration toggle link missing")
	}
	if !strings.Contains(body, "Create account") {
		t.Error("registration submit button missing")
	}
}

func TestServeHTTP_CSRFTokenIncluded(t *testing.T) {
	h, _ := New(Config{Service: newTestService(t)})
	w := httptest.NewRecorder()
	ctx := cqrshtmx.WithCSRFToken(context.Background(), "test-csrf-123")
	r := httptest.NewRequest(http.MethodGet, "/login", nil).WithContext(ctx)
	h.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, "test-csrf-123") {
		t.Error("CSRF token not rendered in page")
	}
	if !strings.Contains(body, `name="csrf-token"`) {
		t.Error("CSRF meta tag missing")
	}
}

func TestServeHTTP_ConfigJSONContainsEndpoints(t *testing.T) {
	h, _ := New(Config{
		Service:  newTestService(t),
		Redirect: "/dashboard",
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, "/auth/webauthn/login/begin") {
		t.Error("config JSON missing loginBegin endpoint")
	}
	if !strings.Contains(body, "/auth/webauthn/login/finish") {
		t.Error("config JSON missing loginFinish endpoint")
	}
	if !strings.Contains(body, "/auth/register") {
		t.Error("config JSON missing register endpoint")
	}
	if !strings.Contains(body, `/dashboard`) {
		t.Error("config JSON missing redirect URL")
	}
}

func TestServeHTTP_AuthPrefix(t *testing.T) {
	h, _ := New(Config{
		Service:    newTestService(t),
		AuthPrefix: "/api",
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, "/api/auth/webauthn/login/begin") {
		t.Error("config JSON should contain prefixed endpoint")
	}
	if strings.Contains(body, `"/auth/webauthn`) {
		t.Error("config JSON should not contain unprefixed endpoints")
	}
}

func TestServeHTTP_AccentColorApplied(t *testing.T) {
	h, _ := New(Config{
		Service:     newTestService(t),
		AccentColor: "#ff0000",
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, "--lp-accent:#ff0000") {
		t.Error("accent color not applied in CSS variable")
	}
}

func TestServeHTTP_CustomCSSPath(t *testing.T) {
	h, _ := New(Config{
		Service: newTestService(t),
		CSSPath: "/css/app.css",
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, `href="/css/app.css"`) {
		t.Error("custom CSS path not linked")
	}
}

func TestServeHTTP_NoCacheHeaders(t *testing.T) {
	h, _ := New(Config{Service: newTestService(t)})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-store")
	}
	if xcto := w.Header().Get("X-Content-Type-Options"); xcto != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", xcto, "nosniff")
	}
}

func TestMount(t *testing.T) {
	h, _ := New(Config{Service: newTestService(t)})
	mux := http.NewServeMux()
	h.Mount(mux, "/login")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSafeRedirectPath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", "/"},
		{"/dashboard", "/dashboard"},
		{"//evil.com", "/"},
		{"https://evil.com", "/"},
		{"relative", "/"},
	}
	for _, tt := range tests {
		if got := safeRedirectPath(tt.in); got != tt.want {
			t.Errorf("safeRedirectPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFirstRune(t *testing.T) {
	if got := firstRune("Acme"); got != "A" {
		t.Errorf("firstRune(Acme) = %q, want %q", got, "A")
	}
	if got := firstRune(""); got != "?" {
		t.Errorf("firstRune() = %q, want %q", got, "?")
	}
	if got := firstRune("Über"); got != "Ü" {
		t.Errorf("firstRune(Über) = %q, want %q", got, "Ü")
	}
}
