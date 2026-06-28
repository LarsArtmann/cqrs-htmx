# cqrs-htmx — Comprehensive Status Report (Round 2)

**Date:** 2026-06-03 12:40
**Session:** Self-review execution — deprecation cleanup, coverage gaps, architecture fixes

---

## a) FULLY DONE (This Session)

### Deprecation Cleanup

| Item                                                      | Status |
| --------------------------------------------------------- | ------ |
| Delete PtrBool function entirely                          | Done   |
| Fix all PtrBool(false) → new(bool) in tests               | Done   |
| Fix all PtrBool(true) → inline closures in tests/examples | Done   |
| Deprecate ClientIP re-export with proper godoc            | Done   |
| KeyExtractorFromClientIP imports httputil directly        | Done   |

### Bug Fixes (Continued from Round 1)

| Item                                                  | Status |
| ----------------------------------------------------- | ------ |
| Fix UpdateRoles ordering (Casbin before user save)    | Done   |
| Nil enforcer protection on all 14 Authz methods       | Done   |
| Response.Status() defers WriteHeader to Apply()       | Done   |
| Rate limiter unbounded heap growth (in-place updates) | Done   |
| Decoder body close error handling                     | Done   |
| decodeFormBody uses PostForm (not Form)               | Done   |
| False-positive rollback test fixed                    | Done   |
| Empty logout test removed                             | Done   |

### New Tests Added

| Test                                              | Module   | What It Covers                     |
| ------------------------------------------------- | -------- | ---------------------------------- |
| TestService_UpdateRoles_AuthzFailurePreservesUser | usermgmt | Rollback on Casbin failure         |
| TestService_UpdateRoles_AuthzApplyError           | usermgmt | nil enforcer in Apply              |
| TestService_UpdateRoles_SaveError                 | usermgmt | Store save failure                 |
| TestService_Login_StoreError                      | usermgmt | Transient vs InvalidCredentials    |
| TestService_Register_RollbackOnGroupPolicyFailure | usermgmt | Real rollback (was false positive) |
| Status defers to Apply                            | root     | WriteHeader not called immediately |
| Status + Redirect chain                           | root     | Fluent API works correctly         |
| WriteString non-StringWriter fallback             | root     | io.Write fallback path             |
| JSON marshal error                                | root     | HTTP 500 on unmarshalable value    |
| Body return value                                 | root     | Method chaining                    |

### Documentation

| Item                                                             | Status |
| ---------------------------------------------------------------- | ------ |
| docs/status/2026-06-03_11-15_post-replace-removal-self-review.md | Done   |
| docs/planning/2026-06-03_11-30_comprehensive-fix-plan.md         | Done   |
| docs/planning/2026-06-03_12-00_round-2-comprehensive-fixes.md    | Done   |
| AGENTS.md updated for replace removal                            | Done   |
| TODO_LIST.md updated (6 items marked done)                       | Done   |

---

## b) PARTIALLY DONE

### Replace Directive Removal

- Root, usermgmt, datastar-demo, integration_test: all go-cqrs-lite replace directives removed
- integration_test still retains cqrs-htmx → ../ and usermgmt → ../usermgmt replaces (library not published)

### Integration Testing

- 5 tests in integration_test/ module
- Tests type bridging (UserID compatibility, Enforcer interface) but NOT HTTP handler integration
- No middleware chaining tests
- No authorization end-to-end tests

---

## c) NOT STARTED

### Critical Security (Pre-v1.1.0)

| #   | Item                                                | File    | Risk     |
| --- | --------------------------------------------------- | ------- | -------- |
| 1   | CSRF proxy bypass — r.TLS == nil trusts all proxies | csrf.go | Security |

### Architecture Improvements

| #   | Item                                                  | Impact    | Effort |
| --- | ----------------------------------------------------- | --------- | ------ |
| 2   | Collapse 6 error handlers to 1 with options           | Medium    | 1 hr   |
| 3   | Deduplicate RequestLogging/RequestLoggingSlog         | Low       | 30 min |
| 4   | Adopt v2 typed dispatch (RegisterTyped/DispatchTyped) | High      | 1.5 hr |
| 5   | Add PaginatedResult[T] support                        | Medium    | 30 min |
| 6   | Real integration HTTP tests                           | High      | 2 hr   |
| 7   | SQL store backend                                     | Very High | 8 hr   |
| 8   | OpenTelemetry integration                             | High      | 3 hr   |
| 9   | Reactive event streams (EventBus)                     | Very High | 4 hr   |

### Type Safety

| #   | Item                                           | Impact |
| --- | ---------------------------------------------- | ------ |
| 10  | WriteJSON[T](w, status, v T) generic variant   | Medium |
| 11  | Response.JSONTyped[T](v T)                     | Medium |
| 12  | Typed EventHandler[T any] instead of event any | High   |
| 13  | Remove EnforceAny/AsEnforcer adapter triad     | Medium |

---

## d) TOTALLY FUCKED UP!

### Real Bugs

1. **CSRF proxy bypass** — `r.TLS == nil` trusts ALL HTTP proxies. Any proxy can bypass CSRF. (NOT FIXED)

### Architecture Smells

2. **6 error handler variants** — `DefaultErrorHandler`, `DefaultErrorHandlerWithRedirect`, `DefaultErrorHandlerWithRequestID`, `DefaultErrorHandlerWithRedirectAndRequestID`, `JSONErrorHandler`, `JSONErrorHandlerWithRedirect`. Should be one type with options.

3. **Split brain: 3 ways to emit CSRF token** — `CSRFTokenHTMLMeta`, `CSRFTokenHXHeaders`, `CSRFTokenFormField` + `Response.CSRFToken()` + `CSRFResponseHeaderMiddleware`. Same concept, 3 APIs.

