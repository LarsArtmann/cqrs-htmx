# Status Report: 2026-05-20 11:36 — Deep Audit & Execution Plan

**Date:** 2026-05-20 11:36 | **Type:** Deep Audit + Prioritized Plan | **Session:** Post-mega-session cleanup

---

## Executive Summary

A comprehensive deep audit of every production `.go` file (root + usermgmt) revealed **31 `fmt.Errorf` calls** that should use `cockroachdb/errors`, **~70 magic strings** needing constants, **0% coverage on `RequestLoggingSlog`**, **70+ missing godoc** in usermgmt, and **`UserIDFromRequest`** returning `string` instead of `UserID`. All tests pass (120 total), lint is clean (0 issues), coverage is 95.2% root / 91.1% usermgmt.

---

## Health Dashboard

| Metric            | Root Module         | usermgmt Submodule |
| ----------------- | ------------------- | ------------------ |
| Coverage          | **95.2%**           | **91.1%**          |
| Test specs        | ~285 (Ginkgo + std) | ~85                |
| Lint issues       | **0**               | **0**              |
| Race detector     | Clean               | Clean              |
| Prod Go files     | 17 (4,066 lines)    | 9 (1,445 lines)    |
| Test Go files     | 20 (7,413 lines)    | 7 (1,393 lines)    |
| Build             | Clean               | Clean              |
| Integration tests | 2 (passing)         | —                  |
| Fuzz tests        | 3                   | —                  |
| Benchmarks        | 8                   | —                  |
| Examples          | 9                   | —                  |

---

## A) FULLY DONE

### All 30 features from FEATURES.md are FULLY_FUNCTIONAL:

1. App Builder (validated, per-App LoginRedirect, timeout)
2. Command Dispatch (HTMX-aware, decode/auth/dispatch pipeline)
3. Query Dispatch (HTMX-aware, render result)
4. JSON Decoding (`DecodeJSON[T]`, `DecodeJSONQuery[T]`)
5. Form Decoding (`DecodeForm[T]`, `DecodeFormQuery[T]`)
6. Casbin Authorization (`Authorize`, `RequireAuth`, `Enforce`)
7. Casbin Middleware (`AuthorizeMiddleware`)
8. User Identity Propagation (branded `UserID`, context → event metadata)
9. HTMX Request Context (`HTMXMiddleware`, `HTMXRequest` struct)
10. HTMX Accessors (standalone, fallback to headers)
11. HTMX Response Builder (fluent `Response`, all headers)
12. Notifications (`NotifySuccess/Error/Warning/Info`, `NotifyWithEvent`)
13. Templ Integration (`RenderTempl`, `RenderTemplResult[T]`)
14. Error Classification (`sync.Once` sentinel registration)
15. Default Error Handler (HTMX-aware, `text/plain`)
16. Middleware Chain (`Chain` composes left-to-right)
17. Handler Options (Redirect, Trigger, PushURL, etc.)
18. Swap Strategies (8 typed constants)
19. Header Constants (all HTMX headers are constants)
20. JSON Error Handler (`JSONErrorHandler`)
21. Lifecycle Hooks (`BeforeDispatchHook`, `AfterDispatchHook`)
22. Correlation ID (branded `id.CorrelationID`, auto-extracted)
23. Request Validation (`ValidateCommand`, `ValidateQuery`)
24. Timeout Propagation (`Config.Timeout`, dispatch-only)
25. Request Logging (`RequestLogging`, `JSONLogFormatter`)
26. Rate Limiting (`RateLimiterMiddleware`, token-bucket)
27. Security Headers (`SecurityHeadersMiddleware`, configurable)
28. Request ID (branded `id.RequestID`, ULID-backed)
29. CSRF Protection (`CSRFMiddleware`, `CSRFProtect`, helpers)
30. Branded UserID (`usermgmt.UserID` via `go-branded-id`)

### Mega Quality Session (45+ tasks — all complete):

- Bug fix: `sanitizeRedirectURL("/")` blocking root redirect
- Type safety: `RateLimiterConfig.Limit/Burst/MaxKeys` → `uint`
- Type safety: `LockoutConfig.MaxAttempts` → `uint`
- Type safety: `type Role string` with constants in usermgmt
- Type safety: `User.Roles` → `[]Role`, all authz methods typed
- Naming: 6 renames (RotateCSRFToken→InvalidateCSRFCookie, etc.)
- Architecture: CSRF split, statusRecorder dedup, http.Pusher, hasNoExplicitBody fix
- Tests: 15+ new tests, 3 fuzz tests, 6 benchmarks
- Docs: 2 ADRs, 33 old reports archived
- Integration: 2 tests bridging root ↔ usermgmt

---

## B) PARTIALLY DONE

