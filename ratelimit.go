package cqrshtmx

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
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

// RateLimiterConfig configures the token-bucket rate limiter per key.
type RateLimiterConfig struct {
	// Limit is the maximum number of requests per Window.
	Limit int
	// Window is the time period for which Limit applies.
	Window time.Duration
	// Burst is the maximum burst size (can be greater than Limit).
	// Zero defaults to Limit.
	Burst int
	// KeyExtractor produces the rate-limit key from a request.
	// If nil, all requests share a single key —effectively a global rate limit.
	// If an extractor returns empty string for a request, that request is
	// exempt from rate limiting (always allowed).
	KeyExtractor KeyExtractor
}

// RateLimiterMiddleware returns HTTP middleware that rate-limits requests
// using a token bucket per key.
//
// If the rate limit is exceeded the middleware responds with 429 Too Many
// Requests and a Retry-After header in seconds.
//
// NOTE: The internal per-key limiter map grows unbounded. For deployments
// with many unique keys (e.g., per-IP limiting on public-facing services),
// consider wrapping this middleware with periodic cleanup or using a bounded
// key space.
func RateLimiterMiddleware(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	if cfg.Limit <= 0 {
		cfg.Limit = 100
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	if cfg.Burst <= 0 {
		cfg.Burst = cfg.Limit
	}

	// rate.Limiter uses events per second; we convert our window to rps.
	limit := rate.Limit(float64(cfg.Limit) / cfg.Window.Seconds())
	retryAfter := strconv.Itoa(int(cfg.Window.Seconds()))

	lim := newPerKeyLimiter(limit, cfg.Burst, cfg.KeyExtractor, retryAfter)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, retryAfter := lim.allow(r)
			if !allowed {
				w.Header().Set("Retry-After", retryAfter)
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("rate limit exceeded"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// perKeyLimiter holds a token-bucket limiter per extracted key.
type perKeyLimiter struct {
	mu           sync.RWMutex
	limit        rate.Limit
	burst        int
	retryAfter   string
	keyExtractor KeyExtractor
	limiters     map[string]*rate.Limiter
}

func newPerKeyLimiter(
	l rate.Limit,
	burst int,
	extractor KeyExtractor,
	retryAfter string,
) *perKeyLimiter {
	return &perKeyLimiter{
		mu:           sync.RWMutex{},
		limit:        l,
		burst:        burst,
		retryAfter:   retryAfter,
		keyExtractor: extractor,
		limiters:     make(map[string]*rate.Limiter),
	}
}

func (p *perKeyLimiter) allow(r *http.Request) (bool, string) {
	var key string
	if p.keyExtractor != nil {
		key = p.keyExtractor(r)
	}
	// Explicitly empty key means "skip rate limiting for this request".
	if key == "" && p.keyExtractor != nil {
		return true, ""
	}

	lim := p.limiter(key)
	if lim.Allow() {
		return true, ""
	}

	return false, p.retryAfter
}

func (p *perKeyLimiter) limiter(key string) *rate.Limiter {
	p.mu.RLock()
	lim, ok := p.limiters[key]
	p.mu.RUnlock()

	if ok {
		return lim
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	// double-check after acquiring write lock
	if lim, ok := p.limiters[key]; ok {
		return lim
	}

	lim = rate.NewLimiter(p.limit, p.burst)
	p.limiters[key] = lim

	return lim
}
