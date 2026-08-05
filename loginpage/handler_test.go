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
	if h.config.Title != "Sign in" {
		t.Errorf("Title = %q, want %q", h.config.Title, "Sign in")
	}
	if h.config.Redirect != "/" {
		t.Errorf("Redirect = %q, want %q", h.config.Redirect, "/")
	}
	if h.config.AccentColor != DefaultAccentColor {
		t.Errorf("AccentColor = %q, want %q", h.config.AccentColor, DefaultAccentColor)
	}
	if h.config.CredentialName != "Passkey" {
		t.Errorf("CredentialName = %q, want %q", h.config.CredentialName, "Passkey")
	}
}

func TestServeHTTP_NoAuthMethodShowsError(t *testing.T) {
	h, _ := New(Config{Service: newTestService(t)})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, "No authentication method is configured") {
		t.Error("page should show no-auth error when no strategies are configured")
	}
	if strings.Contains(body, `id="lp-login-form"`) {
		t.Error("page should not render login form without auth methods")
	}
	if strings.Contains(body, `id="loginpage-config"`) {
		t.Error("page should not include config JS without auth methods")
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

// renderWithWebAuthn renders a page by directly constructing PageData with
// WebAuthn=true. In production this is only true when the Service has a
// WebAuthn provider configured.
func renderWithWebAuthn(t *testing.T, data PageData) string {
	t.Helper()
	data.WebAuthn = true
	if data.inlineCSS == "" {
		data.inlineCSS = "/* test */"
	}
	if data.inlineJS == "" {
		data.inlineJS = "/* test */ navigator.credentials"
	}
	if data.configJSON == "" {
		data.configJSON = `{"redirect":"/","endpoints":{"loginBegin":"/auth/webauthn/login/begin"}}`
	}
	w := httptest.NewRecorder()
	if err := Page(data).Render(context.Background(), w); err != nil {
		t.Fatalf("render: %v", err)
	}
	return w.Body.String()
}

func TestPage_WebAuthnLogin(t *testing.T) {
	body := renderWithWebAuthn(t, PageData{
		Title:    "Test App",
		Brand:    "Acme",
		Subtitle: "Sign in to your account",
		Accent:   "#4f46e5",
	})
	if !strings.Contains(body, "<title>Test App</title>") {
		t.Error("page missing title")
	}
	if !strings.Contains(body, `class="lp-brand">Acme<`) {
		t.Error("page missing brand")
	}
	if !strings.Contains(body, `id="lp-login-form"`) {
		t.Error("page missing login form")
	}
	if !strings.Contains(body, `id="lp-email"`) {
		t.Error("page missing email input")
	}
	if !strings.Contains(body, "Sign in with passkey") {
		t.Error("page missing passkey button")
	}
	if !strings.Contains(body, "loginpage-config") {
		t.Error("page missing config script tag")
	}
	if !strings.Contains(body, "--lp-accent") {
		t.Error("page missing inline CSS")
	}
	if !strings.Contains(body, "navigator.credentials") {
		t.Error("page missing inline JS")
	}
}

func TestPage_RegistrationSection(t *testing.T) {
	body := renderWithWebAuthn(t, PageData{
		Title:   "Test",
		Brand:   "Test",
		Accent:  "#4f46e5",
		ShowReg: true,
	})
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

func TestPage_RegistrationHidden(t *testing.T) {
	body := renderWithWebAuthn(t, PageData{
		Title:   "Test",
		Brand:   "Test",
		Accent:  "#4f46e5",
		ShowReg: false,
	})
	if strings.Contains(body, `id="lp-register-section"`) {
		t.Error("registration section should be hidden when ShowReg is false")
	}
}

func TestPage_OAuth2Buttons(t *testing.T) {
	body := renderWithWebAuthn(t, PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		WebAuthn: true,
		OAuth2Buttons: []OAuth2Button{
			{Provider: "google", Label: "Sign in with Google"},
			{Provider: "github", Label: "Sign in with GitHub"},
		},
	})
	if !strings.Contains(body, "Sign in with Google") {
		t.Error("Google button missing")
	}
	if !strings.Contains(body, "Sign in with GitHub") {
		t.Error("GitHub button missing")
	}
	if !strings.Contains(body, `/auth/oauth/google/begin`) {
		t.Error("Google OAuth begin URL missing")
	}
	if !strings.Contains(body, `/auth/oauth/github/begin`) {
		t.Error("GitHub OAuth begin URL missing")
	}
	if !strings.Contains(body, "lp-divider") {
		t.Error("divider missing between WebAuthn and OAuth2")
	}
}

func TestPage_OAuth2OnlyNoDivider(t *testing.T) {
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		WebAuthn: false,
		OAuth2Buttons: []OAuth2Button{
			{Provider: "google", Label: "Sign in with Google"},
		},
	}
	data.inlineCSS = "/* test */"
	w := httptest.NewRecorder()
	if err := Page(data).Render(context.Background(), w); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Sign in with Google") {
		t.Error("Google button missing")
	}
	if strings.Contains(body, "lp-divider") {
		t.Error("divider should not appear when WebAuthn is false")
	}
	if strings.Contains(body, `id="lp-login-form"`) {
		t.Error("WebAuthn form should not appear when WebAuthn is false")
	}
}

