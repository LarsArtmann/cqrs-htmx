package adminui

import (
	"net/http"
	"strconv"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/templ-components/icons"
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

// SSEHubStatsSource reports live health for the consumer's shared SSE hub:
// connected subscribers and the lifetime count of events dropped for slow
// consumers (silent data loss without reconnect replay — the count existing
// to be seen). Satisfied structurally by an adapter over the consumer's
// broadcaster and drop counter; setup wires one automatically when its
// shared SSE/DataStar hub is enabled.
type SSEHubStatsSource interface {
	// SubscriberCount is the number of live SSE connections right now.
	SubscriberCount() int
	// DroppedEvents is the lifetime total of events dropped for slow
	// consumers since process start.
	DroppedEvents() int64
}

// MaxListRows is the maximum number of rows rendered in a single list page. It
// bounds memory and response size for large datasets. The UI shows a "showing N
// of M" note when a list is truncated. (Real server-side pagination is a future
// enhancement — it needs paginated read-model query methods.)
const MaxListRows = 200

// defaultBasePath is the default URL prefix for the panel when Config.BasePath
// is empty.
const defaultBasePath = "/admin"

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
	TenantID identitymodel.TenantID

	// AccentColor overrides the highlight color. Must be a CSS color literal
	// — #hex (3/4/6/8 digits), rgb()/rgba()/hsl()/hsla() with numeric
	// components, or a named CSS color; the value is interpolated into an
	// inline <style> block, so anything outside that grammar is rejected at
	// [New]. Default [DefaultAccentColor].
	AccentColor string

	// Authorizer decides whether the authenticated user may use the panel.
	// Return a non-nil error to deny access (HTTP 403). When nil, a default
	// role-based authorizer is used — see [defaultAuthorizer]. Override this to
	// match your own role model.
	Authorizer func(user *identitymodel.User) error

	// AuthorizerPosture declares what Config.Authorizer enforces, so the
	// route manifest ([Handler.Routes]) can report the truth. Empty (the
	// default) means role-gated — correct for the default authorizer and for
	// role-based custom authorizers. Set it to
	// [cqrshtmx.PostureAuthenticated] when your custom authorizer admits
	// every authenticated user (e.g. built from [RequireAuthenticated]).
	// Anything else is rejected at [New]: the guard always refuses anonymous
	// callers (401), so public and ungated would be lies.
	AuthorizerPosture cqrshtmx.Posture

	// LogoutURL is the destination of the "Sign out" link. Empty hides the link.
	LogoutURL string

	// SSEURL is the Server-Sent Events endpoint URL. When set, the panel
	// layout includes a data-sse-url attribute and renders the global sync
	// indicator (.sync-bar). Empty disables honest UI sync tracking.
	SSEURL string

	// SSEHubStats, when set, adds two operational cards to the dashboard
	// stats region (and its 30s polled refresh): live SSE subscribers and
	// the lifetime slow-consumer drop count (amber tone when any occurred —
	// drops are silent data loss, so nonzero deserves attention). Nil (the
	// default) adds nothing; wire it only when a hub exists.
	SSEHubStats SSEHubStatsSource

	// NonceFunc returns a per-request CSP nonce for inline scripts (used by
	// ToastContainer and GlobalErrorHandling). Return "" if CSP is not active.
	// When nil, the nonce is read from the request context via
	// httputil.NonceFromRequest, which works automatically when the consumer
	// adds httputil.Nonce middleware (included in [Handler.Middleware]).
	// Set this only to override the default behavior with a custom nonce source.
	NonceFunc func(*http.Request) string

	// ExtraNav appends consumer-owned entries after the built-in sidebar
	// items, so surfaces mounted next door stay discoverable from the panel
	// navigation (the same seam dashboardui ships). Href is rendered
	// VERBATIM — no BasePath prefix. Empty (the default) changes nothing.
	ExtraNav []NavEntry

	// DashboardURL cross-links the sibling dashboardui panel (event-store
	// introspection: journal, projections, time-travel) as a sidebar entry.
	// The two dashboards are complementary — most apps mount both — and this
	// makes the pair one click apart. Empty (the default) hides the entry.
	// For anything more than one link, prefer [Config.ExtraNav].
	DashboardURL string
}

// NavEntry is one consumer-owned sidebar navigation entry, appended after
// the built-in items via [Config.ExtraNav].
type NavEntry struct {
	// Label is the display text ("Support").
	Label string

	// Href is the link target, rendered VERBATIM: unlike built-in nav
	// entries it is NOT prefixed with BasePath. Point it at a sibling
	// surface outside the mounted subtree ("/dashboard"), an absolute URL
	// ("https://docs.example.com"), or build a panel-internal path yourself
	// with Config.BasePath.
	Href string

	// Icon is the sidebar icon (e.g. icons.Link). The zero value renders
	// the link icon.
	Icon icons.Name
}

// withDefaults returns a copy of config with empty fields replaced by defaults and
// validates the result.
func (config Config) withDefaults() (Config, error) {
	if config.Service == nil {
		return config, errConfig("Config.Service is required")
	}
	if config.Title == "" {
		config.Title = "Admin"
	}
	if config.BasePath == "" {
		config.BasePath = defaultBasePath
	}
	config.BasePath = trimTrailingSlash(config.BasePath)
	if config.AccentColor == "" {
		config.AccentColor = DefaultAccentColor
	}
	if !isValidAccentColor(config.AccentColor) {
		return config, errConfig(
			"Config.AccentColor must be a CSS color literal (#hex, rgb()/rgba()/hsl()/hsla() with numeric components, or a named color); got " + config.AccentColor,
		)
	}
	if config.Mode == ModeTenantAdmin && config.TenantID.Get() == "" {
		return config, errConfig("Config.TenantID is required for ModeTenantAdmin")
	}
	switch config.AuthorizerPosture {
	case "", cqrshtmx.PostureAuthenticated, cqrshtmx.PostureRoleGated:
	case cqrshtmx.PosturePublic, cqrshtmx.PostureUngated:
		return config, errConfig(
			"Config.AuthorizerPosture may not be public or ungated — the guard always refuses anonymous callers, so both would be lies",
		)
	default:
		return config, errConfig(
			"Config.AuthorizerPosture must be empty (role-gated default), cqrshtmx.PostureAuthenticated, or cqrshtmx.PostureRoleGated",
		)
	}
	if err := validateExtraNav(config.ExtraNav); err != nil {
		return config, err
	}
	return config, nil
}

// validateExtraNav rejects consumer nav entries without a label or href — a
// sidebar link with empty text or no destination is a silent lie.
func validateExtraNav(entries []NavEntry) error {
	for i, entry := range entries {
		if entry.Label == "" {
			return errConfig("Config.ExtraNav[" + strconv.Itoa(i) + "].Label must not be empty")
		}
		if entry.Href == "" {
			return errConfig("Config.ExtraNav[" + strconv.Itoa(i) + "].Href must not be empty")
		}
	}
	return nil
}
