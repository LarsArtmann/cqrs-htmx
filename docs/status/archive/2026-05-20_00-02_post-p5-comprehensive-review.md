# Comprehensive Status Report — Post-P5 Deduplication

**Date:** 2026-05-20_00-02
**Scope:** Full codebase review — root module + usermgmt submodule
**Trigger:** Post-execution reflection, architecture review, type-safety audit

---

## Executive Summary

The P5 execution plan (25 tasks) is **100% complete**. All deduplication generics are in place, tests pass, lint is clean. The codebase is in good shape — grade **A-** with clear paths to **A+**.

This report identifies what's left: type safety gaps (uint vs int, Role type), a 386-line file that needs splitting, 5 untested public functions in usermgmt, and a subtle bug in `sanitizeRedirectURL` that blocks redirecting to `/`.

---

## A) FULLY DONE

| Item                     | Status      | Details                                                                                                                                     |
| ------------------------ | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| P5 Tasks 1-25            | ✅ Complete | All deduplication, generics, file splits done                                                                                               |
| Root module build        | ✅ Clean    | `go build ./...` passes                                                                                                                     |
| Root module tests        | ✅ 95.6%    | 289 tests, all pass with `-race`                                                                                                            |
| Root module lint         | ✅ 0 issues | `golangci-lint run` clean                                                                                                                   |
| usermgmt build           | ✅ Clean    | Builds and tests pass                                                                                                                       |
| usermgmt tests           | ✅ 84.8%    | All pass with `-race`                                                                                                                       |
| Branded UserID migration | ✅ Complete | `id.go` + all files updated                                                                                                                 |
| Rate limiter MaxKeys     | ✅ Complete | Production-safe bounded map                                                                                                                 |
| CSRF split               | ✅ Complete | `csrf.go` (386 lines) + `csrf_helpers.go` (65 lines)                                                                                        |
| Generic helpers          | ✅ Complete | `htmxBoolField`, `htmxStringField`, `decodeAndSet`, `validateDispatch`, `parseID`, `handleErrorCore`, `contextFields`, `notificationDetail` |

---

## B) PARTIALLY DONE / NEEDS ATTENTION

### 1. `csrf.go` is still 386 lines (threshold: 350)

The split into `csrf_helpers.go` only moved template rendering (65 lines). The main file still contains 4 responsibilities:

- Config + defaults (~130 lines)
- Context helpers (~30 lines)
- Middleware (`CSRFMiddleware`, `CSRFResponseHeaderMiddleware`) (~80 lines)
- Per-handler validation (`CSRFProtect`, `executeCSRFValidation`) (~140 lines)

**Action:** Split into `csrf.go` (config + context + middleware) and `csrf_handler.go` (per-handler validation).

### 2. `usermgmt/authz.go` — 5 public functions at 0% coverage

| Function                     | Coverage |
| ---------------------------- | -------- |
| `AddPolicy`                  | 0.0%     |
| `RemovePolicy`               | 0.0%     |
| `RemoveGroupPolicy`          | 0.0%     |
| `Policies`                   | 0.0%     |
| `GroupPolicies`              | 0.0%     |
| `ImplicitRolesForUser`       | 0.0%     |
| `ImplicitPermissionsForUser` | 0.0%     |

These are exported public API with zero test coverage. Any consumer depending on them has no safety net.

### 3. `usermgmt/user.go:SetPassword` at 0% coverage

`SetPassword` (delegates to `SetPasswordWithCost` with default cost) is untested directly. Covered indirectly via `Register`/`ChangePassword`, but the function itself has no dedicated test.

### 4. `usermgmt/store.go:WithTTL` at 0% coverage

`InMemorySessionStore.WithTTL` (sets custom TTL) is exported but untested.

---

## C) NOT STARTED

### Type Safety Improvements

