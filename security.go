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
	// Use: SecurityHeadersConfig{StrictTransportSecurity: RecommendedHSTS}.
	RecommendedHSTS = "max-age=31536000; includeSubDomains"

	// RecommendedCSP is a baseline Content-Security-Policy for HTMX applications.
	// Allows scripts from self (required for HTMX) and styles from self.
	// Use: SecurityHeadersConfig{ContentSecurityPolicy: RecommendedCSP}.
	RecommendedCSP = "default-src 'self'; script-src 'self'; style-src 'self'"

	// SecurityHeaderSkip is the sentinel value for suppressing a default
	// security header. Set any of ContentTypeOptions, FrameOptions, or
	// ReferrerPolicy to this value to omit that header entirely:
	//
	//	cqrshtmx.SecurityHeadersConfig{
	//	    ContentTypeOptions: cqrshtmx.SecurityHeaderSkip, // do not set X-Content-Type-Options
	//	    ContentSecurityPolicy: cqrshtmx.RecommendedCSP,
	//	}
	SecurityHeaderSkip = "-"
)

// SecurityHeadersConfig configures which security headers to set.
// The three "withDefault" fields (ContentTypeOptions, FrameOptions, ReferrerPolicy)
// fall back to secure defaults when empty. Set any of them to SecurityHeaderSkip
// ("-" sentinel) to suppress that header entirely. All fields are optional.
type SecurityHeadersConfig struct {
	// ContentTypeOptions sets X-Content-Type-Options.
	// Default: "nosniff". Set to SecurityHeaderSkip to omit.
	ContentTypeOptions string

	// FrameOptions sets X-Frame-Options.
	// Default: "DENY". Set to SecurityHeaderSkip to omit.
	FrameOptions string

	// ReferrerPolicy sets Referrer-Policy.
	// Default: "strict-origin-when-cross-origin". Set to SecurityHeaderSkip to omit.
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
	if val == SecurityHeaderSkip {
		return "" // sentinel: suppress this header entirely
	}

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
			if v := cfg.contentTypeOptions(); v != "" {
				w.Header().Set(headerContentTypeOptions, v)
			}

			if v := cfg.frameOptions(); v != "" {
				w.Header().Set(headerFrameOptions, v)
			}

			if v := cfg.referrerPolicy(); v != "" {
				w.Header().Set(headerReferrerPolicy, v)
			}

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