func TestPage_FaviconPresent(t *testing.T) {
	body := renderWithWebAuthn(t, PageData{
		Title:  "Test",
		Brand:  "Acme",
		Accent: "#ff0000",
	})
	if !strings.Contains(body, `rel="icon"`) {
		t.Error("favicon link missing")
	}
	if !strings.Contains(body, "data:image/svg+xml") {
		t.Error("favicon should be an SVG data URI")
	}
}

func TestServeHTTP_CSRFTokenIncluded(t *testing.T) {
	// Use renderWithWebAuthn since CSRF is always rendered, but the page
	// needs at least one auth method to render the body.
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		WebAuthn: true,
	}
	data.inlineCSS = "/* test */"
	data.inlineJS = "/* test */"
	data.configJSON = `{}`
	data.CSRFMeta = `<meta name="csrf-token" content="test-csrf-123">`
	data.CSRFField = `<input type="hidden" name="csrf_token" value="test-csrf-123">`

	w := httptest.NewRecorder()
	if err := Page(data).Render(context.Background(), w); err != nil {
		t.Fatalf("render: %v", err)
	}
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
		OAuth2Buttons: []OAuth2Button{
			{Provider: "google", Label: "Sign in with Google"},
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	body := w.Body.String()
	// OAuth2 buttons are present
	if !strings.Contains(body, "Sign in with Google") {
		t.Error("Google OAuth button missing")
	}
}

func TestServeHTTP_AuthPrefix(t *testing.T) {
	h, _ := New(Config{
		Service:    newTestService(t),
		AuthPrefix: "/api",
		OAuth2Buttons: []OAuth2Button{
			{Provider: "google", Label: "Sign in with Google"},
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, "/api/auth/oauth/google/begin") {
		t.Error("OAuth2 URL should contain prefixed endpoint")
	}
}

func TestServeHTTP_AccentColorApplied(t *testing.T) {
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#ff0000",
		WebAuthn: true,
	}
	data.inlineCSS = "/* test */"
	data.inlineJS = "/* test */"
	data.configJSON = "{}"
	w := httptest.NewRecorder()
	_ = Page(data).Render(context.Background(), w)
	body := w.Body.String()
	if !strings.Contains(body, "--lp-accent:#ff0000") {
		t.Error("accent color not applied in CSS variable")
	}
}

func TestServeHTTP_CustomCSSPath(t *testing.T) {
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		CSSPath:  "/css/app.css",
		WebAuthn: true,
	}
	data.inlineCSS = "/* test */"
	data.inlineJS = "/* test */"
	data.configJSON = "{}"
	w := httptest.NewRecorder()
	_ = Page(data).Render(context.Background(), w)
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

func TestMount_CoexistsWithRootIndex(t *testing.T) {
	h, _ := New(Config{Service: newTestService(t)})

	mux := http.NewServeMux()
	h.Mount(mux, "/login")
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET / status = %d, want 200", rec.Code)
	}
}

func TestNewPageData(t *testing.T) {
	svc := newTestService(t)
	data, err := NewPageData(Config{
		Service: svc,
		Title:   "My App",
	}, nil)
	if err != nil {
		t.Fatalf("NewPageData: %v", err)
	}
	if data.Title != "My App" {
		t.Errorf("Title = %q, want %q", data.Title, "My App")
	}
	if data.WebAuthn != false {
		t.Error("WebAuthn should be false for service without WebAuthn provider")
	}
}

