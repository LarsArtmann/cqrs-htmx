# TODO List — cqrs-htmx

**Date:** 2026-05-07 | **Updated:** 2026-05-19 | **Source:** Self-review session, full codebase audit + comprehensive 9-skill review

## Status Legend

- [ ] OPEN
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
- [x] **CSRF v1.7.3 plaintext HTTP detection** — `CSRFMiddleware` and `executeCSRFValidation` detect non-TLS requests and mark as plaintext via `csrf.PlaintextHTTPRequest`. Commit `9dcae5a`.

## P1 — Code Quality

- [x] **Extract `"true"` string constant** — `headerTrue` constant at htmx.go:35, used in all production code
- [x] **Add doc comment to SwapStrategy const block** — htmx.go:40
- [x] **Deduplicate notification test boilerplate** — `testNotificationTrigger` helper + `notifyOption`/`triggerNotification` private helpers
- [x] **Consolidate duplicate test types** — `mockTemplComponent`, `deleteUserCmd`, `listUsersQuery` → use `bdd*` types
- [x] **Extract helper functions** — `hasNoResponse()`, `hasMinimalResponse()`, `decodeJSONBody`, `decodeRequest`, `decodeFormBody`, `notifyOption`, `triggerNotification`
- [x] **Export `HeaderTrue`** — `HeaderTrue` exported constant in htmx.go. All tests use `cqrshtmx.HeaderTrue` instead of hardcoded `"true"`.
- [x] **Add error context to authorization errors** — `ErrForbidden`, `ErrEnforcerNil`, and `ErrUnauthorized` (with Authorize) now include resource/action context for debugging

## P2 — Architecture Improvements

- [x] **Move remaining mutable globals to per-App config** — `DefaultNotificationEvent` is now an unexported constant; exported var is deprecated. Added `NotifyWithEvent` builder for custom event names per-handler.
- [x] **Extract Casbin interface** — `authz.go` now defines `Enforcer interface { Enforce(...any) (bool, error) }`. `*casbin.Enforcer` satisfies it automatically. Enables mock/fake enforcers in consumer tests.
- [x] **Fix AuthorizeMiddleware ghost system** — Was bypassing App's error handler (raw `http.Error`). Now uses `DefaultErrorHandlerWithRedirect` for HTMX-aware auth error handling. Optional `loginRedirect` parameter.
- [x] **Remove dead sentinels** — `ErrNoUserID` and `ErrRendererMissing` removed. `ErrCommandsNil`, `ErrQueriesNil`, `ErrDecoderMissing` unexported. `DefaultNotificationEvent` removed.
- [x] **Extract shared test helpers** — `testing_test.go` with 11 helpers covering decoders, handlers, capture utilities. Reduced clone groups by 48% (27→14 at t=25).

## P3 — Feature Enhancements

- [x] **Add dispatch lifecycle hooks** — `BeforeDispatchHook` / `AfterDispatchHook` on `Config`. Tested in `hooks_test.go` with 9 specs.
- [x] **Add request validation middleware** — `ValidateCommand` / `ValidateQuery` HandlerOptions with `ErrValidationFailed` sentinel. Tested in `validation_test.go` with 6 specs.
- [x] **Add JSON error response option** — `JSONErrorHandler` writes `{error, status}` JSON. HTMX auth errors redirect via HX-Redirect.
- [x] **Add correlation ID propagation** — `WithCorrelationID` / `CorrelationIDFromContext` (branded `id.CorrelationID`). Auto-extracted from `X-Correlation-ID` header in `ContextEnrichmentMiddleware`.
- [x] **Fix CorrelationID event metadata pipeline** — `EventOptionsFromContext` now propagates branded `id.CorrelationID` into event metadata.
- [x] **Fix AuthorizeMiddleware identity parsing** — now uses branded `UserID` from context, falls back to extractor + `ParseUserID()` validation.
- [x] **Add Request Logging middleware** — `RequestLogging(formatter, writer)` with `DefaultLogFormatter`, captures correlation/user IDs.
- [x] **Add Rate Limiting middleware** — `RateLimiterMiddleware` with token-bucket per-key via `golang.org/x/time/rate`.
- [x] **Add timeout propagation** — `Config.Timeout time.Duration` wraps dispatch with `context.WithTimeout`. Zero/negative = no timeout. Tested in `timeout_test.go` with 7 specs.
- [x] **Add Request ID** — `RequestID` strongly-typed identifier, `NewRequestID`/`ParseRequestID`/`MustParseRequestID`, `WithRequestID`/`RequestIDFromContext`. Propagated into event metadata.
- [x] **Add Security Headers middleware** — `SecurityHeadersMiddleware` / `SecurityHeadersMiddlewareWithConfig` with configurable CSP, HSTS, etc.
- [x] **Add CSRF Protection** — `CSRFMiddleware`, `CSRFProtect`, template helpers, `CSRFResponseHeaderMiddleware`, `RotateCSRFToken`. Uses gorilla/csrf internally.

