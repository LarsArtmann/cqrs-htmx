package loginpage

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
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

	// --- Security (per-request) ---
	CSRFMeta  string
	CSRFField string

	// --- Internal assets (consumers ignore these) ---
	authPrefix string
	inlineCSS  string
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
	cfg  Config
	data PageData
}

// New creates a login page handler from the given Config.
// Returns an error if Config.Service is nil.
func New(cfg Config) (*Handler, error) {
	cfg, err := cfg.withDefaults()
	if err != nil {
		return nil, err
	}
	return &Handler{
		cfg:  cfg,
		data: buildPageData(cfg, nil),
	}, nil
}

// ServeHTTP renders the login page. Only GET and HEAD are supported.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rebuild CSRF fields per-request (the token is request-scoped under nosurf).
	data := h.data
	data.CSRFMeta = cqrshtmx.CSRFTokenHTMLMeta(r)
	data.CSRFField = cqrshtmx.CSRFTokenFormField(r)

	renderPage(w, r, data)
}

// Mount registers the handler at the given pattern on the mux.
// Example: h.Mount(mux, "/login")
func (h *Handler) Mount(mux *http.ServeMux, pattern string) {
	mux.Handle(pattern, h)
}

// NewPageData builds a [PageData] from the given Config and request, suitable
// for rendering [Page] directly in a consumer's own layout (Option B).
//
// Use this when you want full control over the HTML shell but still want the
// login form, embedded WebAuthn JS, and CSRF integration.
func NewPageData(cfg Config, r *http.Request) (PageData, error) {
	cfg, err := cfg.withDefaults()
	if err != nil {
		return PageData{}, err
	}
	data := buildPageData(cfg, r)
	return data, nil
}

// renderPage writes the login page HTML.
func renderPage(w http.ResponseWriter, r *http.Request, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := Page(data).Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "loginpage: render", "error", err)
	}
}

// buildPageData constructs the page data from config. If r is non-nil, CSRF
// fields are populated from the request; otherwise they are left empty.
func buildPageData(cfg Config, r *http.Request) PageData {
	prefix := cfg.AuthPrefix
	endpoints := endpointConfig{
		LoginBegin:     prefix + "/auth/webauthn/login/begin",
		LoginFinish:    prefix + "/auth/webauthn/login/finish",
		Register:       prefix + "/auth/register",
		RegisterBegin:  prefix + "/auth/webauthn/register/begin",
		RegisterFinish: prefix + "/auth/webauthn/register/finish",
	}

	cc := clientConfig{
		Redirect:       safeRedirectPath(cfg.Redirect),
		Endpoints:      endpoints,
		CredentialName: cfg.CredentialName,
	}
	configJSON, err := json.Marshal(cc)
	if err != nil { // cannot fail for this struct
		configJSON = []byte(`{"redirect":"/","endpoints":{}}`)
	}

	hasWebAuthn := cfg.Service.HasWebAuthn()

	// Auto-populate OAuth2 buttons from configured providers when not explicitly set.
	oauth2Buttons := cfg.OAuth2Buttons
	if len(oauth2Buttons) == 0 {
		for _, name := range cfg.Service.ConfiguredOAuth2Providers() {
			oauth2Buttons = append(oauth2Buttons, OAuth2ButtonFromProvider(name))
		}
	}
	hasOAuth2 := len(oauth2Buttons) > 0
	showReg := !cfg.NoRegistration && hasWebAuthn

	subtitle := "Sign in to your account"
	if !hasWebAuthn && !hasOAuth2 {
		subtitle = "No authentication method is configured."
	}

	data := PageData{
		Title:         cfg.Title,
		Brand:         cfg.Brand,
		Subtitle:      subtitle,
		Accent:        cfg.AccentColor,
		CSSPath:       cfg.CSSPath,
		WebAuthn:      hasWebAuthn,
		OAuth2Buttons: oauth2Buttons,
		ShowReg:       showReg,
		authPrefix:    cfg.AuthPrefix,
		inlineCSS:     loginCSS,
		inlineJS:      loginJS,
		configJSON:    string(configJSON),
	}

	if r != nil {
		data.CSRFMeta = cqrshtmx.CSRFTokenHTMLMeta(r)
		data.CSRFField = cqrshtmx.CSRFTokenFormField(r)
	}

	return data
}