4. **Split brain: notifications in 2 places** — `NotifySuccess/Error/Warning/Info` exist as both `HandlerOption` and `Response` methods.

5. **Split brain: recovery in 2 places** — `RecoveryMiddleware` (package-level) and `App.RecoverHandler()` (same concern, two entrypoints).

6. **Split brain: authz enforcement** — `executeAuthorization` (unexported in handler.go) duplicates logic with `Enforce` (exported in authz.go).

---

## e) WHAT WE SHOULD IMPROVE!

### Immediate (Next Session)

1. **Fix CSRF proxy bypass** — Add `TrustedProxies []string` config with IP-based trust check
2. **Collapse 6 error handlers to 1** — Use functional options pattern
3. **Deduplicate RequestLogging/RequestLoggingSlog** — Extract shared formatter logic

### Short Term

4. **Adopt v2 typed dispatch** — `CommandTyped`/`QueryTyped` HandlerOptions
5. **Add PaginatedResult[T]** — Typed pagination for query handlers
6. **Add real integration HTTP tests** — Wire usermgmt.AuthHandler into cqrshtmx.App
7. **Add middleware chaining tests** — Session + ContextEnrichment + Authorize

### Medium Term

8. **SQL store backend** — PostgreSQL UserStore/SessionStore
9. **OpenTelemetry** — Use upstream middleware.NewTracing via MessageAdapter
10. **Reactive streams** — Expose event.EventBus for real-time SSE/HTMX updates

---

## f) TOP 25 THINGS TO DO NEXT

Sorted by impact/effort ratio (highest first):

| #   | Item                                          | Impact    | Effort | Module           |
| --- | --------------------------------------------- | --------- | ------ | ---------------- |
| 1   | Fix CSRF proxy bypass                         | High      | 1 hr   | root             |
| 2   | Collapse 6 error handlers to 1 with options   | Medium    | 1 hr   | root             |
| 3   | Adopt v2 typed dispatch                       | High      | 1.5 hr | root             |
| 4   | Add PaginatedResult[T] support                | Medium    | 30 min | root             |
| 5   | Add real integration HTTP tests               | High      | 2 hr   | integration_test |
| 6   | Deduplicate RequestLogging/RequestLoggingSlog | Low       | 30 min | root             |
| 7   | Add middleware chaining integration tests     | Medium    | 1 hr   | integration_test |
| 8   | Add OpenTelemetry via upstream middleware     | High      | 3 hr   | root             |
| 9   | Reactive event streams (EventBus)             | Very High | 4 hr   | root             |
| 10  | SQL store backend                             | Very High | 8 hr   | usermgmt         |
| 11  | Add WriteJSON[T] generic variant              | Medium    | 15 min | root             |
| 12  | Add Response.JSONTyped[T]                     | Medium    | 15 min | root             |
| 13  | Use samber/lo for slice operations            | Low       | 20 min | various          |
| 14  | Remove EnforceAny/AsEnforcer triad            | Medium    | 30 min | usermgmt         |
| 15  | Fix TriggerWithDetail non-determinism         | Low       | 30 min | root             |
| 16  | Add RequestLoggingSlog tests                  | Low       | 20 min | root             |
| 17  | Add error handler variant tests               | Low       | 15 min | root             |
| 18  | Update ROADMAP.md date/status                 | Low       | 10 min | docs             |
| 19  | Clean stale coverage files                    | Low       | 5 min  | repo             |
| 20  | Redis session store                           | Medium    | 4 hr   | usermgmt         |
| 21  | JWT/OIDC integration                          | High      | 6 hr   | usermgmt         |
| 22  | Fix datastar-demo to use cqrs-htmx            | Medium    | 2 hr   | examples         |
| 23  | Add decodeFormValues json.Marshal error test  | Low       | 10 min | root             |
| 24  | Add HealthHandler unhealthy branch test       | Low       | 10 min | root             |
| 25  | Add enrichUserID extractor error test         | Low       | 10 min | root             |

---

## g) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should we migrate root tests from Ginkgo to standard `testing`?**

The root package uses Ginkgo/Gomega (heavy BDD dependencies, 23 test files). The `usermgmt` module uses standard `testing`. For a library:

**Pros of migrating:**

- Standard `testing` is more idiomatic for libraries
- Eliminates heavy BDD dependency from go.mod (Ginkgo + Gomega)
- Consumers don't transitively import BDD frameworks
- Faster test execution, simpler stack traces
- Consistent with usermgmt module

**Cons of migrating:**

- Rewriting 23 test files is significant work (2-4 hours)
- Ginkgo's `DescribeTable` and nested contexts are expressive
- Some tests rely on Ginkgo's beforeEach/afterEach patterns
- Risk of introducing test regressions during rewrite

**Alternative:** Keep Ginkgo but move it to `go.mod` `test` directive (Go 1.22+ supports test-only dependencies more cleanly).

I don't know which path is better for long-term library health.

---

## Metrics

| Metric        | Root  | Usermgmt | Integration   |
| ------------- | ----- | -------- | ------------- |
| Coverage      | 96.3% | ~89.8%   | N/A (5 tests) |
| Lint issues   | 0     | 0        | 0             |
| Go files      | 19    | 8        | 2             |
| Test files    | 23    | 11       | 2             |
| Tests passing | 390+  | 100+     | 5             |
| Race detector | Pass  | Pass     | Pass          |

## Dependencies

All go-cqrs-lite replace directives removed. v2.0.0 tags resolved from upstream.

PtrBool deleted. ClientIP deprecated (re-export to be removed in future version).