| Issue                                                                | File                                             | Impact                                            |
| -------------------------------------------------------------------- | ------------------------------------------------ | ------------------------------------------------- |
| `RateLimiterConfig.Limit/Burst/MaxKeys` are `int` — should be `uint` | `ratelimit.go:40,45,57`                          | Negative values pass compilation, fail at runtime |
| `LockoutConfig.MaxAttempts` is `int` — should be `uint`              | `usermgmt/lockout.go:14`                         | Same                                              |
| `AccountLockout.attempts` values are `int` — should be `uint`        | `usermgmt/lockout.go:21`                         | Same                                              |
| Roles are `[]string` — should be `type Role string` + `[]Role`       | `usermgmt/user.go:27`, `usermgmt/authz.go:26-30` | Typos like `"admni"` compile fine                 |
| `GroupPolicy.User` is `string` — should be `UserID`                  | `usermgmt/authz.go:57`                           | Type confusion at Casbin boundary                 |
| `Policy.Subject` is `string` — inconsistent with `GroupPolicy.User`  | `usermgmt/authz.go:49`                           | Split brain: same concept, different names        |

### Bug Fix Needed

| Issue                                                | File              | Detail                                                                                                         |
| ---------------------------------------------------- | ----------------- | -------------------------------------------------------------------------------------------------------------- |
| **`sanitizeRedirectURL("/")` returns `("", false)`** | `response.go:201` | `resp.Redirect("/")` silently fails — writes 400 instead. The condition `u.Path != "/"` rejects root redirect. |

### Naming Improvements

