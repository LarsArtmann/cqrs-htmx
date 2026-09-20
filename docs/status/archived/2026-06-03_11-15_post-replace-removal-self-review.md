# cqrs-htmx — Comprehensive Status Report

**Date:** 2026-06-03 11:15
**Session:** Post replace-removal self-review + brutal honest assessment

---

## a) FULLY DONE

### Replace Directive Removal (Just Completed)

| Item                                                                | Status   |
| ------------------------------------------------------------------- | -------- |
| Remove go-cqrs-lite replace directives from root go.mod             | Done     |
| Remove go-cqrs-lite replace directives from usermgmt/go.mod         | Done     |
| Remove go-cqrs-lite replace directives from integration_test/go.mod | Done     |
| Remove go-cqrs-lite replace directives from datastar-demo/go.mod    | Done     |
| go mod tidy all 4 modules                                           | Done     |
| go work sync                                                        | Done     |
| go build ./... all 4 modules                                        | Pass     |
| go test ./... -race all 3 test modules                              | Pass     |
| nix run .#lint (root + usermgmt)                                    | 0 issues |
| nix flake check                                                     | All pass |
| Update AGENTS.md documentation                                      | Done     |
| Update TODO_LIST.md                                                 | Done     |

### v2.0.0 Migration (Previously Completed)

| Item                                              | Status   |
| ------------------------------------------------- | -------- |
| All 4 modules on go-cqrs-lite v2.0.0 import paths | Done     |
| CatalogEntries dead code removed                  | Done     |
| go-error-family v0.3.0 adopted                    | Done     |
| 390+ tests pass with race detector                | Pass     |
| 96.4% root coverage, 90.8% usermgmt               | Verified |

---

## b) PARTIALLY DONE

### Integration Testing

The `integration_test/` module exists but is a **token bridge test**, not real integration:

- 5 tests total (3 in integration_test.go, 2 in bridge_test.go)
- Tests type compatibility (UserID string bridging, Enforcer interface satisfaction)
- Does NOT test HTTP handler integration
- Does NOT test middleware chaining
- Does NOT test authorization end-to-end
- Does NOT test error propagation across modules

### Documentation Currency

| File                      | Status  | Issue                            |
| ------------------------- | ------- | -------------------------------- |
| AGENTS.md                 | Current | Updated for replace removal      |
| TODO_LIST.md              | Current | Updated                          |
| ROADMAP.md                | Stale   | Still says "Updated: 2026-05-27" |
| FEATURES.md               | Current | CatalogEntries marked REMOVED    |
| docs/modularization/\*.md | Stale   | Reference v0.1.1 / v1.6.0        |

### UpdateRoles Consistency

- User saved before Casbin policy updated
- No rollback if Casbin.Apply() fails
- User object has new roles, Casbin does not → split brain

---

## c) NOT STARTED

### Critical Security & Correctness (Pre-v1.1.0)

| # | Item                                   | File                 | Risk               |
| - | -------------------------------------- | -------------------- | ------------------ |
| 1 | Rate limiter unbounded heap growth     | ratelimit.go         | Memory DoS         |
| 2 | CSRF proxy bypass (r.TLS == nil)       | csrf.go              | Security           |
| 3 | Response.Status() breaks fluent chains | response.go          | API bug            |
| 4 | Nil-enforcer + query nil panic tests   | authz.go, handler.go | Missing coverage   |
| 5 | Login error classification tests       | service.go           | Wrong error family |
| 6 | UpdateRoles rollback / ordering fix    | service.go           | Data inconsistency |

### v2 Feature Adoption

| #  | Item                                         | Impact    | Effort |
| -- | -------------------------------------------- | --------- | ------ |
| 7  | Typed dispatch (RegisterTyped/DispatchTyped) | High      | Medium |
| 8  | PaginatedResult[T] for queries               | Medium    | Low    |
| 9  | Reactive event streams (EventBus)            | Very High | High   |
| 10 | Generic middleware adoption                  | Medium    | Medium |

### Architecture

| #  | Item                                 | Impact | Effort  |
| -- | ------------------------------------ | ------ | ------- |
| 11 | SQL store backend for usermgmt       | High   | Large   |
| 12 | Persistent session store (Redis/SQL) | Medium | Large   |
| 13 | OpenTelemetry integration            | High   | Medium  |
| 14 | Remove PtrBool (use new(bool))       | Low    | Trivial |

