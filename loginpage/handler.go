package loginpage

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/httputil"
	"github.com/larsartmann/templ-components/layout"
	"github.com/larsartmann/templ-components/utils"
)

// PageData holds everything the templ page needs to render.
// Exported so consumers can render [Page] directly in their own layout (Option B).
// Construct via [NewPageData] — do not build by hand.
type PageData struct {
	// --- Page metadata ---
	Title    string
	Brand    string
	Subtitle string
	Accent   string
	CSSPath  string

	// --- Feature detection (drives conditional rendering) ---
	WebAuthn      bool           // show passkey login form
	OAuth2Buttons []OAuth2Button // show OAuth2 sign-in buttons
	ShowReg       bool           // show registration section
	// RegisterFirst renders the registration section instead of the login
	// section on load (the JS toggle links still switch between them).
	RegisterFirst bool

	// --- Security & theming (per-request) ---
	CSRFMeta  string
	CSRFField string
	// Nonce is the per-request CSP nonce rendered on the inline script tags.
	// Empty renders the scripts without a nonce attribute.
	Nonce string
	// Theme is the server-resolved color scheme ("light" or "dark", empty
	// for prefers-color-scheme). Drives the body class and ThemeScript.
	Theme string

	// --- Internal assets (consumers ignore these) ---
	authPrefix string
	inlineJS   string
	configJSON string
}

// oauthBeginURL returns the OAuth2 begin-login URL for the given provider.
func (p PageData) oauthBeginURL(provider string) string {
	return p.authPrefix + "/auth/oauth/" + provider + "/begin"
}

// faviconURI returns an inline SVG data-URI favicon using the brand initial
// and accent color. Returns templ.SafeURL to bypass templ's URL sanitizer
// (which rejects data: URIs by default).
func (p PageData) faviconURI() templ.SafeURL {
	svg := "<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'>" +
		"<rect width='100' height='100' rx='20' fill='" + p.Accent + "'/>" +
		"<text x='50' y='70' font-size='60' text-anchor='middle' fill='white'" +
		" font-family='sans-serif' font-weight='bold'>" + firstRune(p.Brand) +
		"</text></svg>"
	return templ.SafeURL("data:image/svg+xml," + svg)
}

// endpointConfig is injected as JSON into the page so the client-side JS knows
// where to send requests.
type endpointConfig struct {
	LoginBegin     string `json:"loginBegin"`
	LoginFinish    string `json:"loginFinish"`
	Register       string `json:"register"`
	RegisterBegin  string `json:"registerBegin"`
	RegisterFinish string `json:"registerFinish"`
}

// clientConfig is the full JSON blob injected via <script type="application/json">.
type clientConfig struct {
	Redirect       string         `json:"redirect"`
	Endpoints      endpointConfig `json:"endpoints"`
	CredentialName string         `json:"credentialName"`
}

// Handler serves the login page. It is safe to use as an http.Handler.
type Handler struct {
	config Config
	data   PageData
}

// New creates a login page handler from the given Config.
// Returns an error if Config.Service is nil.
func New(config Config) (*Handler, error) {
	config, err := config.withDefaults()
	if err != nil {
		return nil, err
	}
	if !config.Service.HasWebAuthn() &&
		len(config.OAuth2Buttons) == 0 &&
		len(config.Service.ConfiguredOAuth2Providers()) == 0 {
		slog.Warn(
			"loginpage: no authentication method is configured; visitors will see a setup notice instead of a login form",
			"fix",
			"configure WebAuthn or OAuth2 providers in the usermgmt ServiceConfig",
		)
	}
	return &Handler{
		config: config,
		data:   buildPageData(config, nil),
	}, nil
}

// ServeHTTP renders the login page. Only GET and HEAD are supported.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rebuild the per-request fields (the CSRF token is request-scoped under
	// nosurf; nonce and theme may come from request hooks or middleware).
	data := h.data
	data.CSRFMeta = httputil.CSRFTokenHTMLMeta(r)
	data.CSRFField = httputil.CSRFTokenFormField(r)
	if h.config.NonceFromRequest != nil {
		data.Nonce = sanitizeNonce(h.config.NonceFromRequest(r))
	}
	data.Theme = h.config.resolveTheme(r)

	renderPage(w, r, data)
}

// Mount registers the handler at the given pattern on the mux.
// Example: h.Mount(mux, "/login")
//
// The pattern is registered without a method, so it conflicts with a
// method-specific "GET /" catch-all on the same mux. Register any site-root
// index as "GET /{$}" or "/" (no method) to avoid a ServeMux panic.
func (h *Handler) Mount(mux *http.ServeMux, pattern string) {
	mux.Handle(pattern, h)
}

// NewPageData builds a [PageData] from the given Config and request, suitable
// for rendering [Page] directly in a consumer's own layout (Option B).
//
// Use this when you want full control over the HTML shell but still want the
// login form, embedded WebAuthn JS, and CSRF integration.
func NewPageData(config Config, r *http.Request) (PageData, error) {
	config, err := config.withDefaults()
	if err != nil {
		return PageData{}, err
	}
	data := buildPageData(config, r)
	return data, nil
}

