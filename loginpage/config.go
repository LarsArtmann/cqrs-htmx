package loginpage

import (
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// DefaultAccentColor is the indigo used for buttons and highlights when
// [Config.AccentColor] is empty.
const DefaultAccentColor = "#4f46e5"

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

	// AccentColor overrides the button/highlight color (any CSS color).
	// Default [DefaultAccentColor].
	AccentColor string

	// CSSPath is an optional URL to a consumer stylesheet, linked after the
	// built-in inline styles so consumers can override.
	CSSPath string

	// NoRegistration hides the registration section. By default, registration
	// is shown when WebAuthn is configured.
	NoRegistration bool

	// AuthPrefix is the URL prefix for auth API endpoints. Default "".
	// Endpoints are at /auth/... by default. Set to "/api" for /api/auth/...
	AuthPrefix string

	// OAuth2Buttons lists the OAuth2 providers to show as sign-in buttons.
	// Each button links to {AuthPrefix}/auth/oauth/{Provider}/begin.
	// Empty slice hides all OAuth2 buttons.
	OAuth2Buttons []OAuth2Button

	// CredentialName is the label stored with newly registered WebAuthn
	// credentials (the "credential_name" query parameter). Default "Passkey".
	CredentialName string
}

func (cfg Config) withDefaults() (Config, error) {
	if cfg.Service == nil {
		return cfg, errConfig("Config.Service is required")
	}
	if cfg.Title == "" {
		cfg.Title = "Sign in"
	}
	if cfg.Brand == "" {
		cfg.Brand = cfg.Title
	}
	if cfg.Redirect == "" {
		cfg.Redirect = "/"
	}
	if cfg.AccentColor == "" {
		cfg.AccentColor = DefaultAccentColor
	}
	if cfg.CredentialName == "" {
		cfg.CredentialName = "Passkey"
	}
	cfg.AuthPrefix = trimTrailingSlash(cfg.AuthPrefix)
	return cfg, nil
}
