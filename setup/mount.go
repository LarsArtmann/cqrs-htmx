package setup

import (
	"net/http"

	"github.com/larsartmann/httputil"
)

// Mount registers all UI routes on the mux with the correct middleware ordering.
//
// Route layout (defaults):
//
//	/auth/*         — registration, login, logout, me (behind session middleware)
//	/admin/*        — admin panel (behind session + CSRF + security middleware)
//	/dashboard/*    — CQRS observability dashboard (behind session + security middleware)
//	/               — login page (no auth — it IS the login page)
//
// Only enabled panels (per Config.Enable*) are mounted.
//
// The middleware ordering follows the documented pattern:
//
//	RecommendedSecurityMiddleware (outer)
//	  └─ SessionMiddleware
//	       └─ CSRFMiddleware (mutations only)
//	            └─ handler
//
// For custom endpoints, use [Bundle.SessionMiddleware] directly.
func (b *Bundle) Mount(mux *http.ServeMux) {
	cfg := b.config

	// Auth routes (register, webauthn, logout, me) — outside CSRF (registration is a POST).
	if b.Auth != nil {
		b.Auth.RegisterRoutes(mux)
	}

	// Login page — no session middleware (it IS the login page).
	if b.Login != nil {
		b.Login.Mount(mux, "/")
	}

	// Admin panel — behind session + CSRF + panel security middleware.
	if b.Admin != nil {
		sessionMW := b.SessionMiddleware()
		csrfMW := httputil.CSRFMiddleware(httputil.CSRFConfig{}) //nolint:exhaustruct // defaults are correct
		panelMW := b.Admin.Middleware()

		mux.Handle(
			cfg.AdminPath,
			sessionMW(csrfMW(panelMW(http.StripPrefix(
				trimTrailing(cfg.AdminPath), b.Admin.Handler(),
			)))),
		)
	}

	// CQRS dashboard — behind session + dashboard security middleware.
	if b.Dashboard != nil {
		sessionMW := b.SessionMiddleware()
		dashMW := b.Dashboard.Middleware()

		mux.Handle(
			cfg.DashboardPath,
			sessionMW(dashMW(http.StripPrefix(
				trimTrailing(cfg.DashboardPath), b.Dashboard.Handler(),
			))),
		)
	}
}

// trimTrailing removes a single trailing slash for use with http.StripPrefix.
func trimTrailing(s string) string {
	if len(s) > 1 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}
