package setup

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/larsartmann/cqrs-htmx/adminui/v4"
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Config is the single entry point for configuring a full-stack cqrs-htmx application.
//
// Only Title is practically required (it defaults to "cqrs-htmx" if empty).
// Everything else has sensible defaults: in-memory stores, no auth providers,
// all UI panels enabled.
//
// Field types are identity-model interfaces (TOTPProvider, WebAuthnProvider, OAuth2Provider)
// so consumers import only the auth strategy sub-modules they need.
type Config struct {
	// Auth providers (all optional). Import the sub-modules and inject:
	//
	//	import totp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
	//	TOTP: totp.New(totp.Config{Issuer: "MyApp"}),
	//
	//	import webauthn "github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4"
	//	WebAuthn: webauthn.New(webauthn.Config{RPID: "myapp.com", ...}),
	//
	//	import oauth2 "github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4"
	//	OAuth2: oauth2.New(oauth2.Config{Providers: ...}),
	TOTP     identitymodel.TOTPProvider
	WebAuthn identitymodel.WebAuthnProvider
	OAuth2   identitymodel.OAuth2Provider

	// Service, when set, adopts an already-constructed *usermgmt.Service
	// instead of building one. This is the composition seam for consumers that
	// construct the service themselves (custom stores, SecurityHooks, MaxUsers,
	// custom AuditLog, snapshotting, ...) and still want the bundle's panels,
	// session middleware, and health endpoint.
	//
	// The bundle then sources its shared infrastructure from the adopted
	// service (see Stores), and the service-construction fields below must
	// stay unset: EventStore, EventBus, ReadModelDB, TOTP, WebAuthn, OAuth2,
	// SessionTTL, AsyncStartup, and OnProjectionFailed are validated as
	// conflicts — they describe a service New would build, not one it adopted.
	//
	// Lifecycle ownership stays with the caller: [Bundle.Close] does NOT close
	// an adopted service. The caller closes it (svc.Close) after the bundle.
	Service *usermgmt.Service

	// Persistence overrides (all optional — defaults to in-memory).
	//
	// EventStore must implement event.SeekableJournal for projectionhost to work.
	// The memory store (storage/memory/v4) satisfies this. If you provide a custom
	// store that does NOT implement SeekableJournal, New returns an error.
	//
	// Ignored (rejected) when Service is set — the adopted service's own
	// infrastructure is used instead.
	EventStore event.Store
	EventBus   event.Bus

	// ReadModelDB enables SQL-backed read models (optional, nil = in-memory).
	ReadModelDB *sql.DB

	// SSEPath mounts a shared Server-Sent Events endpoint that streams every
	// event committed to the event bus as SSE (default: "" = not mounted).
	// The endpoint is session-gated (401 without an authenticated session) —
	// event metadata (stream IDs, types) is not public data.
	//
	// The payload is a small JSON envelope: {type, streamType, streamId,
	// version, occurredAt, eventId}. Use [Bundle.Broadcaster] for custom
	// fan-out (it is the same hub the endpoint serves from).
	SSEPath string

	// UI configuration (all optional — sensible defaults).
	Title       string // page title for all panels (default: "cqrs-htmx")
	AccentColor string // CSS accent color (default: "#0ea5e9")

	// Route paths (all optional — sensible defaults).
	AdminPath     string // default: "/admin/"
	DashboardPath string // default: "/dashboard/"
	LoginRedirect string // default: "/admin/" — where to redirect after login
	HealthPath    string // default: "/health" — health check endpoint

	// Session configuration.
	CookieName string        // default: "session"
	SessionTTL time.Duration // default: 0 (use usermgmt default of 24h)

	// Logger is used for structured auth event logging by the usermgmt service
	// (default: nil = slog.Default()).
	Logger *slog.Logger

	// Admin panel authorization and scope.
	//
	// AdminMode selects a global (ModeSuperAdmin, the default) or tenant-scoped
	// (ModeTenantAdmin) admin panel. In tenant mode, TenantID is required.
	//
	// AdminAuthorizer decides whether an authenticated user may use the admin
	// panel. Return a non-nil error to deny access (HTTP 403). When nil, the
	// default role-based authorizer is used.
	AdminMode       adminui.Mode
	TenantID        usermgmt.TenantID
	AdminAuthorizer func(user *usermgmt.User) error

	// DashboardAuthorizer decides whether an authenticated request may use the
	// CQRS dashboard (runs after the session gate). Return a non-nil error to
	// deny access. When nil, any authenticated user can view the dashboard.
	DashboardAuthorizer func(r *http.Request) error

	// LogoutURL is shown as a link in the admin and dashboard panels (default: "" = hidden).
	LogoutURL string

	// SSEURL enables the admin panel's real-time sync indicator (default: "" = disabled).
	SSEURL string

	// OnProjectionFailed fires when a projection worker exhausts its restart budget.
	// Use for alerting (Slack, PagerDuty, etc.). Nil = no callback.
	OnProjectionFailed func(projectionName, lastError string)

	// AsyncStartup controls whether New blocks until projections finish their
	// initial journal drain. When false (the default), the bundle blocks until
	// all projections catch up before returning — the HTTP server cannot bind
	// until drain completes (multi-minute outage on large journals).
	//
	// Set to true for production: New returns immediately, the HTTP server binds
	// right away, and the /health endpoint returns 503 (not ready) until every
	// projection reaches "live" state, then 200. Point your reverse proxy's
	// health check at /health so it retries during the catch-up window instead
	// of returning 502. See docs/guides/async-projection-startup.md.
	AsyncStartup bool

	// DashboardReadOnly controls whether the CQRS dashboard allows write operations
	// (projection reset, DLQ replay). Nil = true (safe default). Set to false at your
	// own risk — the dashboard will have no authorizer unless you add one manually.
	DashboardReadOnly *bool

	// DashboardPageSize controls the number of rows per page in dashboard tables
	// (default: 0 = use dashboardui default of 50, max 200).
	DashboardPageSize int

	// LoginNoRegistration hides the registration section on the login page (default: false).
	LoginNoRegistration bool

	// Feature flags — control which panels are mounted.
	// Go zero-value (false) = ENABLED. Set true to disable a panel.
	//
	// Disable panels you don't need to reduce the route surface:
	//
	//	DisableDashboard: true, // no CQRS observability panel
	//	DisableAdmin:     true, // no user management panel
	//	DisableLogin:     true, // use your own login page
	DisableAdmin     bool
	DisableDashboard bool
	DisableLogin     bool
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

	if cfg.HealthPath == "" {
		cfg.HealthPath = "/health"
	}

	// The standard mux only treats patterns ending in "/" as subtree patterns:
	// without the slash, "/manage" matches exactly "/manage" and every panel
	// sub-route 404s. Normalize so consumers can pass either form.
	cfg.AdminPath = ensureTrailingSlash(cfg.AdminPath)
	cfg.DashboardPath = ensureTrailingSlash(cfg.DashboardPath)

	// Health checks are exact-match routes; a trailing slash would force an
	// ugly redirect from "/health" to "/health/".
	cfg.HealthPath = trimTrailingSlash(cfg.HealthPath)

	// SSE is an exact-match endpoint, like health.
	cfg.SSEPath = trimTrailingSlash(cfg.SSEPath)

	return cfg
}