| Current                                 | Proposed                                       | File                         |
| --------------------------------------- | ---------------------------------------------- | ---------------------------- |
| `RotateCSRFToken`                       | `InvalidateCSRFCookie` or `ClearCSRFCookie`    | `csrf.go:220`                |
| `AuthHandlers` (plural)                 | `AuthHandler`                                  | `usermgmt/http.go:9`         |
| `GroupPolicy.User`                      | `GroupPolicy.Subject` (match `Policy.Subject`) | `usermgmt/authz.go:57`       |
| `EnforceResult.MatchedRule`             | `EnforceResult.MatchedRules` (it's `[]string`) | `usermgmt/authz.go:64`       |
| `SessionMiddleware`                     | `NewSessionMiddleware` (match Go conventions)  | `usermgmt/middleware.go:32`  |
| Magic number `8` in password validation | `minPasswordLength` constant                   | `usermgmt/service.go:92,295` |

### Architecture Improvements

| Issue                                                                      | Detail                                                                                                                                                                     |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Split brain: `RenderPartial`**                                           | Method on `*HTMXRequest` (`htmx.go:66`) AND standalone function (`htmx.go:137`). Same logic, two locations.                                                                |
| **Split brain: auth flow**                                                 | `AuthorizeMiddleware` (authz.go:86-125) duplicates extract→unauthenticated→enforce flow from `executeAuthorization` (authz.go:63-81).                                      |
| **Split brain: UserID extraction**                                         | `ContextEnrichmentMiddleware` (middleware.go:40-44) and `App.enrichUserID` (app.go:164-179) both extract user IDs. Intentional (fallback), but logic exists in two places. |
| **Duplicate: `statusRecorder` construction**                               | Identical `&statusRecorder{ResponseWriter: w, status: 0, wrote: false}` in `RequestLogging` and `RequestLoggingSlog` (logging.go:108,149).                                 |
| **`executeCSRFValidation` uses `httptest.ResponseRecorder` in production** | `csrf.go:344-385` — test dependency in production code path.                                                                                                               |
| **Role storage dual-source**                                               | `User.Roles []string` AND Casbin grouping policies can diverge on partial failure in `UpdateRoles`.                                                                        |
| **Session TTL in 3 places**                                                | `Service.sessionTTL`, `AuthHandlers.sessionMaxAge`, `InMemorySessionStore.ttl` can drift.                                                                                  |
| **`UpdatedAt` ownership ambiguity**                                        | `AddRole`/`RemoveRole` and `Store.Save`/`Store.Create` both set it.                                                                                                        |

---

## D) TOTALLY FUCKED UP

Nothing is fucked up. The codebase compiles, tests pass, lint is clean. The issues above are improvements, not breakage.

---

## E) WHAT WE SHOULD IMPROVE

### Priority Matrix

1. **Bug fix:** `sanitizeRedirectURL("/")` — 5 min, blocks a real use case
2. **csrf.go split** — 10 min, file over 350 lines
3. **uint for rate limiter fields** — 10 min, type safety at compile time
4. **Role type in usermgmt** — 15 min, prevents typos in role strings
5. **usermgmt authz test coverage** — 30 min, 7 functions at 0%
6. **statusRecorder constructor** — 5 min, trivial dedup
7. **RenderPartial split brain** — 5 min, delegate function to method
8. **Auth flow dedup** — 15 min, extract shared auth subject extraction
9. **Naming cleanup** — 15 min, 6 items above
10. **MinPasswordLength constant** — 3 min, magic number elimination

---

## F) Top 25 Things We Should Get Done Next

| #  | Task                                                           | Impact   | Effort | Category      |
| -- | -------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Fix `sanitizeRedirectURL("/")` bug                             | Critical | 5min   | Bug fix       |
| 2  | Split `csrf.go` → `csrf.go` + `csrf_handler.go`                | High     | 10min  | File size     |
| 3  | `RateLimiterConfig.Limit/Burst/MaxKeys` → `uint`               | High     | 10min  | Type safety   |
| 4  | `LockoutConfig.MaxAttempts` → `uint`                           | High     | 5min   | Type safety   |
| 5  | Introduce `type Role string` in usermgmt + use everywhere      | High     | 15min  | Type safety   |
| 6  | Add tests for 7 untested authz functions                       | High     | 30min  | Test coverage |
| 7  | Add test for `usermgmt/user.go:SetPassword`                    | Medium   | 5min   | Test coverage |
| 8  | Add test for `usermgmt/store.go:WithTTL`                       | Medium   | 5min   | Test coverage |
| 9  | Extract `newStatusRecorder(w)` helper                          | Low      | 5min   | Dedup         |
| 10 | Fix `RenderPartial` split brain (function delegates to method) | Low      | 5min   | Dedup         |
| 11 | Extract shared auth subject extraction (authz.go)              | Medium   | 15min  | Dedup         |
| 12 | Rename `RotateCSRFToken` → `InvalidateCSRFCookie`              | Low      | 3min   | Naming        |
| 13 | Rename `AuthHandlers` → `AuthHandler`                          | Low      | 3min   | Naming        |
| 14 | Rename `GroupPolicy.User` → `GroupPolicy.Subject`              | Medium   | 5min   | Naming        |
| 15 | Rename `EnforceResult.MatchedRule` → `MatchedRules`            | Low      | 2min   | Naming        |
| 16 | Rename `SessionMiddleware` → `NewSessionMiddleware`            | Low      | 2min   | Naming        |
| 17 | Extract `minPasswordLength` constant                           | Low      | 3min   | Magic number  |
| 18 | Fix `UpdatedAt` ownership (store owns it, not domain methods)  | Medium   | 10min  | Architecture  |
| 19 | Consolidate session TTL (single source of truth)               | Medium   | 10min  | Architecture  |
| 20 | Document `Trigger` + `TriggerWithDetail` incompatibility       | Low      | 5min   | Docs          |
| 21 | Consider `GroupPolicy.User` type as `UserID` not `string`      | Medium   | 5min   | Type safety   |
| 22 | Fix `hasNoExplicitBody()` to check `c.render`                  | Low      | 5min   | Correctness   |
| 23 | Add `http.Pusher` to `statusRecorder`                          | Low      | 5min   | HTTP/2 compat |
| 24 | Document `readBody` unlimited behavior when `maxBodySize <= 0` | Low      | 3min   | Docs          |
| 25 | Consider removing `executeCSRFValidation` httptest dependency  | Low      | 15min  | Architecture  |

---

## Metrics

| Metric                          | Value               |
| ------------------------------- | ------------------- |
| Root module coverage            | 95.6%               |
| Root module test count          | 289                 |
| usermgmt coverage               | 84.8%               |
| Production files (root)         | 17                  |
| Production files (usermgmt)     | 9                   |
| Total production LOC            | ~4,054              |
| Lint issues                     | 0                   |
| Build issues                    | 0                   |
| Files over 350 lines            | 1 (`csrf.go`)       |
| Public functions at 0% coverage | 8 (all in usermgmt) |
