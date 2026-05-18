# Comprehensive Status Report — cqrs-htmx Session

**Date:** 2026-05-18 19:47 CEST | **Branch:** master | **Working Tree:** Modified (9 files)

---

## Session Context

This session directly addressed every item listed under "WHAT WE SHOULD IMPROVE" and "Top #25 Things We Should Get Done Next" from the 11:11 CEST status report. The focus was on productionizing the two new middleware features added in the morning session (Request Logging, Rate Limiting) by adding structured output, documentation, benchmarks, godoc examples, and resolving the open design question about rate limiter key extraction.

---

## a) FULLY DONE

### 1. JSONLogFormatter — Structured Request Logging (`logging.go`, `logging_test.go`)

**Problem:** `DefaultLogFormatter` produced plain text only. Modern log pipelines (ELK, Loki, Datadog) require structured JSON.

**Fix:**

- Added `JSONLogFormatter` function matching the `LogFormatter` signature
- Outputs compact JSON: `{"method":"GET","path":"/users","status":"OK","duration":"1.234ms","correlation_id":"...","user_id":"..."}`
- Includes `query` field when `RawQuery` is non-empty
- Handles `json.Marshal` errors gracefully (returns `{"error":"json marshal failed"}`)
- Added 2 BDD test specs for JSON formatting and context ID inclusion
- Added `ExampleJSONLogFormatter` in `example_test.go`

### 2. KeyExtractorFromRemoteAddr — Per-IP Rate Limiting (`ratelimit.go`, `ratelimit_test.go`)

**Problem:** The morning session's open design question ("Should nil KeyExtractor default to RemoteAddr?") was unresolved. Global rate limiting is safe but weak for DoS protection.

**Fix:**

- Added `KeyExtractorFromRemoteAddr() KeyExtractor` helper function
- Returns `r.RemoteAddr` as the rate-limit key
- **Prominently documented** the reverse-proxy warning in godoc: behind nginx/Cloudflare/AWS ALB, `RemoteAddr` is the proxy's IP
- Added BDD test spec proving per-IP isolation: two requests from same IP hit limit, request from different IP succeeds
- Added `ExampleRateLimiterMiddleware` in `example_test.go` demonstrating per-IP usage
- This implements **Option C** from the morning's design question: provide an opt-in helper, keep nil = global default

### 3. Godoc Example Functions for New Middleware (`example_test.go`)

**Problem:** No godoc `Example*` functions existed for `RequestLogging` or `RateLimiterMiddleware`. Consumers had to read source or BDD tests.

**Fix:**

- `ExampleRequestLogging` — demonstrates plain-text logging with `RequestLogging(nil, writer)`
- `ExampleJSONLogFormatter` — demonstrates structured JSON logging
- `ExampleRateLimiterMiddleware` — demonstrates per-IP rate limiting with `KeyExtractorFromRemoteAddr()`
- All examples compile and pass via `go test`

### 4. Benchmarks for New Middleware (`benchmark_test.go`)

**Problem:** No benchmarks existed for `RequestLogging` or `RateLimiterMiddleware`. Both wrap `ResponseWriter` which adds overhead.

**Fix:**

- `BenchmarkRequestLogging/DefaultFormatter` — baseline plain-text logging
- `BenchmarkRequestLogging/JSONFormatter` — JSON marshaling overhead
- `BenchmarkRequestLogging/WithContextIDs` — context value retrieval overhead
- `BenchmarkRateLimiterMiddleware/Global` — single shared limiter
- `BenchmarkRateLimiterMiddleware/PerKey` — fixed-key limiter lookup
- `BenchmarkRateLimiterMiddleware/RemoteAddr` — per-IP key extraction + lookup

**Results** (AMD Ryzen AI MAX+ 395):

| Benchmark                        | ns/op  |
| -------------------------------- | ------ |
| RequestLogging/DefaultFormatter  | ~1,559 |
| RequestLogging/JSONFormatter     | ~2,513 |
| RequestLogging/WithContextIDs    | ~1,955 |
| RateLimiterMiddleware/Global     | ~1,659 |
| RateLimiterMiddleware/PerKey     | ~1,838 |
| RateLimiterMiddleware/RemoteAddr | ~1,679 |

### 5. README Examples for New Middleware (`README.md`)

**Problem:** README had no examples for `RequestLogging` or `RateLimiterMiddleware`.

**Fix:**

- Added "Request Logging" subsection under Middleware with plain-text and JSON examples
- Added "Rate Limiting" subsection with complete config example
- Documented `KeyExtractorFromRemoteAddr()` with proxy warning
- Documented exempt-request pattern (empty string return skips limiting)
- Documented all config fields: Limit, Window, Burst, KeyExtractor

### 6. README ULID Explanation (`README.md`)