// ensureTrailingSlash appends "/" unless the path already ends with one.
func ensureTrailingSlash(s string) string {
	if !strings.HasSuffix(s, "/") {
		return s + "/"
	}

	return s
}

func trimTrailingSlash(s string) string {
	if len(s) > 1 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}

	return s
}

// validate checks the resolved config for common misconfigurations and returns
// a rejection error describing the first issue found, or nil if the config is sound.
func (c Config) validate() error {
	if err := c.validateAdoptedService(); err != nil {
		return err
	}

	if err := c.validatePathShapes(); err != nil {
		return err
	}

	if err := c.validatePathRoots(); err != nil {
		return err
	}

	return requireDistinctPaths(c)
}

// validateAdoptedService rejects configs that both adopt a Service and set
// fields that only apply when New builds the service itself. Silently
// ignoring them would be a footgun: a consumer setting TOTP alongside an
// adopted service would expect TOTP to be enabled and never learn it is not.
func (c Config) validateAdoptedService() error {
	if c.Service == nil {
		return nil
	}

	var conflicts []string

	for _, set := range []struct {
		name string
		set  bool
	}{
		{"EventStore", c.EventStore != nil},
		{"EventBus", c.EventBus != nil},
		{"ReadModelDB", c.ReadModelDB != nil},
		{"TOTP", c.TOTP != nil},
		{"WebAuthn", c.WebAuthn != nil},
		{"OAuth2", c.OAuth2 != nil},
		{"SessionTTL", c.SessionTTL != 0},
		{"AsyncStartup", c.AsyncStartup},
		{"OnProjectionFailed", c.OnProjectionFailed != nil},
	} {
		if set.set {
			conflicts = append(conflicts, set.name)
		}
	}

	if len(conflicts) > 0 {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config",
			"Service adopts an existing *usermgmt.Service, so service-construction fields are ignored — unset them: %s",
			strings.Join(conflicts, ", "))
	}

	return nil
}

