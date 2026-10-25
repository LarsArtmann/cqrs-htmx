package setup

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/larsartmann/cqrs-htmx/adminui/v4"
	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	appkit "github.com/larsartmann/go-appkit"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	"github.com/larsartmann/go-sse"
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

	// SSEFilter optionally scopes the shared SSEPath feed: only events for
	// which the predicate returns true are streamed live AND replayed on
	// reconnect (replay filters fail-closed — a filter that leaked backfill
	// would be a security hole). Nil (the default) = every authenticated
	// session receives the full feed, which stays the documented contract of
	// the shared endpoint (single-tenant default; the 2026-08-30 endpoint
	// shape decision). The predicate operates on the SSE wire event (see
	// transport.DomainEventToSSE for the envelope). The DataStar feed is NOT
	// filtered — it stays a full mirror of the hub.
	SSEFilter func(sse.Event) bool

	// SSEScriptPath controls where the HTMX SSE extension script
	// (cqrshtmx.HTMXExtensionHandler(HTMXExtSSE)) is served when SSEPath is
	// set. Default ("") = "/sse.js" — HTMX SSE consumers self-host nothing,
	// the same asymmetry-free treatment DataStarScriptPath gives the DataStar
	// SDK. Set to "-" to NOT serve the script from the bundle (e.g. you load
	// the extension from a CDN); the hx-ext="sse" script tag must then point
	// at your own location. Ignored when SSEPath is empty.
	SSEScriptPath string

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

	// HealthPath is the readiness endpoint mount (default: "/health"; 503
	// while projections drain, then 200). Set to "-" to NOT mount it at all —
	// for consumers whose own health stack owns probing (go-health via the
	// health module, appkit's /health/* via RunWithAppkit). "" re-defaults to
	// "/health"; "-" is the explicit opt-out.
	HealthPath string

	// HealthChecks adds consumer-owned checks to the mounted readiness
	// endpoint (see [Config.HealthPath]), appended to the built-ins
	// ("projections", "sse-hub") and reported in the same response body —
	// one health surface, one probe, instead of choosing between the
	// bundle's 503-while-draining contract and your own endpoint on the same
	// port. Use cqrshtmx.NewNamedCheck (and NamedCheck.Timeout to bound a
	// hanging check). Names must be unique and must not collide with the
	// built-ins — New rejects duplicates fail-fast. Ignored when HealthPath
	// is "-".
	HealthChecks []cqrshtmx.NamedCheck

	// LivePath mounts a liveness endpoint (default: "" = not mounted —
	// opt-in, no surprise routes). Unlike HealthPath (readiness: 503 while
	// projections catch up), liveness answers 200 whenever the process is
	// serving — point orchestrator restart probes (k8s livenessProbe) here so
	// a draining journal never triggers a restart loop. Public, like health:
	// it exposes no data beyond "the process is up".
	LivePath string

	// Machine endpoints (all opt-in: "" = not mounted — no surprise routes).
	// Session-gated JSON surfaces for monitors, scripts, and runbooks;
	// event metadata (stream IDs, types) is not public data — the /sse 401
	// precedent (ADR-0050 reasoning).

	// EventCatalogPath serves the usermgmt event catalog as immutable JSON
	// (schemas for all domain events; 1-year cache + FNV-1a ETag). The
	// catalog serializes eagerly at New — a failure aborts construction
	// (fail-fast, mirroring attachSSE).
	EventCatalogPath string

	// ProjectionStatusPath serves live projection health as no-cache JSON
	// (per-request ETag): worker states, lag, restart counts.
	ProjectionStatusPath string

	// DebugPath serves build metadata as JSON (version, Go version, title).
	// Keep it session-gated (it is here) — build metadata leaks deployment
	// details.
	DebugPath string

	// DocsPath + DocsHandler mount the consumer's own documentation surface
	// at an opt-in, session-gated path (the shape-B seam, 2026-10-12: setup
	// pulls in zero documentation dependencies — the consumer builds the
	// docserver, e.g. from go-cqrs-lite/catalog, and hands setup the
	// handler). Set both or neither: one without the other is rejected at
	// New. DocsPath keeps the consumer's exact form — a trailing slash
	// mounts a subtree ("/docs/" serves the whole prefix), no slash mounts
	// exactly one route. Session-gated like the other machine endpoints:
	// generated documentation describes the deployment's internals (the
	// event-metadata 401 precedent); consumers wanting public docs mount
	// their handler on their own mux instead.
	DocsPath    string
	DocsHandler http.Handler

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

	// Metrics enables the go-appkit Prometheus surface on the
	// [Bundle.RunWithAppkit] serve path: a dependency-free text-exposition
	// endpoint (default /metrics) plus request-duration histogram, response
	// totals, in-flight gauge, and build-info metrics. Authentication is
	// mandatory by default — set BasicAuthUser/BasicAuthPass, or an explicit
	// AllowUnauthenticated for proxy-authed or loopback-only deployments.
	//
	// RunWithAppkit-only: [Bundle.Run] and [Bundle.RunHandler] serve via
	// httputil and ignore this field — mount your own metrics handler next to
	// the bundle's routes there (e.g. the go-cqrs-lite prometheus module, see
	// docs/guides/leveraging-go-cqrs-lite.md §2.3). Zero value (nil) = no
	// metrics route, byte-identical to before this field existed. appkit
	// validates the config (nil-auth without AllowUnauthenticated is
	// rejected) and the error surfaces from RunWithAppkit.
	//
	// 	Metrics: &appkit.MetricsConfig{
	// 		BasicAuthUser: "metrics",
	// 		BasicAuthPass: os.Getenv("METRICS_PASS"),
	// 	},
	Metrics *appkit.MetricsConfig

	// Version stamps the build served by [Bundle.RunWithAppkit]: GET /version
	// answers {"version": "..."} as JSON and the appkit_build_info metric
	// carries it as a label. RunWithAppkit-only, like Metrics — RunHandler
	// users mount their own build-info endpoint. Zero value ("") = no
	// /version route (rendered as "dev" in the build-info metric only, when
	// Metrics is also set).
	Version string

	// ExtraMiddleware composes consumer-owned middleware into the chain
	// [Bundle.Middleware] assembles, INSIDE the built-in security stack and
	// innermost before the routes: request-logging (see [Config.RequestLogging])
	// stays outermost, then security headers + CSP nonce + panic recovery (see
	// [Config.DisableSecurityMiddleware] to replace that layer), then these —
	// first entry outermost of the extras, matching cqrshtmx.Chain order.
	// Apply auth (API keys, CORS, body limits, rate limiting), metrics, or
	// tracing middleware here without giving up the bundle's serving path.
	//
	// Apps with an ordered security chain that must own the WHOLE outer stack
	// (e.g. CORS outside CSP) set DisableSecurityMiddleware and rebuild their
	// chain in ExtraMiddleware — including cqrshtmx.RecommendedSecurityMiddleware()
	// themselves if they still want parts of it. Nil (the default) = no extras,
	// byte-identical to before this field existed.
	ExtraMiddleware []func(http.Handler) http.Handler

	// DisableSecurityMiddleware removes the built-in outer security layer
	// (cqrshtmx.RecommendedSecurityMiddleware: security headers, per-request
	// CSP nonce, panic recovery) from [Bundle.Middleware]. The chain becomes
	// request-logging (if configured) → [Config.ExtraMiddleware] → routes.
	// The consumer then owns security posture entirely — compose your own
	// chain in ExtraMiddleware, including RecommendedSecurityMiddleware()
	// itself if only the ORDER needs to change. Panels' internal security
	// headers are unaffected; this only unwraps the bundle-level layer.
	DisableSecurityMiddleware bool

	// RequestLogging, when set, composes cqrshtmx.RequestLoggingSlog OUTERMOST
	// in [Bundle.Middleware] — one structured access-log line per request
	// (method, path, status, duration, plus correlation/user/request IDs when
	// present in the context). Because it rides Middleware, every serve path
	// picks it up: [Bundle.Handler], [Bundle.Run], [Bundle.RunHandler], and
	// the bundle's inner chain under RunWithAppkit (appkit's own access logs,
	// if enabled there, are separate).
	//
	// Zero value (nil) = silence, byte-identical to before this field
	// existed. Pass slog.Default() for standard logging or a dedicated
	// buffer/file logger for access logs.
	RequestLogging *slog.Logger

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
	TenantID        identitymodel.TenantID
	AdminAuthorizer func(user *identitymodel.User) error

	// DashboardAuthorizer decides whether an authenticated request may use the
	// CQRS dashboard (runs after the session gate). Return a non-nil error to
	// deny access. When nil, any authenticated user can view the dashboard.
	DashboardAuthorizer func(r *http.Request) error

	// DashboardLayout replaces the dashboard's built-in page shell (sidebar,
	// header, theme, stylesheet links) with a consumer-owned document: every
	// dashboard page renders INSIDE the app's own chrome instead of as a
	// standalone destination. See dashboardui.LayoutFunc for the contract.
	// Nil (the default) keeps the standalone dashboard shell.
	DashboardLayout dashboardui.LayoutFunc

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

	// CommandMiddleware is applied to the usermgmt service's internal command
	// dispatcher (all 20 domain commands), inside the built-in audit chain.
	// Use the factories from go-cqrs-lite/middleware/v4 — Recovery, Retry,
	// CircuitBreaker, TypedMetrics, Tracing — to harden the write path.
	// Nil (the default) keeps the dispatcher bare apart from the audit chain.
	// See docs/guides/leveraging-go-cqrs-lite.md for a recommended chain.
	CommandMiddleware []command.Middleware

	// DashboardReadOnly controls whether the CQRS dashboard allows write operations
	// (projection reset, DLQ replay). Nil = true (safe default). Set to false at your
	// own risk — the dashboard will have no authorizer unless you add one manually.
	DashboardReadOnly *bool

	// DashboardPageSize controls the number of rows per page in dashboard tables
	// (default: 0 = use dashboardui default of 50, max 200).
	DashboardPageSize int

	// LoginNoRegistration hides the registration section on the login page (default: false).
	LoginNoRegistration bool

	// LoginAutoRedirect makes the login page redirect straight to the OAuth2
	// provider when OAuth2 is the page's ONLY sign-in method (no WebAuthn,
	// exactly one provider) — no intermediate button click. With multiple
	// providers or WebAuthn present, the page renders normally (see
	// loginpage.Config.AutoRedirect).
	LoginAutoRedirect bool

	// LoginCSSPath is the URL of the compiled Tailwind stylesheet the login
	// page loads (default: "/app.css"). The login page renders templ-components
	// Tailwind utilities, so the consumer MUST compile a stylesheet scanning
	// github.com/larsartmann/cqrs-htmx/loginpage/v4 (and templ-components) and
	// serve it at this path — see README "Styling". The admin and dashboard
	// panels need nothing: they serve their own embedded bundles.
	LoginCSSPath string

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

	// DisableAuth skips the usermgmt auth handler entirely: Bundle.Auth is
	// nil and Mount registers no /auth/* routes (registration, login,
	// logout, me, webauthn, oauth2). The service is still built or adopted,
	// so the panels and SessionMiddleware keep working — this is the
	// own-login-endpoint mode: your code mints sessions against the service
	// API, the bundle serves everything else.
	//
	// Two combinations are rejected at New (fail-fast, the
	// validateServiceSources posture):
	//
	//   - DisableLogin=false — the login page's form posts to /auth/*;
	//     without the routes it is a dead form. Set DisableLogin too.
	//   - AuthHandlerConfig set — it configures endpoints that will not
	//     exist. Drop it.
	//
	// DisableService implies DisableAuth (see [Config.DisableService]).
	// ADR-0054.
	DisableAuth bool

	// DisableService builds the identity-external shell (ADR-0054): a bundle
	// with NO usermgmt.Service at all. Bundle.Service and Bundle.Auth are
	// nil, no panels are constructed, and Stores comes from EventStore and
	// EventBus — defaulting to memory.NewMemoryStore() and
	// watermill.NewEventBus(), the same defaults the service path applies.
	// What remains is the runtime shell: the serve/drain/close lifecycle
	// (Run/RunHandler/Close), the readiness composition (HealthPath with
	// consumer HealthChecks — the built-in projections/sse-hub checks skip
	// naturally on a nil service and nil hub), opt-in LivePath, and the
	// Handler/Middleware chain composition. Close is a no-op (and stays
	// idempotent) on a shell bundle.
	//
	// For consumers that authenticate against an EXTERNAL authority (PBX
	// directory, corporate SSO, their own OIDC integration) and want the
	// tested lifecycle and readiness composition without carrying a dormant
	// second user database.
	//
	// Every surface whose session gate would dereference the missing
	// service is rejected at New — usermgmt.NewSessionMiddleware(nil, ...)
	// panics on the first request carrying the configured cookie, so a
	// half-mounted gated endpoint would be a latent panic, not a feature:
	//
	//   - Service/ServiceConfig and all service-construction fields
	//     (TOTP, WebAuthn, OAuth2, SessionTTL, Logger, AsyncStartup,
	//     OnProjectionFailed, Observability, ReadModelDB) — except
	//     EventStore/EventBus, which become the shell's own inputs.
	//   - All three panels must be disabled (the dashboard dereferences
	//     Service.ProjectionHost(); admin and login need the service).
	//   - The event feeds (SSEPath, DataStarPath) and the machine
	//     endpoints (EventCatalogPath, ProjectionStatusPath, DebugPath) —
	//     their gate is the bundle session middleware. They return behind
	//     an injectable gate in a future release.
	//   - AuthHandlerConfig — nothing to configure.
	//
	// DisableService implies DisableAuth.
	DisableService bool
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

	cfg = normalizeRoutePaths(cfg)

	if cfg.SSEHeartbeatInterval == 0 {
		cfg.SSEHeartbeatInterval = 15 * time.Second
	}

	return cfg
}
