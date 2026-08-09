// middleware-showcase demonstrates every httputil HTTP middleware in a single
// runnable server. It complements examples/middleware-demo (which shows
// go-cqrs-lite *dispatch* middleware like retry and circuit breaker) by focusing
// on the HTTP-layer middleware that httputil provides.
//
// Middleware demonstrated (outer → inner order):
//
//  1. Recovery        — catch panics, return 500          (cqrshtmx.RecoveryMiddleware)
//  2. SecurityHeaders — X-Content-Type-Options, etc.      (httputil.SecurityHeaders)
//  3. Metrics         — per-request method/path/status/duration recorder
//  4. CORS            — cross-origin requests with preflight caching
//  5. ClientIP        — extract real IP from X-Forwarded-For / X-Real-IP
//  6. RateLimiter     — per-client rate limiting (100 req/min, burst 20)
//  7. Compression     — gzip/deflate response compression
//  8. ETag            — 304 Not Modified for repeat queries
//
// All eight are composed via httputil.NewMiddlewareStack — a validated, named
// stack builder that enforces "Recovery outermost" and rejects duplicate names.
//
// Run: go run . and open http://localhost:8099
//
//	curl -v --compressed http://localhost:8099/api/data
//	curl -v -H 'If-None-Match: "abc"' http://localhost:8099/api/data
//	curl -v -H 'Origin: https://app.example.com' http://localhost:8099/api/data
package main

import (
	"encoding/json/v2"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	etag "github.com/larsartmann/go-etag"
	"github.com/larsartmann/httputil"
)

// logRecorder implements httputil.MetricsRecorder by logging each request.
// In production you'd replace this with a Prometheus counter/histogram.
type logRecorder struct {
	log *slog.Logger
}

func (r *logRecorder) Record(method, path string, status int, duration time.Duration) {
	r.log.Info("request",
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", duration.Milliseconds(),
	)
}

type dataResponse struct {
	Message    string    `json:"message"`
	Count      int64     `json:"count"`
	ServerTime time.Time `json:"serverTime"`
}

func main() {
	var requestCount atomic.Int64

	mux := http.NewServeMux()

	// --- Endpoints ---

	mux.HandleFunc("GET /api/data", func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)

		resp := dataResponse{
			Message:    "Hello from the middleware showcase!",
			Count:      count,
			ServerTime: time.Now().UTC(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.MarshalWrite(w, resp)
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<!DOCTYPE html>
<html><body>
  <h1>httputil Middleware Showcase</h1>
  <p>Try: <a href="/api/data">/api/data</a> (compressed, ETagged, CORS-enabled, rate-limited, metrics-tracked)</p>
</body></html>`)
	})

	// Standard health endpoints via httputil.RegisterHealth.
	httputil.RegisterHealth(mux)

	// --- Build the middleware stack ---
	//
	// NewMiddlewareStack provides a named, validated alternative to bare Chain.
	// It enforces "Recovery outermost" and rejects duplicate names — useful for
	// production stacks with 6+ middleware where ordering bugs are easy to make.

	stack := httputil.NewMiddlewareStack()

	// 1. Recovery — must be first (Validate enforces this).
	if err := stack.Add(httputil.MiddlewareRecovery, cqrshtmx.RecoveryMiddleware); err != nil {
		panic(fmt.Sprintf("add recovery: %v", err))
	}

	// 2. Security headers (X-Content-Type-Options, X-Frame-Options, Referrer-Policy).
	secCfg := httputil.DefaultSecurityHeadersConfig()
	secCfg.PermissionsPolicy = "geolocation=(), microphone=(), camera=()"

	if err := stack.Add(httputil.MiddlewareSecurityHeaders, httputil.SecurityHeaders(secCfg)); err != nil {
		panic(fmt.Sprintf("add security-headers: %v", err))
	}

	// 3. Metrics — logs every request with method/path/status/duration.
	//    Replace logRecorder with a Prometheus recorder in production.
	logger := slog.Default()

	if err := stack.Add("metrics", httputil.Metrics(httputil.MetricsConfig{ //nolint:exhaustruct // example only
		Recorder: &logRecorder{log: logger},
	})); err != nil {
		panic(fmt.Sprintf("add metrics: %v", err))
	}

	// 4. CORS — allows cross-origin browser requests with preflight caching.
	//    DefaultCORSConfig allows all origins (dev-friendly). Restrict
	//    AllowedOrigins in production.
	if err := stack.Add(httputil.MiddlewareCORS, httputil.CORS(httputil.DefaultCORSConfig())); err != nil {
		panic(fmt.Sprintf("add cors: %v", err))
	}

	// 5. ClientIP — extracts the real client IP from X-Forwarded-For / X-Real-IP.
	//    Required before KeyedRateLimiterMiddleware when behind a reverse proxy
	//    so rate limiting keys on the real client, not the proxy.
	if err := stack.Add(httputil.MiddlewareClientIP, httputil.ClientIPMiddleware); err != nil {
		panic(fmt.Sprintf("add client-ip: %v", err))
	}

	// 6. Rate limiting — 100 req/min per client IP, burst 20.
	//    KeyExtractorFromClientIP reads the IP set by ClientIPMiddleware above.
	rlCfg := httputil.DefaultKeyedRateLimiterConfig()
	rlCfg.Limit = 100
	rlCfg.Window = time.Minute
	rlCfg.Burst = 20

	if err := stack.Add(httputil.MiddlewareKeyedRateLimit,
		httputil.KeyedRateLimiterMiddleware(rlCfg),
	); err != nil {
		panic(fmt.Sprintf("add rate-limit: %v", err))
	}

	// 7. Compression — gzip/deflate with RFC 7231 q-value negotiation.
	//    Automatically skips small responses and already-compressed content types.
	if err := stack.Add(httputil.MiddlewareCompression,
		httputil.Compression(httputil.DefaultCompressionConfig()),
	); err != nil {
		panic(fmt.Sprintf("add compression: %v", err))
	}

	// 8. ETag — buffers GET/HEAD responses, computes FNV-64a hash, returns
	//    304 Not Modified on If-None-Match match.
	//    MUST be inside (after) Compression so the hash is on uncompressed data.
	if err := stack.Add(httputil.MiddlewareETag,
		httputil.ETag(etag.DefaultETagConfig()),
	); err != nil {
		panic(fmt.Sprintf("add etag: %v", err))
	}

	// Validate enforces structural rules (Recovery at position 0, no duplicates).
	if err := stack.Validate(); err != nil {
		panic(fmt.Sprintf("stack validation: %v", err))
	}

	logger.Info("middleware stack", "order", stack.Names())

	handler := stack.Build(mux)

	addr := ":8099"
	log.Printf("httputil Middleware Showcase — listening on http://localhost%s (stack: %v)", addr, stack.Names())

	// Serve with real timeouts via httputil.NewServer (never bare http.ListenAndServe).
	serverCfg := httputil.DefaultServerConfig() //nolint:exhaustruct // example only
	serverCfg.Addr = addr

	srv, err := httputil.NewServer(serverCfg, handler)
	if err != nil {
		panic(fmt.Sprintf("invalid server config: %v", err))
	}

	if err := <-srv.Start(); err != nil {
		panic(err)
	}
}
