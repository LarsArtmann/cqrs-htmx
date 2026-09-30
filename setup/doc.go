// Package setup provides a one-call composition root for cqrs-htmx applications.
//
// It eliminates the boilerplate of manually wiring usermgmt.Service, adminui.Handler,
// dashboardui.Dashboard, and loginpage.Handler — plus the middleware chain that connects them.
//
// # Quick start
//
//	bundle, err := setup.New(setup.Config{
//	    Title:   "My App",
//	    TOTP:    totp.New(totp.Config{Issuer: "MyApp"}),
//	})
//	if err != nil { log.Fatal(err) }
//
//	if err := bundle.Run(ctx, ":8080"); err != nil { log.Fatal(err) }
//
// [Bundle.Run] mounts all routes, serves with safe timeouts (SSE-compatible),
// drains gracefully when ctx is cancelled, and closes the bundle.
//
// # What you get
//
//   - /auth/* — registration, login (WebAuthn/TOTP/OAuth2), logout, me
//   - /admin/* — admin dashboard (session + CSRF gated, 401 otherwise)
//   - /dashboard/* — CQRS/ES observability (session gated, 401 otherwise)
//   - /health — readiness check (503 while projections are draining)
//   - / — login page
//
// # Convenience methods
//
// [Bundle.Run] is the one-liner: mount, serve, graceful shutdown, close.
// [Bundle.RunHandler] does the same for a handler you compose yourself
// (e.g. [Bundle.Handler] (mux) with your own routes added).
//
// Alternatively, call [Bundle.Mount] and [Bundle.Middleware] separately for full control:
//
//	mux := http.NewServeMux()
//	bundle.Mount(mux)
//	http.ListenAndServe(":8080", bundle.Middleware()(mux))
//
// The chain [Bundle.Middleware] builds is request-logging (if set) →
// security → [Config.ExtraMiddleware] → routes; setting
// [Config.DisableSecurityMiddleware] drops the security layer so you can
// rebuild it in [Config.ExtraMiddleware]. Deciding between this bundle and
// hand-wiring the modules? Read docs/guides/setup-vs-hand-wiring.md.
//
// # Customization
//
// Every sub-component is exposed on the [Bundle] struct. Override panels, add custom
// routes, or swap middleware after construction:
//
//	bundle.Admin.SetAccentColor("#ff0000")
//	mux.Handle("POST /orders", bundle.SessionMiddleware(app.Command("CreateOrder", ...)))
//
// # Configuration
//
// Config fields cover the most common production needs:
//
//   - [Config.SessionTTL] — session cookie lifetime (default: 24h via usermgmt)
//   - [Config.AuthHandlerConfig] — HTTP-layer auth hardening (rate limits for
//     the passwordless ceremonies, cookie flags, OAuth2 redirects) merged into
//     the /auth/* handler config; nil keeps the historical defaults
//   - [Config.Metrics] / [Config.Version] — Prometheus /metrics (Basic-Auth
//     by default) and the /version JSON build stamp on the RunWithAppkit
//     serve path; nil/"" mount nothing
//   - [Config.EventCatalogPath] / [Config.ProjectionStatusPath] /
//     [Config.DebugPath] — opt-in session-gated JSON machine endpoints
//     (event catalog, live projection statuses, build metadata)
//   - [Config.LivePath] — opt-in public liveness probe (always-200);
//     [Config.HealthPath] "-" opts the readiness endpoint out entirely;
//     [Config.HealthChecks] appends consumer checks to the built-ins on
//     that endpoint (one probe surface for dependencies too)
//   - [Config.SSEScriptPath] — HTMX SSE extension served at "/sse.js" when
//     SSEPath is set ("-" opts out)
//   - [Config.Logger] — structured auth event logging (default: slog.Default())
//   - [Config.LogoutURL] — logout link shown in admin and dashboard panels
//   - [Config.SSEURL] — enables admin panel real-time sync indicator
//   - [Config.OnProjectionFailed] — callback when a projection exhausts restarts
//   - [Config.AsyncStartup] — bind immediately; /health gates readiness during drain
//   - [Config.DashboardReadOnly] — nil = true (safe); set false at your own risk
//   - [Config.DashboardPageSize] — rows per page in dashboard tables (default: 50)
//   - [Config.LoginNoRegistration] — hide registration section on login page
//   - [Config.HealthPath] — health endpoint path (default: "/health")
//   - [Config.SSEPath] — opt-in session-gated SSE feed of all committed events;
//     reconnects resume from Last-Event-ID and journal-backed stores also
//     backfill first-time subscribers
//   - [Config.SSEHeartbeatInterval] — keep-alive comment frames on the shared
//     SSE feed (default: 15s; non-positive disables)
//   - [Config.Service] — adopt an already-built *usermgmt.Service instead of
//     constructing one (caller keeps lifecycle ownership)
//   - [Config.ServiceConfig] — escape hatch for usermgmt.ServiceConfig knobs
//     the flattened fields cannot express (MaxUsers, TokenPepper,
//     SecurityHooks, ...); mutually exclusive with [Config.Service]
//   - [Config.AdminMode] / [Config.TenantID] — tenant-scoped admin panel
//   - [Config.AdminAuthorizer] / [Config.DashboardAuthorizer] — custom access control
//   - [Config.ExtraMiddleware] — consumer HTTP middleware composed inside
//     the built-in security stack (first entry outermost of the extras);
//     [Config.DisableSecurityMiddleware] removes that layer for consumers
//     who own the whole chain
//
// Paths are normalized and validated at [New]: panel mount paths gain a
// trailing slash (so the standard mux registers them as subtrees), "/" is
// reserved for the login page, and colliding paths are rejected with a
// descriptive error instead of panicking inside Mount. Custom AdminPath and
// DashboardPath values are passed to the panels as their BasePath, so all
// internal links and HTMX targets match the mount location.
//
// # Persistence
//
// By default, everything runs in-memory (lost on restart). Provide your own [event.Store]
// or SQL database for production:
//
//	bundle, err := setup.New(setup.Config{
//	    EventStore:  mySQLStore,
//	    ReadModelDB: db,
//	})
//
// # Feature flags
//
// Disable panels you don't need to reduce the route surface:
//
//	setup.Config{
//	    DisableAdmin:     true,  // no user management panel
//	    DisableDashboard: true,  // no CQRS dashboard
//	    DisableLogin:     true,  // use your own login page
//	}
//
// # Identity-external apps
//
// Applications that authenticate against an external authority (a PBX
// directory, corporate SSO, their own OIDC integration) can adopt the
// bundle's runtime shell — the serve/drain/close lifecycle, the readiness
// composition, opt-in liveness, shared Stores — without carrying a dormant
// second user database (ADR-0054):
//
//	setup.Config{
//	    DisableService:   true, // no usermgmt.Service, no /auth/*, no panels
//	    DisableAdmin:     true, // required with DisableService
//	    DisableDashboard: true, // required with DisableService
//	    DisableLogin:     true, // required with DisableService
//	    HealthPath:       "/healthz",
//	    HealthChecks:     []cqrshtmx.NamedCheck{...}, // your readiness checks
//	    LivePath:         "/livez",
//	}
//
// DisableAuth alone (service stays) is the own-login-endpoint mode: your
// code mints sessions against the service API, the bundle's panels keep
// working, and no /auth/* routes mount. The session-gated feeds and machine
// endpoints are rejected in shell mode (their gate is the bundle session
// middleware); they return behind an injectable gate in a future release.
//
// # Graceful shutdown
//
// [Bundle.Close] closes the dashboard's SSE broadcaster and the usermgmt service
// (projections, eviction goroutines, event bus, event store). Call on server shutdown.
// Safe to call multiple times. A shell bundle ([Config.DisableService]) owns
// neither — Close is a no-op there and stays idempotent.
package setup
