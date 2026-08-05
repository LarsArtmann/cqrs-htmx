# Leveraging httputil — The HTTP Middleware Partner

> How to pair `cqrs-htmx` with its sibling library [`github.com/larsartmann/httputil`](https://github.com/larsartmann/httputil) for the HTTP concerns cqrs-htmx deliberately does **not** re-implement: CORS, compression, body limits, client IP, production serving, metrics, ETag, and health-readiness.

## Why this guide exists

`cqrs-htmx` follows a **duck-typing** philosophy: it re-exports the three concerns it migrated into httputil (CSRF, keyed rate limiting, Server-Timing — see `csrf_reexport.go`, `ratelimit_reexport.go`, `server_timing_reexport.go`) and keeps richer, domain-aware re-implementations where a generic version would lose value (Recovery with errorfamily, Logging with dispatch-error capture, context enrichment with domain IDs).

For everything else — the everyday HTTP middleware a browser-facing CQRS app needs — `cqrs-htmx` intentionally stays out of the way. That middleware lives in **httputil**. This guide maps each concern to the exact httputil symbol so you import what you need with zero glue.

> **Import once, compose freely.** Every httputil middleware is a `func(http.Handler) http.Handler`, so it drops straight into `cqrshtmx.Chain(...)`.

## Concern → middleware map

| Concern | httputil symbol | Notes |
| --- | --- | --- |
| Cross-origin browser API access | `httputil.CORS(cfg)` + `DefaultCORSConfig()` | `DenyUnmatched: true` by default since v0.7.0; wildcard `*.example.com` rejects lookalikes. |
| Response compression (htmx.js, JSON, SSE text) | `httputil.Compression(cfg)` + `DefaultCompressionConfig()` | Pooled gzip/deflate writers; skips already-compressed content types. |
| Body-size guard for JSON/form decoders | `httputil.MaxBodySize(maxBytes)` | Rejects oversized bodies *before* decode → pairs with `cqrshtmx.ErrRequestTooLarge` (413). |
| Client IP behind a reverse proxy | `httputil.ClientIPMiddleware`, `httputil.ClientIP(r)`, `httputil.IsTrustedProxy` | Trusted-proxy CIDR matching; feeds `KeyExtractorFromClientIP` for correct rate limiting. |
| Production server with timeouts | `httputil.NewServer(cfg, h)` + `DefaultServerConfig()` | `Validate()`d config; sane `Read`/`Write`/`Idle`/`ReadHeader` timeouts. **Never use bare `http.ListenAndServe`.** |
| Metrics recording | `httputil.Metrics(cfg)` + `DefaultMetricsConfig()` + `MetricsRecorder` interface | Plug in Prometheus/etc. via the interface. |
| Dynamic response ETagging | `httputil.ETag(cfg)` + `DefaultETagConfig()` | Buffers responses, hashes, serves `304` on `If-None-Match`. (cqrs-htmx already does *static-asset* ETagging for htmx.js / OpenAPI / event catalog.) |
| Validated middleware stack | `httputil.NewMiddlewareStack()` | Enforces "Recovery outermost", rejects duplicate names — a safer alternative to bare `Chain`. |
| Health: liveness | `httputil.LiveHandler()` | Static 200 — process is up. |
| Health: readiness | `httputil.ReadyHandlerWithProbe(func() bool)` | 200/503 based on a probe (deps ready). |
| Health: mount conventional endpoints | `httputil.RegisterHealth(mux)` | Mounts `/healthz`, `/livez`, `/readyz` in one call. |
| Request-level timeout | `httputil.Timeout(d)` | Bounds the **whole** HTTP request. Distinct from `cqrshtmx.Config.Timeout`, which bounds only the CQRS dispatch. Use both: request-level as a safety net, dispatch-level for per-command SLAs. |
| Error classification registration | `httputil.RegisterErrorClassifications()` | Call once at startup so stdlib HTTP errors and httputil codes (`http.write_failed`, `http.compress_write_failed`, …) classify through `cqrshtmx.MapError` via `errorfamily.Classify`. |
| HTTP-spec compliance testing | `httputil/httpspec.Run(t, h, opts...)` | 18 standard specs (index reachable, unknown→404, HEAD/OPTIONS handled, TRACE/CONNECT rejected, error responses carry Content-Type, no version leak, `X-Content-Type-Options` present). `cqrs-htmx` uses this internally — see `httpspec_compliance_test.go`. |
| Response capture in tests | `httputil.NewResponseRecorder(w)` | Lightweight status/body capture. (cqrs-htmx's own `StatusRecorder` is richer — captures dispatch errors — prefer it inside the pipeline.) |

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
    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
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
    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
    cqrshtmx.HTMXMiddleware,
    app.Middleware(),
)(mux)

// Serve with real timeouts (never bare http.ListenAndServe).
srv, err := httputil.NewServer(httputil.DefaultServerConfig(), handler)
if err != nil {
    log.Fatal(err)
}
log.Fatal(srv.ListenAndServe())
```

### 3. Correct rate limiting behind a proxy

Without client-IP extraction, `KeyExtractorFromClientIP` sees the proxy's IP and all clients share one bucket. Put `ClientIPMiddleware` in the chain (after the proxy trust list is configured):

```go
handler := cqrshtmx.Chain(
    cqrshtmx.SecurityHeadersMiddleware,
    httputil.ClientIPMiddleware,            // populates client IP in context
    cqrshtmx.RateLimiterMiddleware(cqrshtmx.DefaultRateLimiterConfig()),
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

```go
func main() {
    // Makes stdlib HTTP errors + httputil middleware codes classify through
    // cqrshtmx.MapError (via errorfamily.Classify). Idempotent.
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
- **Security headers** (`cqrshtmx.SecurityHeadersMiddleware`) — superset config (`PermissionsPolicy`, `Custom` map, `SecurityHeaderSkip` sentinel). *Note: httputil has a weaker parallel version — a known split brain tracked in `docs/research/2026-08-05_httputil-deep-dive.html`.*
- **Context enrichment** (`cqrshtmx.ContextEnrichmentMiddleware`) — domain-aware (UserID + RequestID + CorrelationID).
- **Chain** (`cqrshtmx.Chain`) — curried signature (`Chain(mw...) func(http.Handler) http.Handler`) for composable stacking.

For these, use the `cqrshtmx.*` version. For everything in the table above, use `httputil.*`.

## See also

- `docs/research/2026-08-05_httputil-deep-dive.html` — full adoption audit of httputil in this codebase.
- `docs/guides/production-readiness.md` — production checklist (middleware stack, observability, security).
- `docs/guides/dispatch-middleware-ordering.md` — CQRS-layer (not HTTP-layer) middleware ordering.
- httputil's own `FEATURES.md` and `CHANGELOG.md` for the full 16-middleware catalogue.
