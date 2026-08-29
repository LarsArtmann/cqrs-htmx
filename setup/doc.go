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
// # Graceful shutdown
//
// [Bundle.Close] closes the dashboard's SSE broadcaster and the usermgmt service
// (projections, eviction goroutines, event bus, event store). Call on server shutdown.
// Safe to call multiple times.
package setup
