package setup

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/larsartmann/cqrs-htmx/adminui/v4"
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/httputil"
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
	//	// secret-less PKCE clients: ProviderConfig.ClientType = oauth2.ClientTypePublic
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
	// SessionTTL, Logger, AsyncStartup, and OnProjectionFailed are validated
	// as conflicts — they describe a service New would build, not one it
	// adopted.
	//
	// Lifecycle ownership stays with the caller: [Bundle.Close] does NOT close
	// an adopted service. The caller closes it (svc.Close) after the bundle.
	//
	// Precedence: Service (adopt) > ServiceConfig (full override) > flattened
	// fields (convenience). See [Config.ServiceConfig].
	Service *usermgmt.Service

	// ServiceConfig, when set (and Service is nil), is passed to
	// usermgmt.NewService verbatim. This is the escape hatch for service knobs
	// the flattened fields cannot express — MaxUsers, TokenPepper,
	// SecurityHooks, CheckpointStore, SnapshotConfig, SessionStore, Lockout,
	// EmailVerification, custom SessionStore/AuditLog, ... — while still
	// getting the bundle's panels, session middleware, and health endpoint.
	//
	// ServiceConfig and Service are mutually exclusive, and ServiceConfig
	// conflicts with the same flattened service-construction fields as Service
	// does (EventStore, EventBus, ReadModelDB, TOTP, WebAuthn, OAuth2,
	// SessionTTL, Logger, AsyncStartup, OnProjectionFailed) — set those inside
	// ServiceConfig instead, where EventStore defaults to in-memory and
	// EventBus to the watermill bus when nil.
	//
	// Unlike an adopted Service, a service built from ServiceConfig IS closed
	// by [Bundle.Close]. The bundle applies exactly one default on top of the
	// override: when ServiceConfig.AuditLog is nil, an in-memory audit log
	// (usermgmt.NewAuditLog) is registered, matching the flattened-path
	// behavior.
	ServiceConfig *usermgmt.ServiceConfig

	// Observability, when set, wires a pre-built OTel middleware bundle
	// (github.com/larsartmann/go-cqrs-lite/middleware) into the event bus of
	// the service New constructs: bundle.Publish() runs on every event
	// publish (producer spans) and bundle.Event() on every event handle
	// (consumer spans + metrics). Zero value (nil) = no OTel middleware, so
	// default behavior is byte-identical to before this field existed.
	//
	// This is a flattened-path convenience, like TOTP or EventStore: it is
	// REJECTED as a conflict when Service or ServiceConfig is also set.
	// ServiceConfig users wire the same middleware themselves:
	//
	//	svcCfg.SecurityHooks.PublishMiddleware = bundle.Publish()
	//	svcCfg.SecurityHooks.HandlerMiddleware = bundle.Event()
	//
	// When both the bundle and ServiceConfig-style SecurityHooks would apply
	// (flattened path has no SecurityHooks, so this is composition with the
	// manual path above), the bundle's middleware runs OUTERMOST: the OTel
	// span wraps any signing/encryption work. Create the bundle after
	// cqrsotel.Setup so its tracer resolves the registered global provider:
	//
	//	cqrsotel.Setup(cqrsotel.WithService("my-app", "1.0.0", "local"))
	//	bundle, err := middleware.NewOTelBundle(
	//		cqrsotel.NewTracer("my-app"), cqrsotel.NewMeter("my-app"))
	//	// ...
	//	setup.New(setup.Config{Observability: bundle, /* ... */})
	//
	// See docs/guides/leveraging-go-cqrs-lite.md §2 for the full OTel wiring
	// story (free domain spans, HTTP root spans, correlation).
	Observability *middleware.OTelBundle

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
	// The flattened path always uses the SQLite read-model dialect: the
	// handle must speak SQLite (modernc.org/sqlite "sqlite" or
	// mattn/go-sqlite3 "sqlite3"), which New verifies with a one-query
	// dialect probe. For Postgres or MySQL read models, use
	// [Config.ServiceConfig] with ReadModelDB AND ReadModelDialect set
	// together ("postgres"/"pgx"/"mysql").
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

	// SSEHeartbeatInterval controls how often connected SSE clients receive
	// keep-alive comment frames on the SSEPath endpoint. A non-positive value
	// disables heartbeats. Default: 15 seconds.
	SSEHeartbeatInterval time.Duration

	// SSEMaxReplay caps the number of events replayed to a first-time SSE
	// subscriber (no Last-Event-ID). A value of 0 means "use the transport
	// default" (1000). Set to a positive int to bound the initial backfill
	// window and prevent sending the entire journal history on first connect.
	SSEMaxReplay int

	// DataStarPath mounts a DataStar SSE feed that streams the same events as
	// the SSEPath feed, encoded as DataStar patches for the Datastar SDK
	// (default: "" = not mounted). Both feeds fan out from ONE shared hub
	// ([Bundle.Broadcaster]) — a single broadcast reaches HTMX and DataStar
	// clients simultaneously (ADR-0050).
	//
	// The endpoint is session-gated (401 without an authenticated session),
	// mirroring the SSEPath contract: event metadata (stream IDs, types) is
	// not public data. The feed is live fan-out only — /sse remains the
	// replay-capable endpoint (Last-Event-ID backfill from the journal).
	//
	// Requires the event-bus bridge (created automatically): with only
	// DataStarPath set, [Bundle.Broadcaster] still exists and bridges the
	// event bus, but no /sse route is mounted.
	DataStarPath string

	// DataStarScriptPath controls where the DataStar SDK script
	// ([datastar.ScriptHandler]) is served when DataStarPath is set.
	// Default ("") = "/datastar.js". Set to "-" to NOT serve the script from
	// the bundle (e.g. you load the SDK from a CDN); the SDK script tag must
	// then point at your own location. Ignored when DataStarPath is empty.
	DataStarScriptPath string

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

	// CSRF overrides the CSRF protection applied to the admin panel's mutation
	// endpoints (default: nil = httputil.CSRFConfig{}, which issues cookies
	// without the Secure flag and logs a warning — fine for local HTTP dev).
	//
	// For HTTPS deployments set a production config so cookies carry the Secure
	// flag and the warning disappears:
	//
	//	CSRF: &httputil.CSRFConfig{Secure: true},
	//
	// The value is passed to httputil.CSRFMiddleware verbatim (which validates
	// it at mount time), so every httputil knob — TrustedOrigins, TrustedProxies,
	// cookie/header names — is available.
	CSRF *httputil.CSRFConfig

	// AuthHandlerConfig, when set, is merged into the usermgmt.HandlerConfig
	// the bundle constructs for its auth endpoints (/auth/*). This is the seam
	// for the HTTP-layer hardening usermgmt offers but the flattened fields
	// cannot express: the six per-group rate limiters
	// (WebAuthnRateLimit, RegistrationRateLimit, TOTPRateLimit,
	// VerificationRateLimit, ImportRateLimit, OAuthRateLimit — brute-force
	// protection for the passwordless ceremonies), the cookie Secure flag and
	// SessionMaxAge, the per-request handler Timeout, the OAuth2 redirect
	// URLs, and the ImportExportAuthorizer.
	//
	// This does NOT violate the ServiceConfig-verbatim policy
	// (see [Config.ServiceConfig]): usermgmt.HandlerConfig is a different
	// struct — HTTP-layer, constructed only by this bundle — so there is no
	// verbatim form a consumer could pass instead.
	//
	// Zero value (nil) = today's literal: HandlerConfig{CookieName: cfg.CookieName}
	// — byte-identical defaults, no rate limits.
	//
	// Merging: an empty CookieName inside AuthHandlerConfig inherits
	// [Config.CookieName] (the auth handler MUST write the same cookie the
	// session middleware reads); a non-empty CookieName that differs from
	// Config.CookieName is rejected at New. The value is copied — later
	// mutations of the caller's struct do not leak into the bundle. It composes
	// with all three service sources (Service, ServiceConfig, flattened).
	//
	// 	AuthHandlerConfig: &usermgmt.HandlerConfig{
	// 		WebAuthnRateLimit: usermgmt.RateLimitConfig{
	// 			Enabled: true, MaxRequests: 10, Window: time.Minute,
	// 		},
	// 	},
	AuthHandlerConfig *usermgmt.HandlerConfig

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

	// DataStar endpoints are exact-match endpoints, like SSE.
	cfg.DataStarPath = trimTrailingSlash(cfg.DataStarPath)
	cfg.DataStarScriptPath = trimTrailingSlash(cfg.DataStarScriptPath)

	if cfg.DataStarPath != "" && cfg.DataStarScriptPath == "" {
		cfg.DataStarScriptPath = "/datastar.js"
	}

	if cfg.SSEHeartbeatInterval == 0 {
		cfg.SSEHeartbeatInterval = 15 * time.Second
	}

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
	if err := c.validateServiceSources(); err != nil {
		return err
	}

	if err := c.validateReadModelDialect(); err != nil {
		return err
	}

	if err := c.validateAuthHandlerConfig(); err != nil {
		return err
	}

	if c.SSEMaxReplay < 0 {
		return errorfamily.Newf(
			errorfamily.Rejection,
			"setup.invalid_config",
			"SSEMaxReplay must not be negative — 0 uses the transport default (1000), a positive value caps the backfill; got %d",
			c.SSEMaxReplay,
		)
	}

	if err := c.validatePathShapes(); err != nil {
		return err
	}

	if err := c.validatePathRoots(); err != nil {
		return err
	}

	return requireDistinctPaths(c)
}

