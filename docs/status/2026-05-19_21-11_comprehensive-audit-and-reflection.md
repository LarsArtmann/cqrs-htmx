# Comprehensive Status Report — Deep Audit & Reflection

**Date:** 2026-05-19 21:11 CEST
**Branch:** master (3 commits ahead of origin)
**Reporter:** Crush (AI Engineering Partner)

---

## 1. Overall Health

|| Metric               | Value           | Status |
|| -------------------- | --------------- | ------ |
|| Root Tests           | 289 / 289       | Green  |
|| Root Coverage        | 95.7%           | Green  |
|| usermgmt Tests       | 45 / 45         | Green  |
|| Race Detector        | Clean (both)    | Green  |
|| go build ./...       | Pass (both)     | Green  |
|| golangci-lint        | 0 issues        | Green  |
|| Prod Files           | 22 (15 root + 7 usermgmt) | — |
|| Total LOC            | ~10,451         | —      |

**Overall Assessment:** `STABLE` — Both modules build, test, and lint cleanly. Coverage is strong at 95.7%. No regressions. But the self-critique below reveals significant architectural debt and missed opportunities.

---

## a) FULLY DONE ✅

### Root Library (cqrs-htmx)

All features from the 19 status reports in `docs/status/` are complete:

- App builder with validation, timeout, lifecycle hooks
- Command/Query dispatch with HTMX response handling
- JSON/Form decoding with body size limits
- Casbin authorization (Enforcer interface, Authorize, Enforce, AuthorizeMiddleware)
- User identity propagation (strongly-typed UserID, CorrelationID, RequestID)
- HTMX request context, response builder, swap strategies
- Notifications (HandlerOption + Response methods, NotifyWithEvent builder)
- Error classification with sentinels and HTMX-aware error handlers
- CSRF protection via gorilla/csrf
- Rate limiting with per-key token bucket, configurable TTL, hooks
- Request logging (plain text + JSON + slog)
- Security headers (configurable builder)
- Middleware chain, context enrichment, RequestID auto-generation
- Per-handler timeout override (WithTimeout)
- 9 godoc examples, 16 benchmarks

### usermgmt Submodule

- Authz wrapper around Casbin (RBAC with domains, deny-override)
- Service layer (register, login, logout, authenticate, authorize, updateRoles)
- User/Session types with bcrypt password hashing
- In-memory UserStore/SessionStore implementations
- HTTP handlers (AuthHandlers with route registration)
- SessionMiddleware (cookie + Bearer token)
- 45 tests, fast execution (bcrypt cost 4 in tests)

---

## b) PARTIALLY DONE ⚠️

| Item                                      | Why Partial                                                    |
| ----------------------------------------- | -------------------------------------------------------------- |
| usermgmt ↔ cqrs-htmx integration          | Two separate authorization paths; no bridge between them       |
| usermgmt input validation                 | No email/password/display name validation anywhere             |
| usermgmt context bridging                 | SessionMiddleware stores `*User` but doesn't set `cqrshtmx.UserID` in context |
| `statusRecorder` in logging.go            | Missing `http.Flusher`/`http.Hijacker` — SSE/WebSocket break   |
| Error handler duplication                 | `DefaultErrorHandlerWithRedirect` / `JSONErrorHandlerWithRedirect` share 90% logic |

---

## c) NOT STARTED ❌

### High-Value Missing Work

| #   | Item                                                                  | Impact     |
| --- | --------------------------------------------------------------------- | ---------- |
| 1   | usermgmt: `Service` methods don't accept `context.Context`            | Critical   |
| 2   | usermgmt: TOCTOU race in `Register()` (FindByEmail → Save unlocked)   | Security   |
| 3   | usermgmt: No email/password validation                                | Security   |
| 4   | usermgmt: `errorStatus()` duplicates parent's `MapError()`            | DRY        |
| 5   | usermgmt: Cookie `MaxAge` hardcoded (86400) vs session TTL (24h)      | Correctness|
| 6   | Root: `ValidateCommand`/`ValidateQuery` silently no-op on nil decoder | Security   |
| 7   | Root: Rate limiter magic numbers (100, 1m, 10m) → package constants   | Clarity    |
| 8   | Root: Content-Type strings scattered as magic strings                 | DRY        |
| 9   | usermgmt: `UpdateRoles()` is non-atomic (remove all, add all)        | Correctness|
| 10  | usermgmt: `FindByEmail` is O(n) — needs email index                  | Performance|

### Lower-Priority Missing Work