| Item                          | Current                    | Target                          | Gap                              |
| ----------------------------- | -------------------------- | ------------------------------- | -------------------------------- |
| Error wrapping consistency    | `fmt.Errorf` in 31 sites   | `cockroachdb/errors` everywhere | 31 calls to fix                  |
| Magic string constants        | ~20 constants defined      | ~90 total needed                | ~70 missing                      |
| usermgmt godoc                | 2 functions documented     | ~72 exported symbols            | ~70 missing                      |
| `RequestLoggingSlog` coverage | 0%                         | 80%+                            | Entire function untested         |
| usermgmt `context.Context`    | Accepted but ignored (`_`) | Propagated to stores            | Store interfaces need ctx params |

---

## C) NOT STARTED

| Item                                                       | Effort | Impact                                  |
| ---------------------------------------------------------- | ------ | --------------------------------------- |
| `UserIDFromRequest` → return `UserID` instead of `string`  | 30m    | HIGH — type safety at module boundary   |
| `GroupPolicy.Subject` → `UserID` type                      | 1h     | MED — consistency with rest of usermgmt |
| `JSONErrorHandlerWithRedirect` test coverage               | 15m    | MED — 0% coverage on exported function  |
| usermgmt store/authz interfaces → accept `context.Context` | 2h     | MED — enables cancellation, tracing     |
| Email/SessionToken branded types                           | 1h     | LOW — nice-to-have type safety          |
| Persistent session store (cookie/Redis)                    | 2h     | HIGH — production readiness             |
| Password reset flow                                        | 2h     | MED                                     |
| Email verification flow                                    | 2h     | MED                                     |
| SSE/EventStream helper                                     | 3h     | HIGH                                    |
| OAuth2/OIDC hooks                                          | 3h     | HIGH                                    |
| Rate limiter min-heap eviction                             | 2h     | MED                                     |
| Multi-tenancy via Casbin domains                           | 2h     | MED                                     |

---

## D) TOTALLY FUCKED UP

| Issue                                           | Severity   | Details                                                                                                                                 |
| ----------------------------------------------- | ---------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `RequestLoggingSlog` 0% coverage                | **HIGH**   | 68-line exported function with zero tests. Code could be broken and we'd never know.                                                    |
| 31 `fmt.Errorf` instead of `cockroachdb/errors` | **MEDIUM** | Loses stack traces on every error wrapping in authz, handler, decoder, csrf, options. This is the #1 consistency violation.             |
| ~70 magic strings in security headers           | **LOW**    | Header names like `"X-Content-Type-Options"` are raw strings. Works but fragile.                                                        |
| `UserIDFromRequest` returns `string`            | **MEDIUM** | The bridge function between usermgmt and root returns untyped `string` when it has `UserID` available. Defeats the purpose of branding. |
| usermgmt `context.Context` silently ignored     | **LOW**    | All service methods accept ctx but discard it (`_`). Stores can't be cancelled or traced.                                               |

---

## E) WHAT WE SHOULD IMPROVE

### HIGH PRIORITY

1. **Fix all `fmt.Errorf` → `cockroachdb/errors`** — 31 sites across 8 files. Stack traces are lost every time. This is a library — consumers deserve proper error chains.
2. **Test `RequestLoggingSlog`** — 0% coverage on a 68-line public function is unacceptable for a library.
3. **Fix `UserIDFromRequest` return type** — One-line fix, huge type safety improvement.

### MEDIUM PRIORITY

4. **Extract security header constants** — 9 header names + default values should be typed.
5. **Add usermgmt godoc** — ~70 exported symbols undocumented. Library consumers need docs.
6. **Test `JSONErrorHandlerWithRedirect`** — Exported function, 0% coverage.
7. **`GroupPolicy.Subject` → `UserID`** — Currently `string`, inconsistent with the rest of usermgmt.

### LOW PRIORITY

8. **Extract remaining magic strings** — log keys, `"Bearer "`, `"session_token"` default.
9. **Add `context.Context` to store/authz interfaces** — Future-proofing for persistent stores.
10. **Branded `Email` and `SessionToken` types** — Nice-to-have.

---

## F) Top 25 Things to Get Done Next

### P0 — Immediate (This Session)

| #   | Item                                                              | Effort | Impact |
| --- | ----------------------------------------------------------------- | ------ | ------ |
| 1   | Fix `fmt.Errorf` → `cockroachdb/errors` in root module (20 calls) | 30m    | HIGH   |
| 2   | Fix `fmt.Errorf` → `cockroachdb/errors` in usermgmt (11 calls)    | 15m    | HIGH   |
| 3   | Add `RequestLoggingSlog` tests (0% → 80%+)                        | 30m    | HIGH   |
| 4   | Fix `UserIDFromRequest` to return `UserID`                        | 15m    | HIGH   |
| 5   | Extract security header constants in `security.go`                | 15m    | MED    |

### P1 — This Week

