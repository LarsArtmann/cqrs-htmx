# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased] - 2026-05-18

### Added

- **Panic recovery middleware**: `RecoveryMiddleware` recovers from panics, logs stack traces via `slog.ErrorContext`, and writes 500. `App.RecoveryMiddleware()` uses the App's configured `ErrorHandler` for consistent error formatting (JSON, redirects, request ID correlation). `http.ErrAbortHandler` is re-raised without recovery (Go net/http convention).
- **JSON result rendering**: `RenderJSON[T]()` renders query results as JSON with 200 OK. `RenderJSONStatus[T](status)` renders with a custom status code (e.g., 201 Created). Both include runtime type assertion for compile-time documentation.
- **Request ID in error responses**: `JSONErrorHandlerWithRedirect` now includes `"request_id"` field when a `RequestID` is present in context. `DefaultErrorHandlerWithRequestID` and `DefaultErrorHandlerWithRedirectAndRequestID` prefix plain-text errors with `[request_id: RID]`.
- **Config field**: `IncludeRequestIDInErrors` — when `true` and no custom `ErrorHandler` is set, the default handler automatically includes request IDs in error responses.
- `CorrelationID` type alias (`type CorrelationID = id.CorrelationID`) with `NewCorrelationID()`, `ParseCorrelationID()`, `MustParseCorrelationID()` helpers
- E2E test verifying event.NewEvent accepts options from EventOptionsFromContext
- `ContextEnrichmentMiddleware` now validates `X-Correlation-ID` as ULID; non-ULID values are silently dropped

### Changed

- **`WithCorrelationID` / `CorrelationIDFromContext`**: context now stores strongly-typed `id.CorrelationID` (ULID-backed branded type) instead of `string`. **Breaking change for consumers** passing raw strings — use `MustParseCorrelationID()` in tests, `NewCorrelationID()` to generate, or `ParseCorrelationID()` in production.
- `AuthorizeMiddleware` now prefers branded `UserID` from context over raw extractor string; falls back to extractor + `ParseUserID()` validation. Unparseable ULIDs now return 401 instead of passing raw strings to Casbin.
- Context keys are now empty-struct sentinel types (`userIDKey{}`, `correlationIDKey{}`, `htmxKey{}`) instead of string-based types — standard Go pattern for collision-free context values.

### Fixed

- `CorrelationIDFromContext` → `EventOptionsFromContext` → `event.WithCorrelationID()` pipe now fully wired with branded `id.CorrelationID` type
- Dead test: "returns error when casbin enforce fails" now actually tests failure (asserts error instead of asserting success)
- `EventOptionsFromContext` now propagates `CorrelationID` alongside `UserID` into event metadata

## [1.0.0] - 2026-05-16

### Added

- BDD test suite using Ginkgo/Gomega (`bdd_test.go`)
- `DecodeFormQuery` handler option for query parameter form decoding (symmetry with `DecodeForm`)
- `docs/` directory with architecture reviews, planning docs, and status reports
- `NewUserID()`, `ParseUserID()`, `MustParseUserID()` helpers; `type UserID = id.UserID` re-export
- `NotificationLevel` type with `LevelSuccess`, `LevelError`, `LevelWarning`, `LevelInfo` constants
- `JSONErrorHandlerWithRedirect` for JSON error responses with custom login redirect
- Dispatch lifecycle hooks: `BeforeDispatchHook` / `AfterDispatchHook` on `Config`
- Request validation: `ValidateCommand` / `ValidateQuery` HandlerOptions with `ErrValidationFailed`
- Correlation ID propagation: `WithCorrelationID` / `CorrelationIDFromContext`
- Timeout propagation: `Config.Timeout` wraps dispatch with `context.WithTimeout`
- Godoc examples (7 `Example*` functions)
- Benchmark tests (10 sub-benchmarks)
- `CONTRIBUTING.md` contribution guide
- GitHub Actions CI pipeline (build + test + lint + coverage gate)
- `authMode` typed enum for handler authorization (makes impossible states unrepresentable)

### Changed