func TestNewPageData_WithCSRF(t *testing.T) {
	svc := newTestService(t)
	ctx := cqrshtmx.WithCSRFToken(context.Background(), "token-abc")
	r := httptest.NewRequest(http.MethodGet, "/login", nil).WithContext(ctx)
	data, err := NewPageData(Config{Service: svc}, r)
	if err != nil {
		t.Fatalf("NewPageData: %v", err)
	}
	if !strings.Contains(data.CSRFMeta, "token-abc") {
		t.Error("CSRFMeta should contain the token")
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

func TestProviderDisplayName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"google", "Google"},
		{"github", "GitHub"},
		{"microsoft", "Microsoft"},
		{"custom", "Custom"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ProviderDisplayName(tt.in); got != tt.want {
			t.Errorf("ProviderDisplayName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestOAuth2ButtonFromProvider(t *testing.T) {
	btn := OAuth2ButtonFromProvider("google")
	if btn.Provider != "google" {
		t.Errorf("Provider = %q, want %q", btn.Provider, "google")
	}
	if btn.Label != "Sign in with Google" {
		t.Errorf("Label = %q, want %q", btn.Label, "Sign in with Google")
	}
}

func TestNewPageData_NilService(t *testing.T) {
	_, err := NewPageData(Config{}, nil)
	if err == nil {
		t.Fatal("expected error for nil Service")
	}
}

func TestBuildPageData_AutoPopulateOAuth2(t *testing.T) {
	svc := newTestService(t)
	h, _ := New(Config{Service: svc})
	// Service has no oauth2 provider, so auto-populate should produce no buttons.
	if len(h.data.OAuth2Buttons) != 0 {
		t.Errorf("expected no OAuth2 buttons, got %d", len(h.data.OAuth2Buttons))
	}
}

func TestRenderPage_SetsContentType(t *testing.T) {
	h, _ := New(Config{Service: newTestService(t)})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/html; charset=utf-8")
	}
}

func TestPage_BrowserUnsupportedFallback(t *testing.T) {
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		WebAuthn: true,
	}
	data.inlineCSS = "/* test */"
	data.inlineJS = "/* test */"
	data.configJSON = "{}"
	w := httptest.NewRecorder()
	_ = Page(data).Render(context.Background(), w)
	body := w.Body.String()
	if !strings.Contains(body, `id="lp-no-webauthn"`) {
		t.Error("browser-unsupported fallback div missing")
	}
	if !strings.Contains(body, "lp-hidden") {
		t.Error("fallback div should be hidden by default")
	}
}

func TestPage_CSSPathLinked(t *testing.T) {
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		CSSPath:  "/css/custom.css",
		WebAuthn: true,
	}
	data.inlineCSS = "/* test */"
	data.inlineJS = "/* test */"
	data.configJSON = "{}"
	w := httptest.NewRecorder()
	_ = Page(data).Render(context.Background(), w)
	body := w.Body.String()
	if !strings.Contains(body, `href="/css/custom.css"`) {
		t.Error("custom CSS path not linked")
	}
}

func TestPage_NoAuthState(t *testing.T) {
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		WebAuthn: false,
	}
	data.inlineCSS = "/* test */"
	w := httptest.NewRecorder()
	_ = Page(data).Render(context.Background(), w)
	body := w.Body.String()
	if !strings.Contains(body, "No authentication method is configured") {
		t.Error("no-auth error state missing")
	}
}

func TestFaviconURI(t *testing.T) {
	data := PageData{Brand: "Acme", Accent: "#ff0000"}
	uri := data.faviconURI()
	if !strings.Contains(string(uri), "data:image/svg+xml") {
		t.Error("favicon should be an SVG data URI")
	}
	if !strings.Contains(string(uri), "fill='#ff0000'") {
		t.Error("favicon should use the accent color")
	}
	if !strings.Contains(string(uri), ">A<") {
		t.Error("favicon should use the brand initial")
	}
}

func TestOauthBeginURL(t *testing.T) {
	data := PageData{authPrefix: "/api"}
	url := data.oauthBeginURL("google")
	if url != "/api/auth/oauth/google/begin" {
		t.Errorf("oauthBeginURL = %q, want %q", url, "/api/auth/oauth/google/begin")
	}
}

func TestServeHTTP_HeadMethod(t *testing.T) {
	h, _ := New(Config{Service: newTestService(t)})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodHead, "/login", nil)
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("HEAD should return 200, got %d", w.Code)
	}
}

func TestServeHTTP_NoRegistration(t *testing.T) {
	h, _ := New(Config{
		Service:        newTestService(t),
		NoRegistration: true,
	})
	data := h.data
	data.WebAuthn = true
	data.ShowReg = false
	data.inlineCSS = "/* test */"
	data.inlineJS = "/* test */"
	data.configJSON = "{}"
	w := httptest.NewRecorder()
	_ = Page(data).Render(context.Background(), w)
	if strings.Contains(w.Body.String(), "Create one") {
		t.Error("registration toggle should be hidden when NoRegistration is true")
	}
}

func TestPage_AllAuthMethods(t *testing.T) {
	body := renderWithWebAuthn(t, PageData{
		Title:   "Test",
		Brand:   "Test",
		Accent:  "#4f46e5",
		ShowReg: true,
		OAuth2Buttons: []OAuth2Button{
			{Provider: "google", Label: "Sign in with Google"},
		},
	})
	if !strings.Contains(body, "lp-divider") {
		t.Error("divider missing when both WebAuthn and OAuth2 present")
	}
	if !strings.Contains(body, "Create one") {
		t.Error("registration toggle missing")
	}
	if !strings.Contains(body, "Sign in with Google") {
		t.Error("OAuth2 button missing")
	}
	if !strings.Contains(body, "Sign in with passkey") {
		t.Error("passkey button missing")
	}
}

func TestFaviconURI_EmptyBrand(t *testing.T) {
	data := PageData{Brand: "", Accent: "#000"}
	uri := data.faviconURI()
	if !strings.Contains(string(uri), ">?<") {
		t.Error("empty brand should show '?' as favicon initial")
	}
}

func TestRenderPage_RenderError(t *testing.T) {
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		WebAuthn: true,
	}
	data.inlineCSS = "/* test */"
	data.inlineJS = "/* test */"
	data.configJSON = "{}"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	renderPage(w, r, data)
	// Should not panic, even if render fails due to cancelled context.
}
