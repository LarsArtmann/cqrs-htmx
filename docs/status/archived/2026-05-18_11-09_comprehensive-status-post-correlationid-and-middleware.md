# Comprehensive Status Report — cqrs-htmx

**Date:** 2026-05-18 11:09 CEST | **Branch:** master | **Commits since last report:** 8

---

## a) FULLY DONE

### CorrelationID Pipeline Fix

- `EventOptionsFromContext` now reads `CorrelationIDFromContext`, validates as `id.CorrelationID`, and appends `event.WithCorrelationID(cid)` — the previously phantom pipe is now wired.
- Branded `CorrelationID` type exported with `NewCorrelationID()`, `ParseCorrelationID()`, `MustParseCorrelationID()` helpers.
- `ContextEnrichmentMiddleware` validates `X-Correlation-ID` as ULID; non-ULID values silently dropped (matching UserID behavior).

### AuthorizeMiddleware Identity Fix

- Now prefers branded `UserID` from context (`UserIDFromContext`) over raw extractor string.
- Falls back to extractor + `ParseUserID()` — unparseable ULIDs now return 401 instead of passing raw strings to Casbin.
- Handles nil extractor gracefully with 401.

### Context Key Type Safety

- Replaced string-based `contextKey`/`htmxContextKey` with empty-struct sentinel types (`userIDKey{}`, `correlationIDKey{}`, `htmxKey{}`).

### New Features

- **Request Logging middleware** (`logging.go`): `RequestLogging(formatter, writer)` with `DefaultLogFormatter`. Captures method, path, status, duration, correlation ID, and user ID. 6 test specs.
- **Rate Limiting middleware** (`ratelimit.go`): `RateLimiterMiddleware` with token-bucket per-key via `golang.org/x/time/rate`. 5 test specs. Global, per-key, and exempt-request modes supported.

### Documentation & Meta

- README updated for branded CorrelationID API.
- CHANGELOG `[Unreleased]` section captures all breaking changes.
- AGENTS.md updated with new gotchas and API contract.
- FEATURES.md test count updated (170→197), prod files (10→12), all outdated descriptions corrected.
- TODO_LIST.md fully completed — all items marked DONE.

### Tests

- Dead test in `coverage_test.go:237` fixed — "returns error when casbin enforce fails" now actually asserts error instead of success.
- 197 total specs (up from ~170).

---

## b) PARTIALLY DONE

- Nothing. All items in TODO_LIST.md and FEATURES.md are complete.

---

## c) NOT STARTED

- WebSocket Support (explicitly NOT_PLANNED).
- No open bugs or feature requests remaining from the TODO list.

---

## d) TOTALLY FUCKED UP!

- **Nothing currently broken.** All tests pass, lint is clean, race detector is clean.
- **LSP false positives remain:** The `golangci_lint_ls` LSP shows stale warnings for `golang.org/x/time/rate` import and `exhaustruct`/`gci` on test files that the CLI linter does not report. This is a known LSP cache issue (documented in AGENTS.md). CLI `golangci-lint run` reports 0 issues.

---

## e) WHAT WE SHOULD IMPROVE!

1. **CorrelationID README example narrative**: The README says "Correlation ID (strongly-typed, ULID-backed, auto-extracted...)" but consumers might still expect to pass arbitrary strings. Consider adding a section explaining ULID enforcement and how to generate valid correlation IDs (`NewCorrelationID()`).
2. **Rate limiter cleanup**: The per-key limiter map grows unbounded. Add a periodic cleanup of stale entries (or document that consumers should set a reasonable number of keys).
3. **Logging middleware lacks structured output**: `DefaultLogFormatter` outputs plain text. Consider a `JSONLogFormatter` for structured logging pipelines.
4. **Missing `exhaustruct` exclusion for `RateLimiterConfig`**: Already added to `.golangci.yml` but LSP cache may lag.
5. **Test count in FEATURES.md**: Set to 197 but will drift. Consider automating or removing exact counts.

---

## f) Top #25 Things We Should Get Done Next

### High Impact, Low Effort (Do Next)

1. Add `JSONLogFormatter` for structured request logging
2. Document rate limiter map cleanup strategy in godoc comments
3. Add README example for `RequestLogging` middleware usage
4. Add README example for `RateLimiterMiddleware` usage
5. Add godoc Example functions for new middleware (logging, rate limiting)

