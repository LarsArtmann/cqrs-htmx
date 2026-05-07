# TODO List — cqrs-htmx

**Date:** 2026-05-07 | **Source:** Code review, architecture review, quality scan, features audit, all .md docs

## Status Legend

- [ ] TODO
- [x] DONE
- [~] PARTIALLY DONE
- [-] NOT APPLICABLE

## Files Read

- [x] AGENTS.md
- [x] README.md
- [x] CHANGELOG.md
- [x] app.go
- [x] handler.go
- [x] options.go
- [x] response.go
- [x] authz.go
- [x] context.go
- [x] errors.go
- [x] htmx.go
- [x] middleware.go
- [x] notify.go
- [x] suite_test.go
- [x] app_test.go
- [x] htmx_test.go
- [x] middleware_test.go
- [x] context_test.go
- [x] errors_test.go
- [x] integration_test.go
- [x] coverage_test.go
- [x] .golangci.yml
- [x] go.mod

---

## P0 — Security & Correctness

- [ ] **Fix XSS in DefaultErrorHandler** — `errors.go:97` writes `err.Error()` unsanitized to response body. Sanitize or HTML-escape error messages before writing. (Code Quality Scan #1, Full Code Review)
- [x] **Remove dead empty-block in handleCommandDispatch** — Fixed in commit 3ad5efc (handler.go)
- [x] **Wrap decodeFormValues errors** — Fixed + fixed nil-wrapping bug (options.go)

## P1 — Code Quality

- [x] **Reduce handleCommandDispatch complexity to ≤10** — Fixed in commit 3ad5efc (extracted executePreDispatchChecks + applyCommandResponse)
- [ ] **Extract `"true"` string constant** — `htmx.go:71` has 8 occurrences of `"true"`. Extract to `const headerValueTrue = "true"`. (Code Quality Scan #7, goconst)
- [ ] **Add doc comment to SwapStrategy const block** — `htmx.go:39` missing exported comment. (Code Quality Scan #8, revive)
- [ ] **Add nolint directives for intentional patterns** — `app.go:131` exhaustruct, `errors.go:10` gochecknoinits. These are intentional; add `//nolint:` with reason. (Lint Hygiene)
- [ ] **Deduplicate notification test boilerplate** — `coverage_test.go:429-505` has 3x identical test structure for NotifyError/Warning/Info. Extract table-driven test. (Code Quality Scan #12, dupl)

## P2 — Architecture Improvements

- [ ] **Move LoginRedirect to per-App config** — `errors.go:82` is a mutable global. `Config.LoginRedirect` mutates it in `New()`, causing race conditions with multiple Apps. Store on App and pass through handlerConfig. (Architecture Review #1, Full Code Review)
- [ ] **Move NotificationEvent to per-App config** — `notify.go:5` is a mutable global. Same race concern as LoginRedirect. Store on App, pass through handlerConfig. (Architecture Review #1)
- [ ] **Extract Casbin interface** — `authz.go` uses `*casbin.Enforcer` concrete type. Define `Enforcer interface { Enforce(...) (bool, error) }` for testability. (Architecture Review #2)

## P3 — BDD Tests (User-Facing Scenarios)

- [x] **BDD: Consumer creates a command handler with all features** — Done (bdd_test.go)
- [x] **BDD: Consumer queries data with templ rendering** — Done (bdd_test.go)
- [x] **BDD: Consumer handles authentication errors gracefully** — Done (bdd_test.go)
- [x] **BDD: Consumer uses the Response builder for custom responses** — Done (bdd_test.go)

## P4 — Feature Enhancements

- [ ] **Add dispatch lifecycle hooks** — `OnBeforeDispatch` / `OnAfterDispatch` options for logging, metrics, tracing. Non-breaking addition. (Architecture Review #3)
- [ ] **Add request validation middleware** — Optional schema validation in the decode pipeline. (Full Code Review)
- [ ] **Add observability/logging hooks** — No logging or metrics middleware exists. Add hooks for monitoring. (Architecture Review)
- [ ] **Add correlation ID propagation** — No request/correlation ID in context. Add `WithCorrelationID` / `CorrelationIDFromContext`. (Full Code Review)
- [ ] **Add JSON error response option** — DefaultErrorHandler only returns plain text. Add JSON error format option. (Features Audit)
- [ ] **Add timeout propagation** — Library doesn't set deadlines on context. Consumers must handle externally. (Architecture Review)

## Already Done

- [x] **App builder with validation** — `New(Config)` validates at least one dispatcher. Tests pass. (app.go:37)
- [x] **Generic decoders** — `DecodeJSON[T]`, `DecodeJSONQuery[T]`, `DecodeForm[T]` with error handling. (options.go)
- [x] **Casbin authorization** — `Authorize`, `RequireAuth`, `Enforce`, `AuthorizeMiddleware`. All tested. (authz.go)
- [x] **HTMX request context** — `HTMXMiddleware`, `HTMXRequest` struct, all accessors with fallback. (htmx.go)
- [x] **HTMX response builder** — Fluent `Response` with all HTMX headers supported. (response.go)
- [x] **Notifications** — Both HandlerOptions and Response methods. Standard `{level, message}` pattern. (notify.go, response.go)
- [x] **Error classification** — `init()` registers all sentinels. `MapError` maps families to HTTP status. (errors.go)
- [x] **Context propagation** — User ID → context → event metadata. Dedup in handlers. (context.go)
- [x] **Templ duck-typing** — `RenderTempl`, `RenderTemplResult[T]` without importing templ. (options.go)
- [x] **Middleware chain** — `Chain` composes middleware left-to-right. (middleware.go)
- [x] **92.8% test coverage** — 121 tests, race-safe. (all test files)
