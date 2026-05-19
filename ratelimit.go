package cqrshtmx

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Rate limiter defaults.
const (
	DefaultRateLimit  = 100
	DefaultRateWindow = time.Minute
	DefaultRateTTL    = 10 * time.Minute
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
	// TTL is how long an idle limiter entry is kept before eviction.
	// Zero defaults to 10 minutes.
	TTL time.Duration
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
func RateLimiterMiddleware(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	if cfg.Limit <= 0 {
		cfg.Limit = DefaultRateLimit
	}
	if cfg.Window <= 0 {
		cfg.Window = DefaultRateWindow
	}
	if cfg.Burst <= 0 {
		cfg.Burst = cfg.Limit
	}

	// rate.Limiter uses events per second; we convert our window to rps.
	limit := rate.Limit(float64(cfg.Limit) / cfg.Window.Seconds())
	retryAfter := strconv.Itoa(int(cfg.Window.Seconds()))

	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = DefaultRateTTL
	}

	lim := newPerKeyLimiter(limit, cfg.Burst, cfg.KeyExtractor, retryAfter, ttl)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, retryAfter := lim.allow(r)
			if !allowed {
				if cfg.OnRejected != nil {
					cfg.OnRejected(r, retryAfter)
				}

				if cfg.RejectionHandler != nil {
					cfg.RejectionHandler(w, r, retryAfter)
				} else {
					w.Header().Set("Retry-After", retryAfter)
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = w.Write([]byte("rate limit exceeded"))
				}

				return
			}

			if cfg.OnAllowed != nil {
				cfg.OnAllowed(r)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// limiterEntry holds a rate.Limiter and its last access time for TTL eviction.
type limiterEntry struct {
	lim      *rate.Limiter
	lastUsed time.Time
}

// perKeyLimiter holds a token-bucket limiter per extracted key.
// Stale entries are evicted when the map is accessed after their TTL expires.
type perKeyLimiter struct {
	mu           sync.RWMutex
	limit        rate.Limit
	burst        int
	retryAfter   string
	keyExtractor KeyExtractor
	limiters     map[string]*limiterEntry
	ttl          time.Duration
}

func newPerKeyLimiter(
	l rate.Limit,
	burst int,
	extractor KeyExtractor,
	retryAfter string,
	ttl time.Duration,
) *perKeyLimiter {
	return &perKeyLimiter{
		mu:           sync.RWMutex{},
		limit:        l,
		burst:        burst,
		retryAfter:   retryAfter,
		keyExtractor: extractor,
		limiters:     make(map[string]*limiterEntry),
		ttl:          ttl,
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
	entry, ok := p.limiters[key]
	p.mu.RUnlock()

	if ok && time.Since(entry.lastUsed) < p.ttl {
		return entry.lim
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Evict stale entries while holding write lock.
	now := time.Now()
	for k, v := range p.limiters {
		if now.Sub(v.lastUsed) > p.ttl {
			delete(p.limiters, k)
		}
	}

	// double-check after acquiring write lock
	if entry, ok := p.limiters[key]; ok {
		entry.lastUsed = time.Now()
		return entry.lim
	}

	lim := rate.NewLimiter(p.limit, p.burst)
	p.limiters[key] = &limiterEntry{lim: lim, lastUsed: time.Now()}

	return lim
}
