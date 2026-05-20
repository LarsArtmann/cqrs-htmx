# Full Code Review — cqrs-htmx

**Date:** 2026-05-07 | **Reviewer:** Senior Staff Architect | **Coverage:** 92.8% | All 18 files reviewed

## Files Reviewed

| File                  | Lines | Role                                                   |
| --------------------- | ----- | ------------------------------------------------------ |
| `app.go`              | 137   | App builder, Config, Command/Query factories           |
| `handler.go`          | 92    | Command/Query dispatch orchestration                   |
| `options.go`          | 228   | HandlerOption types, decoders, renderers, auth helpers |
| `response.go`         | 184   | HTMX response builder (fluent API)                     |
| `authz.go`            | 76    | Casbin authorization                                   |
| `context.go`          | 42    | User ID context propagation                            |
| `errors.go`           | 98    | Error classification, mapping, sentinel errors         |
| `htmx.go`             | 167   | HTMX request headers, context, accessors               |
| `middleware.go`       | 47    | HTTP middleware (HTMX, context enrichment, chain)      |
| `notify.go`           | 37    | Notification HandlerOptions                            |
| `suite_test.go`       | 13    | Ginkgo test suite                                      |
| `app_test.go`         | 459   | App/command/query/authorization tests                  |
| `htmx_test.go`        | 456   | HTMX request/response/notification tests               |
| `middleware_test.go`  | 109   | Middleware tests                                       |
| `context_test.go`     | 55    | Context propagation tests                              |
| `errors_test.go`      | 84    | Error mapping tests                                    |
| `integration_test.go` | 277   | End-to-end integration tests                           |
| `coverage_test.go`    | 534   | Coverage gap tests                                     |

## Findings by File

### app.go (137 lines)

**Assessment: GOOD with minor issues**

- `buildHandlerConfig` creates zero-value struct — exhaustruct linter complains but this is intentional pattern (options fill in fields). Consider `nolint:exhaustruct` directive.
- `enrichUserID` correctly deduplicates user ID extraction (checks context first).
- `Config.LoginRedirect` mutates package-level global `LoginRedirect` inside `New()`. This is a side effect in a constructor — surprising. Should document or restructure.
- Missing package-level doc comment (revive).

### handler.go (92 lines)

**Assessment: NEEDS WORK**

- `handleCommandDispatch` has cyclomatic complexity 11 (max 10). Root cause: the response-finalization logic at the end has too many branches.
- Empty `else if` block at line 44 — dead code path where non-HTMX redirect was already written by `applyHTMXResponse`. This does nothing.
- Command and query dispatchers share a similar pattern (authorize → decode → dispatch → respond). The duplication is acceptable given the different types, but the response-finalization could be extracted.

### options.go (228 lines)

**Assessment: GOOD with minor issues**

- `decodeFormValues` returns unwrapped `json.Marshal`/`json.Unmarshal` errors (wrapcheck). Should wrap with context.
- `DecodeJSON` and `DecodeJSONQuery` are nearly identical — could share a generic decoder, but the different return types (command.Command vs query.Query) make this tricky. Acceptable duplication.
- `executeAuthorization` checks both `cfg.authorize` and `cfg.requireAuth` together. Clean separation.
- `applyHTMXResponse` creates a new `Response` builder for every successful dispatch — fine for a library.

### response.go (184 lines)

**Assessment: GOOD**

- Fluent API is clean and idiomatic.
- `setTriggerWithDetail` has 82.4% coverage — the merge-failure fallback path is untested.
- `NotifySuccess/Error/Warning/Info` duplicate the same pattern 4x. Could use a private helper, but the public API surface is clean.
- `Apply()` sets `text/html` content-type for HTMX requests. This is correct for partial HTML responses.

### authz.go (76 lines)

**Assessment: GOOD**