| #   | Item                                                                  | Impact     |
| --- | --------------------------------------------------------------------- | ---------- |
| 11  | `handleErr` helper to deduplicate error handler + afterDispatchHook   | DRY (6×)   |
| 12  | Generic context accessor helper (3× repeated pattern)                 | DRY        |
| 13  | `Response.CSRFToken()` has no direct test                             | Coverage   |
| 14  | `Response.NotifySuccess/Error/Warning/Info` have no direct tests       | Coverage   |
| 15  | `DefaultLogFormatter` has no direct test                               | Coverage   |
| 16  | `HTMXRequest.RenderPartial()` method has no direct test                | Coverage   |
| 17  | `bcryptCost` is a mutable `var` — should be `ServiceConfig` field     | Correctness|
| 18  | `RawEnforcer()` leaks implementation detail                           | API design |
| 19  | `Authz` should satisfy parent's `Enforcer` interface                  | Integration|
| 20  | usermgmt: Password reset / forgot password flow                       | Feature    |
| 21  | usermgmt: Email verification                                          | Feature    |
| 22  | usermgmt: Account lockout after N failed attempts                     | Security   |
| 23  | usermgmt: Audit logging (policy changes, logins, role assignments)    | Observability|
| 24  | usermgmt: Profile update / user deletion via Service                   | Feature    |
| 25  | usermgmt: No logging anywhere (failed auth silently swallowed)         | Debugging  |

---

## d) TOTALLY FUCKED UP 🔥

### 1. Two Competing Authorization Systems — No Bridge

**usermgmt** has its own `Authz` struct wrapping `*casbin.Enforcer`.
**cqrs-htmx** has `Enforcer` interface + `Authorize`/`Enforce`/`AuthorizeMiddleware`.

They do NOT interoperate:
- `usermgmt.Authz` does NOT satisfy `cqrshtmx.Enforcer` (different parameter count)
- `usermgmt.SessionMiddleware` stores `*User` in context but does NOT set `cqrshtmx.UserID`
- Downstream handlers using `cqrshtmx.Authorize()` or `cqrshtmx.AuthorizeMiddleware` will see NO user identity from usermgmt sessions

**Impact:** Consumers using both packages together will have auth that silently fails. This is a split brain.

### 2. `statusRecorder` Missing Interface Support

`/home/lars/projects/cqrs-htmx/logging.go` wraps `http.ResponseWriter` but does NOT implement `http.Flusher`, `http.Hijacker`, or `http.Pusher`. Any SSE, WebSocket, or HTTP/2 push behind `RequestLogging` or `RequestLoggingSlog` will break silently.

### 3. `bcryptCost` Is a Package-Level Mutable Variable

`user.go:21` declares `var bcryptCost = defaultBcryptCost`. Any goroutine can change this at runtime without synchronization. The `TestMain` workaround sets it once at test start, but this is fragile — parallel tests or imported packages could modify it.

---

## e) WHAT WE SHOULD IMPROVE!

### Critical Fixes (P0)

1. **Bridge usermgmt → cqrs-htmx identity**: `SessionMiddleware` must set `cqrshtmx.UserID` in context (parse from `user.ID`). Otherwise the parent's auth/CSRF/context system is invisible to usermgmt sessions.

2. **Fix `statusRecorder`**: Implement `http.Flusher` and `http.Hijacker` interfaces. Use `ok` type assertion pattern from chi/stdlib.

3. **Make `bcryptCost` immutable**: Remove package-level `var`, add `BcryptCost int` to `ServiceConfig`, default in `NewService`.

### High Impact (P1)

4. **Add `context.Context` to all `Service` methods**: `Register(ctx, req)`, `Login(ctx, req)`, `Authenticate(ctx, token)`, etc. Without context, there's no cancellation, no timeout, no tracing, no distributed context propagation.

5. **Fix TOCTOU race in `Register()`**: Lock across `FindByEmail` + `Save` in `InMemoryUserStore`, or add `CreateIfNotExists` method.

6. **Validate user input**: Email format (`net/mail`), password minimum length, display name sanitization. Add `Validate() error` on request types.

7. **Make `Authz` satisfy parent `Enforcer` interface**: Either adapt the 4-param `Enforce(sub, dom, obj, act)` to the parent's `Enforce(...any) (bool, error)`, or provide a wrapper.

8. **Extract `handleErr` helper**: Deduplicate `a.errorHandler(w, r, err); a.afterDispatchHook(ctx, r, err); return` from 6 call sites.

### Medium Impact (P2)

9. **Rate limiter constants**: Extract `100`, `time.Minute`, `10*time.Minute` to exported `DefaultRateLimit`, `DefaultRateWindow`, `DefaultRateTTL`.

10. **Content-Type constants**: Extract `"text/plain; charset=utf-8"`, `"application/json; charset=utf-8"`, `"text/html; charset=utf-8"`.

11. **Cookie MaxAge from sessionTTL**: `http.go:113` hardcodes `86400`. Should use `int(s.sessionTTL.Seconds())`.

12. **Use `strings.CutPrefix`** in `extractToken()`: Replace `auth[:7] == "Bearer "` with `strings.CutPrefix(auth, "Bearer ")`.

13. **Add `FindByEmail` index** to `InMemoryUserStore`: Add `emails map[string]string` field. O(n) → O(1).