// scriptSafeJSON escapes characters that let a JSON string value terminate the
// enclosing <script> element ("</script>") or start a tag ("<"), using the
// same \u003c-style escapes encoding/json v1 applied by default. Safe to apply
// to marshaled JSON: < > & can only occur inside string literals, so the
// escapes never corrupt structure.
func scriptSafeJSON(s string) string {
	return strings.NewReplacer(
		"<", `\u003c`,
		">", `\u003e`,
		"&", `\u0026`,
	).Replace(s)
}

// renderPage writes the login page HTML.
func renderPage(w http.ResponseWriter, r *http.Request, data PageData) {
	w.Header().Set("Content-Type", cqrshtmx.ContentTypeHTML)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := Page(data).Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "loginpage: render", "error", err)
	}
}

// buildPageData constructs the page data from config. If r is non-nil, CSRF
// fields are populated from the request; otherwise they are left empty.
func buildPageData(config Config, r *http.Request) PageData {
	prefix := config.AuthPrefix
	endpoints := endpointConfig{
		LoginBegin:     prefix + "/auth/webauthn/login/begin",
		LoginFinish:    prefix + "/auth/webauthn/login/finish",
		Register:       prefix + "/auth/register",
		RegisterBegin:  prefix + "/auth/webauthn/register/begin",
		RegisterFinish: prefix + "/auth/webauthn/register/finish",
	}

	cc := clientConfig{
		Redirect:       safeRedirectPath(config.Redirect),
		Endpoints:      endpoints,
		CredentialName: config.CredentialName,
	}
	configJSON, err := json.Marshal(cc)
	if err != nil { // cannot fail for this struct
		configJSON = []byte(`{"redirect":"/","endpoints":{}}`)
	}
	// The JSON is embedded in a raw <script type="application/json"> block;
	// encoding/json/v2 does not HTML-escape, so escape manually — a config
	// string containing "</script>" would otherwise break out of the block.
	configJSON = []byte(scriptSafeJSON(string(configJSON)))

	hasWebAuthn := config.Service.HasWebAuthn()

	// Auto-populate OAuth2 buttons from configured providers when not explicitly set.
	oauth2Buttons := config.OAuth2Buttons
	if len(oauth2Buttons) == 0 {
		for _, name := range config.Service.ConfiguredOAuth2Providers() {
			oauth2Buttons = append(oauth2Buttons, OAuth2ButtonFromProvider(name))
		}
	}
	hasOAuth2 := len(oauth2Buttons) > 0
	showReg := !config.NoRegistration && hasWebAuthn

	subtitle := "Sign in to your account"
	if !hasWebAuthn && !hasOAuth2 {
		subtitle = "No authentication method is configured."
	}

	data := PageData{
		Title:         config.Title,
		Brand:         config.Brand,
		Subtitle:      subtitle,
		Accent:        config.AccentColor,
		CSSPath:       config.CSSPath,
		WebAuthn:      hasWebAuthn,
		OAuth2Buttons: oauth2Buttons,
		ShowReg:       showReg,
		RegisterFirst: config.RegisterFirst,
		authPrefix:    config.AuthPrefix,
		inlineJS:      loginJS,
		configJSON:    string(configJSON),
	}

	if r != nil {
		data.CSRFMeta = httputil.CSRFTokenHTMLMeta(r)
		data.CSRFField = httputil.CSRFTokenFormField(r)
	}

	return data
}

// pageProps translates PageData into the templ-components layout props: the
// consumer's compiled Tailwind stylesheet, the brand SVG favicon, noindex,
// the per-request CSP nonce, and the resolved theme. A resolved theme wins
// SSR-first (body class + no ThemeScript, so the consumer stays the single
// source of truth); an empty theme falls back to the library's
// prefers-color-scheme behavior.
func pageProps(p PageData) layout.PageProps {
	props := layout.DefaultPageProps()
	props.Title = p.Title
	props.Description = p.Subtitle
	props.Nonce = p.Nonce
	props.CSSPath = p.CSSPath
	props.Favicon = "" // favicon renders in headExtras as a templ.SafeURL data: URI
	props.HeadContent = headExtras(p)
	props.HTMXVersion = "" // the login page ships its own script, no htmx runtime
	props.HTMXSrc = ""
	props.SEO.NoIndex = true
	props.NoThemeScript = p.Theme != ""
	if p.Theme != "" {
		props.BodyClass = utils.Class(p.Theme, props.BodyClass)
	}
	return props
}

// scriptTag renders the inline WebAuthn script with an optional nonce. The
// nonce is sanitized (base64url characters only), so templ.Raw cannot be
// abused for attribute injection even with a hostile NonceFromRequest.
func scriptTag(js, nonce string) string {
	if nonce == "" {
		return "<script>" + js + "</script>"
	}
	return `<script nonce="` + nonce + `">` + js + "</script>"
}

// jsonScriptTag renders the client-config JSON block with an optional nonce.
// The JSON itself is already script-safe (see scriptSafeJSON).
func jsonScriptTag(cfgJSON, nonce string) string {
	if nonce == "" {
		return `<script id="loginpage-config" type="application/json">` + cfgJSON + "</script>"
	}
	return `<script id="loginpage-config" type="application/json" nonce="` + nonce + `">` + cfgJSON + "</script>"
}