- `Enforce` correctly handles nil enforcer, error from Casbin, and denied policy.
- `AuthorizeMiddleware` is a standalone convenience — doesn't depend on `App`. Good separation.
- Uses `errors.Wrapf` correctly for Casbin errors.

### context.go (42 lines)

**Assessment: GOOD**

- Clean, minimal. `EventOptionsFromContext` gracefully handles invalid user IDs (returns empty `id.UserID{}`).
- `contextKey` type prevents collisions.

### errors.go (98 lines)

**Assessment: NEEDS FIX**

- **XSS (gosec G705):** `DefaultErrorHandler` writes `err.Error()` directly to response body at line 97. While `text/plain` content-type mitigates most risks, some browsers may still render HTML. Should sanitize or truncate.
- `init()` registers error classifications — necessary for CQRS integration but linter complains. This is acceptable; add `nolint` directive.
- `LoginRedirect` is a mutable package-level global. The `Config.LoginRedirect` field mutates it in `New()`, which means creating two Apps with different login redirects will race. Should store on App instead.

### htmx.go (167 lines)

**Assessment: GOOD with minor issues**

- Accessor functions check context first, then fall back to headers. This dual-path is well-designed.
- `"true"` string literal repeated 8x (goconst). Should extract to a constant.
- Swap strategy const block missing doc comment.
- `HTMXRequest` struct is well-designed with `RenderPartial()` method.

### middleware.go (47 lines)

**Assessment: EXCELLENT**

- Minimal, correct. `Chain` uses `slices.Backward` — idiomatic Go 1.26.
- Both middleware are framework-agnostic (`net/http` only).

### notify.go (37 lines)

**Assessment: GOOD**

- Duplicate of `Response.Notify*` methods as `HandlerOption` functions. This is intentional — provides both imperative and declarative notification paths.
- `NotificationEvent` is a mutable global. Same concern as `LoginRedirect`.

### Test Files

**Assessment: GOOD — 92.8% coverage, comprehensive**

- Tests are well-structured using Ginkgo BDD style.
- `coverage_test.go` (534 lines) specifically targets uncovered paths. Well done.
- 102 lint warnings in tests — mostly exhaustruct (test configs intentionally partial), unused parameters (lambda args), and dot-imports (Ginkgo convention). These are acceptable in test code.
- Test duplication in notification tests (3x similar boilerplate in coverage_test.go) — could use table-driven tests.

## Split Brain Analysis

**No split brains detected.** All types are defined in exactly one place. The notification pattern appears in both `notify.go` (HandlerOptions) and `response.go` (Response methods), but these are intentionally different API surfaces, not duplicated definitions.

## Type Safety Assessment

| Concern                                | Status                                                    |
| -------------------------------------- | --------------------------------------------------------- |
| Generic decoders (DecodeJSON[T], etc.) | GOOD — compile-time type safety                           |
| TemplComponent duck-typing             | GOOD — matches templ without import                       |
| handlerConfig field access             | GOOD — unexported struct, only modified via HandlerOption |
| Error sentinels                        | GOOD — typed errors with Is() checking                    |
| Config validation                      | GOOD — New() validates at least one dispatcher            |

## Architecture Smells

1. **Mutable globals**: `LoginRedirect` and `NotificationEvent` are package-level vars. `Config.LoginRedirect` mutates the global in `New()` — race-prone if multiple Apps are created concurrently.
2. **Dead code**: Empty `else if` block in `handleCommandDispatch:44`.
3. **Cyclomatic complexity**: `handleCommandDispatch` at 11 — just over the 10 threshold.
4. **XSS surface**: Error messages written directly to response without sanitization.

## What's Missing (Priority Order)

1. No request validation middleware/schema enforcement
2. No logging/observability hooks
3. No request ID/correlation ID propagation
4. No timeout/cancellation context propagation to CQRS dispatch
5. No built-in JSON error response format (only plain text)
6. No metrics/hooks for monitoring dispatch latency