**Problem:** README stated "Correlation ID (strongly-typed, ULID-backed...)" but didn't explain _why_ ULID enforcement exists or how it benefits consumers.

**Fix:**

- Added "Why ULID?" subsection under Context Propagation
- Lists four benefits: collision-free keys, lexicographic sortability, type safety, consistency across event metadata/Casbin/context
- Documents silent-drop behavior for invalid IDs
- References `NewUserID()` / `NewCorrelationID()` for generation
- References `ParseUserID()` / `ParseCorrelationID()` for safe parsing

### 7. FEATURES.md Update (`FEATURES.md`)

**Problem:** FEATURES.md had 25 features, no Rate Limiting entry, stale metrics.

**Fix:**

- Added Feature #26: Rate Limiting (FULLY_FUNCTIONAL)
- Updated Feature #25 description to include `JSONLogFormatter`
- Updated test specs: 197 → **200**
- Updated benchmarks: 10 → **16**
- Updated date: 2026-05-16 → **2026-05-18**

### 8. Rate Limiter Map Growth Documentation (`ratelimit.go`, `AGENTS.md`)

**Problem:** `perKeyLimiter.limiters` is an unbounded `map[string]*rate.Limiter`. Memory leak risk for deployments with many unique keys.

**Fix:**

- Added godoc NOTE on `RateLimiterMiddleware` documenting the unbounded growth risk
- Added AGENTS.md Gotcha #25 documenting the limitation and recommending bounded key spaces or periodic cleanup
- No implementation change — this is a documented architectural constraint

### 9. Lint Fixes (3 issues across 3 files)

| File                | Issue                                         | Fix                                      |
| ------------------- | --------------------------------------------- | ---------------------------------------- |
| `logging.go`        | `errchkjson` — unchecked `json.Marshal` error | Check error, return fallback JSON string |
| `example_test.go`   | `revive` — unused `line` parameter            | Rename to `_`                            |
| `ratelimit_test.go` | `goconst` — `"192.168.1.1:1234"` repeated 3×  | Extract to `const ip1`, add `const ip2`  |

---

## b) PARTIALLY DONE

- Nothing. All identified work from the morning session's improvement list is complete.

---

## c) NOT STARTED

- WebSocket/SSE support (explicitly NOT_PLANNED — no HTMX native support yet).
- No open bugs or feature requests from TODO_LIST.md remaining.

---

## d) TOTALLY FUCKED UP!

- **Nothing currently broken.** All 200 tests pass, lint is clean (0 issues), race detector is clean.
- **LSP false positives:** The `golangci_lint_ls` LSP may show stale warnings for `golang.org/x/time/rate` import and `exhaustruct`/`gci` on test files. The CLI linter (`golangci-lint run`) reports **0 issues**. This is a known LSP cache issue documented in AGENTS.md.

---

## e) WHAT WE SHOULD IMPROVE!

1. **Rate limiter map grows unbounded** — `perKeyLimiter.limiters` is a `map[string]*rate.Limiter` with no cleanup. Documented but not fixed. For long-running deployments with per-IP limiting, memory will grow linearly with unique visitors.
2. **No `slog` integration** — The logging middleware uses a callback-based `LogWriter`. A `SlogLogFormatter` that returns `slog.Record` or integrates with `slog.Handler` would fit modern Go logging patterns.
3. **Test counts in FEATURES.md will drift** — Currently hardcoded to 200. Consider removing exact counts or automating via CI.
4. **No benchmark for new middleware under high concurrency** — Benchmarks are single-threaded. A thundering-herd benchmark for the rate limiter would reveal contention on the `sync.RWMutex`.
5. **go.work friction** — `GOWORK=off` is still required on every command due to parent `go.work`.
6. **CorrelationID README example narrative could be stronger** — The "Why ULID?" section exists but could include a migration snippet for consumers upgrading from raw strings.
7. **No `ExampleAuthorizeMiddleware`** — The standalone Casbin middleware lacks a godoc example.

---

## f) Top #25 Things We Should Get Done Next

### Immediate (Next 1-2 Weeks)

1. Implement LRU or time-based eviction for rate limiter map (fix memory leak)
2. Add request body size limit middleware
3. Add CORS middleware (framework-agnostic)
4. Add compression middleware (gzip/brotli response)
5. Add `ExampleAuthorizeMiddleware` to `example_test.go`

### Short-term (Next Month)

6. Peer review CorrelationID breaking change impact on known consumers
7. Add OpenTelemetry span creation in middleware
8. Add metrics middleware (request count, latency histograms)
9. Add health check handler (`/healthz`)
10. Add graceful shutdown helper
11. Add `SlogLogFormatter` for `log/slog` integration
12. Thundering-herd benchmark for rate limiter

### Medium-term (Next Quarter)

