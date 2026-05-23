# Comprehensive Execution Plan & Progress — cqrs-htmx

**Date:** 2026-05-21 02:00 CEST
**Session:** Comprehensive plan creation, CI fix, coverage improvements, documentation update

---

## Plan Overview

All remaining TODOs from TODO_LIST.md, status reports, and coverage gaps consolidated into a prioritized 23-task execution plan. Each task estimated ≤12 min.

## Plan Table (sorted by importance/impact/effort)

| #   | Task                                                                     | Priority | Est | Impact   | Status                                  |
| --- | ------------------------------------------------------------------------ | -------- | --- | -------- | --------------------------------------- |
| T01 | Fix CI pipeline (GOWORK=off, remove GOFLAGS=-insecure, add usermgmt job) | P0       | 10m | Critical | **DONE**                                |
| T02 | Integration test: root+usermgmt E2E                                      | P0       | 12m | Critical | **DEFERRED** — needs separate Go module |
| T03 | CSRF coverage: fieldName, sameSite, csrfTokenFromContext                 | P1       | 10m | High     | **DONE**                                |
| T04 | Response coverage: sanitizeRedirectURL edge cases                        | P1       | 8m  | Medium   | **DONE**                                |
| T05 | Logging coverage: Push formatter (66.7%)                                 | P1       | 8m  | Medium   | **DEFERRED** — needs HTTP/2 Pusher mock |
| T06 | Usermgmt handler coverage (handleLogout, handleMe, handleLogin)          | P1       | 10m | High     | **DEFERRED** — next session             |
| T07 | Usermgmt authz coverage (RolesForUser, EnforceEx, Apply)                 | P1       | 10m | High     | **DEFERRED** — next session             |
| T08 | Usermgmt service coverage (Register, Login, generateToken)               | P1       | 10m | High     | **DEFERRED** — next session             |
| T09 | CSRFConfig.Secure default warning                                        | P1       | 8m  | High     | **DEFERRED** — needs API design         |
| T10 | policyWrapErr coverage (0%→100%)                                         | P2       | 5m  | Low      | **DEFERRED** — next session             |
| T11 | TypedHandler[T] on App                                                   | P1       | 10m | High     | **DEFERRED** — needs API design         |
| T12 | errorStatus dedup (root↔usermgmt)                                        | P2       | 8m  | Medium   | **DEFERRED** — next session             |
| T13 | RateLimiterConfig signedness unify                                       | P2       | 6m  | Low      | **DEFERRED** — next session             |
| T14 | Usermgmt HTTP timeout                                                    | P2       | 8m  | Medium   | **DEFERRED** — next session             |
| T15 | BrandNamer adoption for root markers                                     | P3       | 8m  | Low      | **DEFERRED** — next session             |
| T16 | Dependabot investigation                                                 | P2       | 5m  | Medium   | **BLOCKED** — gh auth expired           |
| T17 | ValidateID adoption                                                      | P3       | 8m  | Low      | **DEFERRED** — next session             |
| T18 | Rate limiter eviction O(n)→min-heap                                      | P3       | 12m | Low      | **DEFERRED** — next session             |
| T19 | CSRF fuzz tests                                                          | P3       | 10m | Low      | **DEFERRED** — next session             |
| T20 | Publisher/Subscriber ISP adoption                                        | P3       | 10m | Low      | **DEFERRED** — next session             |
| T21 | UserID type split resolution                                             | P2       | 8m  | Medium   | **DEFERRED** — owner decision needed    |
| T22 | Update TODO_LIST.md                                                      | P2       | 8m  | Medium   | **DONE**                                |
| T23 | Write this status report and commit                                      | P2       | 5m  | Medium   | **DONE**                                |

## Summary

| Category                       | Count                                           |
| ------------------------------ | ----------------------------------------------- |
| **DONE**                       | 5 (T01, T03, T04, T22, T23)                     |
| **DEFERRED** (next session)    | 14 (T05-T15, T17-T20)                           |
| **BLOCKED** (external)         | 1 (T16 — gh auth)                               |
| **DEFERRED** (design decision) | 2 (T02 — separate module, T21 — owner decision) |
| **Total**                      | 22 tasks                                        |

## What Was Done This Session

