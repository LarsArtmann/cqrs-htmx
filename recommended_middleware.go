package cqrshtmx

import (
	"net/http"

	"github.com/larsartmann/httputil"
)

// RecommendedPermissionsPolicy is a Permissions-Policy header value that
// disables browser features commonly unnecessary in server-rendered web apps
// (geolocation, microphone, camera, payment, USB). Use it as a starting point
// and tighten further for APIs or single-page apps that need even fewer
// capabilities.
const RecommendedPermissionsPolicy = "geolocation=(), microphone=(), camera=(), payment=(), usb=()"

// RecommendedSecurityMiddleware returns the baseline middleware chain
// recommended for production HTTP services built with cqrs-htmx:
//
//   - SecurityHeaders — X-Content-Type-Options, X-Frame-Options,
//     Referrer-Policy, and RecommendedPermissionsPolicy.
//   - Nonce — generates a per-request CSP nonce and sets a
//     Content-Security-Policy header that allows 'self' + nonce for scripts
//     and styles.
//   - RecoveryMiddleware — catches panics and returns 500 instead of crashing.
//
// Both adminui.Handler.Middleware and dashboardui.Dashboard.Middleware delegate
// to this function so that mounting either UI gives the same security posture.
// Consumers building their own UI can call it directly:
//
//	http.ListenAndServe(":8080", cqrshtmx.Chain(
//	    cqrshtmx.RecommendedSecurityMiddleware(),
//	    sessionMW,
//	    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
//	    app.Middleware(),
//	)(mux))
func RecommendedSecurityMiddleware() func(http.Handler) http.Handler {
	securityCfg := httputil.DefaultSecurityHeadersConfig()
	securityCfg.PermissionsPolicy = RecommendedPermissionsPolicy

	return Chain(
		httputil.SecurityHeaders(securityCfg),
		httputil.Nonce(httputil.DefaultNonceConfig()),
		RecoveryMiddleware,
	)
}
