package cqrshtmx

import (
	"github.com/larsartmann/httputil"
)

// Keyed rate limiting now lives in httputil. These aliases preserve backward
// compatibility for cqrs-htmx consumers.
//
// Deprecated: import github.com/larsartmann/httputil directly. These aliases
// will be removed in cqrs-htmx v5.

type (
	// RateLimiterConfig is an alias for httputil.KeyedRateLimiterConfig.
	//
	// Deprecated: use httputil.KeyedRateLimiterConfig.
	RateLimiterConfig = httputil.KeyedRateLimiterConfig
	// RateLimiter is an alias for httputil.KeyedRateLimiter.
	//
	// Deprecated: use httputil.KeyedRateLimiter.
	RateLimiter       = httputil.KeyedRateLimiter
	// KeyExtractor is an alias for httputil.KeyExtractor.
	//
	// Deprecated: use httputil.KeyExtractor.
	KeyExtractor      = httputil.KeyExtractor
)

var (
	// RateLimiterMiddleware is an alias for httputil.KeyedRateLimiterMiddleware.
	//
	// Deprecated: use httputil.KeyedRateLimiterMiddleware.
	RateLimiterMiddleware = httputil.KeyedRateLimiterMiddleware
	// NewRateLimiter is an alias for httputil.NewKeyedRateLimiter.
	//
	// Deprecated: use httputil.NewKeyedRateLimiter.
	NewRateLimiter        = httputil.NewKeyedRateLimiter
	// DefaultRateLimiterConfig is an alias for httputil.DefaultKeyedRateLimiterConfig.
	//
	// Deprecated: use httputil.DefaultKeyedRateLimiterConfig.
	DefaultRateLimiterConfig   = httputil.DefaultKeyedRateLimiterConfig
	// KeyExtractorFromRemoteAddr is an alias for httputil.KeyExtractorFromRemoteAddr.
	//
	// Deprecated: use httputil.KeyExtractorFromRemoteAddr.
	KeyExtractorFromRemoteAddr = httputil.KeyExtractorFromRemoteAddr
	// KeyExtractorFromClientIP is an alias for httputil.KeyExtractorFromClientIP.
	//
	// Deprecated: use httputil.KeyExtractorFromClientIP.
	KeyExtractorFromClientIP   = httputil.KeyExtractorFromClientIP
)

const (
	// DefaultRateLimit mirrors httputil.DefaultRateLimit.
	//
	// Deprecated: use httputil.DefaultRateLimit.
	DefaultRateLimit  = httputil.DefaultRateLimit
	// DefaultRateWindow mirrors httputil.DefaultRateWindow.
	//
	// Deprecated: use httputil.DefaultRateWindow.
	DefaultRateWindow = httputil.DefaultRateWindow
	// DefaultRateTTL mirrors httputil.DefaultRateTTL.
	//
	// Deprecated: use httputil.DefaultRateTTL.
	DefaultRateTTL    = httputil.DefaultRateTTL
)
