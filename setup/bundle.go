package setup

import (
	"net/http"

	"github.com/larsartmann/cqrs-htmx/adminui/v4"
	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	"github.com/larsartmann/cqrs-htmx/loginpage/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/httputil"
)

// Bundle is the result of [New] — a fully wired application with all sub-modules connected.
//
// Every sub-component is exported so consumers can customize after construction:
//
//	bundle.Admin.SetAccentColor("#ff0000")
//	bundle.Dashboard.SetReadOnly(true)
//
// Use [Bundle.Mount] to register all routes on a mux, [Bundle.Middleware] for the middleware
// chain, and [Bundle.Close] for graceful shutdown.
type Bundle struct {
	// Service is the event-sourced user management service.
	// Use it for registration, authentication, session management, and read-model queries.
	Service *usermgmt.Service

	// Auth is the HTTP handler for auth routes (/auth/register, /auth/webauthn/*, etc.).
	// Already registered by [Bundle.Mount] — exposed for advanced route customization.
	Auth *usermgmt.AuthHandler

	// Admin is the admin panel handler (user/tenant/membership management).
	// Nil if Config.DisableAdmin is true.
	Admin *adminui.Handler

	// Dashboard is the CQRS/ES observability dashboard.
	// Nil if Config.DisableDashboard is true.
	Dashboard *dashboardui.Dashboard

	// Login is the login page handler.
	// Nil if Config.DisableLogin is true.
	Login *loginpage.Handler

	// Stores holds the shared event infrastructure.
	// Use EventStore and EventBus for your own projections, read models, or SSE endpoints.
	Stores *Stores

	// config holds the resolved configuration (defaults applied).
	config Config
}

// Stores holds the shared event infrastructure created by [New].
//
// These are the SAME stores used by the usermgmt.Service and dashboardui.Dashboard.
// Use them for your own endpoints, projections, or read models.
type Stores struct {
	// EventStore is the shared event store (also satisfies event.Journal and
	// event.SeekableJournal when using the default memory store).
	EventStore event.Store

	// EventBus is the shared event bus for pub/sub.
	EventBus event.Bus
}

// SessionMiddleware returns the session authentication middleware.
// It reads the session cookie, validates it, and injects *usermgmt.User into the request context.
//
// Apply this to any route that requires authentication:
//
//	mux.Handle("/api/", bundle.SessionMiddleware(myHandler))
func (b *Bundle) SessionMiddleware() func(http.Handler) http.Handler {
	return usermgmt.NewSessionMiddleware(b.Service, b.config.CookieName)
}

// CSRFMiddleware returns the CSRF protection middleware (via httputil).
// Apply this to mutation endpoints (POST/PUT/DELETE) that use form submissions.
func (b *Bundle) CSRFMiddleware() func(http.Handler) http.Handler {
	return httputil.CSRFMiddleware(httputil.CSRFConfig{})
}

// Middleware returns the outer middleware chain for the entire application:
// security headers + per-request CSP nonce + panic recovery.
//
// Wrap your mux with this:
//
//	http.ListenAndServe(":8080", bundle.Middleware()(mux))
func (b *Bundle) Middleware() func(http.Handler) http.Handler {
	return cqrshtmx.RecommendedSecurityMiddleware()
}

// Handler is a convenience method that mounts all routes and wraps the mux with
// [Bundle.Middleware]. It eliminates the boilerplate of calling Mount + Middleware separately:
//
//	mux := http.NewServeMux()
//	http.ListenAndServe(":8080", bundle.Handler(mux))
//
// If you need custom middleware between security and your routes, use [Bundle.Mount]
// and [Bundle.Middleware] directly.
func (b *Bundle) Handler(mux *http.ServeMux) http.Handler {
	b.Mount(mux)

	return b.Middleware()(mux)
}

// Close gracefully shuts down all background resources (projections, eviction goroutines,
// dashboard SSE broadcaster). Call on server shutdown. Safe to call multiple times.
func (b *Bundle) Close() error {
	if b.Dashboard != nil {
		b.Dashboard.Close()
	}

	if b.Service != nil {
		if err := b.Service.Close(); err != nil {
			return errorfamily.WrapInfrastructure(err, "setup.bundle_close", "failed to close service")
		}
	}

	return nil
}
