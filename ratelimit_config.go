package cqrshtmx

import (
	"net/http"
	"strconv"
	"time"

	httputil "github.com/larsartmann/httputil"
	"golang.org/x/time/rate"
)

// Rate limiter defaults.
const (
	DefaultRateLimit  = 100
	DefaultRateWindow = time.Minute
	DefaultRateTTL    = 10 * time.Minute

	headerRetryAfter     = "Retry-After"
	rateLimitExceededMsg = "rate limit exceeded"
)

// KeyExtractor extracts a rate-limit key from an HTTP request.
// Return "" if the request should not be rate-limited (always allowed).
// Common extractors: RemoteAddr, header value, user ID from context.
type KeyExtractor func(r *http.Request) string

// KeyExtractorFromRemoteAddr returns a KeyExtractor that uses the request's
// RemoteAddr as the rate-limit key.
//
// WARNING: Behind reverse proxies (nginx, Cloudflare, AWS ALB), RemoteAddr
// is the proxy's IP, not the client's. Use this only when the server is
// exposed directly or when the proxy forwards the real client IP in a
// header (e.g., X-Forwarded-For) and you parse that instead.
func KeyExtractorFromRemoteAddr() KeyExtractor {
	return func(r *http.Request) string {
		return r.RemoteAddr
	}
}

// KeyExtractorFromClientIP returns a KeyExtractor that uses ClientIP to
// resolve the real client IP, respecting X-Forwarded-For and X-Real-IP
// headers. Use this when deployed behind a trusted reverse proxy.
func KeyExtractorFromClientIP() KeyExtractor {
	return httputil.ClientIP
}

// RateLimiterConfig configures the token-bucket rate limiter per key.
type RateLimiterConfig struct {
	// Limit is the maximum number of requests per Window.
	Limit uint
	// Window is the time period for which Limit applies.
	Window time.Duration
	// Burst is the maximum burst size (can be greater than Limit).
	// Zero defaults to Limit.
	Burst uint
	// KeyExtractor produces the rate-limit key from a request.
	// If nil, all requests share a single key —effectively a global rate limit.
	// If an extractor returns empty string for a request, that request is
	// exempt from rate limiting (always allowed).
	KeyExtractor KeyExtractor
	// TTL is how long an idle limiter entry is kept before eviction.
	// Zero defaults to 10 minutes.
	TTL time.Duration
	// MaxKeys caps the number of tracked rate-limit keys.
	// When exceeded, the oldest entry (by last access time) is evicted.
	// Zero means no cap (unbounded growth).
	MaxKeys uint
	// OnAllowed is called when a request passes rate limiting.
	OnAllowed func(r *http.Request)
	// OnRejected is called when a request is rejected due to rate limiting.
	OnRejected func(r *http.Request, retryAfter string)
	// RejectionHandler writes the response for rejected requests.
	// Default: writes 429 Too Many Requests with Retry-After header.
	RejectionHandler func(w http.ResponseWriter, r *http.Request, retryAfter string)
}

// RateLimiterMiddleware returns HTTP middleware that rate-limits requests
// using a token bucket per key.
//
// If the rate limit is exceeded the middleware responds with 429 Too Many
// Requests and a Retry-After header in seconds.
//
// The internal per-key limiter map uses TTL-based eviction (default 10 min).
// Entries not accessed within the TTL are cleaned up on the next cache miss.
// For very high cardinality key spaces, consider increasing the TTL or using
// a bounded key extractor.
//
// The returned middleware has an ActiveKeys method for monitoring.
// RateLimiter wraps rate-limiting middleware and exposes monitoring.
type RateLimiter struct {
	middleware func(http.Handler) http.Handler
	limiter    *perKeyLimiter
}

// ActiveKeys returns the number of currently tracked rate-limit keys.
func (rl *RateLimiter) ActiveKeys() int {
	return rl.limiter.Len()
}

// Middleware returns the underlying middleware function.
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return rl.middleware
}

// NewRateLimiter creates a RateLimiter with monitoring capabilities.
// Use this when you need to track the number of active rate-limit keys.
// For simple usage without monitoring, use RateLimiterMiddleware instead.
func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	return buildRateLimiter(cfg)
}

// RateLimiterMiddleware returns HTTP middleware that rate-limits requests
// using a token bucket per key.
//
// If the rate limit is exceeded the middleware responds with 429 Too Many
// Requests and a Retry-After header in seconds.
//
// The internal per-key limiter map uses TTL-based eviction (default 10 min).
// Entries not accessed within the TTL are cleaned up on the next cache miss.
// For very high cardinality key spaces, consider increasing the TTL or using
// a bounded key extractor.
//
// For monitoring, use NewRateLimiter instead.
func RateLimiterMiddleware(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	return buildRateLimiter(cfg).Middleware()
}

func buildRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	if cfg.Limit == 0 {
		cfg.Limit = uint(DefaultRateLimit)
	}
	if cfg.Window <= 0 {
		cfg.Window = DefaultRateWindow
	}
	if cfg.Burst == 0 {
		cfg.Burst = cfg.Limit
	}

	limit := rate.Limit(float64(cfg.Limit) / cfg.Window.Seconds())
	retryAfter := strconv.Itoa(int(cfg.Window.Seconds()))

	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = DefaultRateTTL
	}

	lim := newPerKeyLimiter(
		limit, cfg.Burst,
		cfg.KeyExtractor, retryAfter, ttl, cfg.MaxKeys,
	)

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, retryAfter := lim.allow(r)
			if !allowed {
				if cfg.OnRejected != nil {
					cfg.OnRejected(r, retryAfter)
				}

				if cfg.RejectionHandler != nil {
					cfg.RejectionHandler(w, r, retryAfter)
				} else {
					w.Header().Set(headerRetryAfter, retryAfter)
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = w.Write([]byte(rateLimitExceededMsg))
				}

				return
			}

			if cfg.OnAllowed != nil {
				cfg.OnAllowed(r)
			}

			next.ServeHTTP(w, r)
		})
	}

	return &RateLimiter{middleware: mw, limiter: lim}
}

// limiterEntry holds a rate.Limiter and its last access time for TTL eviction.
