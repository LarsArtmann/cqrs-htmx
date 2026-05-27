package cqrshtmx

import "net/http"

const (
	headerContentTypeOptions = "X-Content-Type-Options"
	headerFrameOptions       = "X-Frame-Options"
	headerReferrerPolicy     = "Referrer-Policy"
	headerCSP                = "Content-Security-Policy"
	headerHSTS               = "Strict-Transport-Security"
	headerPermissionsPolicy  = "Permissions-Policy"

	defaultContentTypeOptions = "nosniff"
	defaultFrameOptions       = "DENY"
	defaultReferrerPolicy     = "strict-origin-when-cross-origin"

	// RecommendedHSTS is a recommended Strict-Transport-Security value for production.
	// Use: SecurityHeadersConfig{StrictTransportSecurity: RecommendedHSTS}
	RecommendedHSTS = "max-age=31536000; includeSubDomains"

	// RecommendedCSP is a baseline Content-Security-Policy for HTMX applications.
	// Allows scripts from self (required for HTMX) and styles from self.
	// Use: SecurityHeadersConfig{ContentSecurityPolicy: RecommendedCSP}
	RecommendedCSP = "default-src 'self'; script-src 'self'; style-src 'self'"
)

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
	// Default: "" (not set). See RecommendedCSP for a sensible baseline.
	ContentSecurityPolicy string

	// StrictTransportSecurity sets Strict-Transport-Security.
	// Default: "" (not set). See RecommendedHSTS for a production value.
	StrictTransportSecurity string

	// PermissionsPolicy sets Permissions-Policy.
	// Default: "" (not set)
	PermissionsPolicy string

	// Custom headers are applied after all other headers.
	Custom map[string]string
}

func withDefault(val, def string) string {
	if val != "" {
		return val
	}
	return def
}

func (c *SecurityHeadersConfig) contentTypeOptions() string {
	return withDefault(c.ContentTypeOptions, defaultContentTypeOptions)
}

func (c *SecurityHeadersConfig) frameOptions() string {
	return withDefault(c.FrameOptions, defaultFrameOptions)
}

func (c *SecurityHeadersConfig) referrerPolicy() string {
	return withDefault(c.ReferrerPolicy, defaultReferrerPolicy)
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
			w.Header().Set(headerContentTypeOptions, cfg.contentTypeOptions())
			w.Header().Set(headerFrameOptions, cfg.frameOptions())
			w.Header().Set(headerReferrerPolicy, cfg.referrerPolicy())

			if cfg.ContentSecurityPolicy != "" {
				w.Header().Set(headerCSP, cfg.ContentSecurityPolicy)
			}

			if cfg.StrictTransportSecurity != "" {
				w.Header().Set(headerHSTS, cfg.StrictTransportSecurity)
			}

			if cfg.PermissionsPolicy != "" {
				w.Header().Set(headerPermissionsPolicy, cfg.PermissionsPolicy)
			}

			for k, v := range cfg.Custom {
				w.Header().Set(k, v)
			}

			next.ServeHTTP(w, r)
		})
	}
}
