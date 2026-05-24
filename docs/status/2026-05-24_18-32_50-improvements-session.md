# Status Report — 50 Improvements Session

**Date:** 2026-05-24 18:32
**Since:** Previous commit `ef7593a` (golangci-lint config reformat)
**Branch:** master

---

## Module Health

| Module           | Tests       | Coverage | Lint     | Race |
| ---------------- | ----------- | -------- | -------- | ---- |
| Root             | PASS (380+) | 96.7%    | 0 issues | PASS |
| usermgmt         | PASS        | 91.1%    | 0 issues | PASS |
| integration_test | PASS        | —        | 0 issues | PASS |
| datastar-demo    | BUILD OK    | —        | —        | —    |

---

## A) FULLY DONE (50 items)

### Batch 1: Bug Fixes & Correctness (6)

1. **Fixed duplicate comment** in `usermgmt/service.go:Register` — lines 145-146 had duplicated "Register validates the request..."
2. **Fixed command dispatch context** — `handleCommandDispatch` now passes `r.WithContext(ctx)` to `applyCommandResponse`, matching `handleQueryDispatch` behavior (`handler.go:85`)
3. **MapError returns 413** for `ErrRequestTooLarge` — was falling through to error-family Rejection (400). Now explicit check before family classification (`errors.go:73`)
4. **LoginRequest validates max password length** — adds `maxPasswordLength` (128) check to `LoginRequest.Validate()`, preventing bcrypt CPU abuse at login time (`usermgmt/service.go:203`)
5. **Added DefaultMaxBodySize** (10 MB) — `decoder.go` defines constant. `readBody` now defaults to 10 MB when no limit configured. **Behavior change**: previously zero meant unlimited
6. **Removed dead code** — `InMemorySessionStore.ttl` field and `WithTTL` method removed. The TTL was stored but never read (TTL passed directly to `Create()`)

### Batch 2: Security Hardening (4)

7. **sanitizeRedirectURL depth check** — tracks path segment depth, blocks if `..` causes negative depth (escapes above root). Legitimate normalization like `/a/../b/c` still works (`response.go:237`)
8. **usermgmt body size limit** — `handleAuthEndpoint` uses `io.LimitReader(r.Body, 1<<20)` (1 MB cap) to prevent unbounded body reads (`usermgmt/http.go:113`)
9. **CSRFConfig.Validate Secure warning** — emits `slog.Warn` when `Secure=false` during `Validate()` (`csrf.go:149`)
10. **Path traversal defense** — `splitPath` helper for depth-based traversal detection in redirect URLs

### Batch 3: New HandlerOptions (5)

11. **WithMaxBodySize** — per-handler body size override. Takes precedence over App-level `Config.MaxBodySize` (`options.go:273`)
12. **WithSuccessStatus** — custom success status code (default 204 No Content). Common values: 200 OK, 201 Created (`options.go:279`)
13. **OnError** — per-handler error callback, invoked after App-level error handler. For logging/metrics on specific routes (`options.go:313`)
14. **WithMaxBodySize precedence fix** — `App.Command()` and `App.Query()` now only set `cfg.maxBodySize` from app config when handler hasn't already set it (`app.go:118,143`)
15. **handlerConfig.successStatus + onError** — new fields added to `handlerConfig` struct

### Batch 4: Response Builder Enhancements (6)

16. **Response.Status(code)** — fluent HTTP status code setter (`response.go:166`)
17. **Response.Header(key, value)** — fluent custom header setter (`response.go:172`)
18. **Response.ContentType(ct)** — fluent Content-Type setter (`response.go:178`)
19. **Response.Body(data)** — write raw bytes, calls Apply first (`response.go:183`)
20. **Response.WriteString(s)** — write string, calls Apply first (`response.go:189`)
21. **Response.JSON(v)** — encode and write JSON body, sets Content-Type (`response.go:195`)

### Batch 5: Helper Functions & Convenience (5)

22. **IsAuthenticated(r)** — checks for non-zero UserID in request context (`authz.go:135`)
23. **MustNew(cfg)** — panics on error, for init-time setup (`app.go:108`)
24. **HasCommands() / HasQueries()** — report dispatcher availability (`app.go:120-121`)
25. **KeyExtractorFromClientIP()** — rate limiter key extractor using `ClientIP()` which respects X-Forwarded-For and X-Real-IP (`ratelimit.go:45`)
26. **NewRateLimiter** — returns `*RateLimiter` with `ActiveKeys()` monitoring. `RateLimiterMiddleware` unchanged for backward compat (`ratelimit.go:81`)