- **`WithUserID` / `UserIDFromContext`**: context now stores strongly-typed `id.UserID` (ULID-backed branded type) instead of `string`. `UserIDExtractor` still returns `string`; middleware parses to `UserID`. **Breaking change for consumers** passing string literals or plain strings — use `MustParseUserID()` in tests, `ParseUserID()` in production.
- Extract helper functions (`hasNoResponse`, `hasMinimalResponse`, `decodeJSONBody`, `decodeRequest`, `decodeFormBody`, `notifyOption`, `triggerNotification`) to reduce duplication
- Extract notification helpers to dedicated `notify.go` module
- Consolidate duplicate test types across test files into shared helpers
- Remove local-path `replace` directives from `go.mod` — resolve from GitHub
- Notification levels now use `NotificationLevel` type instead of magic strings
- Remove `headerTrue` alias — use `HeaderTrue` everywhere
- `"X-Correlation-ID"` header extracted to `headerCorrelationID` constant
- `JSONErrorHandler` now delegates to `JSONErrorHandlerWithRedirect` with default redirect
- Authorization config consolidated from 4 fields (`authorize bool` + `requireAuth bool` + `resource` + `action`) to typed `authMode` enum + `resource` + `action`

### Fixed

- Use `headerRedirect` constant instead of hardcoded `"HX-Redirect"` string in `DefaultErrorHandlerWithRedirect`
- Thread `Config.LoginRedirect` into per-App error handler (was dead code — `New()` now creates a closure that captures the resolved loginRedirect)
- Use `headerTrue` constant in `Response.Refresh()` instead of hardcoded `"true"`
- Fix README compile-breaking example (`cqrshtmx.LoginRedirect` → `Config.LoginRedirect`)
- Fix error wrapping: `errors.Wrapf` with `%s` on sentinels → `fmt.Errorf("%w: ...")` throughout
- `Enforce(nil, ...)` error now includes all three fields (subject, resource, action) — was missing subject
- `JSONErrorHandler` now respects `Config.LoginRedirect` (was hardcoded to `/login`)

### Removed

- Remove dead `enrichContext()` no-op stub
- Remove redundant gocritic `disabled-checks` entries (`dupImport`, `octalLiteral`, `whyNoLint` — already disabled by default)
- Remove unused `io` and `event` imports from test files
- Remove dead sentinels `ErrNoUserID` and `ErrRendererMissing` (exported but never returned by any code path)
- Remove deprecated `DefaultNotificationEvent` var (race risk, unexported constant used internally)
- Unexport internal sentinels: `ErrCommandsNil` → `errCommandsNil`, `ErrQueriesNil` → `errQueriesNil`, `ErrDecoderMissing` → `errDecoderMissing`

## [0.2.0] - 2026-05-07

### Added

- Eliminate all 103 golangci-lint issues → 0 issues, project is lint-clean
- `.golangci.yml` v2 format with proper exclusion rules
- Comprehensive test coverage at 93.5% (138 specs)

## [0.1.0] - 2026-05-04

### Added

- **App builder**: `App` struct with `Config`, `Command()`, `Query()`, per-App `ErrorHandler` and `LoginRedirect`
- **CQRS dispatch**: `handleCommandDispatch()` and `handleQueryDispatch()` with automatic error handling
- **Handler options**: `DecodeJSON`, `DecodeForm`, `Render`, `RenderTempl`, `RenderTemplResult`, `Authorize`, `Enforce`, `UserIDExtractor`
- **HTMX response builder**: Fluent API with `Response` struct — `StatusCode()`, `Header()`, `Redirect()`, `Refresh()`, `Retarget()`, `Reswap()`, `Trigger()`, `TriggerAfterSettle()`, `TriggerAfterSwap()`
- **HTMX middleware**: `HTMXMiddleware` parses `HX-*` headers once, stores `HTMXRequest` in context
- **HTMX context accessors**: `IsHTMXRequest()`, `GetHTMXPrompt()`, `GetHTMXTarget()`, `GetHTMXTrigger()`, `GetHTMXTriggerName()`, `RenderPartial()`
- **Notification system**: `NotifySuccess`, `NotifyError`, `NotifyWarning`, `NotifyInfo` — standard `{level, message}` trigger pattern via `notify.go`
- **Casbin authorization**: `Authorize()`, `Enforce()`, `AuthorizeMiddleware()` using `casbin/casbin/v3`
- **Context enrichment**: `ContextEnrichmentMiddleware` + `UserIDExtractor` → context → event metadata
- **Error classification**: CQRS error → HTTP status mapping with `RegisterClassification`, sentinel errors (`ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrBadRequest`, `ErrConflict`, `ErrInternal`), and `LoginRedirect` support
- **templ integration**: `TemplComponent` duck-typed interface (no `a-h/templ` import dependency) with `RenderTempl` and `RenderTemplResult` options
- **Middleware chain**: `Chain()` utility for composing `net/http` middleware
- **Git Town integration**: `git-town.toml` configuration

### Changed

- Deduplicate `UserIDExtractor` calls — handlers check context first, skip if middleware already set user ID
