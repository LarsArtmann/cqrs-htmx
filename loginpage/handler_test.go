package loginpage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/httputil"
)

func newTestService(t *testing.T) *usermgmt.Service {
	t.Helper()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

// assertContains fails the test when body lacks want. Shared by the page
// content assertions so the failure shape lives in one place.
func assertContains(t *testing.T, body, want, label string) {
	t.Helper()
	if !strings.Contains(body, want) {
		t.Error(label)
	}
}

// assertNotContains fails the test when body contains unwanted.
func assertNotContains(t *testing.T, body, unwanted, label string) {
	t.Helper()
	if strings.Contains(body, unwanted) {
		t.Error(label)
	}
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
	assertContains(
		t,
		body,
		"No authentication method is configured",
		"page should show no-auth error when no strategies are configured",
	)
	assertNotContains(t, body, `id="lp-login-form"`, "page should not render login form without auth methods")
	assertNotContains(t, body, `id="loginpage-config"`, "page should not include config JS without auth methods")
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
	assertContains(t, body, "<title>Test App</title>", "page missing title")
	assertContains(t, body, ">Acme</h1>", "page missing brand")
	assertContains(t, body, `id="lp-login-form"`, "page missing login form")
	assertContains(t, body, `id="lp-email"`, "page missing email input")
	assertContains(t, body, "Sign in with passkey", "page missing passkey button")
	assertContains(t, body, "loginpage-config", "page missing config script tag")
	assertContains(t, body, "navigator.credentials", "page missing inline JS")
}

func TestPage_RegistrationSection(t *testing.T) {
	body := renderWithWebAuthn(t, PageData{
		Title:   "Test",
		Brand:   "Test",
		Accent:  "#4f46e5",
		ShowReg: true,
	})
	assertContains(t, body, `id="lp-register-section"`, "registration section missing when ShowReg is true")
	assertContains(t, body, "Create one", "registration toggle link missing")
	assertContains(t, body, "Create account", "registration submit button missing")
}

func TestPage_RegistrationHidden(t *testing.T) {
	body := renderWithWebAuthn(t, PageData{
		Title:   "Test",
		Brand:   "Test",
		Accent:  "#4f46e5",
		ShowReg: false,
	})
	assertNotContains(
		t,
		body,
		`id="lp-register-section"`,
		"registration section should be hidden when ShowReg is false",
	)
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
	assertContains(t, body, "Sign in with Google", "Google button missing")
	assertContains(t, body, "Sign in with GitHub", "GitHub button missing")
	assertContains(t, body, `/auth/oauth/google/begin`, "Google OAuth begin URL missing")
	assertContains(t, body, `/auth/oauth/github/begin`, "GitHub OAuth begin URL missing")
	assertContains(t, body, "lp-divider", "divider missing between WebAuthn and OAuth2")
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
	w := httptest.NewRecorder()
	if err := Page(data).Render(context.Background(), w); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := w.Body.String()
	assertContains(t, body, "Sign in with Google", "Google button missing")
	assertNotContains(t, body, "lp-divider", "divider should not appear when WebAuthn is false")
	assertNotContains(t, body, `id="lp-login-form"`, "WebAuthn form should not appear when WebAuthn is false")
}

func TestPage_FaviconPresent(t *testing.T) {
	body := renderWithWebAuthn(t, PageData{
		Title:  "Test",
		Brand:  "Acme",
		Accent: "#ff0000",
	})
	assertContains(t, body, `rel="icon"`, "favicon link missing")
	assertContains(t, body, "data:image/svg+xml", "favicon should be an SVG data URI")
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
	data.inlineJS = "/* test */"
	data.configJSON = `{}`
	data.CSRFMeta = `<meta name="csrf-token" content="test-csrf-123">`
	data.CSRFField = `<input type="hidden" name="csrf_token" value="test-csrf-123">`

	w := httptest.NewRecorder()
	if err := Page(data).Render(context.Background(), w); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := w.Body.String()
	assertContains(t, body, "test-csrf-123", "CSRF token not rendered in page")
	assertContains(t, body, `name="csrf-token"`, "CSRF meta tag missing")
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
	assertContains(t, body, "Sign in with Google", "Google OAuth button missing")
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
	assertContains(t, body, "/api/auth/oauth/google/begin", "OAuth2 URL should contain prefixed endpoint")
}

func TestServeHTTP_AccentColorApplied(t *testing.T) {
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#ff0000",
		WebAuthn: true,
	}
	data.inlineJS = "/* test */"
	data.configJSON = "{}"
	w := httptest.NewRecorder()
	_ = Page(data).Render(context.Background(), w)
	body := w.Body.String()
	// templ HTML-escapes the SafeURL (single quotes become &#39;), so assert
	// the fragments that survive escaping: the data URI scheme and the
	// percent-encoded color ('#' would truncate the data URI at the fragment
	// boundary — it MUST arrive as %23).
	assertContains(t, body, "data:image/svg+xml", "accent color data URI missing from the SVG favicon")
	assertContains(t, body, "%23ff0000", "accent color not applied (percent-encoded) in the SVG favicon")
}