| #   | Item                                                         | Effort | Impact |
| --- | ------------------------------------------------------------ | ------ | ------ |
| 6   | Add usermgmt godoc (~70 exported symbols)                    | 1h     | MED    |
| 7   | Test `JSONErrorHandlerWithRedirect`                          | 15m    | MED    |
| 8   | Extract remaining magic strings (logging, ratelimit, notify) | 30m    | LOW    |
| 9   | `GroupPolicy.Subject` → `UserID` type                        | 1h     | MED    |
| 10  | Raise usermgmt coverage to 92%+                              | 1h     | HIGH   |

### P2 — Next Sprint

| #   | Item                                            | Effort | Impact |
| --- | ----------------------------------------------- | ------ | ------ |
| 11  | Add persistent session store interface          | 2h     | HIGH   |
| 12  | Add `context.Context` to store/authz interfaces | 2h     | MED    |
| 13  | Password reset flow in usermgmt                 | 2h     | MED    |
| 14  | SSE/EventStream helper for HTMX real-time       | 3h     | HIGH   |
| 15  | OAuth2/OIDC integration hooks                   | 3h     | HIGH   |

### P3 — Backlog

| #   | Item                                     | Effort | Impact |
| --- | ---------------------------------------- | ------ | ------ |
| 16  | Email verification flow                  | 2h     | MED    |
| 17  | Rate limiter min-heap eviction           | 2h     | MED    |
| 18  | Multi-tenancy via Casbin domains         | 2h     | MED    |
| 19  | Branded `Email` and `SessionToken` types | 1h     | LOW    |
| 20  | Visual architecture diagram (D2)         | 1h     | MED    |

### P4 — Nice to Have

| #   | Item                                   | Effort | Impact |
| --- | -------------------------------------- | ------ | ------ |
| 21  | Performance profiling and optimization | 2h     | LOW    |
| 22  | Expand benchmark suite                 | 1h     | MED    |
| 23  | OpenTelemetry tracing integration      | 3h     | HIGH   |
| 24  | Example application (full usage demo)  | 2h     | HIGH   |
| 25  | Migrate to flake.nix build system      | 2h     | MED    |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should `fmt.Errorf("%w: ...")` be replaced with `errors.Wrapf` / `errors.Newf` from `cockroachdb/errors` throughout?**

The codebase already imports `cockroachdb/errors` for sentinel creation. But 31 sites use `fmt.Errorf` for wrapping. The question is:

- **`fmt.Errorf("%w: ...", sentinel, err)`** — Standard Go wrapping. Works with `errors.Is()`/`errors.As()`. No stack trace.
- **`errors.Wrapf(err, "context")`** — CockroachDB style. Adds stack trace. Compatible with `errors.Is()`/`errors.As()`.
- **`errors.WithMessagef(err, "context")`** — Adds message but no stack trace. Similar to `fmt.Errorf`.

**My recommendation:** Use `errors.Newf` for new errors and `errors.Wrapf` for wrapping existing errors. This is what cockroachdb/errors is designed for. But changing 31 sites is a significant diff — I'll proceed with this assumption.

---

## Dependency Status

| Dependency         | Version | Status              |
| ------------------ | ------- | ------------------- |
| go-cqrs-lite/core  | v1.2.0  | Current             |
| casbin/casbin/v3   | v3.10.0 | Current             |
| gorilla/csrf       | v1.7.3  | Fixed               |
| cockroachdb/errors | v1.13.0 | Current (underused) |
| golang.org/x/time  | v0.15.0 | Current             |
| go-branded-id      | v0.1.0  | Current (usermgmt)  |

---

## Audit Findings Summary

### Root Module — `fmt.Errorf` Calls (20 total)

| File       | Calls | Lines                    |
| ---------- | ----- | ------------------------ |
| decoder.go | 6     | 21, 41, 78, 82, 100, 104 |
| authz.go   | 4     | 44, 56, 71, 84           |
| handler.go | 3     | 77, 152, 162             |
| csrf.go    | 3     | 22, 133, 140             |
| options.go | 2     | 156, 222                 |
| logging.go | 1     | 143                      |
| context.go | 1     | 24                       |

### usermgmt — `fmt.Errorf` Calls (11 total)

| File       | Calls | Lines                                  |
| ---------- | ----- | -------------------------------------- |
| authz.go   | 8     | 106, 112, 116, 123, 172, 175, 197, 207 |
| service.go | 2     | 50, 89 (rest already use cockroachdb)  |
| store.go   | 2     | 81, 122                                |
| user.go    | 2     | 50, 103                                |

### Zero Coverage Functions

| Function                       | File       | Lines                       |
| ------------------------------ | ---------- | --------------------------- |
| `RequestLoggingSlog`           | logging.go | 141-208 (entirely untested) |
| `JSONErrorHandlerWithRedirect` | errors.go  | 172 (untested)              |

---

_Arte in Aeternum_
