package cqrshtmx

import (
	"github.com/larsartmann/httputil"
)

// Keyed rate limiting now lives in httputil. These aliases preserve backward
// compatibility for cqrs-htmx consumers.

type RateLimiterConfig = httputil.KeyedRateLimiterConfig
type RateLimiter = httputil.KeyedRateLimiter
type KeyExtractor = httputil.KeyExtractor

var (
	RateLimiterMiddleware      = httputil.KeyedRateLimiterMiddleware
	NewRateLimiter             = httputil.NewKeyedRateLimiter
	DefaultRateLimiterConfig   = httputil.DefaultKeyedRateLimiterConfig
	KeyExtractorFromRemoteAddr = httputil.KeyExtractorFromRemoteAddr
	KeyExtractorFromClientIP   = httputil.KeyExtractorFromClientIP
)

const (
	DefaultRateLimit  = httputil.DefaultRateLimit
	DefaultRateWindow = httputil.DefaultRateWindow
	DefaultRateTTL    = httputil.DefaultRateTTL
)
