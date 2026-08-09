package setup

import (
	"database/sql"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Config is the single entry point for configuring a full-stack cqrs-htmx application.
//
// Only Title is practically required (it defaults to "cqrs-htmx" if empty).
// Everything else has sensible defaults: in-memory stores, no auth providers,
// all UI panels enabled.
//
// Field types use usermgmt interfaces (TOTPProvider, WebAuthnProvider, OAuth2Provider)
// so consumers import only the auth strategy sub-modules they need.
type Config struct {
	// Auth providers (all optional). Import the sub-modules and inject:
	//
	//   import totp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
	//   TOTP: totp.New(totp.Config{Issuer: "MyApp"}),
	//
	//   import webauthn "github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4"
	//   WebAuthn: webauthn.New(webauthn.Config{RPID: "myapp.com", ...}),
	//
	//   import oauth2 "github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4"
	//   OAuth2: oauth2.New(oauth2.Config{Providers: ...}),
	TOTP     usermgmt.TOTPProvider
	WebAuthn usermgmt.WebAuthnProvider
	OAuth2   usermgmt.OAuth2Provider

	// Persistence overrides (all optional — defaults to in-memory).
	//
	// EventStore must implement event.SeekableJournal for projectionhost to work.
	// The memory store (storage/memory/v4) satisfies this. If you provide a custom
	// store that does NOT implement SeekableJournal, New returns an error.
	EventStore event.Store
	EventBus   event.Bus

	// ReadModelDB enables SQL-backed read models (optional, nil = in-memory).
	ReadModelDB *sql.DB

	// UI configuration (all optional — sensible defaults).
	Title       string // page title for all panels (default: "cqrs-htmx")
	AccentColor string // CSS accent color (default: "#0ea5e9")

	// Route paths (all optional — sensible defaults).
	AdminPath     string // default: "/admin/"
	DashboardPath string // default: "/dashboard/"
	LoginRedirect string // default: "/admin/" — where to redirect after login

	// Session configuration.
	CookieName string // default: "session"

	// Feature flags — control which panels are mounted (default: all true).
	//
	// Disable panels you don't need to reduce the route surface:
	//
	//   EnableDashboard: false, // no CQRS observability panel
	//   EnableAdmin:     false, // no user management panel
	//   EnableLogin:     false, // use your own login page
	EnableAdmin     bool
	EnableDashboard bool
	EnableLogin     bool
}

func (c Config) withDefaults() Config {
	cfg := c
	if cfg.Title == "" {
		cfg.Title = "cqrs-htmx"
	}
	if cfg.AccentColor == "" {
		cfg.AccentColor = "#0ea5e9"
	}
	if cfg.AdminPath == "" {
		cfg.AdminPath = "/admin/"
	}
	if cfg.DashboardPath == "" {
		cfg.DashboardPath = "/dashboard/"
	}
	if cfg.LoginRedirect == "" {
		cfg.LoginRedirect = "/admin/"
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "session"
	}
	return cfg
}