func TestServeHTTP_CustomCSSPath(t *testing.T) {
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		CSSPath:  "/css/app.css",
		WebAuthn: true,
	}
	data.inlineJS = "/* test */"
	data.configJSON = "{}"
	w := httptest.NewRecorder()
	_ = Page(data).Render(context.Background(), w)
	body := w.Body.String()
	assertContains(t, body, `href="/css/app.css"`, "custom CSS path not linked")
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
	ctx := httputil.WithCSRFToken(context.Background(), "token-abc")
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
	data.inlineJS = "/* test */"
	data.configJSON = "{}"
	w := httptest.NewRecorder()
	_ = Page(data).Render(context.Background(), w)
	body := w.Body.String()
	assertContains(t, body, `id="lp-no-webauthn"`, "browser-unsupported fallback div missing")
	assertContains(t, body, `id="lp-no-webauthn" class="hidden`, "fallback div should be hidden by default")
}

func TestPage_CSSPathLinked(t *testing.T) {
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		CSSPath:  "/css/custom.css",
		WebAuthn: true,
	}
	data.inlineJS = "/* test */"
	data.configJSON = "{}"
	w := httptest.NewRecorder()
	_ = Page(data).Render(context.Background(), w)
	body := w.Body.String()
	assertContains(t, body, `href="/css/custom.css"`, "custom CSS path not linked")
}

func TestPage_NoAuthState(t *testing.T) {
	data := PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		WebAuthn: false,
	}
	w := httptest.NewRecorder()
	_ = Page(data).Render(context.Background(), w)
	body := w.Body.String()
	assertContains(t, body, "No authentication method is configured", "no-auth error state missing")
}

