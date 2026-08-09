package setup

import (
	"net/http"

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
				trimTrailing(cfg.AdminPath), b.Admin.Handler(),
			))),
		)
	}

	// CQRS dashboard — behind session.
	// No CSRF: the dashboard is read-only by default. If you enable write mode
	// (Config.DashboardReadOnly = false), add CSRF yourself.
	if b.Dashboard != nil {
		sessionMW := b.SessionMiddleware()

		mux.Handle(
			cfg.DashboardPath,
			sessionMW(http.StripPrefix(
				trimTrailing(cfg.DashboardPath), b.Dashboard.Handler(),
			)),
		)
	}

	// Health check — public, no auth.
	if cfg.HealthPath != "" {
		mux.Handle(cfg.HealthPath, b.healthHandler())
	}
}

// healthHandler builds a readiness check handler that verifies the projection
// host is not in a failed state. If the service is nil (should not happen in
// normal usage), it returns a simple 200 OK.
func (b *Bundle) healthHandler() http.HandlerFunc {
	if b.Service == nil {
		return cqrshtmx.ReadinessHandler()
	}

	return cqrshtmx.ReadinessHandler(
		cqrshtmx.NewNamedCheck("projections", func() error {
			for _, s := range b.Service.ProjectionStatuses() {
				if s.Status == "failed" {
					return errorfamily.NewInfrastructure(
						"setup.projection_failed",
						"projection %q has failed: %s", s.Name, s.LastError,
					)
				}
			}

			return nil
		}),
	)
}

// trimTrailing removes a single trailing slash for use with http.StripPrefix.
func trimTrailing(s string) string {
	if len(s) > 1 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}

	return s
}
