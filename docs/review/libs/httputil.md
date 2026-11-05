# httputil Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/httputil` (root module; `server_timing` submodule has its own report) · Checkout: `/home/lars/projects/httputil`

## Verdict at a glance

| Pin | Latest (verified 2026-11-05) | Prod import sites | Adoption | Score |
| --- | --- | --- | --- | --- |
| v1.8.0 | v1.8.0 (= latest) | 53 files | **Used — the HTTP middleware substrate** | **73/100 count · ~85 effective (library-principle discounts)** |

## What the library is

The LarsArtmann HTTP middleware toolkit: CSRF (nosurf-based), keyed rate limiting, per-request CSP nonces, security headers, server lifecycle (`NewServer` with validated `ServerConfig`, TLS, shutdown), client-IP/trusted proxies, CORS, compression/decompression, request/body limits, request IDs, logging, metrics, Server-Timing (submodule), i18n language negotiation, K8s health endpoints, `MiddlewareStack` composition with validation. cqrs-htmx's CSRF/Server-Timing/rate-limit re-exports are thin deprecated aliases over this library (v5 removes them; internal modules already import httputil directly — 0 deprecated-alias uses verified).

## How cqrs-htmx uses it today

Direct symbol census: CSRF suite (`CSRFConfig` ×32, `CSRFMiddleware` ×29, token formatters/test helper, `CSRFResponseHeaderMiddleware`, `WithCSRFToken`, `ErrCSRFInvalid`), keyed rate limiting (`KeyedRateLimiter*` ×45 across limiter/config/middleware/key extractors), nonce CSP (`Nonce`/`NonceFromRequest` ×13, `RecommendedCSPWithNonce` ×4), `SecurityHeaders` ×11 (+configs/defaults), server lifecycle (`NewServer` ×13, `ServerConfig` ×7), client IP (`ClientIP`, `KeyExtractorFromClientIP/FromRemoteAddr`, middleware), `ParseUintQuery` ×4, `ForbiddenHandler` ×4, `RegisterHealth` ×3, `RegisterErrorClassifications` (wired from `app.go` via `sync.Once`). `examples/middleware-showcase` demonstrates all 8 middleware in one validated `MiddlewareStack`. Consumer guide: [docs/guides/leveraging-httputil.md](../../guides/leveraging-httputil.md).

## Capability inventory

| # | Capability | Status | Evidence / reason |
| --- | --- | --- | --- |
| 1 | CSRF suite (middleware, config, token surface, test helper) | 🟢 Fully leveraged | 100+ refs; the deprecated root re-export is on the documented v5 glide path |
| 2 | Keyed rate limiting (+key extractors) | 🟢 Fully leveraged | 45+ refs across root/examples/setup |
| 3 | Per-request CSP nonce | 🟢 Fully leveraged | `Nonce`/`NonceFromRequest`/`NonceConfig`/`RecommendedCSPWithNonce` |
| 4 | Security headers | 🟢 Fully leveraged | `RecommendedSecurityMiddleware()` builds on `SecurityHeaders` + `RecommendedPermissionsPolicy` |
| 5 | Server lifecycle (`NewServer`, `ServerConfig`, TLS, shutdown) | 🟢 Fully leveraged | e2e/examples/setup run paths; `RunHandler` keeps no-timeout SSE posture |
| 6 | Client IP + trusted proxies | 🟢 Fully leveraged | extractors + `ClientIP` middleware; rate-limit keys derive through them |
| 7 | CORS | 🟡 Partially used | Documented wrap pattern for cross-origin dashboards + showcase example; not a library default **by design** (never enforce origins consumers disagree with) |
| 8 | Compression/Decompression | 🟡 Partially used | Showcase + guide; SSE interplay makes it consumer opt-in |
| 9 | HTTP metrics (`Metrics`, `MetricsConfig`, `MetricsRecorder`) | 🟡 Partially used | Adopted indirectly: `setup` forwards `appkit.ServiceConfig.Metrics` (appkit composes httputil metrics) — `setup/run_appkit.go:82` |
| 10 | K8s health endpoints (`RegisterHealth`, `LiveHandler`, `ReadyHandler*`) | 🟡 Partially used | `RegisterHealth` used; readiness is owned by the go-health bridge + `ProjectionReadinessCheck` (deliberate, richer) |
| 11 | `MiddlewareStack` composition + validation | 🔴 Missed | Showcase only; `RecommendedSecurityMiddleware` builds chains by hand — stack validation (recovery-first checks, duplicate names) would harden the recommended chains. Low effort, low-medium impact |
| 12 | `MaxBodySize` middleware | ⚪ N/A | Root enforces body limits at the decode layer with per-handler `WithMaxBodySize` override (`options_validate.go:99`, `decoder.go:22-56`) — semantics a global middleware cannot express |
| 13 | Language negotiation suite | ⚪ N/A | The library has no localized strings |
| 14 | `RequestID` middleware | ⚪ N/A | Root's `ContextEnrichmentMiddleware` mints branded ULIDs via go-cqrs-lite `id` (type-consistency wins over httputil's time-ordered IDs) |
| 15 | `Logging` middleware | ⚪ N/A | `RequestLoggingSlog` is slog-native and pre-dates; consolidation possible but low value |
| 16 | `Timeout` middleware | ⚪ N/A | SSE-safe: server-wide WriteTimeout severs streams (gotcha 46); setup pins `appkit.NoTimeout` |
| 17 | ETag middleware | ⚪ N/A | `internal/hx` composes `go-etag/server` primitives directly for immutable JSON + `HashTag` (see [go-etag report](go-etag.md)) — the lower-level choice fits the embed/cache contract |

## Score derivation

17 rows − 6 N/A = **11 applicable**; 6 full ×1 + 4 partial ×0.5 + 1 missed ×0 = **8 → 73/100** (band: high). Every "partial" is a **deliberate library-principle choice** (opt-in, never default-enforced), so the effective posture for a library is **~85 very high**; the one actionable miss is `MiddlewareStack` validation.

## Version currency

v1.8.0 pinned = v1.8.0 latest (verified `git ls-remote --tags origin`, 2026-11-05).

## Should cqrs-htmx use it (more)?

Yes — correctly and increasingly so (deprecated re-exports shrink the surface toward v5). One TODO-worthy item: build `RecommendedSecurityMiddleware()` through `httputil.NewMiddlewareStack` to inherit validation.

## How to verify

```bash
rg -oN 'httputil\.[A-Z][A-Za-z0-9]*' --type go -g '!scripts/**' . | sed 's/.*://' | sort | uniq -c | sort -rn | head -30
rg -c 'cqrshtmx\.(CSRF|RateLimiter|ServerTiming|SecurityHeaders)' adminui dashboardui loginpage setup usermgmt examples -g '*.go'   # 0 = no deprecated-alias dogfooding
git -C ~/projects/httputil ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- [docs/LIBRARY_INTEGRATIONS.md](../../LIBRARY_INTEGRATIONS.md) (2026-05-13, superseded) — pre-dated the httputil extraction of CSRF/rate-limiting.