// validateServiceSources rejects configs that set more than one of the three
// service sources, or combine an explicit source (Service or ServiceConfig)
// with flattened service-construction fields. Silently ignoring them would be
// a footgun: a consumer setting TOTP alongside an adopted service would
// expect TOTP to be enabled and never learn it is not.
func (c Config) validateServiceSources() error {
	switch {
	case c.Service != nil && c.ServiceConfig != nil:
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"Service and ServiceConfig are mutually exclusive — Service adopts an existing *usermgmt.Service, ServiceConfig builds one; pick one",
		)
	case c.Service != nil:
		if conflicts := c.serviceConstructionConflicts(); len(conflicts) > 0 {
			return errorfamily.Newf(
				errorfamily.Rejection,
				"setup.invalid_config",
				"Service adopts an existing *usermgmt.Service, so service-construction fields are ignored — unset them: %s",
				strings.Join(conflicts, ", "),
			)
		}
	case c.ServiceConfig != nil:
		if conflicts := c.serviceConstructionConflicts(); len(conflicts) > 0 {
			return errorfamily.Newf(
				errorfamily.Rejection,
				"setup.invalid_config",
				"ServiceConfig fully specifies the service, so flattened service-construction fields are ignored — set them inside ServiceConfig instead: %s",
				strings.Join(conflicts, ", "),
			)
		}
	}

	return nil
}