1. **CI pipeline fixed** — Removed broken `GOFLAGS=-insecure` (removed in Go 1.26), added `GOWORK=off` (parent go.work doesn't include this module), added separate usermgmt build+test+coverage jobs
2. **CSRF coverage improved** — Added 4 tests for default field name, default SameSite (zero value), context fallback token, and empty context
3. **Redirect URL coverage improved** — Added 3 test cases for `data:` URLs, scheme-relative URLs (`//evil.com`), and unparseable URLs (`://\x00bad`)
4. **TODO_LIST.md updated** — Added 15 new open items from this planning session, marked 6 items done from today's work
5. **Comprehensive 22-task plan created** — All remaining work prioritized, estimated, and tracked

## Metrics

| Metric            | Value |
| ----------------- | ----- |
| Root coverage     | 95.9% |
| Usermgmt coverage | 91.7% |
| Lint issues (CLI) | 0     |
| Test suites       | PASS  |
| Race detector     | PASS  |

## Files Changed This Session

| File                       | Change                                                      |
| -------------------------- | ----------------------------------------------------------- |
| `.github/workflows/ci.yml` | Fixed broken GOFLAGS, added GOWORK=off, added usermgmt jobs |
| `csrf_test.go`             | Added 4 CSRF helper coverage tests                          |
| `coverage_test.go`         | Added 3 redirect URL edge case tests                        |
| `TODO_LIST.md`             | Updated with 15 new open items + 6 completed items          |

---

## Appendix: Status as of 2026-05-23

Re-evaluated against current `master` branch, `TODO_LIST.md`, git history since plan, and CI configuration.

### Task-by-Task Reassessment

| #   | Task                                                                     | Status at Plan | Status Now     | Evidence                                                                                                                                                                                                                                                                         |
| --- | ------------------------------------------------------------------------ | -------------- | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| T01 | Fix CI pipeline (GOWORK=off, remove GOFLAGS=-insecure, add usermgmt job) | **DONE**       | ✅ **DONE**    | `.github/workflows/ci.yml` has `GOWORK=off`, `GONOSUMCHECK`, no `GOFLAGS=-insecure`. Jobs for root, usermgmt, integration_test, datastar-demo + coverage gates.                                                                                                                  |
| T02 | Integration test: root+usermgmt E2E                                      | DEFERRED       | ✅ **DONE**    | `integration_test/bridge_test.go` with `TestUsermgmtBridge_AsEnforcer` + `TestUsermgmtBridge_UserIDFromRequest` (commit `7a10c9d`). Separate Go module.                                                                                                                          |
| T03 | CSRF coverage: fieldName, sameSite, csrfTokenFromContext                 | **DONE**       | ✅ **DONE**    | 4 tests in `csrf_test.go`.                                                                                                                                                                                                                                                                                                      |
| T04 | Response coverage: sanitizeRedirectURL edge cases                        | **DONE**       | ✅ **DONE**    | 3 tests in `coverage_test.go`.                                                                                                                                                                                                                                                                                                   |
| T05 | Logging coverage: Push formatter (66.7%)                                 | DEFERRED       | ✅ **DONE**    | `mockPusher`, `pusherRecorder`, `hijackRecorder` in `testing_test.go`. Push + Hijack delegation tests in `logging_test.go` (commit `7a10c9d`).                                                                                                                                   |
| T06 | Usermgmt handler coverage (handleLogout, handleMe, handleLogin)          | DEFERRED       | ✅ **DONE**    | `TestHandlers_Login_Success`, `TestHandlers_Logout_Success`, `TestHandlers_Me_*`, `TestHandlers_Register_Success` in `usermgmt/coverage_test.go` (commit `7a10c9d`).                                                                                                            |
| T07 | Usermgmt authz coverage (RolesForUser, EnforceEx, Apply)                 | DEFERRED       | ✅ **DONE**    | `TestAuthz_EnforceEx_Error`, `TestAuthz_Apply_RemovePolicies` + prior session tests for RolesForUser (commit `7a10c9d`).                                                                                                                                                         |
| T08 | Usermgmt service coverage (Register, Login, generateToken)               | DEFERRED       | ✅ **DONE**    | `TestService_Login_AccountLocked`, `TestService_Login_UserNotFound`, `TestService_Register_DuplicateUserID` + SessionMiddleware tests (commit `7a10c9d`).                                                                                                                       |
| T09 | CSRFConfig.Secure default warning                                        | DEFERRED       | ✅ **DONE**    | `CSRFMiddleware` emits `slog.Warn` when `Secure=false` (commit `7a10c9d`, `csrf.go`).                                                                                                                                                                                            |
| T10 | policyWrapErr coverage (0%→100%)                                         | DEFERRED       | ✅ **DONE**    | `TestPolicyWrapErr` in `usermgmt/coverage_test.go` (commit `7a10c9d`).                                                                                                                                                                                                           |
| T11 | TypedHandler[T] on App                                                   | DEFERRED       | ❌ **OPEN**    | Requires top-level generic function (Go methods can't add type params on non-generic receiver). No `TypedHandler` references found in codebase. API design decision still pending.                                                                                              |
| T12 | errorStatus dedup (root↔usermgmt)                                        | DEFERRED       | ✅ **CLOSED**  | Closed as "NOT RECOMMENDED" — would couple usermgmt to go-cqrs-lite's event classification system. Modules serve different purposes.                                                                                                                                             |
| T13 | RateLimiterConfig signedness unify                                       | DEFERRED       | ✅ **DONE**    | `perKeyLimiter.burst` and `perKeyLimiter.maxKeys` changed from `int` to `uint` (commit `7a10c9d`, `ratelimit.go`).                                                                                                                                                              |
| T14 | Usermgmt HTTP timeout                                                    | DEFERRED       | ✅ **DONE**    | `HandlerConfig.Timeout` with `context.WithTimeout` in `handleAuthEndpoint` + `handleLogout` (commit `7a10c9d`, `usermgmt/http.go`).                                                                                                                                             |
| T15 | BrandNamer adoption for root markers                                     | DEFERRED       | ❌ **BLOCKED** | Upstream `go-cqrs-lite/core/pkg/id` marker types (`userMarker`, `correlationMarker`) are unexported. No `BrandNamer` in codebase.                                                                                                                                                |
| T16 | Dependabot investigation                                                 | BLOCKED        | ❌ **BLOCKED** | No `.github/dependabot.yml` exists. Original blocker (gh auth expired) likely still applies.                                                                                                                                                                                     |
| T17 | ValidateID adoption                                                      | DEFERRED       | ✅ **CLOSED**  | Closed as "NOT NEEDED" — `ParseUserID` already validates ULID format via `id.ParseUserID`. `ValidateID` only checks non-zero, already done via `IsZero()`.                                                                                                                      |
| T18 | Rate limiter eviction O(n)→min-heap                                      | DEFERRED       | ✅ **DONE**    | `container/heap` min-heap via `evictionHeap` type. O(log n) eviction (commit `7a10c9d`, `ratelimit.go`).                                                                                                                                                                         |
| T19 | CSRF fuzz tests                                                          | DEFERRED       | ✅ **DONE**    | `FuzzCSRFConfigValidation` in `fuzz_test.go` (commit `7a10c9d`).                                                                                                                                                                                                                 |
| T20 | Publisher/Subscriber ISP adoption                                        | DEFERRED       | ✅ **CLOSED**  | Closed as "NOT APPLICABLE" — cqrs-htmx dispatches commands/queries, doesn't publish events. Publisher/Subscriber interfaces are for event sourcing infrastructure.                                                                                                              |
| T21 | UserID type split resolution                                             | DEFERRED       | ❌ **OPEN**    | `usermgmt.UserID` (string-backed) vs `cqrshtmx.UserID` (ULID-backed) remain incompatible. Owner decision needed on whether usermgmt is standalone or always paired with cqrs-htmx.                                                                                              |
| T22 | Update TODO_LIST.md                                                      | **DONE**       | ✅ **DONE**    | Per plan.                                                                                                                                                                                                                                                                                                                         |
| T23 | Write this status report and commit                                      | **DONE**       | ✅ **DONE**    | Per plan.                                                                                                                                                                                                                                                                                                                         |

### Summary

| Category                   | Count | Tasks                                                                |
| -------------------------- | ----- | -------------------------------------------------------------------- |
| **DONE** (was done)        | 5     | T01, T03, T04, T22, T23                                              |
| **DONE** (completed since) | 11    | T02, T05, T06, T07, T08, T09, T10, T13, T14, T18, T19               |
| **CLOSED** (won't-do/N/A)  | 3     | T12, T17, T20                                                        |
| **STILL OPEN**             | 2     | T11 (TypedHandler API design), T21 (UserID type split decision)      |
| **STILL BLOCKED**          | 2     | T15 (BrandNamer — upstream unexported), T16 (Dependabot — gh auth)   |
| **Total resolved**         | 19/22 | 86%                                                                  |

### Current Metrics (from latest status report 2026-05-22)

| Metric            | Value   |
| ----------------- | ------- |
| Root coverage     | 96.6%   |
| Usermgmt coverage | 88.6%   |
| Lint issues       | 0       |
| Race detector     | PASS    |

### Remaining Work (Priority Order)

1. **T21** — Owner decision: resolve `usermgmt.UserID` vs `cqrshtmx.UserID` type split
2. **T11** — Design `TypedHandler[T]` top-level generic function API
3. **T15** — Blocked on upstream `go-cqrs-lite` exporting marker types or providing `BrandNamer`
4. **T16** — Blocked on `gh auth` token refresh; then add `.github/dependabot.yml`
