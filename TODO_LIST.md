# TODO List — cqrs-htmx

**Date:** 2026-05-07 | **Source:** Self-review session, full codebase audit

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
- [ ] **Verify XSS safety in DefaultErrorHandler** — `html.EscapeString` on `text/plain` Content-Type is arguably unnecessary (distorts `foo < bar`). Decide: remove it or change Content-Type to text/html.

## P1 — Code Quality

- [x] **Extract `"true"` string constant** — `headerTrue` constant at htmx.go:35, used in all production code
- [x] **Add doc comment to SwapStrategy const block** — htmx.go:40
- [x] **Deduplicate notification test boilerplate** — `testNotificationTrigger` helper + `notifyOption`/`triggerNotification` private helpers
- [x] **Consolidate duplicate test types** — `mockTemplComponent`, `deleteUserCmd`, `listUsersQuery` → use `bdd*` types
- [x] **Extract helper functions** — `hasNoResponse()`, `hasMinimalResponse()`, `decodeJSONBody`, `decodeRequest`, `decodeFormBody`, `notifyOption`, `triggerNotification`
- [ ] **Export `HeaderTrue` or provide test helper** — Tests still hardcode `"true"` (34 occurrences); `headerTrue` is unexported

## P2 — Architecture Improvements

- [ ] **Move remaining mutable globals to per-App config** — `DefaultNotificationEvent` is still a package-level mutable var. Race condition risk with multiple Apps.
- [ ] **Extract Casbin interface** — `authz.go` uses `*casbin.Enforcer` concrete type. Define `Enforcer interface { Enforce(...) (bool, error) }` for testability.
- [ ] **Remove dead sentinels** — `ErrNoUserID` and `ErrRendererMissing` are exported but never returned by any code path. Breaking change — defer to v2.

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
- [x] **Notifications** — Both HandlerOptions and Response methods via shared helpers
- [x] **Error classification** — `sync.Once` registers all sentinels. `MapError` maps families to HTTP status
- [x] **Context propagation** — User ID → context → event metadata. Dedup in handlers
- [x] **Templ duck-typing** — `RenderTempl`, `RenderTemplResult[T]` without importing templ
- [x] **Middleware chain** — `Chain` composes middleware left-to-right
- [x] **92.6% test coverage** — 137 tests, race-safe
- [x] **0 lint issues** — golangci-lint clean
- [x] **All header constants consolidated** — No hardcoded HTMX header strings in production code
- [x] **Per-App LoginRedirect** — Config field now actually works via closure