13. Investigate `go.work` removal to eliminate `GOWORK=off` requirement
14. Explore HTMX v2 compatibility
15. Add SSE helper when HTMX adds native support
16. Investigate `go 1.27` upgrade
17. Consider `slog` integration throughout the library (not just middleware)
18. Property-based tests for rate limiter (gopter)
19. Integration test with real Casbin policy file loading

### Long-term / Research

20. Evaluate `contextKey` type aliasing for consumer extensions
21. Migration guide for v0.x → v1.0+ consumers
22. Split-brain audit: are there other ID types that should be branded?
23. Security audit: gosec/govulncheck in CI (add to GitHub Actions)
24. Consider middleware ordering documentation/guide
25. Add request ID middleware (separate from correlation ID)

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should we implement automatic cleanup for the rate limiter map, or is documenting the limitation sufficient?**

The `perKeyLimiter.limiters` map grows without bound. For a public API with per-IP limiting, every unique visitor creates a new `*rate.Limiter` that lives forever.

**Option A (document only):** Keep current implementation. Add strong warnings in godoc and README. Consumers with high-cardinality key spaces must implement their own rate limiter or use a bounded key space. Simple, zero complexity added.

**Option B (time-based eviction):** Add `lastUsed time.Time` per entry and a background goroutine that sweeps stale limiters. Accurate but adds goroutine lifecycle management and requires `Stop()` or context cancellation.

**Option C (LRU cache):** Replace the map with an LRU cache (e.g., `github.com/hashicorp/golang-lru`) with a configurable max size. Oldest entries are evicted automatically. Adds a dependency but is deterministic and self-contained.

**Option D (hybrid):** Document the limitation AND provide a `BoundedRateLimiterConfig` variant that uses LRU. Keep the current middleware simple for the common case.

I lean toward **D** — the current middleware should stay simple (documented limitation), but we should offer a bounded variant for production deployments with unbounded key spaces. The decision depends on whether we want to add `hashicorp/golang-lru` as a dependency.

---

## Metrics Snapshot

| Metric         | Before Session | After Session | Delta |
| -------------- | -------------- | ------------- | ----- |
| Test specs     | 197            | 200           | +3    |
| Prod files     | 12             | 12            | —     |
| Coverage       | 95.9%          | 95.8%         | -0.1% |
| Lint issues    | 0              | 0             | —     |
| Benchmarks     | 10             | 16            | +6    |
| Features       | 25             | 26            | +1    |
| Godoc examples | 6              | 9             | +3    |

## File Inventory (Production)

| #   | File            | Lines | Purpose                                     |
| --- | --------------- | ----- | ------------------------------------------- |
| 1   | `app.go`        | ~190  | App builder, Config, Command/Query handlers |
| 2   | `handler.go`    | ~136  | dispatch orchestration, timeout             |
| 3   | `options.go`    | ~341  | HandlerOption, decoders, validation, authz  |
| 4   | `response.go`   | ~179  | HTMX response builder (fluent API)          |
| 5   | `authz.go`      | ~112  | Enforcer, Authorize, Enforce, middleware    |
| 6   | `context.go`    | ~114  | UserID/CorrelationID context + event opts   |
| 7   | `errors.go`     | ~158  | Sentinels, MapError, error handlers         |
| 8   | `htmx.go`       | ~173  | HTMXRequest, accessors, context storage     |
| 9   | `notify.go`     | ~78   | Notification HandlerOptions + builder       |
| 10  | `middleware.go` | ~58   | ContextEnrichment, HTMX, Chain              |
| 11  | `logging.go`    | ~115  | RequestLogging + JSONLogFormatter           |
| 12  | `ratelimit.go`  | ~120  | RateLimiterMiddleware + KeyExtractor helper |

## Files Changed in This Session

| File                | Change                                                                                   |
| ------------------- | ---------------------------------------------------------------------------------------- |
| `logging.go`        | Added `JSONLogFormatter`, handled `json.Marshal` error                                   |
| `logging_test.go`   | Added 2 JSON formatter test specs                                                        |
| `ratelimit.go`      | Added `KeyExtractorFromRemoteAddr()`, documented unbounded map growth                    |
| `ratelimit_test.go` | Added per-IP rate limiting spec, extracted constants for lint                            |
| `example_test.go`   | Added `ExampleRequestLogging`, `ExampleJSONLogFormatter`, `ExampleRateLimiterMiddleware` |
| `benchmark_test.go` | Added `BenchmarkRequestLogging` (3 subs), `BenchmarkRateLimiterMiddleware` (3 subs)      |
| `README.md`         | Added Request Logging + Rate Limiting sections, "Why ULID?" explanation                  |
| `FEATURES.md`       | Added Feature #26, updated metrics (200 specs, 16 benchmarks)                            |
| `AGENTS.md`         | Added Gotcha #25 for rate limiter unbounded map                                          |

---

_Generated with Crush_ | _Assisted-by: Crush:kimi-for-coding_