### High Impact, Medium Effort

6. Peer review of `CorrelationID` breaking change impact on consumers
7. Add rate limiter map entry expiration (LRU or time-based)
8. Add request body size limit middleware
9. Add CORS middleware (framework-agnostic version)
10. Add compression middleware (gzip/brotli response)

### Medium Impact

11. Add OpenTelemetry span creation in middleware
12. Add metrics middleware (request count, latency histograms)
13. Add health check handler (`/healthz`)
14. Add graceful shutdown helper
15. Investigate `go.work` removal to eliminate `GOWORK=off` requirement

### Long-term / Research

16. Explore HTMX v2 compatibility
17. Add SSE (Server-Sent Events) helper (when HTMX adds native support)
18. Add WebSocket helper (if go-cqrs-lite adds event streaming)
19. Investigate `go 1.27` upgrade (runtime improvements)
20. Consider `slog` integration for structured logging throughout the library
21. Benchmark the `perKeyLimiter` under high concurrency (thundering herd)
22. Add property-based tests for rate limiter (gopter)
23. Add integration test with real Casbin policy file loading
24. Evaluate whether `contextKey` type aliasing for consumer extensions is needed
25. Add migration guide doc for consumers upgrading from v0.x to v1.0+

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the `RateLimiterMiddleware` provide a `RateLimiterConfig.KeyExtractor` that defaults to `RemoteAddr` instead of the empty-string global key?**

Right now, when `KeyExtractor` is nil, all requests share one key (`""`). This is a safe default but provides poor protection against distributed DoS. A `RemoteAddr`-based default would be more protective but could cause issues behind reverse proxies (the remote addr is the proxy, not the client). We don't know the typical deployment topology of consumers.

Options:

- A) Keep nil = global (safest, simplest)
- B) Change nil = RemoteAddr (more protective but could surprise proxy users)
- C) Add a `KeyExtractorFromRemoteAddr` helper function and document it prominently

I lean toward **C** but want confirmation before adding it.

---

## Metrics Snapshot

| Metric      | Before | After | Delta |
| ----------- | ------ | ----- | ----- |
| Test specs  | 170    | 197   | +27   |
| Prod files  | 10     | 12    | +2    |
| Coverage    | 95.7%  | 95.7% | —     |
| Lint issues | 0      | 0     | —     |
| Benchmarks  | 10     | 10    | —     |

## Files Changed in This Session

| File                | Change                                                                   |
| ------------------- | ------------------------------------------------------------------------ |
| `context.go`        | Branded CorrelationID, EventOptionsFromContext wiring, empty-struct keys |
| `context_test.go`   | E2E event pipeline test, CorrelationID type tests                        |
| `middleware.go`     | ParseCorrelationID validation in ContextEnrichmentMiddleware             |
| `authz.go`          | AuthorizeMiddleware now prefers UserID from context                      |
| `app_test.go`       | 3 new AuthorizeMiddleware identity specs                                 |
| `hooks_test.go`     | CorrelationID through middleware test, non-ULID drop test                |
| `coverage_test.go`  | Fixed dead Enforce error test                                            |
| `htmx.go`           | Empty-struct htmxKey sentinel type                                       |
| `logging.go`        | **NEW** RequestLogging middleware                                        |
| `logging_test.go`   | **NEW** 6 request logging specs                                          |
| `ratelimit.go`      | **NEW** RateLimiterMiddleware                                            |
| `ratelimit_test.go` | **NEW** 5 rate limiting specs                                            |
| `FEATURES.md`       | Updated to 26 features, corrected descriptions                           |
| `README.md`         | CorrelationID examples updated for branded types                         |
| `CHANGELOG.md`      | [Unreleased] section with all changes                                    |
| `TODO_LIST.md`      | All items marked DONE                                                    |
| `AGENTS.md`         | New gotchas for CorrelationID, context keys, identity parsing            |
| `.golangci.yml`     | Added RateLimiterConfig to exhaustruct exclusion                         |

---

_Generated with Crush_ | _Assisted-by: Crush:hf:moonshotai/Kimi-K2.6_
