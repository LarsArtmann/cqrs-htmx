package setup

import (
	"net/http"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/httputil"
)

// Mount registers all UI routes on the mux with the correct middleware ordering.
//
// Route layout (defaults):
//
//	/auth/*         — registration, login, logout, me (public)
//	/admin/*        — admin panel (behind session + CSRF)
//	/dashboard/*    — CQRS observability dashboard (behind session)
//	/health         — readiness check (public)
//	/               — login page (public)
//
// Only enabled panels (per Config.Disable* flags) are mounted.
//
// The middleware ordering follows the documented pattern:
//
//	RecommendedSecurityMiddleware (outer — applied via [Bundle.Middleware] or [Bundle.Handler])
//	  └─ SessionMiddleware
//	       └─ CSRFMiddleware (mutations only)
//	            └─ handler
//
// For custom endpoints, use [Bundle.SessionMiddleware] directly.
func (b *Bundle) Mount(mux *http.ServeMux) {
	cfg := b.config

	// Auth routes (register, webauthn, logout, me) — public (no session needed for registration).
	if b.Auth != nil {
		b.Auth.RegisterRoutes(mux)
	}

	// Login page — public (it IS the login page).
	if b.Login != nil {
		b.Login.Mount(mux, "/")
	}

	// Admin panel — behind session + CSRF.
	// Security middleware (headers, nonce, recovery) is applied at the bundle level
	// via [Bundle.Middleware] or [Bundle.Handler], not duplicated per-panel.
	if b.Admin != nil {
		sessionMW := b.SessionMiddleware()
		csrfMW := httputil.CSRFMiddleware(httputil.CSRFConfig{})

		mux.Handle(
			cfg.AdminPath,
			sessionMW(csrfMW(http.StripPrefix(
				trimTrailingSlash(cfg.AdminPath), b.Admin.Handler(),
			))),
		)
	}

	// CQRS dashboard — behind an authenticated session. The dashboard renders
	// event payloads and stream IDs, so it must never be public: the session
	// middleware only enriches the context, it does not block, so an explicit
	// requireSession gate is applied (401, mirroring the admin panel's behavior).
	// A custom [Config.DashboardAuthorizer] can refine access further.
	// No CSRF: the dashboard is read-only by default. If you enable write mode
	// (Config.DashboardReadOnly = false), add CSRF yourself.
	if b.Dashboard != nil {
		sessionMW := b.SessionMiddleware()

		mux.Handle(
			cfg.DashboardPath,
			sessionMW(requireSession(http.StripPrefix(
				trimTrailingSlash(cfg.DashboardPath), b.Dashboard.Handler(),
			))),
		)
	}

	// Health check — public, no auth.
	if cfg.HealthPath != "" {
		mux.Handle(cfg.HealthPath, b.healthHandler())
	}
}

// requireSession blocks requests that carry no authenticated user, responding
// 401 (the admin panel's convention for unauthenticated access).
func requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := usermgmt.UserFromContext(r.Context()); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// healthHandler builds a readiness check handler that verifies all projection
// workers have caught up and none have failed. It returns 503 while any
// projection is still draining its initial journal backlog (essential for
// async startup — see Config.AsyncStartup), and 200 once every worker reaches
// "live" state. If the service is nil (should not happen in normal usage), it
// returns a simple 200 OK.
func (b *Bundle) healthHandler() http.HandlerFunc {
	if b.Service == nil {
		return cqrshtmx.ReadinessHandler()
	}

	return cqrshtmx.ReadinessHandler(
		cqrshtmx.ProjectionReadinessCheck(b.Service),
	)
}
