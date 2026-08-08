# Leveraging httputil — The HTTP Middleware Partner

> How to pair `cqrs-htmx` with its sibling library [`github.com/larsartmann/httputil`](https://github.com/larsartmann/httputil) for the HTTP concerns cqrs-htmx deliberately does **not** re-implement: CORS, compression, body limits, client IP, production serving, metrics, ETag, and health-readiness.

## Why this guide exists

`cqrs-htmx` follows a **duck-typing** philosophy: the three concerns it migrated into httputil (CSRF, keyed rate limiting, Server-Timing) are now **deprecated re-exports** (`csrf_reexport.go`, `ratelimit_reexport.go`, `server_timing_reexport.go` — 39 symbols with `// Deprecated:` markers, removal planned for v5). Import `httputil` directly for these. cqrs-htmx keeps richer, domain-aware re-implementations where a generic version would lose value (Recovery with errorfamily, Logging with dispatch-error capture, context enrichment with domain IDs).

For everything else — the everyday HTTP middleware a browser-facing CQRS app needs — `cqrs-htmx` intentionally stays out of the way. That middleware lives in **httputil**. This guide maps each concern to the exact httputil symbol so you import what you need with zero glue.

> **Import once, compose freely.** Every httputil middleware is a `func(http.Handler) http.Handler`, so it drops straight into `cqrshtmx.Chain(...)`.

## Concern → middleware map

| Concern                                        | httputil symbol                                                                  | Notes                                                                                                                                                                                                                                                        |
| ---------------------------------------------- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Cross-origin browser API access                | `httputil.CORS(cfg)` + `DefaultCORSConfig()`                                     | `DenyUnmatched: true` by default since v0.7.0; wildcard `*.example.com` rejects lookalikes.                                                                                                                                                                  |
| Response compression (htmx.js, JSON, SSE text) | `httputil.Compression(cfg)` + `DefaultCompressionConfig()`                       | Pooled gzip/deflate writers; skips already-compressed content types.                                                                                                                                                                                         |
| Body-size guard for JSON/form decoders         | `httputil.MaxBodySize(maxBytes)`                                                 | Rejects oversized bodies _before_ decode → pairs with `cqrshtmx.ErrRequestTooLarge` (413). Use as an outer middleware guard (defense-in-depth) alongside the per-handler `cqrshtmx.WithMaxBodySize`. See Recipe 8 below.                                                                   |
| Decompression bomb protection                  | `httputil.Decompression(cfg)` + `DefaultDecompressionConfig()`                    | Transparently decompresses gzip/deflate request bodies with bomb protection (`MaxDecompressionSize`, default 16 MiB). Essential for endpoints that accept `Content-Encoding: gzip`. See Recipe 7 below.                                                     |
| CSP nonce per request                           | `httputil.Nonce(cfg)` + `DefaultNonceConfig()` + `NonceFromRequest(r)`            | Generates a cryptographic nonce per request, stores in context, optionally sets CSP header. Used by adminui's `Middleware()` chain. `NonceAttr(r)` returns `nonce="..."` for templates.                                                                   |
| Client IP behind a reverse proxy               | `httputil.ClientIPMiddleware`, `httputil.ClientIP(r)`, `httputil.IsTrustedProxy` | Trusted-proxy CIDR matching; feeds `KeyExtractorFromClientIP` for correct rate limiting.                                                                                                                                                                     |
| Production server with timeouts                | `httputil.NewServer(cfg, h)` + `DefaultServerConfig()`                           | `Validate()`d config; sane `Read`/`Write`/`Idle`/`ReadHeader` timeouts. **Never use bare `http.ListenAndServe`.**                                                                                                                                            |
| Metrics recording                              | `httputil.Metrics(cfg)` + `DefaultMetricsConfig()` + `MetricsRecorder` interface | Plug in Prometheus/etc. via the interface.                                                                                                                                                                                                                   |
| Dynamic response ETagging                      | `httputil.ETag(cfg)` + `DefaultETagConfig()`                                     | Buffers responses, hashes, serves `304` on `If-None-Match`. (cqrs-htmx already does _static-asset_ ETagging for htmx.js / OpenAPI / event catalog.)                                                                                                          |
| Validated middleware stack                     | `httputil.NewMiddlewareStack()`                                                  | Enforces "Recovery outermost", rejects duplicate names — a safer alternative to bare `Chain`.                                                                                                                                                                |
| Health: liveness                               | `httputil.LiveHandler()`                                                         | Static 200 — process is up.                                                                                                                                                                                                                                  |
| Health: readiness                              | `httputil.ReadyHandlerWithProbe(func() bool)`                                    | 200/503 based on a probe (deps ready).                                                                                                                                                                                                                       |
| Health: mount conventional endpoints           | `httputil.RegisterHealth(mux)`                                                   | Mounts `/healthz`, `/livez`, `/readyz` in one call.                                                                                                                                                                                                          |
| Request-level timeout                          | `httputil.Timeout(d)`                                                            | Bounds the **whole** HTTP request. Distinct from `cqrshtmx.Config.Timeout`, which bounds only the CQRS dispatch. Use both: request-level as a safety net, dispatch-level for per-command SLAs.                                                               |
| Error classification registration              | `httputil.RegisterErrorClassifications()`                                        | Call once at startup so stdlib HTTP errors and httputil codes (`http.write_failed`, `http.compress_write_failed`, …) classify through `cqrshtmx.MapError` via `errorfamily.Classify`.                                                                        |
| HTTP-spec compliance testing                   | `httputil/httpspec.Run(t, h, opts...)`                                           | 18 standard specs (index reachable, unknown→404, HEAD/OPTIONS handled, TRACE/CONNECT rejected, error responses carry Content-Type, no version leak, `X-Content-Type-Options` present). `cqrs-htmx` uses this internally — see `httpspec_compliance_test.go`. |
| Response capture in tests                      | `httputil.NewResponseRecorder(w)`                                                | Lightweight status/body capture. (cqrs-htmx's own `StatusRecorder` is richer — captures dispatch errors — prefer it inside the pipeline.)                                                                                                                    |

## Recipes

### 1. CORS for a browser-facing API

```go
import "github.com/larsartmann/httputil"

handler := cqrshtmx.Chain(
    cqrshtmx.SecurityHeadersMiddleware,
    cqrshtmx.RecoveryMiddleware,
    httputil.CORS(httputil.CORSConfig{
        AllowedOrigins:   []string{"https://app.example.com"},
        AllowCredentials: true,
        AllowedMethods:   []string{"GET", "POST"},
        AllowedHeaders:   []string{"Content-Type", "X-CSRF-Token", "HX-Request"},
    }),
    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
    app.Middleware(),
)(mux)
```

### 2. Compression + body-size guard + production server

```go
// Guard the JSON/form decoders cqrs-htmx exposes, then compress responses.
handler := cqrshtmx.Chain(
    cqrshtmx.SecurityHeadersMiddleware,
    cqrshtmx.RecoveryMiddleware,
    httputil.MaxBodySize(1 << 20),          // 1 MiB — rejects before decode (413)
    httputil.Compression(httputil.DefaultCompressionConfig()),
    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
    cqrshtmx.HTMXMiddleware,
    app.Middleware(),
)(mux)

// Serve with real timeouts (never bare http.ListenAndServe).
srv, err := httputil.NewServer(httputil.DefaultServerConfig(), handler)
if err != nil {
    log.Fatal(err)
}
if err := <-srv.Start(); err != nil {
    log.Fatal(err)
}
```

### 3. Correct rate limiting behind a proxy

Without client-IP extraction, `KeyExtractorFromClientIP` sees the proxy's IP and all clients share one bucket. Put `ClientIPMiddleware` in the chain (after the proxy trust list is configured):

```go
handler := cqrshtmx.Chain(
    cqrshtmx.SecurityHeadersMiddleware,
    httputil.ClientIPMiddleware,            // populates client IP in context
    httputil.RateLimiterMiddleware(httputil.DefaultRateLimiterConfig()),
    app.Middleware(),
)(mux)
```

### 4. Liveness vs readiness (Kubernetes)

```go
// Liveness: the process is up.
mux.Handle("/healthz", httputil.LiveHandler())

// Readiness: deps are ready. cqrs-htmx's app.HealthHandler() already reports
// dispatcher availability — use it as the readiness probe, or compose a probe:
mux.Handle("/readyz", httputil.ReadyHandlerWithProbe(func() bool {
    return db.Ping() == nil
}))

// Or mount the conventional trio in one call:
httputil.RegisterHealth(mux)
```

### 5. Register error classifications once at startup

> **Note:** As of v4.x, `cqrshtmx.New()` calls `httputil.RegisterErrorClassifications()` automatically (via `sync.Once`). You only need to call it manually if you use httputil middleware without creating a `cqrshtmx.App`.

```go
func main() {
    // Already called by cqrshtmx.New() — but if you use httputil standalone:
    httputil.RegisterErrorClassifications()
    // ... build app, serve ...
}
```

### 6. Lock in HTTP-spec compliance in your own test suite

```go
import "github.com/larsartmann/httputil/httpspec"

func TestMyAppIsHTTPSpecCompliant(t *testing.T) {
    // Use a specific path, not "GET /", so unknown-path detection sees a real 404.
    httpspec.Run(t, myHandler, httpspec.WithIndexPath("/index"))
}
```

## What cqrs-htmx keeps (and why)

These have **richer, domain-aware** implementations than httputil's generic equivalents, so cqrs-htmx keeps its own:

- **Recovery** (`cqrshtmx.RecoveryMiddleware`) — classifies panics via errorfamily, routes through the configured `ErrorHandler`, recovers Request/Correlation IDs.
- **Logging** (`cqrshtmx.RequestLoggingSlog`) — captures dispatch errors and domain IDs (user/correlation/request).
- **Security headers** (`cqrshtmx.SecurityHeadersMiddleware`) — now a **deprecated alias** over `httputil.SecurityHeaders`. httputil's `SecurityHeadersConfig` is the single source of truth (gained `PermissionsPolicy`, `Custom`, `ContentTypeOptions`, `SecurityHeaderSkip`, `RecommendedHSTS`/`RecommendedCSP` in v0.9.0). The split brain is resolved. Use `httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig())` directly.

## Re-export deprecation migration table

The following 39 symbols in `cqrs-htmx/v4` are **deprecated** (type/var aliases over `github.com/larsartmann/httputil`). Import `httputil` directly; the aliases will be removed in v5.

### CSRF (`csrf_reexport.go`)

| Deprecated `cqrshtmx.*`        | Replacement `httputil.*`                |
| ------------------------------ | --------------------------------------- |
| `CSRFConfig`                   | `httputil.CSRFConfig`                   |
| `CSRFMiddleware`               | `httputil.CSRFMiddleware`               |
| `CSRFResponseHeaderMiddleware` | `httputil.CSRFResponseHeaderMiddleware` |
| `CSRFTokenFromContext`         | `httputil.CSRFTokenFromContext`         |
| `WithCSRFToken`                | `httputil.WithCSRFToken`                |
| `CSRFTestToken`                | `httputil.CSRFTestToken`                |
| `InvalidateCSRFCookie`         | `httputil.InvalidateCSRFCookie`         |
| `CSRFTokenHTMLMeta`            | `httputil.CSRFTokenHTMLMeta`            |
| `CSRFTokenHXHeaders`           | `httputil.CSRFTokenHXHeaders`           |
| `CSRFTokenFormField`           | `httputil.CSRFTokenFormField`           |
| `ForbiddenErrorHandler`        | `httputil.ForbiddenErrorHandler`        |
| `ErrCSRFInvalid`               | `httputil.ErrCSRFInvalid`               |
| `ErrCSRFConfig`                | `httputil.ErrCSRFConfig`                |
| `ErrorHandler`                 | `httputil.ErrorHandler`                 |

### Rate limiting (`ratelimit_reexport.go`)

| Deprecated `cqrshtmx.*`      | Replacement `httputil.*`              |
| ---------------------------- | ------------------------------------- |
| `RateLimiterConfig`          | `httputil.RateLimiterConfig`          |
| `RateLimiter`                | `httputil.RateLimiter`                |
| `KeyExtractor`               | `httputil.KeyExtractor`               |
| `RateLimiterMiddleware`      | `httputil.RateLimiterMiddleware`      |
| `NewRateLimiter`             | `httputil.NewRateLimiter`             |
| `DefaultRateLimiterConfig`   | `httputil.DefaultRateLimiterConfig`   |
| `KeyExtractorFromRemoteAddr` | `httputil.KeyExtractorFromRemoteAddr` |
| `KeyExtractorFromClientIP`   | `httputil.KeyExtractorFromClientIP`   |
| `DefaultRateLimit`           | `httputil.DefaultRateLimit`           |
| `DefaultRateWindow`          | `httputil.DefaultRateWindow`          |
| `DefaultRateTTL`             | `httputil.DefaultRateTTL`             |

### Server-Timing (`server_timing_reexport.go`)

| Deprecated `cqrshtmx.*`      | Replacement `httputil.*`              |
| ---------------------------- | ------------------------------------- |
| `ServerTiming`               | `httputil.ServerTiming`               |
| `ServerTimingMiddleware`     | `httputil.ServerTimingMiddleware`     |
| `ServerTimingMiddlewareWhen` | `httputil.ServerTimingMiddlewareWhen` |
| `ServerTimingFromContext`    | `httputil.ServerTimingFromContext`    |
| `WithServerTiming`           | `httputil.WithServerTiming`           |
| `RecordServerTiming`         | `httputil.RecordServerTiming`         |
| `MeasureServerTiming`        | `httputil.MeasureServerTiming`        |

> **Migration is mechanical:** add `"github.com/larsartmann/httputil"` to your imports, change the `cqrshtmx.` prefix to `httputil.` for any of the symbols above, and remove the `cqrshtmx.` import if it's no longer needed. The types are aliases, so no behavior change.

### Security headers (`security.go`)

| Deprecated `cqrshtmx.*`               | Replacement `httputil.*`                                            |
| ------------------------------------- | ------------------------------------------------------------------- |
| `SecurityHeadersConfig`               | `httputil.SecurityHeadersConfig` (type alias)                       |
| `SecurityHeadersMiddleware`           | `httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig())` |
| `SecurityHeadersMiddlewareWithConfig` | `httputil.SecurityHeaders(cfg)`                                     |
| `RecommendedHSTS`                     | `httputil.RecommendedHSTS`                                          |
| `RecommendedCSP`                      | `httputil.RecommendedCSP`                                           |
| `SecurityHeaderSkip`                  | `httputil.SecurityHeaderSkip`                                       |

> **Note:** `cqrshtmx.SecurityHeadersMiddleware` applies secure defaults on a zero-value config (nosniff, DENY, strict-origin-when-cross-origin). `httputil.SecurityHeaders` does not — pass `httputil.DefaultSecurityHeadersConfig()` for the equivalent behavior, or construct your own `SecurityHeadersConfig` explicitly.

- **Context enrichment** (`cqrshtmx.ContextEnrichmentMiddleware`) — domain-aware (UserID + RequestID + CorrelationID).
- **Chain** (`cqrshtmx.Chain`) — curried signature (`Chain(mw...) func(http.Handler) http.Handler`) for composable stacking.

For these, use the `cqrshtmx.*` version. For everything in the table above, use `httputil.*`.

### 7. Decompression — security hardening against compressed-body bombs

cqrs-htmx's `decoder.go` reads request bodies with `io.LimitReader` to cap size, but this sees the **compressed** byte stream — a 1 KB gzipped body can decompress to gigabytes, bypassing the limit. `httputil.Decompression` middleware transparently decompresses `Content-Encoding: gzip/deflate` request bodies **with a hard ceiling** (`MaxDecompressionSize`, default 16 MiB) before the decoder sees them.

```go
handler := cqrshtmx.Chain(
    cqrshtmx.RecoveryMiddleware,
    httputil.Decompression(httputil.DefaultDecompressionConfig()), // bomb protection
    httputil.MaxBodySize(1 << 20),                                 // 1 MiB post-decompress
    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
    app.Middleware(),
)(mux)
```

The middleware is safe to leave in the chain even when no clients send compressed bodies — it passes through requests without `Content-Encoding` unchanged.

### 8. MaxBodySize — outer middleware guard (defense-in-depth)

cqrs-htmx provides per-handler body-size limiting via `cqrshtmx.WithMaxBodySize(n)`, which caps the body during JSON/form decoding. But a middleware-level guard rejects oversized bodies **before any processing** — before CSRF validation, before context enrichment, before the handler pipeline runs. This is defense-in-depth: even if a handler forgets `WithMaxBodySize`, the middleware catches it.

```go
handler := cqrshtmx.Chain(
    cqrshtmx.RecoveryMiddleware,
    httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig()),
    httputil.MaxBodySize(10 << 20), // 10 MiB hard ceiling for the entire API
    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
    cqrshtmx.HTMXMiddleware,
    app.Middleware(),
)(mux)
```

For handlers that need a smaller limit (e.g., a login form at 4 KB), compose with `WithMaxBodySize` — the tighter per-handler limit wins.

## See also

- `docs/research/2026-08-09_httputil-deep-dive.html` — latest adoption audit of httputil v0.11.0 in this codebase.
- `docs/research/2026-08-05_httputil-deep-dive.html` — prior adoption audit (v0.8.0).
- `examples/middleware-showcase/` — runnable example demonstrating Compression, CORS, ETag, RateLimiting, Metrics, and MiddlewareStack.
- `docs/guides/production-readiness.md` — production checklist (middleware stack, observability, security).
- `docs/guides/dispatch-middleware-ordering.md` — CQRS-layer (not HTTP-layer) middleware ordering.
- httputil's own `FEATURES.md` and `CHANGELOG.md` for the full 16-middleware catalogue.
