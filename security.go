package cqrshtmx

import (
	"net/http"

	"github.com/larsartmann/httputil"
)

// SecurityHeadersConfig is an alias for httputil.SecurityHeadersConfig.
//
// Deprecated: Import github.com/larsartmann/httputil and use
// httputil.SecurityHeadersConfig directly. Remove in v5.
type SecurityHeadersConfig = httputil.SecurityHeadersConfig

// RecommendedHSTS is a recommended Strict-Transport-Security value for production.
//
// Deprecated: Use httputil.RecommendedHSTS. Remove in v5.
var RecommendedHSTS = httputil.RecommendedHSTS

// RecommendedCSP is a baseline Content-Security-Policy for HTMX applications.
//
// Deprecated: Use httputil.RecommendedCSP. Remove in v5.
var RecommendedCSP = httputil.RecommendedCSP

// SecurityHeaderSkip is the sentinel value for suppressing a default security header.
//
// Deprecated: Use httputil.SecurityHeaderSkip. Remove in v5.
var SecurityHeaderSkip = httputil.SecurityHeaderSkip

// SecurityHeadersMiddleware returns HTTP middleware that sets security headers
// on every response. These headers provide defense-in-depth against common
// web attacks and are recommended for all production deployments.
//
// A zero-value SecurityHeadersConfig applies secure defaults:
// X-Content-Type-Options: nosniff, X-Frame-Options: DENY,
// Referrer-Policy: strict-origin-when-cross-origin.
//
// Deprecated: Use httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig())
// instead. Remove in v5.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return SecurityHeadersMiddlewareWithConfig(SecurityHeadersConfig{})(next)
}

// SecurityHeadersMiddlewareWithConfig returns HTTP middleware that sets
// security headers based on the provided configuration. Empty fields fall
// back to secure defaults; set any field to SecurityHeaderSkip to suppress
// that header.
//
// Deprecated: Use httputil.SecurityHeaders instead. Remove in v5.
func SecurityHeadersMiddlewareWithConfig(
	config SecurityHeadersConfig,
) func(http.Handler) http.Handler {
	applySecurityDefaults(&config)

	return httputil.SecurityHeaders(config)
}

// applySecurityDefaults fills in secure defaults for empty fields, matching
// cqrs-htmx's zero-value-equals-secure-defaults contract. httputil's zero
// value sets no headers, so this transformation preserves backward
// compatibility for cqrs-htmx consumers.
func applySecurityDefaults(config *SecurityHeadersConfig) {
	if config.ContentTypeOptions == "" && !config.ContentTypeNosniff {
		config.ContentTypeOptions = "nosniff"
	}

	if config.FrameOptions == "" {
		config.FrameOptions = "DENY"
	}

	if config.ReferrerPolicy == "" {
		config.ReferrerPolicy = "strict-origin-when-cross-origin"
	}
}