## P4 — Polish

- [x] **Add godoc examples** — 9 `Example*` functions in `example_test.go`: `New`, `App_Command`, `App_Query`, `NewResponse`, `SwapStrategy`, `HTMXMiddleware`, `RequestLogging`, `RateLimiterMiddleware`.
- [x] **Add benchmark tests** — 16 sub-benchmarks in `benchmark_test.go`: `MapError` (6), `ParseHTMXRequest` (2), `CommandDispatch`, `QueryDispatch`, etc.
- [x] **Create CONTRIBUTING.md** — Document lint config, test patterns, naming conventions
- [x] **Add `golangci-lint` to CI/CD** — GitHub Actions enforcement (.github/workflows/ci.yml)
- [x] **Document `.golangci.yml` decisions** — Inline comments explaining exclusions

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
- [x] **94.7% test coverage** — 289 specs, race-safe
- [x] **0 lint issues** — golangci-lint clean
- [x] **All header constants consolidated** — No hardcoded HTMX header strings in production code
- [x] **Per-App LoginRedirect** — Config field now actually works via closure
- [x] **Enforcer interface** — Enables testability without concrete Casbin dependency

## P5 — Open Items from 2026-05-19 Comprehensive Review

### Correctness

- [ ] **Fix context mismatch in `applyQueryResponse`** — `handler.go:124` uses `r.Context()` instead of the enriched `ctx` from timeout wrapping. Low risk (render rarely fails from timeout), but inconsistent with `handleCommandDispatch`.
- [ ] **Fix nil context in usermgmt tests** — `usermgmt/handler_test.go:314,322` uses `nil` context; `gopls SA1012` warns. Should use `context.TODO()`.

### Production Safety

- [ ] **Add max-keys cap to rate limiter** — `ratelimit.go` `perKeyLimiter.limiters` map grows unbounded under pathological key patterns. Add `MaxKeys int` to `RateLimiterConfig` with eviction when exceeded.
- [ ] **Cache CSRF Protect instance** — `csrf.go:413-444` `executeCSRFValidation` creates `csrf.Protect()` per request when `CSRFProtect` is used. Cache the Protect instance at handler creation time.

### Deduplication (High Impact)

- [ ] **Generic HTMX accessor** — Collapse 8 accessor functions in `htmx.go:96-170` to a single `htmxField[T]` generic helper.
- [ ] **Generic decoder** — Collapse `DecodeJSON`/`DecodeJSONQuery`/`DecodeForm`/`DecodeFormQuery` in `options.go:66-107` to a single generic decoder.
- [ ] **Generic validation** — Collapse `ValidateCommand`/`ValidateQuery` in `options.go:201-250` to a single generic validation wrapper.
- [ ] **Unified notification implementation** — Merge `notifyOption` and `triggerNotification` in `notify.go`+`response.go` to a single implementation.

### Deduplication (Medium Impact)

- [ ] **Shared logging context extraction** — Extract `appendContextAttrs(r)` helper from `logging.go` (3 blocks).
- [ ] **Shared error handler core** — Extract `handleErrorWithRedirect(w, r, err, redirect, writeBody)` from `errors.go:127-183`.
- [ ] **Generic ID helpers** — Collapse `ParseUserID`/`ParseCorrelationID`/`ParseRequestID` to a `parseID[T]` generic.

### File Organization

- [ ] **Split `csrf.go` (445 lines)** — Move template helpers (`CSRFTokenHTMLMeta`, `CSRFTokenHXHeaders`, `CSRFTokenFormField`, `RotateCSRFToken`) to `csrf_helpers.go`.
