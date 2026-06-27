package adminui

import (
	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
)

// Mode controls the scope of the admin panel.
type Mode int

const (
	// ModeSuperAdmin shows a global view: all users, tenants, and audit events.
	// Intended for platform operators.
	ModeSuperAdmin Mode = iota
	// ModeTenantAdmin shows a view scoped to a single tenant: dashboard,
	// members, and audit. Requires [Config.TenantID].
	ModeTenantAdmin
)

// DefaultAccentColor is the indigo used for buttons, links, and highlights when
// [Config.AccentColor] is empty.
const DefaultAccentColor = "#4f46e5"

// Config configures an admin panel. Only [Config.Service] is required; every
// other field has a sensible default applied by [New].
type Config struct {
	// Service backs the panel. Required.
	Service *usermgmt.Service

	// Title is shown in the sidebar and the browser tab. Default "Admin".
	Title string

	// BasePath is the URL prefix the panel is mounted under, without a trailing
	// slash (e.g. "/admin"). Used for every internal link. Default "/admin".
	BasePath string

	// Mode selects the panel scope. Default [ModeSuperAdmin].
	Mode Mode

	// TenantID scopes a [ModeTenantAdmin] panel to one tenant. Ignored in
	// [ModeSuperAdmin] mode. Required when Mode == [ModeTenantAdmin].
	TenantID usermgmt.TenantID

	// AccentColor overrides the highlight color (any CSS color). Default
	// [DefaultAccentColor].
	AccentColor string

	// Authorizer decides whether the authenticated user may use the panel.
	// Return a non-nil error to deny access (HTTP 403). When nil, a default
	// role-based authorizer is used — see [defaultAuthorizer]. Override this to
	// match your own role model.
	Authorizer func(user *usermgmt.User) error

	// LogoutURL is the destination of the "Sign out" link. Empty hides the link.
	LogoutURL string
}

// withDefaults returns a copy of cfg with empty fields replaced by defaults and
// validates the result.
func (cfg Config) withDefaults() (Config, error) {
	if cfg.Service == nil {
		return cfg, errConfig("Config.Service is required")
	}
	if cfg.Title == "" {
		cfg.Title = "Admin"
	}
	if cfg.BasePath == "" {
		cfg.BasePath = "/admin"
	}
	cfg.BasePath = trimTrailingSlash(cfg.BasePath)
	if cfg.AccentColor == "" {
		cfg.AccentColor = DefaultAccentColor
	}
	if cfg.Mode == ModeTenantAdmin && cfg.TenantID.Get() == "" {
		return cfg, errConfig("Config.TenantID is required for ModeTenantAdmin")
	}
	return cfg, nil
}