// validatePathShapes rejects paths that do not start with a slash (or, for
// LoginRedirect, a URL scheme).
func (c Config) validatePathShapes() error {
	if !startsWithSlash(c.AdminPath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config", "AdminPath must start with %q (got %q)", "/", c.AdminPath)
	}

	if !startsWithSlash(c.DashboardPath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config", "DashboardPath must start with %q (got %q)", "/", c.DashboardPath)
	}

	if !startsWithSlash(c.HealthPath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config", "HealthPath must start with %q (got %q)", "/", c.HealthPath)
	}

	if c.SSEPath != "" && !startsWithSlash(c.SSEPath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config", "SSEPath must start with %q (got %q)", "/", c.SSEPath)
	}

	if !startsWithSlash(c.LoginRedirect) && !startsWithScheme(c.LoginRedirect) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config",
			"LoginRedirect must start with %q or a URL scheme (got %q)", "/", c.LoginRedirect)
	}

	if c.CookieName == "" {
		return errorfamily.NewRejection("setup.invalid_config", "CookieName must not be empty")
	}

	return nil
}

// validatePathRoots rejects mounts on the site root: the login page owns "/"
// as its catch-all, so any panel or health endpoint there would collide at
// Mount time.
func (c Config) validatePathRoots() error {
	if c.AdminPath == "/" || c.DashboardPath == "/" || c.HealthPath == "/" || c.SSEPath == "/" {
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"AdminPath, DashboardPath, HealthPath, and SSEPath must not be \"/\" — the site root is reserved for the login page",
		)
	}

	return nil
}

// requireDistinctPaths rejects configs where two mount paths resolve to the
// same route after normalization. Equal paths would make http.ServeMux panic
// inside Mount ("conflicts with pattern"); rejecting here surfaces the
// misconfiguration at New, not at first request. Overlapping-but-distinct
// paths ("/app/" vs "/app/admin/") are fine: the mux resolves them by
// longest prefix.
func requireDistinctPaths(c Config) error {
	paths := []struct{ name, path string }{
		{"AdminPath", trimTrailingSlash(c.AdminPath)},
		{"DashboardPath", trimTrailingSlash(c.DashboardPath)},
		{"HealthPath", trimTrailingSlash(c.HealthPath)},
		{"SSEPath", trimTrailingSlash(c.SSEPath)},
	}

	for i := range paths {
		for j := i + 1; j < len(paths); j++ {
			if paths[i].path == paths[j].path {
				return errorfamily.Newf(errorfamily.Rejection,
					"setup.invalid_config",
					"%s and %s both resolve to %q — routes would conflict",
					paths[i].name, paths[j].name, paths[i].path)
			}
		}
	}

	return nil
}

func startsWithSlash(s string) bool {
	return len(s) > 0 && s[0] == '/'
}

func startsWithScheme(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