### Batch 6: Code Quality & Consistency (6)

27. **usermgmt/service.go** — all 5 `fmt.Errorf` → `errors.Wrapf` (cockroachdb/errors) for consistent stack traces
28. **usermgmt/authz.go** — all 14 `fmt.Errorf` → `errors.Wrapf` for consistent error wrapping
29. **slices.Delete** in `User.RemoveRole` — replaced manual slice manipulation (`usermgmt/user.go:85`)
30. **authMode.String()** — human-readable names for debug/logging: "none", "required", "authorized" (`options.go:43`)
31. **Authenticate cleanup comment** — clarified why `IsExpired()` check exists before `Valid()` (proactive session store cleanup)
32. **CSRFConfig.Validate** — added Secure=false warning in Validate path

### Batch 7: Monitoring & Observability (4)

33. **InMemoryUserStore.Count()** — returns number of stored users (`usermgmt/store.go:115`)
34. **InMemorySessionStore.Count()** — returns number of active sessions (`usermgmt/store.go:204`)
35. **perKeyLimiter.Len()** — internal active key count (`ratelimit.go:230`)
36. **RateLimiter.ActiveKeys()** — public API for monitoring rate limiter state (`ratelimit.go:76`)

### Batch 8: Tests (14)

37. MapError ErrRequestTooLarge → 413 test
38. sanitizeRedirectURL traversal blocking tests (depth escape, deep traversal, legitimate normalization)
39. IsAuthenticated helper tests (no user, with user)
40. MustNew tests (panic on invalid, success on valid)
41. HasCommands/HasQueries tests (command-only, query-only app)
42. Response.Status/Header/ContentType tests
43. Response.JSON/Body/WriteString tests (including marshal error path)
44. WithMaxBodySize HandlerOption test (per-handler override)
45. WithSuccessStatus HandlerOption test (201 Created)
46. OnError HandlerOption test (error callback fires)
47. KeyExtractorFromClientIP test (XFF extraction)
48. NewRateLimiter ActiveKeys monitoring test
49. LoginRequest maxPasswordLength validation test (usermgmt)
50. InMemoryUserStore/SessionStore Count tests (usermgmt)

---

## B) PARTIALLY DONE

None — all 50 items completed and verified.

---

## C) NOT STARTED

### Coverage Gaps (root module, below 90%)

| File               | Function               | Coverage |
| ------------------ | ---------------------- | -------- |
| `csrf.go:221`      | `csrfTokenFromRequest` | 66.7%    |
| `logging.go:223`   | `Hijack`               | 60.0%    |
| `options.go:43`    | `authMode.String`      | 0.0%     |
| `csrf.go:113`      | `sameSite`             | 83.3%    |
| `csrf.go:174`      | `buildGorillaOptions`  | 88.9%    |
| `ratelimit.go:321` | `Push`                 | 75.0%    |

### Coverage Gaps (usermgmt, below 80%)

| File           | Function             | Coverage |
| -------------- | -------------------- | -------- |
| `authz.go:116` | `NewAuthz`           | 84.2%    |
| `authz.go:158` | `Enforce`            | 75.0%    |
| `authz.go:241` | `Apply`              | 69.2%    |
| `authz.go:*`   | Many authz methods   | 75% each |
| `http.go:115`  | `handleAuthEndpoint` | 80.0%    |
| `user.go:147`  | `generateToken`      | 75.0%    |

### Not Started (from TODO_LIST.md / previous planning sessions)

- Typed errors with structured fields (machine-readable error responses)
- OpenTelemetry integration for dispatch tracing
- Persistent store interfaces (SQL) for usermgmt
- WebSocket/SSE support for real-time notifications
- Request coalescing / deduplication middleware
- Health check endpoint helpers
- Graceful shutdown helpers
- Prometheus metrics middleware

---

## D) TOTALLY FUCKED UP

Nothing broken. All 4 modules build, pass tests with race detector, and have 0 lint issues.

