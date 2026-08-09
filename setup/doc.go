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
//	defer bundle.Close()
//
//	mux := http.NewServeMux()
//	http.ListenAndServe(":8080", bundle.Handler(mux))
//
// # What you get
//
//   - /auth/* — registration, login (WebAuthn/TOTP/OAuth2), logout, me
//   - /admin/* — admin dashboard (user/tenant/membership management)
//   - /dashboard/* — CQRS/ES observability (events, projections, DLQ)
//   - /health — readiness check (verifies projection health)
//   - / — login page
//
// # Convenience methods
//
// [Bundle.Handler] mounts all routes and wraps the mux with [Bundle.Middleware] in one call.
// Alternatively, call [Bundle.Mount] and [Bundle.Middleware] separately for custom middleware:
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
//   - [Config.LogoutURL] — logout link shown in admin and dashboard panels
//   - [Config.SSEURL] — enables admin panel real-time sync indicator
//   - [Config.OnProjectionFailed] — callback when a projection exhausts restarts
//   - [Config.DashboardReadOnly] — nil = true (safe); set false at your own risk
//   - [Config.DashboardPageSize] — rows per page in dashboard tables (default: 50)
//   - [Config.LoginNoRegistration] — hide registration section on login page
//   - [Config.HealthPath] — health endpoint path (default: "/health")
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