---

## d) TOTALLY FUCKED UP!

### Real Bugs

1. **UpdateRoles inconsistent state** — Saves user BEFORE updating Casbin. If Casbin fails, user has new roles but authz doesn't reflect it. No rollback.

2. **TestHandlers_Logout_ServiceError is an empty test** — Creates a service and mux, then does nothing. Passes but tests zero behavior. False confidence.

3. **TestService_Register_RollbackOnGroupPolicyFailure is a false positive** — Creates real Authz (which succeeds), Register succeeds, `regErr == nil`, so the rollback assertion block is **never entered**. Test passes while covering 0 rollback logic.

4. **Rate limiter unbounded memory growth** — Heap entries never removed on refresh. Documented, admitted, not fixed.

5. **CSRF proxy bypass** — `r.TLS == nil` trusts ALL HTTP proxies. Any proxy can bypass CSRF.

6. **Response.Status() breaks fluent API** — Calls WriteHeader() immediately, so subsequent Redirect()/PushURL() calls that set headers silently fail.

### Architecture Smells

7. **19 root package files** — Intentionally flat, but `go-structure-linter` flags all as "should be in internal/ or pkg/". This is by design per AGENTS.md, but it means we fight tooling.

8. **datastar-demo doesn't use cqrs-htmx** — Lives in our repo, imports only go-cqrs-lite + datastar-go. Doesn't demonstrate our library at all.

9. **6 error handler variants** — `DefaultErrorHandler`, `DefaultErrorHandlerWithRedirect`, `DefaultErrorHandlerWithRequestID`, `DefaultErrorHandlerWithRedirectAndRequestID`, `JSONErrorHandler`, `JSONErrorHandlerWithRedirect`. Should be one type with options.

10. **Split brain: 3 ways to emit CSRF token** — `CSRFTokenHTMLMeta`, `CSRFTokenHXHeaders`, `CSRFTokenFormField` (csrf_helpers.go) + `Response.CSRFToken()` (response.go) + `CSRFResponseHeaderMiddleware` (csrf.go). Same concept, 3 different APIs.

11. **Split brain: notifications in 2 places** — `NotifySuccess/Error/Warning/Info` exist as both `HandlerOption` (notify.go) and `Response` methods (response.go).

12. **Split brain: recovery in 2 places** — `RecoveryMiddleware` (package-level, uses DefaultErrorHandler) and `App.RecoverHandler()` (uses App.errorHandler). Same concern, two entrypoints.

13. **Split brain: authz enforcement** — `executeAuthorization` (unexported in handler.go) duplicates logic with `Enforce` (exported in authz.go).

---

## e) WHAT WE SHOULD IMPROVE!

### Immediate (This Session)

1. **Fix the 3 false-positive/empty tests** — `TestHandlers_Logout_ServiceError`, `TestService_Register_RollbackOnGroupPolicyFailure`, and add real rollback test.

2. **Fix UpdateRoles ordering** — Move Casbin update before user save, or add rollback.

3. **Add missing tests** — nil-enforcer bypass, query nil panic, login error classification.

### Short Term (Next Few Sessions)

4. **Fix rate limiter heap growth** — Add heapIndex map, use heap.Fix for in-place updates.

5. **Fix CSRF proxy bypass** — Add TrustedProxies config with IP-based trust check.

6. **Fix Response.Status()** — Defer WriteHeader to Apply().

7. **Collapse 6 error handlers to 1** — Use functional options pattern.

8. **Remove PtrBool** — `new(bool)` is idiomatic, `PtrBool` is redundant.

9. **Adopt v2 typed dispatch** — Add `CommandTyped`/`QueryTyped` HandlerOptions wrapping `command.RegisterTyped`/`query.RegisterTyped`.

### Medium Term

10. **Real integration tests** — Wire usermgmt.AuthHandler into cqrshtmx.App, test over actual HTTP.

11. **SQL store backend** — Implement PostgreSQL UserStore/SessionStore.

12. **OpenTelemetry** — Use upstream middleware.NewTracing[M] via MessageAdapter.

13. **Reactive streams** — Expose event.EventBus for real-time SSE/HTMX updates.

### Documentation

14. **Update ROADMAP.md** — Date and status are stale.

15. **Clean stale coverage files** — usermgmt/cov.out, usermgmt/coverage.out, reports/coverage.out.