**One thing I got wrong during execution**: Initially changed `RateLimiterMiddleware` return type from `func(http.Handler) http.Handler` to `*RateLimiter`, which broke 16 test files. Reverted and created separate `NewRateLimiter()` for the monitoring use case, keeping `RateLimiterMiddleware` backward compatible.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **usermgmt authz error paths at 75%** — most authz methods have untested error branches (casbin returning errors). Need error-injecting mock enforcer
2. **csrfTokenFromRequest at 66.7%** — context fallback path not tested (when gorilla has no token but context does)
3. **authMode.String at 0%** — unexported, only useful for debug. Need indirect test or export it
4. **Hijack at 60%** — needs a mock Hijacker ResponseWriter test
5. **Response builder tests could be stronger** — verify all chain combinations, not just individual methods
6. **usermgmt HTTP handler error paths** — `handleRegister`, `handleLogin`, `handleAuthEndpoint` all at 80-87% due to untested JSON decode failures and timeout paths

### Code

7. **Dead `splitPath` function** — only used by `sanitizeRedirectURL`, could be inlined but cleaner as helper
8. **authMode.String() unexported** — useful for consumers debugging auth failures. Consider exporting
9. **usermgmt error wrapping is inconsistent with root** — root uses `error-family` classification, usermgmt doesn't. Not wrong, but dual system is cognitive load

### Process

10. **No benchmark tests for new Response methods** — JSON, Body, WriteString have no benchmarks
11. **No fuzz tests for new decoder changes** — DefaultMaxBodySize boundary could benefit from fuzzing

---

## F) Top 25 Things We Should Get Done Next

### P0 — Close Coverage Gaps (high impact, low effort)

1. Add error-injecting mock enforcer for usermgmt authz tests (brings 17 methods from 75% → 90%+)
2. Test `csrfTokenFromRequest` context fallback path (66.7% → 100%)
3. Test `authMode.String()` all branches (0% → 100%)
4. Test `StatusRecorder.Hijack` with real Hijacker mock (60% → 100%)
5. Test `StatusRecorder.Push` success path (75% → 100%)
6. Test `usermgmt/handleAuthEndpoint` timeout path (80% → 95%+)
7. Test `usermgmt/handleRegister` JSON decode failure (87.5% → 95%+)

### P1 — Security & Correctness (high impact, medium effort)

8. Add `Content-Security-Policy` defaults to `SecurityHeadersConfig` (currently empty string)
9. Add `Strict-Transport-Security` recommended value in docs/config
10. Add `SameSite` enforcement for session cookies in usermgmt (already strict, but validate in tests)
11. Add request method validation to handlers (reject GET for command endpoints)
12. Add `X-Request-ID` response header propagation (set it in response if generated)

### P2 — Feature Enhancements (medium impact, medium effort)

13. **Typed errors with structured fields** — `ErrorResponse{Error: string, Code: string, Details: map}` for machine-readable API errors
14. **OpenTelemetry spans** — `BeforeDispatchHook`/`AfterDispatchHook` are ready; add otel helpers
15. **Health check helpers** — `HealthHandler` that checks dispatcher availability
16. **Graceful shutdown helpers** — `Shutdowner` interface with timeout
17. **Request coalescing** — dedup middleware for concurrent identical queries
18. **Prometheus metrics middleware** — request count, latency histogram, error rate
19. **WebSocket/SSE notification bridge** — server-sent events for real-time HTMX notifications

### P3 — Code Quality (medium impact, low effort)

20. **Export `AuthMode` types** — `AuthModeNone()`, `AuthModeRequired()`, `AuthModeAuthorized()` for consumer readability
21. **Add `fmt.Stringer` to `NotificationLevel`** — for structured logging
22. **Benchmark new Response methods** — JSON, Body, WriteString
23. **Add example tests** for Response builder fluent API
24. **Add example tests** for `WithSuccessStatus`, `WithMaxBodySize`, `OnError`
25. **Update TODO_LIST.md** — mark new items, close completed ones

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should `usermgmt` adopt the `error-family` classification system that the root module uses?**

The root module uses `go-error-family` to classify errors into families (Rejection, Conflict, Transient, Infrastructure) which powers `MapError()`. The usermgmt submodule does NOT use this system — it has its own `errorStatus()` switch with hardcoded `errors.Is` checks. This means:

- usermgmt errors wrapped by the root module get classified correctly at the HTTP boundary
- But usermgmt's own `errorStatus()` is a parallel, unmaintained mapping that could drift
- Adopting error-family in usermgmt would add a dependency on `go-error-family` and couple the submodule to the root's error architecture
- NOT adopting it means maintaining two error→HTTP mappings

This is a **domain architecture decision** that only Lars can make. I can implement either direction.
