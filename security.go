package cqrshtmx

import "net/http"

// SecurityHeadersMiddleware returns HTTP middleware that sets security headers
// on every response. These headers provide defense-in-depth against common
// web attacks and are recommended for all production deployments.
//
// Headers set:
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: DENY
//   - Referrer-Policy: strict-origin-when-cross-origin
//
// Usage:
//
//	handler := cqrshtmx.SecurityHeadersMiddleware(mux)
//
// Or in a Chain:
//
//	handler := cqrshtmx.Chain(
//	    cqrshtmx.SecurityHeadersMiddleware,
//	    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
//	)(mux)
//
// Note: This middleware does NOT set Content-Security-Policy or
// Strict-Transport-Security. Those headers require application-specific
// configuration and should be set by the consumer or a dedicated middleware.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}