14. **Remove `RawEnforcer()`**: It leaks `*casbin.Enforcer`. If needed, expose specific operations instead.

### Lower Impact (P3)

15. **Test `Response.CSRFToken()`**: Direct test for `resp.CSRFToken(token)` setting header.
16. **Test `Response.NotifySuccess/Error/Warning/Info`**: Direct tests for Response builder notification methods.
17. **Test `DefaultLogFormatter`**: Direct formatter output test.
18. **Test `HTMXRequest.RenderPartial()` method**: Struct method test.
19. **Replace `decodeFormValues` with `go-playground/form/v4`**: Better form decoding with proper type conversion.
20. **usermgmt: Use `cockroachdb/errors`** instead of `std errors`: Consistent error wrapping with stack traces.

---

## f) Top #25 Things To Get Done Next

Sorted by **Impact / Effort** ratio (highest first):

| #   | Task                                                           | Effort | Impact    | Type          |
| --- | -------------------------------------------------------------- | ------ | --------- | ------------- |
| 1   | Bridge usermgmt→cqrs-htmx: SessionMiddleware sets UserID      | 10min  | Critical  | Integration   |
| 2   | Fix statusRecorder: add Flusher/Hijacker                      | 10min  | Critical  | Bug fix       |
| 3   | Extract rate limiter default constants                        | 5min   | Medium    | DRY           |
| 4   | Extract Content-Type string constants                         | 5min   | Medium    | DRY           |
| 5   | Cookie MaxAge from sessionTTL                                 | 5min   | Medium    | Correctness   |
| 6   | Use strings.CutPrefix in extractToken                         | 3min   | Low       | Modernize     |
| 7   | Extract handleErr helper (deduplicate 6×)                     | 10min  | Medium    | DRY           |
| 8   | Make bcryptCost immutable via ServiceConfig                   | 10min  | Medium    | Correctness   |
| 9   | Add context.Context to Service methods                        | 20min  | High      | API design    |
| 10  | Validate user input (email, password, display name)           | 15min  | High      | Security      |
| 11  | Fix TOCTOU race in Register()                                 | 10min  | High      | Security      |
| 12  | Add FindByEmail index to InMemoryUserStore                    | 10min  | Medium    | Performance   |
| 13  | Test Response.CSRFToken() directly                            | 5min   | Low       | Coverage      |
| 14  | Test Response.NotifySuccess/Error/Warning/Info                | 10min  | Low       | Coverage      |
| 15  | Test DefaultLogFormatter directly                             | 5min   | Low       | Coverage      |
| 16  | Make Authz satisfy parent Enforcer interface                  | 15min  | High      | Integration   |
| 17  | Remove RawEnforcer(), expose specific ops                     | 10min  | Medium    | API design    |
| 18  | usermgmt: use cockroachdb/errors                              | 10min  | Medium    | Consistency   |
| 19  | Atomic UpdateRoles via Authz.Apply                            | 10min  | Medium    | Correctness   |
| 20  | usermgmt: Add slog-based logging                              | 15min  | Medium    | Observability |
| 21  | Generic context accessor helper                               | 15min  | Low       | DRY           |
| 22  | ValidateCommand/ValidateQuery: warn on nil decoder            | 5min   | Medium    | Security      |
| 23  | Short CSRF secret warning in Validate()                       | 5min   | Medium    | Security      |
| 24  | usermgmt: Password change method                              | 15min  | Medium    | Feature       |
| 25  | usermgmt: Account lockout after N failures                    | 20min  | High      | Security      |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should usermgmt be a separate Go module (`usermgmt/go.mod`) or should it be a sub-package of cqrs-htmx (`github.com/larsartmann/cqrs-htmx/usermgmt`)?**

Arguments for separate module (current):
- Independent versioning
- Consumers who don't need user management don't pull it in
- Clean dependency isolation (usermgmt only needs casbin + crypto)

Arguments for sub-package:
- **Direct access to internal types** (`decodeJSONBody`, `MapError`, `Enforcer` interface)
- **No integration bridge needed** — SessionMiddleware can directly call `WithUserID()`
- **No duplicate sentinels** — share `ErrForbidden`, `ErrUnauthorized`
- **Simpler consumer experience** — one import, one version
- The current separate module has ZERO imports from the parent — it's completely standalone, which means all shared concepts are duplicated

The current architecture forces a painful choice: either usermgmt imports cqrs-htmx (circular dependency risk since usermgmt is in a subdirectory), or they stay isolated with duplicated types and no integration. Either the directory structure needs to change (separate repo), or the module structure needs to change (sub-package of cqrs-htmx).

**What is the intended relationship between these two packages?**

---

## Commit History (This Session)

```
1c5d3fb chore: update test coverage, add usermgmt bcrypt test override, reformat long lines
c2bac30 feat(usermgmt): rebuild as superb ES/CQRS auth library
```

---

_Updated: 2026-05-19 21:11 CEST_
