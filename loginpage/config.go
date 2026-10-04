package loginpage

import (
	"net/http"
	"strings"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// Theme values accepted by [Config.Theme] and [Config.ThemeFromRequest].
const (
	ThemeDark  = "dark"
	ThemeLight = "light"
)

// DefaultAccentColor is the indigo filling the generated SVG favicon's
// brand square when [Config.AccentColor] is empty. (It no longer colors
// buttons or highlights: since the templ-components rewrite all page
// styling is Tailwind.)
const DefaultAccentColor = "#4f46e5"

// knownProviderLabels maps common provider names to user-friendly display
// labels for sign-in buttons. Providers not in this map get a capitalized name.
//
//nolint:gochecknoglobals // static lookup table, provider names are intrinsic strings
var knownProviderLabels = map[string]string{
	"google":    "Google",
	"github":    "GitHub",
	"microsoft": "Microsoft",
	"apple":     "Apple",
	"gitlab":    "GitLab",
	"facebook":  "Facebook",
	"amazon":    "Amazon",
	"linkedin":  "LinkedIn",
	"twitter":   "Twitter",
	"discord":   "Discord",
}

// ProviderDisplayName returns a user-friendly label for an OAuth2 provider.
// Known providers (google, github, etc.) use a curated label; unknown
// providers get a title-cased name.
func ProviderDisplayName(provider string) string {
	if label, ok := knownProviderLabels[provider]; ok {
		return label
	}
	if provider == "" {
		return ""
	}
	return strings.ToUpper(provider[:1]) + provider[1:]
}

// OAuth2ButtonFromProvider creates an OAuth2Button with an auto-generated
// label from the provider name.
func OAuth2ButtonFromProvider(provider string) OAuth2Button {
	return OAuth2Button{
		Provider: provider,
		Label:    "Sign in with " + ProviderDisplayName(provider),
	}
}

// OAuth2Button describes a single OAuth2/OIDC sign-in button rendered on the
// login page. The [Provider] string matches the {provider} path segment in the
// usermgmt OAuth2 routes: GET /auth/oauth/{provider}/begin.
type OAuth2Button struct {
	// Provider is the provider name used in the URL path (e.g. "google",
	// "github"). Must match a key in the oauth2.Provider configuration.
	Provider string

	// Label is the button text (e.g. "Sign in with Google").
	Label string
}

// Config configures the login page handler. Only [Config.Service] is required;
// every other field has a sensible default applied by [New].
type Config struct {
	// Service provides user management and auth-method detection. Required.
	// The page reads Service.HasWebAuthn / Service.HasOAuth2 to decide which
	// UI sections to render.
	Service *usermgmt.Service

	// Title is the page <title> and main heading. Default "Sign in".
	Title string

	// Brand is the app/site name shown above the form. Default: same as Title.
	Brand string

	// Redirect is the URL to navigate to after successful login or registration.
	// Must be a root-relative path (e.g. "/dashboard"). Default "/".
	Redirect string

	// AccentColor overrides the accent color of the generated SVG favicon
	// (the brand-initial square; any CSS color without markup or quotes).
	// It does NOT affect buttons or other page styling — those are Tailwind
	// classes compiled by the consumer (see README "Styling").
	// Default [DefaultAccentColor].
	AccentColor string

	// CSSPath is the URL of the consumer's compiled Tailwind stylesheet. The
	// page is built entirely on templ-components, whose classes only exist
	// after the consumer compiles Tailwind v4 with @source scanning of this
	// package (see README "Styling"). Default "/app.css".
	CSSPath string

	// Theme forces a color scheme ("light" or "dark") instead of following
	// prefers-color-scheme. Only set it when the consumer owns the theme
	// decision; for cookie-driven theming use [Config.ThemeFromRequest].
	Theme string

	// ThemeFromRequest resolves the theme per request (e.g. from a cookie).
	// When set it wins over [Config.Theme]; returning "" falls back to
	// prefers-color-scheme. A resolved theme is applied SSR-first (body
	// class, no ThemeScript), so the consumer stays the single source of truth.
	ThemeFromRequest func(*http.Request) string

	// NonceFromRequest returns the per-request CSP nonce for the page's
	// inline script tags. Required by consumers whose CSP has no
	// 'unsafe-inline' for script-src (e.g. a nonce-based middleware); the
	// value must match the nonce the middleware wrote into the
	// Content-Security-Policy header, or the browser blocks the scripts.
	// The value is sanitized to base64url characters before rendering.
	// Empty (default) renders the scripts without a nonce attribute.
	NonceFromRequest func(*http.Request) string

	// NoRegistration hides the registration section. By default, registration
	// is shown when WebAuthn is configured.
	NoRegistration bool

	// AuthPrefix is the URL prefix for auth API endpoints. Default "".
	// Endpoints are at /auth/... by default. Set to "/api" for /api/auth/...
	AuthPrefix string

	// OAuth2Buttons lists the OAuth2 providers to show as sign-in buttons.
	// Each button links to {AuthPrefix}/auth/oauth/{Provider}/begin.
	// A nil or empty slice auto-detects buttons from the providers configured
	// on [Config.Service]; set [Config.NoOAuth2] to hide them all.
	OAuth2Buttons []OAuth2Button

	// NoOAuth2 force-hides every OAuth2 sign-in button on the page, including
	// the buttons auto-detected from [Config.Service]'s configured providers —
	// for consumers that expose OAuth2 only elsewhere (a dedicated SSO page,
	// a different route) but still configure providers on the service.
	// Mutually exclusive with [Config.OAuth2Buttons] (rejected at [New]).
	NoOAuth2 bool

	// CredentialName is the label stored with newly registered WebAuthn
	// credentials (the "credential_name" query parameter). Default "Passkey".
	CredentialName string

	// RegisterFirst renders the registration section instead of the login
	// section on load (e.g. when /register is a distinct route). The JS
	// toggle links still switch between sections.
	RegisterFirst bool
}

func (config Config) withDefaults() (Config, error) {
	if config.Service == nil {
		return config, errConfig("Config.Service is required")
	}
	if config.Title == "" {
		config.Title = "Sign in"
	}
	if config.Brand == "" {
		config.Brand = config.Title
	}
	if config.Redirect == "" {
		config.Redirect = "/"
	}
	if config.AccentColor == "" {
		config.AccentColor = DefaultAccentColor
	}
	if config.CredentialName == "" {
		config.CredentialName = "Passkey"
	}
	if err := validateAccentColor(config.AccentColor); err != nil {
		return config, err
	}
	if err := validateStylesheetURL(config.CSSPath); err != nil {
		return config, err
	}
	if config.NoOAuth2 && len(config.OAuth2Buttons) > 0 {
		return config, errConfig("Config.NoOAuth2 and Config.OAuth2Buttons are mutually exclusive: NoOAuth2 hides every OAuth2 button, an explicit list shows specific ones")
	}
	if config.CSSPath == "" {
		config.CSSPath = "/app.css"
	}
	if config.Theme != "" && config.Theme != ThemeLight && config.Theme != ThemeDark {
		return config, errConfig(`Config.Theme must be "light" or "dark" (or empty to follow prefers-color-scheme)`)
	}
	config.AuthPrefix = trimTrailingSlash(config.AuthPrefix)
	return config, nil
}

// resolveTheme applies the per-request theme hook over the static theme.
// Returns "" when neither is set or the value is unrecognized — the page
// then follows prefers-color-scheme via the library's ThemeScript.
func (config Config) resolveTheme(r *http.Request) string {
	if config.ThemeFromRequest != nil && r != nil {
		if theme := config.ThemeFromRequest(r); theme == ThemeLight || theme == ThemeDark {
			return theme
		}
	}
	if config.Theme == ThemeLight || config.Theme == ThemeDark {
		return config.Theme
	}
	return ""
}