// readModelDialectProbeTimeout bounds the dialect probe so a hung database
// cannot stall New indefinitely. The flattened path connects to ReadModelDB
// during New anyway (view-store auto-migration), so the probe adds no new
// connection behavior — only a bound on how long the first one may take.
const readModelDialectProbeTimeout = 5 * time.Second

// validateReadModelDialect rejects a ReadModelDB handle that does not speak
// SQLite. The flattened convenience path never sets ReadModelDialect, so
// usermgmt resolves "" to the SQLite read-model constructor family — against
// a Postgres or MySQL handle that family's DDL mostly parses and silently
// creates SQLite-schema tables, corrupting the deployment far from the
// misconfiguration. Failing at New with a pointer to the supported path
// (ServiceConfig.ReadModelDialect) is the honest alternative.
//
// Detection is a dialect probe, not a driver-name check: database/sql does
// not expose the registered driver name, so the validator asks the database
// itself. "SELECT sqlite_version()" answers successfully on every SQLite
// engine and fails on Postgres/MySQL. An unreachable database passes
// validation (it cannot be disproven) and fails later inside usermgmt with
// its own infrastructure error — a rejection here would lie about the cause.
//
// Must run after validateServiceSources: a non-nil ReadModelDB can only
// reach this check on the flattened path (the explicit service sources
// reject it as a conflict there).
func (c Config) validateReadModelDialect() error {
	if c.ReadModelDB == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), readModelDialectProbeTimeout)
	defer cancel()

	var version string
	probeErr := c.ReadModelDB.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version)
	if probeErr == nil {
		return nil
	}

	// The probe failed for one of two reasons: the database is reachable and
	// does not speak SQLite (reject with the actionable pointer), or the
	// connection itself failed (pass — see the doc comment). Ping separates
	// the two cases.
	if err := c.ReadModelDB.PingContext(ctx); err != nil {
		return nil
	}

	return errorfamily.Newf(
		errorfamily.Rejection,
		"setup.invalid_config",
		"ReadModelDB does not speak SQLite (dialect probe failed: %s) — the flattened config path always builds SQLite-dialect read models; for this database use Config.ServiceConfig with ReadModelDB and ReadModelDialect set together (\"postgres\", \"pgx\", or \"mysql\")",
		probeErr,
	)
}