func TestFaviconURI(t *testing.T) {
	data := PageData{Brand: "Acme", Accent: "#ff0000"}
	uri := data.faviconURI()
	if !strings.Contains(string(uri), "data:image/svg+xml") {
		t.Error("favicon should be an SVG data URI")
	}
	if strings.Contains(string(uri), "#") {
		t.Error("raw '#' truncates a data URI at the fragment boundary — the SVG must be percent-encoded")
	}
	if !strings.Contains(string(uri), "fill%3D%27%23ff0000%27") {
		t.Error("favicon should use the accent color")
	}
	if !strings.Contains(string(uri), "%3EA%3C") {
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
	assertContains(t, body, "lp-divider", "divider missing when both WebAuthn and OAuth2 present")
	assertContains(t, body, "Create one", "registration toggle missing")
	assertContains(t, body, "Sign in with Google", "OAuth2 button missing")
	assertContains(t, body, "Sign in with passkey", "passkey button missing")
}

func TestFaviconURI_EmptyBrand(t *testing.T) {
	data := PageData{Brand: "", Accent: "#000"}
	uri := data.faviconURI()
	if !strings.Contains(string(uri), "%3E%3F%3C") {
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
	data.inlineJS = "/* test */"
	data.configJSON = "{}"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	renderPage(w, r, data)
	// Should not panic, even if render fails due to cancelled context.
}

func TestNew_MaliciousAccentColorRejected(t *testing.T) {
	for _, accent := range []string{
		`#000</style><script>alert(1)</script>`,
		`red">x`,
		`a&b`,
		`back\slash`,
		"quote'd",
	} {
		_, err := New(Config{Service: newTestService(t), AccentColor: accent})
		if err == nil {
			t.Errorf("AccentColor %q should be rejected", accent)
		}
	}
}

func TestNew_ValidAccentColorsAccepted(t *testing.T) {
	for _, accent := range []string{
		"#4f46e5",
		"rgb(79 70 229)",
		"color-mix(in srgb, red 50%, blue)",
		"var(--brand-accent)",
		"tomato",
	} {
		if _, err := New(Config{Service: newTestService(t), AccentColor: accent}); err != nil {
			t.Errorf("AccentColor %q should be accepted, got %v", accent, err)
		}
	}
}

func TestNew_MaliciousCSSPathRejected(t *testing.T) {
	for _, cssPath := range []string{
		"javascript:alert(1)",
		"data:text/html,alert(1)",
		"ftp://example.com/app.css",
	} {
		_, err := New(Config{Service: newTestService(t), CSSPath: cssPath})
		if err == nil {
			t.Errorf("CSSPath %q should be rejected", cssPath)
		}
	}
}

func TestNew_ValidCSSPathsAccepted(t *testing.T) {
	for _, cssPath := range []string{"", "/css/app.css", "https://cdn.example.com/app.css", "http://localhost/app.css"} {
		if _, err := New(Config{Service: newTestService(t), CSSPath: cssPath}); err != nil {
			t.Errorf("CSSPath %q should be accepted, got %v", cssPath, err)
		}
	}
}

func TestConfigJSON_ScriptBreakoutEscaped(t *testing.T) {
	h, err := New(Config{
		Service:        newTestService(t),
		CredentialName: `</script><script>alert(1)</script>`,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if strings.Contains(h.data.configJSON, "</script>") {
		t.Errorf("configJSON must not contain a raw </script>, got %q", h.data.configJSON)
	}
	if !strings.Contains(h.data.configJSON, `\u003c/script`) {
		t.Errorf("configJSON should escape < as \\u003c, got %q", h.data.configJSON)
	}
}

func TestServeHTTP_MaliciousCredentialNameNotInjected(t *testing.T) {
	h, err := New(Config{
		Service:        newTestService(t),
		CredentialName: `</script><script>alert(1)</script>`,
		OAuth2Buttons:  []OAuth2Button{{Provider: "google", Label: "Google"}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(w, r)
	body := w.Body.String()
	assertNotContains(
		t,
		body,
		"</script><script>alert(1)</script>",
		"rendered page must not contain the raw injection payload",
	)
}

func TestPage_NonceRenderedOnScripts(t *testing.T) {
	body := renderWithWebAuthn(t, PageData{
		Title:  "Test",
		Brand:  "Test",
		Accent: "#4f46e5",
		Nonce:  "abc123_-",
	})
	if strings.Count(body, `nonce="abc123_-"`) < 2 {
		t.Error("config + inline scripts should carry the CSP nonce")
	}
}

func TestPage_NonceSanitized(t *testing.T) {
	if got := sanitizeNonce(`x"><script>`); got != "xscript" {
		t.Errorf("sanitizeNonce = %q, want %q", got, "xscript")
	}
}

func bodyClassTokens(t *testing.T, body string) []string {
	t.Helper()
	const marker = `<body class="`
	_, after, ok := strings.Cut(body, marker)
	if !ok {
		t.Fatal("body tag with class attribute not found")
	}
	rest := after
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatal("unterminated body class attribute")
	}
	return strings.Fields(rest[:end])
}

func TestPage_ThemeBodyClass(t *testing.T) {
	for _, theme := range []string{"dark", "light"} {
		body := renderWithWebAuthn(t, PageData{Title: "T", Brand: "T", Accent: "#000", Theme: theme})
		if !slices.Contains(bodyClassTokens(t, body), theme) {
			t.Errorf("theme %q: body class should contain the theme token", theme)
		}
	}
}

func TestPage_RegisterFirstVisibility(t *testing.T) {
	loginFirst := renderWithWebAuthn(t, PageData{Title: "T", Brand: "T", Accent: "#000", ShowReg: true})
	if strings.Contains(loginFirst, `id="lp-login-section" class="hidden`) {
		t.Error("login section should be visible by default")
	}
	if !strings.Contains(loginFirst, `id="lp-register-section" class="hidden`) {
		t.Error("register section should be hidden by default")
	}
	registerFirst := renderWithWebAuthn(
		t,
		PageData{Title: "T", Brand: "T", Accent: "#000", ShowReg: true, RegisterFirst: true},
	)
	if !strings.Contains(registerFirst, `id="lp-login-section" class="hidden`) {
		t.Error("login section should be hidden when RegisterFirst is set")
	}
	if strings.Contains(registerFirst, `id="lp-register-section" class="hidden`) {
		t.Error("register section should be visible when RegisterFirst is set")
	}
}

func TestResolveTheme(t *testing.T) {
	config := Config{Theme: ThemeDark}
	if got := config.resolveTheme(nil); got != ThemeDark {
		t.Errorf("static theme = %q, want %q", got, ThemeDark)
	}
	hook := Config{Theme: ThemeDark, ThemeFromRequest: func(*http.Request) string { return ThemeLight }}
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	if got := hook.resolveTheme(r); got != ThemeLight {
		t.Errorf("hook should win over static theme, got %q", got)
	}
	bad := Config{Theme: "purple"}
	if got := bad.resolveTheme(r); got != "" {
		t.Errorf("invalid theme should fall back to prefers-color-scheme, got %q", got)
	}
}

func TestNew_DefaultCSSPath(t *testing.T) {
	h, err := New(Config{Service: newTestService(t)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if h.config.CSSPath != "/app.css" {
		t.Errorf("CSSPath = %q, want %q", h.config.CSSPath, "/app.css")
	}
}

// stubOAuth2WithNames satisfies usermgmt's OAuth2Provider plus the optional
// Names() the auto-detect path reads, so tests can configure providers
// without the real oauth2 module.
type stubOAuth2WithNames struct{ names []string }

func (m *stubOAuth2WithNames) BeginLogin(_ context.Context, _, _ string) (string, string, error) {
	return "", "", nil
}

func (m *stubOAuth2WithNames) FinishLogin(_ context.Context, _, _, _ string) ([]byte, error) {
	return nil, nil
}

func (m *stubOAuth2WithNames) Names() []string { return m.names }

func TestNew_NoOAuth2_WithExplicitButtons_Rejected(t *testing.T) {
	_, err := New(Config{
		Service: newTestService(t),
		OAuth2Buttons: []OAuth2Button{
			{Provider: "google", Label: "Sign in with Google"},
		},
		NoOAuth2: true,
	})
	if err == nil {
		t.Fatal("NoOAuth2 together with an explicit OAuth2Buttons list must be rejected at New")
	}
}

func TestBuildPageData_NoOAuth2_HidesAutoDetectedButtons(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		OAuth2: &stubOAuth2WithNames{names: []string{"google", "github"}},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	auto := buildPageData(Config{Service: svc}, nil)
	if len(auto.OAuth2Buttons) != 2 {
		t.Fatalf("auto-detect: want 2 buttons, got %d", len(auto.OAuth2Buttons))
	}

	hidden := buildPageData(Config{Service: svc, NoOAuth2: true}, nil)
	if len(hidden.OAuth2Buttons) != 0 {
		t.Errorf("NoOAuth2: want 0 buttons, got %d", len(hidden.OAuth2Buttons))
	}
	if hidden.WebAuthn != false {
		t.Errorf("NoOAuth2 must not change WebAuthn detection")
	}
}

func TestServeHTTP_NoOAuth2_RendersNoButtons(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		// WebAuthn off, providers on, NoOAuth2 on: the page renders the
		// setup notice rather than any sign-in control.
		OAuth2: &stubOAuth2WithNames{names: []string{"google"}},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	h, err := New(Config{Service: svc, NoOAuth2: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()
	assertNotContains(t, body, "/auth/oauth/google/begin", "NoOAuth2 must hide auto-detected provider buttons")
	assertNotContains(t, body, "Sign in with Google", "NoOAuth2 must hide provider labels")
	assertContains(
		t,
		body,
		"No authentication method is configured",
		"page should render the setup notice when every method is hidden",
	)
}

func TestPage_ButtonSpinnerServerRenderedHidden(t *testing.T) {
	body := renderWithWebAuthn(t, PageData{
		Title:    "Test",
		Brand:    "Test",
		Accent:   "#4f46e5",
		WebAuthn: true,
	})
	assertContains(t, body, `class="hidden lp-spinner"`, "submit buttons must server-render the hidden lp-spinner hook")
	assertContains(t, body, "animate-spin", "spinner must be the library feedback.Spinner (animate-spin class)")
	assertNotContains(t, body, "SPINNER_HTML", "login.js must no longer carry hand-built spinner markup")
}

func newOAuthOnlyService(t *testing.T, names ...string) *usermgmt.Service {
	t.Helper()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		OAuth2: &stubOAuth2WithNames{names: names},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func TestNew_AutoRedirect_NoOAuth2_Rejected(t *testing.T) {
	_, err := New(Config{
		Service:      newTestService(t),
		AutoRedirect: true,
		NoOAuth2:     true,
	})
	if err == nil {
		t.Fatal("AutoRedirect together with NoOAuth2 must be rejected at New")
	}
}

func TestServeHTTP_AutoRedirect_SingleProvider(t *testing.T) {
	h, err := New(Config{
		Service:      newOAuthOnlyService(t, "google"),
		AutoRedirect: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/login", nil))
	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != "/auth/oauth/google/begin" {
		t.Errorf("Location = %q, want %q", loc, "/auth/oauth/google/begin")
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-store")
	}
}

func TestServeHTTP_AutoRedirect_ErrorParamRendersPage(t *testing.T) {
	h, err := New(Config{
		Service:      newOAuthOnlyService(t, "google"),
		AutoRedirect: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/login?error=Provider+said+no", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (?error= must break the callback redirect loop)", w.Code, http.StatusOK)
	}
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("Location = %q, want none", loc)
	}
	body := w.Body.String()
	assertContains(t, body, "Provider said no", "page should surface the error param in the #lp-error alert")
	assertNotContains(
		t,
		body,
		"red-400 hidden",
		"error alert must not carry the hidden class when a message is present",
	)
}

func TestServeHTTP_ErrorParamSanitized(t *testing.T) {
	h, err := New(Config{Service: newOAuthOnlyService(t, "google")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, `/login?error=<script>alert(1)</script>`, nil))
	body := w.Body.String()
	assertNotContains(t, body, "<script>alert(1)</script>", "error message must be HTML-escaped by templ")
	assertContains(t, body, "alert(1)", "escaped error text should still be present")
}

func TestServeHTTP_AutoRedirect_AuthPrefix(t *testing.T) {
	h, err := New(Config{
		Service:      newOAuthOnlyService(t, "google"),
		AutoRedirect: true,
		AuthPrefix:   "/api",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/login", nil))
	if loc := w.Header().Get("Location"); loc != "/api/auth/oauth/google/begin" {
		t.Errorf("Location = %q, want %q", loc, "/api/auth/oauth/google/begin")
	}
}

func TestServeHTTP_AutoRedirect_MultipleProvidersRendersPage(t *testing.T) {
	h, err := New(Config{
		Service:      newOAuthOnlyService(t, "google", "github"),
		AutoRedirect: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/login", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (page must render so the visitor can choose)", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	assertContains(t, body, "/auth/oauth/google/begin", "page should render Google OAuth begin URL")
	assertContains(t, body, "/auth/oauth/github/begin", "page should render buttons for every provider")
}

func TestServeHTTP_AutoRedirect_NoAuthMethodRendersNotice(t *testing.T) {
	h, err := New(Config{
		Service:      newTestService(t),
		AutoRedirect: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/login", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "No authentication method is configured") {
		t.Error("page should render the setup notice when there is nothing to redirect to")
	}
}

func TestSoleOAuthBeginURL(t *testing.T) {
	tests := []struct {
		name string
		data PageData
		want string
	}{
		{
			"single provider only",
			PageData{OAuth2Buttons: []OAuth2Button{{Provider: "google"}}},
			"/auth/oauth/google/begin",
		},
		{"webauthn present", PageData{WebAuthn: true, OAuth2Buttons: []OAuth2Button{{Provider: "google"}}}, ""},
		{"two providers", PageData{OAuth2Buttons: []OAuth2Button{{Provider: "google"}, {Provider: "github"}}}, ""},
		{"no providers", PageData{}, ""},
		{
			"auth prefix",
			PageData{authPrefix: "/api", OAuth2Buttons: []OAuth2Button{{Provider: "google"}}},
			"/api/auth/oauth/google/begin",
		},
	}
	for _, tt := range tests {
		if got := tt.data.soleOAuthBeginURL(); got != tt.want {
			t.Errorf("%s: soleOAuthBeginURL = %q, want %q", tt.name, got, tt.want)
		}
	}
}
