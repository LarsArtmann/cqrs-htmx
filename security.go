package cqrshtmx

import "net/http"

// SecurityHeadersConfig configures which security headers to set.
// All fields are optional; zero values use secure defaults.
type SecurityHeadersConfig struct {
	// ContentTypeOptions sets X-Content-Type-Options.
	// Default: "nosniff"
	ContentTypeOptions string

	// FrameOptions sets X-Frame-Options.
	// Default: "DENY"
	FrameOptions string

	// ReferrerPolicy sets Referrer-Policy.
	// Default: "strict-origin-when-cross-origin"
	ReferrerPolicy string

	// ContentSecurityPolicy sets Content-Security-Policy.
	// Default: "" (not set)
	ContentSecurityPolicy string

	// StrictTransportSecurity sets Strict-Transport-Security.
	// Default: "" (not set)
	StrictTransportSecurity string

	// PermissionsPolicy sets Permissions-Policy.
	// Default: "" (not set)
	PermissionsPolicy string

	// Custom headers are applied after all other headers.
	Custom map[string]string
}

func (c *SecurityHeadersConfig) contentTypeOptions() string {
	if c.ContentTypeOptions != "" {
		return c.ContentTypeOptions
	}
	return "nosniff"
}

func (c *SecurityHeadersConfig) frameOptions() string {
	if c.FrameOptions != "" {
		return c.FrameOptions
	}
	return "DENY"
}

func (c *SecurityHeadersConfig) referrerPolicy() string {
	if c.ReferrerPolicy != "" {
		return c.ReferrerPolicy
	}
	return "strict-origin-when-cross-origin"
}

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
// For custom headers, use SecurityHeadersMiddlewareWithConfig.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return SecurityHeadersMiddlewareWithConfig(SecurityHeadersConfig{})(next)
}

// SecurityHeadersMiddlewareWithConfig returns HTTP middleware that sets
// security headers based on the provided configuration.
func SecurityHeadersMiddlewareWithConfig(
	cfg SecurityHeadersConfig,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", cfg.contentTypeOptions())
			w.Header().Set("X-Frame-Options", cfg.frameOptions())
			w.Header().Set("Referrer-Policy", cfg.referrerPolicy())

			if cfg.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}

			if cfg.StrictTransportSecurity != "" {
				w.Header().Set("Strict-Transport-Security", cfg.StrictTransportSecurity)
			}

			if cfg.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", cfg.PermissionsPolicy)
			}

			for k, v := range cfg.Custom {
				w.Header().Set(k, v)
			}

			next.ServeHTTP(w, r)
		})
	}
}