// validateAuthHandlerConfig rejects an AuthHandlerConfig whose CookieName
// disagrees with the flattened Config.CookieName. The auth handler must write
// the same cookie the session middleware reads — a mismatch silently breaks
// every authenticated request while the login flow still looks fine (the
// historical default-composition bug the SQL restart contract test guards).
// An empty CookieName inside AuthHandlerConfig is NOT a conflict: it inherits
// Config.CookieName at construction (see resolveAuthHandlerConfig).
//
// Must run after withDefaults (New applies defaults first), so an unset
// Config.CookieName has already become "session" by the time this compares.
func (c Config) validateAuthHandlerConfig() error {
	if c.AuthHandlerConfig == nil {
		return nil
	}

	if name := c.AuthHandlerConfig.CookieName; name != "" && name != c.CookieName {
		return errorfamily.Newf(
			errorfamily.Rejection,
			"setup.invalid_config",
			"AuthHandlerConfig.CookieName %q does not match Config.CookieName %q — the auth handler must write the same cookie the session middleware reads; leave it empty to inherit",
			name,
			c.CookieName,
		)
	}

	return nil
}

// serviceConstructionConflicts lists the flattened service-construction fields
// that are set. Only meaningful when Service or ServiceConfig is also set.
func (c Config) serviceConstructionConflicts() []string {
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
		{"Logger", c.Logger != nil},
		{"AsyncStartup", c.AsyncStartup},
		{"OnProjectionFailed", c.OnProjectionFailed != nil},
		{"Observability", c.Observability != nil},
	} {
		if set.set {
			conflicts = append(conflicts, set.name)
		}
	}

	return conflicts
}

// validatePathShapes rejects paths that do not start with a slash (or, for
// LoginRedirect, a URL scheme).
func (c Config) validatePathShapes() error {
	if !startsWithSlash(c.AdminPath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config", "AdminPath must start with %q (got %q)", "/", c.AdminPath)
	}

	if !startsWithSlash(c.DashboardPath) {
		return errorfamily.Newf(
			errorfamily.Rejection,
			"setup.invalid_config",
			"DashboardPath must start with %q (got %q)",
			"/",
			c.DashboardPath,
		)
	}

	if !startsWithSlash(c.HealthPath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config", "HealthPath must start with %q (got %q)", "/", c.HealthPath)
	}

	if err := c.validateOptionalFeedPaths(); err != nil {
		return err
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

// validateOptionalFeedPaths checks the optional SSE/DataStar mount paths —
// empty means "feature disabled", which is always valid.
func (c Config) validateOptionalFeedPaths() error {
	if c.SSEPath != "" && !startsWithSlash(c.SSEPath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config", "SSEPath must start with %q (got %q)", "/", c.SSEPath)
	}

	if c.DataStarPath != "" && !startsWithSlash(c.DataStarPath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config", "DataStarPath must start with %q (got %q)", "/", c.DataStarPath)
	}

	if c.DataStarScriptPath != "" && c.DataStarScriptPath != "-" && !startsWithSlash(c.DataStarScriptPath) {
		return errorfamily.Newf(errorfamily.Rejection,
			"setup.invalid_config",
			"DataStarScriptPath must start with %q, be empty, or be \"-\" to disable (got %q)",
			"/",
			c.DataStarScriptPath,
		)
	}

	return nil
}

// validatePathRoots rejects mounts on the site root: the login page owns "/"
// as its catch-all, so any panel or health endpoint there would collide at
// Mount time.
func (c Config) validatePathRoots() error {
	if c.AdminPath == "/" || c.DashboardPath == "/" || c.HealthPath == "/" || c.SSEPath == "/" ||
		c.DataStarPath == "/" || (c.DataStarScriptPath == "/") {
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"AdminPath, DashboardPath, HealthPath, SSEPath, DataStarPath, and DataStarScriptPath must not be \"/\" — the site root is reserved for the login page",
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
		{"DataStarPath", trimTrailingSlash(c.DataStarPath)},
		{"DataStarScriptPath", trimTrailingSlash(c.DataStarScriptPath)},
	}

	for i := range paths {
		for j := i + 1; j < len(paths); j++ {
			// Unset optional paths ("") never conflict with each other — two
			// disabled features are not a route collision.
			if paths[i].path == "" || paths[j].path == "" {
				continue
			}

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
