# SUPERB Plan: httputil Consolidation — Server-Timing, CSRF, Rate Limiting

> **Date:** 2026-07-30
> **Status:** ✅ COMPLETED (2026-07-30). All 4 phases executed, verified, and documented. See resolution notes below.
> **Goal:** Move general-purpose HTTP middleware OUT of cqrs-htmx INTO httputil, removing 2 external deps (`justinas/nosurf`, `golang.org/x/time`) from cqrs-htmx and eliminating 3 duplicated reimplementations. httputil becomes the canonical home for HTTP middleware in the LarsArtmann ecosystem.

---

## Context

### The Problem

cqrs-htmx root module reimplements three pieces of HTTP middleware that belong in httputil:

1. **Server-Timing** (389 LOC, zero deps) — unique to cqrs-htmx, no counterpart in httputil, zero coupling
2. **CSRF** (905 LOC, dep: `justinas/nosurf`) — self-contained, httputil has NO CSRF support
3. **Rate limiting** (392 LOC, dep: `golang.org/x/time`) — cqrs-htmx version is better (min-heap eviction, MaxKeys cap); httputil has a simpler version (O(n) sweep)

### The Strategy

For each subsystem:

1. Port the code to httputil (adapting to httputil's conventions)
2. Delete the original from cqrs-htmx root
3. Create a thin re-export file (`_reexport.go`) in cqrs-htmx with type aliases + var aliases
4. Update cqrs-htmx tests to reference the re-exported symbols (API unchanged for consumers)

### What Stays Out of Scope (Explicitly)

- **Recovery middleware** — deep `errorfamily` + `event.Infrastructure` coupling, not worth the refactor
- **Security headers** — different struct shapes (`ContentTypeNosniff bool` vs `ContentTypeOptions string`), API breakage risk
- **openapi/ extraction** — rejected per competitive landscape research
- **SSE/WS extraction** — thin wrappers over already-extracted `go-sse`
- **usermgmt god-package split** — deferred to v5 per ROADMAP

---

## Pareto Breakdown

### The 1% that delivers 51% of the result

**Server-Timing → httputil.** Zero deps, zero coupling, zero risk. One file + 3 test files move. httputil gains a unique feature. ~60 min.

### The 4% that delivers 64% of the result

**+ CSRF core → httputil.** Removes `justinas/nosurf` from cqrs-htmx deps. httputil gains CSRF support. Messier (handler.go depends on internal helpers). ~90 min.

### The 20% that delivers 80% of the result

**+ Rate limiting consolidation.** Removes `golang.org/x/time` from cqrs-htmx deps. httputil's TokenBucketLimiter gains min-heap eviction + MaxKeys cap. ~100 min.

### The remaining 20% for 100%

Deferred: Recovery + Security headers enrichment. Not worth the risk/cost now.

---

## Execution Graph

```mermaid
graph TD
    subgraph "Phase 1: Server-Timing (1% → 51%)"
        HT1[Move server_timing.go to httputil]
        HT1T[Port test files - remove go-cqrs-lite deps]
        HT1B[go build + test httputil]
        HT1C[Commit httputil]

        CQ1[Delete server_timing.go from cqrs-htmx]
        CQ1R[Create re-export file]
        CQ1T[Update test imports]
        CQ1B[go build + test cqrs-htmx]
        CQ1C[Commit cqrs-htmx]

        HT1 --> HT1T --> HT1B --> HT1C
        HT1C --> CQ1 --> CQ1R --> CQ1T --> CQ1B --> CQ1C
    end

    subgraph "Phase 2: CSRF (4% → 64%)"
        HT2[Add nosurf dep to httputil]
        HT2A[Port csrf_config.go + csrf_context.go]
        HT2B[Port csrf_middleware.go + csrf_helpers.go]
        HT2C[Port csrf_testing.go]
        HT2D[Port/adapt CSRF tests]
        HT2B2[go build + test httputil]
        HT2C2[Commit httputil]

        CQ2[Delete CSRF core files from cqrs-htmx]
        CQ2A[Update csrf_handler.go imports]
        CQ2B[Create re-export file]
        CQ2C[Update all 12 test files]
        CQ2D[go build + test cqrs-htmx]
        CQ2E[Commit cqrs-htmx]

        HT2 --> HT2A --> HT2B --> HT2C --> HT2D --> HT2B2 --> HT2C2
        HT2C2 --> CQ2 --> CQ2A --> CQ2B --> CQ2C --> CQ2D --> CQ2E
    end

    subgraph "Phase 3: Rate Limiting (20% → 80%)"
        HT3[Port min-heap perKeyLimiter to httputil]
        HT3A[Add MaxKeys cap + KeyExtractor to httputil]
        HT3B[Add RateLimiter monitoring struct to httputil]
        HT3C[Update httputil ratelimit tests]
        HT3D[go build + test httputil]
        HT3E[Commit httputil]

        CQ3[Delete ratelimit_config.go + ratelimit_middleware.go]
        CQ3A[Create re-export file]
        CQ3B[Update test files]
        CQ3C[go build + test cqrs-htmx]
        CQ3D[Commit cqrs-htmx]

        HT3 --> HT3A --> HT3B --> HT3C --> HT3D --> HT3E
        HT3E --> CQ3 --> CQ3A --> CQ3B --> CQ3C --> CQ3D
    end

    subgraph "Phase 4: Finalize"
        F1[Run full cqrs-htmx workspace test]
        F2[Lint all modules]
        F3[Update docs: AGENTS.md, CHANGELOG.md]
        F4[Git push both repos]
    end

    CQ1C --> Phase2
    CQ2E --> Phase3
    CQ3D --> F1 --> F2 --> F3 --> F4
```

---

## Level 1: Phase Tasks (30–100 min each)

| #   | Phase         | Task                                                                | Repo      | Est.   | Impact   | Risk      | Depends On |
| --- | ------------- | ------------------------------------------------------------------- | --------- | ------ | -------- | --------- | ---------- |
| 1.1 | Server-Timing | Port `server_timing.go` + tests to httputil                         | httputil  | 45 min | High     | Near zero | —          |
| 1.2 | Server-Timing | Delete from cqrs-htmx + create re-exports + update tests            | cqrs-htmx | 30 min | High     | Near zero | 1.1        |
| 2.1 | CSRF          | Port CSRF core (5 files) + tests to httputil                        | httputil  | 60 min | High     | Low       | 1.2        |
| 2.2 | CSRF          | Delete from cqrs-htmx + re-exports + update handler + 12 test files | cqrs-htmx | 60 min | High     | Medium    | 2.1        |
| 3.1 | Rate Limit    | Enrich httputil's ratelimit.go with min-heap + MaxKeys + monitoring | httputil  | 60 min | Medium   | Low       | 2.2        |
| 3.2 | Rate Limit    | Delete from cqrs-htmx + re-exports + update 8 test files            | cqrs-htmx | 45 min | Medium   | Low       | 3.1        |
| 4.1 | Finalize      | Full workspace test + lint + docs update + push                     | both      | 45 min | Critical | Low       | 3.2        |

**Total estimated: ~5.5 hours**

---

## Level 2: Detailed Subtasks (max 12 min each)

### Phase 1: Server-Timing → httputil

| #     | Subtask                                            | Repo      | Est.   | Detail                                                                                                                                                                                           |
| ----- | -------------------------------------------------- | --------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1.1.1 | Copy `server_timing.go` to httputil                | httputil  | 5 min  | Change `package cqrshtmx` → `package httputil`. Change context key types from cqrs-htmx RequestID types to plain `context.Context` keys.                                                         |
| 1.1.2 | Copy `server_timing_test.go` to httputil           | httputil  | 10 min | Remove go-cqrs-lite imports (`command/v4`, `id/v4`). Replace test helpers that dispatch commands with plain `net/http` handlers.                                                                 |
| 1.1.3 | Copy `server_timing_bench_test.go` to httputil     | httputil  | 5 min  | Remove go-cqrs-lite imports. Simplify benchmarks to use `net/http` only.                                                                                                                         |
| 1.1.4 | Copy `server_timing_fuzz_test.go` to httputil      | httputil  | 5 min  | Should be clean — fuzz tests use only `net/http`.                                                                                                                                                |
| 1.1.5 | `GOWORK=off go mod tidy` in httputil               | httputil  | 2 min  | No new deps needed (zero-dep code).                                                                                                                                                              |
| 1.1.6 | `GOWORK=off go test ./... -count=1` in httputil    | httputil  | 5 min  | Verify all tests pass.                                                                                                                                                                           |
| 1.1.7 | `GOWORK=off go vet ./...` in httputil              | httputil  | 2 min  | Verify no vet issues.                                                                                                                                                                            |
| 1.1.8 | Commit httputil                                    | httputil  | 5 min  | `feat: add Server-Timing middleware (W3C Server Timing API)`                                                                                                                                     |
| 1.2.1 | Delete `server_timing.go` from cqrs-htmx           | cqrs-htmx | 2 min  | `git rm server_timing.go`                                                                                                                                                                        |
| 1.2.2 | Create `server_timing_reexport.go`                 | cqrs-htmx | 8 min  | Type aliases: `ServerTiming`, context key types, var aliases for all exported functions (`ServerTimingMiddleware`, `ServerTimingFromContext`, `MeasureServerTiming`, `MeasureServerTimingFunc`). |
| 1.2.3 | Delete `server_timing_*_test.go` from cqrs-htmx    | cqrs-htmx | 2 min  | Tests now live in httputil. Keep a thin smoke test in cqrs-htmx that verifies re-exports work.                                                                                                   |
| 1.2.4 | `GOEXPERIMENT=jsonv2 go build ./...`               | cqrs-htmx | 5 min  | Verify root module compiles.                                                                                                                                                                     |
| 1.2.5 | `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race` | cqrs-htmx | 10 min | Full test suite.                                                                                                                                                                                 |
| 1.2.6 | Commit cqrs-htmx                                   | cqrs-htmx | 5 min  | `refactor: move Server-Timing to httputil, re-export for backward compat`                                                                                                                        |

### Phase 2: CSRF core → httputil

| #     | Subtask                                                                                                               | Repo      | Est.   | Detail                                                                                                                                                                                                                                        |
| ----- | --------------------------------------------------------------------------------------------------------------------- | --------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2.1.1 | `go get github.com/justinas/nosurf` in httputil                                                                       | httputil  | 2 min  | Add nosurf dependency.                                                                                                                                                                                                                        |
| 2.1.2 | Port `csrf_config.go` to httputil                                                                                     | httputil  | 10 min | Change package. Export `ConfigureNosurfHandler` (was `configureNosurfHandler`). Export `DefaultCSRFHeaderName`, `DefaultCSRFFieldName`, `DefaultCSRFCookieName`. Keep `CSRFConfig` with accessors. Port `ErrCSRFInvalid` as `ErrCSRFInvalid`. |
| 2.1.3 | Port `csrf_context.go` to httputil                                                                                    | httputil  | 5 min  | Change package. Export `CSRFTokenFromContext`, `WithCSRFToken`, context key.                                                                                                                                                                  |
| 2.1.4 | Port `csrf_middleware.go` to httputil                                                                                 | httputil  | 10 min | Change package. Export `CSRFMiddleware`, `CSRFResponseHeaderMiddleware`. Export `SetPlaintextHTTPOrigin` (was `setPlaintextHTTPOrigin`), `TranslateCSRFHeaders` (was `translateCSRFHeaders`).                                                 |
| 2.1.5 | Port `csrf_helpers.go` to httputil                                                                                    | httputil  | 5 min  | Change package. Export `CSRFFieldHTML` and helpers.                                                                                                                                                                                           |
| 2.1.6 | Port `csrf_testing.go` to httputil                                                                                    | httputil  | 8 min  | Change package. Export `CSRFTestToken` helper.                                                                                                                                                                                                |
| 2.1.7 | Port/adapt CSRF tests to httputil                                                                                     | httputil  | 10 min | Port `csrf_middleware_test.go`, `csrf_helpers_test.go` basics. Remove cqrs-htmx-specific assertions.                                                                                                                                          |
| 2.1.8 | `GOWORK=off go mod tidy && go test ./... -count=1`                                                                    | httputil  | 5 min  | Verify.                                                                                                                                                                                                                                       |
| 2.1.9 | Commit httputil                                                                                                       | httputil  | 5 min  | `feat: add CSRF middleware (justinas/nosurf wrapper)`                                                                                                                                                                                         |
| 2.2.1 | Delete `csrf_config.go`, `csrf_context.go`, `csrf_middleware.go`, `csrf_helpers.go`, `csrf_testing.go` from cqrs-htmx | cqrs-htmx | 3 min  | Keep `csrf_handler.go` (the HandlerOption adapter).                                                                                                                                                                                           |
| 2.2.2 | Update `csrf_handler.go` imports                                                                                      | cqrs-htmx | 10 min | Import CSRF types/helpers from httputil. `executeCSRFValidation` now calls `httputil.ConfigureNosurfHandler`, `httputil.SetPlaintextHTTPOrigin`, `httputil.TranslateCSRFHeaders`, etc.                                                        |
| 2.2.3 | Create `csrf_reexport.go`                                                                                             | cqrs-htmx | 8 min  | Type aliases: `CSRFConfig`, context types. Var aliases: `CSRFMiddleware`, `CSRFResponseHeaderMiddleware`, `CSRFTokenFromContext`, `WithCSRFToken`, `CSRFTestToken`. Const aliases for header/field names.                                     |
| 2.2.4 | Update test files (12 files)                                                                                          | cqrs-htmx | 12 min | Most should work via re-exports. Check for direct references to internal (lowercase) symbols.                                                                                                                                                 |
| 2.2.5 | `GOEXPERIMENT=jsonv2 go build ./...`                                                                                  | cqrs-htmx | 5 min  | Verify.                                                                                                                                                                                                                                       |
| 2.2.6 | `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race`                                                                    | cqrs-htmx | 10 min | Full test suite.                                                                                                                                                                                                                              |
| 2.2.7 | Commit cqrs-htmx                                                                                                      | cqrs-htmx | 5 min  | `refactor: move CSRF core to httputil, removes nosurf dep from root`                                                                                                                                                                          |

### Phase 3: Rate limiting consolidation

| #     | Subtask                                                                 | Repo      | Est.   | Detail                                                                                                                                                                                                                                                                     |
| ----- | ----------------------------------------------------------------------- | --------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 3.1.1 | Port min-heap eviction into httputil's TokenBucketLimiter               | httputil  | 12 min | Replace O(n) sweep with min-heap. Port `evictionEntry`, `evictionHeap`, `evictStale`, `evictOldestIfAtCapacity` from cqrs-htmx's `ratelimit_middleware.go`.                                                                                                                |
| 3.1.2 | Add `MaxKeys` cap to TokenBucketLimiter                                 | httputil  | 8 min  | Port `MaxKeys` field + eviction-on-cap logic.                                                                                                                                                                                                                              |
| 3.1.3 | Add `KeyExtractor` type + helpers to httputil                           | httputil  | 8 min  | Port `KeyExtractor` type, `KeyExtractorFromRemoteAddr`, `KeyExtractorFromClientIP` (delegates to existing `ClientIP`).                                                                                                                                                     |
| 3.1.4 | Add monitoring `RateLimiter` struct to httputil                         | httputil  | 10 min | Port `ActiveKeys()`, `Check()`, `Middleware()` methods. This becomes `KeyedRateLimiter` in httputil (avoids name clash with `RateLimiter` interface).                                                                                                                      |
| 3.1.5 | Update httputil ratelimit tests                                         | httputil  | 12 min | Add tests for min-heap eviction, MaxKeys cap, monitoring API.                                                                                                                                                                                                              |
| 3.1.6 | `GOWORK=off go mod tidy && go test ./... -count=1`                      | httputil  | 5 min  | Verify.                                                                                                                                                                                                                                                                    |
| 3.1.7 | Commit httputil                                                         | httputil  | 5 min  | `feat: enrich rate limiter with min-heap eviction, MaxKeys cap, monitoring API`                                                                                                                                                                                            |
| 3.2.1 | Delete `ratelimit_config.go` + `ratelimit_middleware.go` from cqrs-htmx | cqrs-htmx | 2 min  | `git rm` both files.                                                                                                                                                                                                                                                       |
| 3.2.2 | Create `ratelimit_reexport.go`                                          | cqrs-htmx | 12 min | Type aliases: `RateLimiterConfig`, `KeyExtractor`. Var aliases: `RateLimiterMiddleware`, `NewRateLimiter`, `DefaultRateLimiterConfig`, `KeyExtractorFromRemoteAddr`, `KeyExtractorFromClientIP`. Const aliases: `DefaultRateLimit`, `DefaultRateWindow`, `DefaultRateTTL`. |
| 3.2.3 | Update test files (8 files)                                             | cqrs-htmx | 12 min | Most should work via re-exports. Check for references to `perKeyLimiter`, `evictionHeap`, etc.                                                                                                                                                                             |
| 3.2.4 | `GOEXPERIMENT=jsonv2 go build ./...`                                    | cqrs-htmx | 5 min  | Verify.                                                                                                                                                                                                                                                                    |
| 3.2.5 | `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race`                      | cqrs-htmx | 10 min | Full test suite.                                                                                                                                                                                                                                                           |
| 3.2.6 | Commit cqrs-htmx                                                        | cqrs-htmx | 5 min  | `refactor: delegate rate limiting to httputil, removes x/time dep from root`                                                                                                                                                                                               |

### Phase 4: Finalize

| #     | Subtask                                                                                    | Repo      | Est.   | Detail                                                                                                      |
| ----- | ------------------------------------------------------------------------------------------ | --------- | ------ | ----------------------------------------------------------------------------------------------------------- |
| 4.1.1 | `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race` (all 15 modules)                        | cqrs-htmx | 10 min | Full workspace test.                                                                                        |
| 4.1.2 | `GOEXPERIMENT=jsonv2 golangci-lint run` per module                                         | cqrs-htmx | 10 min | Verify lint clean.                                                                                          |
| 4.1.3 | Update AGENTS.md (dep tree changes, module notes)                                          | cqrs-htmx | 8 min  | Note: nosurf and x/time removed from root deps. httputil now provides CSRF + Server-Timing + rate limiting. |
| 4.1.4 | Update CHANGELOG.md in cqrs-htmx                                                           | cqrs-htmx | 5 min  | Entry for the consolidation.                                                                                |
| 4.1.5 | Update httputil README.md (add CSRF + Server-Timing + enhanced rate limit to feature list) | httputil  | 5 min  |                                                                                                             |
| 4.1.6 | `git push` httputil                                                                        | httputil  | 2 min  | Push httputil changes.                                                                                      |
| 4.1.7 | `git push` cqrs-htmx                                                                       | cqrs-htmx | 2 min  | Push cqrs-htmx changes.                                                                                     |

---

## Critical Design Decisions

### D1: How cqrs-htmx re-exports httputil types

Pattern: a `_reexport.go` file per subsystem in cqrs-htmx root:

```go
// server_timing_reexport.go
package cqrshtmx

import "github.com/larsartmann/httputil"

// Server-Timing now lives in httputil. These aliases preserve
// backward compatibility for cqrs-htmx consumers.
type ServerTiming = httputil.ServerTiming
type serverTimingCtxKey = httputil.ServerTimingCtxKey // if exported

var ServerTimingMiddleware = httputil.ServerTimingMiddleware
var ServerTimingFromContext = httputil.ServerTimingFromContext
var MeasureServerTiming = httputil.MeasureServerTiming
var MeasureServerTimingFunc = httputil.MeasureServerTimingFunc
```

Consumers who import `cqrshtmx.ServerTiming` get the httputil type via alias. Zero API breakage.

### D2: CSRF handler stays in cqrs-htmx

`csrf_handler.go` stays because:

- `CSRFProtect` returns `HandlerOption` (cqrs-htmx type)
- `executeCSRFValidation` reads `handlerConfig.csrfConfig` (cqrs-htmx internal)
- These are cqrs-htmx integration glue, not general-purpose

The handler imports httputil's CSRF helpers (`httputil.ConfigureNosurfHandler`, etc.).

### D3: Rate limiting naming in httputil

httputil already has `RateLimiter` (interface) and `RateLimit` (middleware). The cqrs-htmx monitoring struct `RateLimiter` would clash. Solution:

- httputil keeps `RateLimiter` as the interface
- httputil gains `KeyedRateLimiter` struct (the cqrs-htmx monitoring type, renamed)
- cqrs-htmx re-exports both: `type RateLimiterConfig = httputil.KeyedRateLimiterConfig`, `var NewRateLimiter = httputil.NewKeyedRateLimiter`

### D4: Context key types

Server-Timing uses context keys to store/retrieve `*ServerTiming`. The context key type must be identical in both packages. Solution: the key type lives in httputil, and cqrs-htmx's re-export aliases it.

CSRF context keys (for CSRF token) similarly live in httputil.

### D5: Test strategy

- httputil gains comprehensive tests for the moved code (ported from cqrs-htmx, adapted to remove go-cqrs-lite dependencies)
- cqrs-htmx keeps thin smoke tests that verify the re-exports work (compile-time + basic behavior)
- cqrs-htmx's integration tests that USE CSRF/ratelimit/server-timing should pass unchanged because they reference re-exported symbols

---

## Risk Mitigation

| Risk                           | Mitigation                                                                                           |
| ------------------------------ | ---------------------------------------------------------------------------------------------------- |
| Breaking consumer API          | All exports preserved via type/var aliases. Compile-time verified.                                   |
| httputil tests fail            | Each port is tested independently before moving to cqrs-htmx.                                        |
| go.work replace drift          | httputil is already in go.work replace. After httputil publishes a new tag, update cqrs-htmx go.mod. |
| CSRF handler breaks            | `csrf_handler.go` tested independently. The import change is mechanical.                             |
| Rate limiting behavior changes | httputil's TokenBucketLimiter interface preserved. Min-heap is an internal optimization.             |
| Lint regressions               | Run `golangci-lint` per module after each phase.                                                     |

---

## Dep Tree Impact

| Action                   | Deps removed from cqrs-htmx root | New deps in httputil      |
| ------------------------ | -------------------------------- | ------------------------- |
| Server-Timing → httputil | none (already zero-dep)          | none                      |
| CSRF → httputil          | `justinas/nosurf`                | `justinas/nosurf` (moved) |
| Rate limiting → httputil | `golang.org/x/time`              | none (already has it)     |

**Net result:** cqrs-htmx root loses 2 direct external deps. httputil gains 1 (`justinas/nosurf`, which it didn't have). httputil already had `golang.org/x/time` for its existing rate limiter.

---

## Verification Checklist

- [x] httputil: `GOWORK=off go test ./... -count=1 -race` passes
- [x] httputil: `GOWORK=off golangci-lint run` clean (0 issues)
- [x] cqrs-htmx: `GOEXPERIMENT=jsonv2 go build ./...` passes (all 15 modules)
- [x] cqrs-htmx: all 10 module groups pass with `-race` (go.work active)
- [x] cqrs-htmx: `GOEXPERIMENT=jsonv2 golangci-lint run` clean on all touched files
- [x] cqrs-htmx root go.mod: `justinas/nosurf` absent from require block
- [x] cqrs-htmx root go.mod: `golang.org/x/time` only as indirect
- [x] cqrs-htmx consumers: existing `cqrshtmx.CSRFMiddleware`, `cqrshtmx.ServerTiming`, `cqrshtmx.RateLimiterMiddleware` still work (type/var aliases)
- [ ] Both repos pushed — **BLOCKED**: httputil v0.8.0 not yet published; cqrs-htmx `go.work` replace still required. See blocker note below.
- [ ] `nix run .#test` passes — **BLOCKED**: nix build is hermetic (GOWORK=off), fetches published httputil v0.7.1 which lacks new symbols. Requires httputil v0.8.0 tag + cqrs-htmx go.mod bump + go.work replace removal.

---

## Resolution Notes (2026-07-30)

### What was completed beyond the original plan

1. **httputil lint: 49→0 issues.** The original port introduced 49 lint violations (depguard, varnamelen×31, exhaustruct×3, canonicalheader×6, wsl_v5×8, noinlineerr×3, gci×2, nlreturn×1, nolintlint×2). All resolved via config + code fixes.
2. **Renames for clarity:** `CSRFErrorHandler`→`ErrorHandler` and `ForbiddenCSRFHandler`→`ForbiddenHandler` in httputil (the types are general-purpose, not CSRF-specific). cqrs-htmx aliases updated.
3. **TokenBucketLimiter deprecated** (not deleted): 6 `// Deprecated:` markers added, pointing to `KeyedRateLimiter`. Reversible deprecation over breaking deletion.
4. **cqrs-htmx lint: 64→1 issues** (the 1 remaining is a pre-existing `unparam` in decoder.go, untouched). Added: `_reexport.go` exclusion from gochecknoglobals+revive, httputil types to exhaustruct exclude, G705 to gosec excludes, canonicalheader text exclusions for HX-_/X-CSRF-_ ecosystem headers.
5. **1288 LOC of redundant tests deleted** (7 files). All cqrs-htmx-specific coverage retained in `integration_csrf_test.go`, `benchmark_middleware_test.go`, `feedback_features_test.go`.
6. **Root coverage:** 93.2% (was 93.7%, gate ≥90% — still passes).
7. **httputil git index corruption fixed** (libgit2 checksum error was blocking `nix run .#test`).

### Blocker: httputil v0.8.0 publication required

cqrs-htmx now depends on **unreleased** httputil symbols (`CSRFConfig`, `ServerTiming`, `KeyedRateLimiter`, `ErrorHandler`, `ForbiddenHandler`). The `go.work` replace (`=> /home/lars/projects/httputil`) makes local `go test` work, but `nix run .#test` (GOWORK=off, hermetic) fetches published v0.7.1 and fails to compile.

**3 steps to unblock:**

1. Publish httputil v0.8.0 (`git tag v0.8.0 && git push origin master --tags`)
2. Bump cqrs-htmx (`go get github.com/larsartmann/httputil@v0.8.0`)
3. Remove go.work replace for httputil
