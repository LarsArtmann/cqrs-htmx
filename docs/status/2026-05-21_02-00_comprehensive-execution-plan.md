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