---

## f) TOP 25 THINGS TO DO NEXT

Sorted by impact/effort ratio (highest impact, lowest effort first):

| #  | Item                                          | Impact    | Effort | Module           |
| -- | --------------------------------------------- | --------- | ------ | ---------------- |
| 1  | Fix false-positive Register rollback test     | High      | 15 min | usermgmt         |
| 2  | Fix empty Logout service error test           | High      | 10 min | usermgmt         |
| 3  | Fix UpdateRoles ordering (save after Casbin)  | High      | 15 min | usermgmt         |
| 4  | Add nil-enforcer + query nil panic tests      | High      | 20 min | root             |
| 5  | Add Login error classification tests          | High      | 20 min | usermgmt         |
| 6  | Remove PtrBool, use new(bool)                 | Low       | 10 min | usermgmt         |
| 7  | Clean stale coverage files                    | Low       | 5 min  | repo             |
| 8  | Update ROADMAP.md date/status                 | Low       | 10 min | docs             |
| 9  | Fix Response.Status() fluent chain            | Medium    | 30 min | root             |
| 10 | Fix rate limiter heap growth                  | Medium    | 45 min | root             |
| 11 | Collapse 6 error handlers to 1 with options   | Medium    | 1 hr   | root             |
| 12 | Add real integration HTTP tests               | High      | 2 hr   | integration_test |
| 13 | Fix CSRF proxy bypass                         | Medium    | 1 hr   | root             |
| 14 | Adopt v2 typed dispatch                       | High      | 1.5 hr | root             |
| 15 | Add PaginatedResult[T] support                | Medium    | 30 min | root             |
| 16 | Remove ClientIP re-export (dead weight)       | Low       | 5 min  | root             |
| 17 | Fix decodeFormBody to use PostForm            | Low       | 10 min | root             |
| 18 | Fix \_ = r.Body.Close() in decoder            | Low       | 10 min | root             |
| 19 | Add RequestLogging / RequestLoggingSlog dedup | Low       | 20 min | root             |
| 20 | SQL store backend                             | Very High | 8 hr   | usermgmt         |
| 21 | OpenTelemetry middleware                      | High      | 3 hr   | root             |
| 22 | Reactive event streams                        | Very High | 4 hr   | root             |
| 23 | Redis session store                           | Medium    | 4 hr   | usermgmt         |
| 24 | JWT/OIDC integration                          | High      | 6 hr   | usermgmt         |
| 25 | datastar-demo actually uses cqrs-htmx         | Medium    | 2 hr   | examples         |

---

## g) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should we fix the root package flatness (19 files) or accept the tooling fight?**

The AGENTS.md explicitly defends the flat package: "Root module is intentionally a single flat package... Sub-package extraction would harm consumer UX." But this means:

- `go-structure-linter` flags every file as "should be in internal/ or pkg/"
- 19 files in one package is a lot of surface area
- Consumers import everything or nothing — no granular imports

The alternative:

- Split into `cqrshtmx/authz`, `cqrshtmx/htmx`, `cqrshtmx/middleware`, `cqrshtmx/security`
- Consumers can import only what they need
- Better tooling compliance
- But more complex module boundary, potential import cycles, worse UX for simple use cases

**I don't know which is better for a library's long-term health.** The "flat is simpler" argument is strong for adoption, but the "structure matters" argument is strong for maintenance.

---

## Metrics

| Metric        | Root  | Usermgmt | Integration   |
| ------------- | ----- | -------- | ------------- |
| Coverage      | 96.4% | 90.8%    | N/A (5 tests) |
| Lint issues   | 0     | 0        | 0             |
| Go files      | 19    | 9        | 2             |
| Test files    | 23    | 11       | 2             |
| Tests passing | 390+  | 100+     | 5             |
| Race detector | Pass  | Pass     | Pass          |

## Dependencies

| Module        | go-cqrs-lite                          | Status                 |
| ------------- | ------------------------------------- | ---------------------- |
| Root          | command/v2, event/v2, id/v2, query/v2 | Resolved from upstream |
| Usermgmt      | event/v2                              | Resolved from upstream |
| Integration   | command/v2                            | Resolved from upstream |
| Datastar-demo | command/v2, id/v2, query/v2           | Resolved from upstream |

**All replace directives removed.** Only `integration_test` retains local cqrs-htmx replaces (library not yet published).
