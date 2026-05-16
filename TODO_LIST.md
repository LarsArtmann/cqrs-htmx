# TODO List — cqrs-htmx

**Date:** 2026-05-07 | **Updated:** 2026-05-07 | **Source:** Self-review session, full codebase audit

## Status Legend

- [ ] TODO
- [x] DONE
- [~] PARTIALLY DONE
- [-] NOT APPLICABLE

---

## P0 — Security & Correctness

- [x] **Use headerTrue constant in Response.Refresh()** — Fixed in commit `98616cb` (response.go:56)
- [x] **Use headerRedirect constant in DefaultErrorHandlerWithRedirect** — Fixed in commit `21c22c9` (errors.go:110)
- [x] **Fix Config.LoginRedirect dead code** — Was stored but never read; now threads into per-App error handler closure. Commit `ab52b05`.
- [x] **Fix README compile-breaking example** — `cqrshtmx.LoginRedirect` didn't exist. Fixed in commit `1a44e8c`.
- [x] **Verify XSS safety in DefaultErrorHandler** — Removed `html.EscapeString` from `text/plain` responses. `text/plain` Content-Type prevents browser HTML rendering; escaping distorted error messages like "foo < bar". Added `//nolint:gosec` with explanation.

## P1 — Code Quality

- [x] **Extract `"true"` string constant** — `headerTrue` constant at htmx.go:35, used in all production code
- [x] **Add doc comment to SwapStrategy const block** — htmx.go:40
- [x] **Deduplicate notification test boilerplate** — `testNotificationTrigger` helper + `notifyOption`/`triggerNotification` private helpers
- [x] **Consolidate duplicate test types** — `mockTemplComponent`, `deleteUserCmd`, `listUsersQuery` → use `bdd*` types
- [x] **Extract helper functions** — `hasNoResponse()`, `hasMinimalResponse()`, `decodeJSONBody`, `decodeRequest`, `decodeFormBody`, `notifyOption`, `triggerNotification`
- [~] **Export `HeaderTrue` or provide test helper** — Tests still hardcode `"true"` (34 occurrences); `headerTrue` is unexported. Created `testing_test.go` with helpers but this item still open.
- [x] **Add error context to authorization errors** — `ErrForbidden`, `ErrEnforcerNil`, and `ErrUnauthorized` (with Authorize) now include resource/action context for debugging

## P2 — Architecture Improvements

- [x] **Move remaining mutable globals to per-App config** — `DefaultNotificationEvent` is now an unexported constant; exported var is deprecated. Added `NotifyWithEvent` builder for custom event names per-handler.
- [x] **Extract Casbin interface** — `authz.go` now defines `Enforcer interface { Enforce(...any) (bool, error) }`. `*casbin.Enforcer` satisfies it automatically. Enables mock/fake enforcers in consumer tests.
- [x] **Fix AuthorizeMiddleware ghost system** — Was bypassing App's error handler (raw `http.Error`). Now uses `DefaultErrorHandlerWithRedirect` for HTMX-aware auth error handling. Optional `loginRedirect` parameter.
- [~] **Remove dead sentinels** — `ErrNoUserID` and `ErrRendererMissing` are exported but never returned by any code path. Breaking change — defer to v2.
- [~] **Extract shared test helpers** — `testing_test.go` created with 11 helpers covering decoders, handlers, capture utilities. Reduced clone groups by 48% (27→14 at t=25).

## P3 — Feature Enhancements

- [ ] **Add dispatch lifecycle hooks** — `OnBeforeDispatch` / `OnAfterDispatch` for logging/metrics/tracing
- [ ] **Add request validation middleware** — Optional schema validation in decode pipeline
- [ ] **Add JSON error response option** — DefaultErrorHandler only returns plain text
- [ ] **Add correlation ID propagation** — `WithCorrelationID` / `CorrelationIDFromContext`
- [ ] **Add timeout propagation** — Library doesn't set deadlines on context

## P4 — Polish

- [ ] **Add godoc examples** — `SwapStrategy`, `Config`, `Response`, `HTMXRequest`
- [ ] **Create CONTRIBUTING.md** — Document lint config, test patterns, naming conventions
- [ ] **Add `golangci-lint` to CI/CD** — GitHub Actions enforcement
- [ ] **Add benchmark tests** — `MapError`, `parseHTMXRequest`, `HTMXMiddleware`
- [ ] **Document `.golangci.yml` decisions** — Inline comments explaining exclusions

## Already Done

- [x] **App builder with validation** — `New(Config)` validates at least one dispatcher
- [x] **Generic decoders** — `DecodeJSON[T]`, `DecodeJSONQuery[T]`, `DecodeForm[T]`, `DecodeFormQuery[T]`
- [x] **Casbin authorization** — `Authorize`, `RequireAuth`, `Enforce`, `AuthorizeMiddleware`
- [x] **HTMX request context** — `HTMXMiddleware`, `HTMXRequest` struct, all accessors with fallback
- [x] **HTMX response builder** — Fluent `Response` with all HTMX headers supported
- [x] **Notifications** — Both HandlerOptions and Response methods via shared helpers; `NotifyWithEvent` builder for custom events
- [x] **Error classification** — `sync.Once` registers all sentinels. `MapError` maps families to HTTP status
- [x] **Context propagation** — User ID → context → event metadata. Dedup in handlers
- [x] **Templ duck-typing** — `RenderTempl`, `RenderTemplResult[T]` without importing templ
- [x] **Middleware chain** — `Chain` composes middleware left-to-right
- [x] **95.7% test coverage** — 148 tests, race-safe
- [x] **0 lint issues** — golangci-lint clean
- [x] **All header constants consolidated** — No hardcoded HTMX header strings in production code
- [x] **Per-App LoginRedirect** — Config field now actually works via closure
- [x] **Enforcer interface** — Enables testability without concrete Casbin dependency
