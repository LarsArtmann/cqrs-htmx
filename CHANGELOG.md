# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [2.4.0] - 2026-06-17

### Changed

- **go-cqrs-lite v2.4.0**: All modules upgraded to the latest per-module tags (command, event, id, query, codec, decider, dispatcher, memory, projection, snapshot, otel).
- **go-error-family v0.4.0**: Upgraded with idiomatic `event.Wrapf` / `event.WrapTransient` replacing all `NewTransient(fmt.Sprintf(...)).WithCause(err)` patterns.
- **go-branded-id v0.3.1**: Latest branded ID library.
- **ginkgo v2.31.0 + gomega v1.42.0**: Latest BDD test framework.
- **Lint: 0 issues**: Fixed all 16 pre-existing lint warnings (errorlint, exhaustruct, contextcheck false positives).
- **Email validation consolidated**: `ImportUser.Validate()` now delegates to `ParseEmail()` instead of reimplementing identical logic.

### Security

- **Import/export authorization**: Endpoints now require admin role by default. Previously any authenticated user could export all user data or create accounts.
- **TOTP disable protection**: `DisableTOTP` now requires a valid code, preventing MFA stripping if a session is hijacked.
- **Per-endpoint rate limiting**: `HandlerConfig.ImportRateLimit`, `TOTPRateLimit`, `VerificationRateLimit` for per-IP rate limiting.

### Changed

- **TOTP library**: Replaced hand-rolled HMAC-SHA1 with `pquerna/otp/totp`.
- **DisableTOTP signature**: Now takes a code parameter.
- **ExportFormat → UserDataFormat**: Honest naming for dual import/export use.
- **Email value type**: `ParseEmail`/`MustParseEmail` for centralized validation. Used in `ExportUser`.
- **withTimeout helper**: Eliminates duplicated context-timeout boilerplate.

### Added

- **TOTP multi-factor authentication**: RFC 6238 TOTP via `pquerna/otp/totp` library. Events: `TOTPEnabled`/`TOTPDisabled`. Two-phase setup: `EnableTOTP` → pending secret → `VerifyTOTPSetup` confirms. Generates `otpauth://` URIs for QR codes. `Service.VerifyTOTP` validates codes as second factor. `DisableTOTP` requires a valid code to prevent MFA stripping.
- **Email verification flow**: Token-based email confirmation with `EmailVerified` event. `SendVerificationEmail` generates token (configurable TTL, default 24h) with optional SMTP callback. `VerifyEmail` consumes single-use token. Email change resets verification status.
- **User import/export**: `Service.ImportUsersFromJSON`/`ImportUsersFromCSV` for batch user creation (skips existing emails). Input validation via `ImportUser.Validate()`. `Service.ExportUsersToJSON`/`ExportUsersToCSV` for data export. Admin-only by default via `ImportExportAuthorizer`. Per-IP rate limiting via `HandlerConfig.ImportRateLimit`.
- **Virtual authenticator integration test**: Full WebAuthn registration → login ceremony using W3C spec test vectors. Tests: successful full flow, registration replay rejection, login with wrong challenge.
- **SQL event store**: `SQLEventStore` implements `event.Store` + `event.Journal` for Postgres/SQLite/MySQL. Auto-migrates schema, supports optimistic concurrency, parameterized queries per dialect.
- **Audit log projection**: `AuditLog` records all user events as `AuditEntry` structs. Queryable by aggregate ID, recent N, or total count. Optional via `ServiceConfig.AuditLog`.
- **Rate-limited registration**: Per-IP fixed-window rate limiting on the registration endpoint. Configurable via `HandlerConfig.RegistrationRateLimitConfig`.
- **Session rotation on privilege change**: `UpdateRoles` deletes all user sessions after role change, forcing re-authentication.
- **Credential listing pagination**: `GET /auth/credentials?page=N&page_size=N` with configurable page size (default 20, max 100).
- **Structured logging in WebAuthn ceremonies**: All 4 ceremony methods log at debug/info/warn with user_id, email, error context.
- **CI pipeline (GitHub Actions)**: Multi-module CI with separate lint jobs for root and usermgmt, race detector, `GONOSUMCHECK` env vars, concurrency group.
- **doc.go**: Comprehensive root package documentation with Quick Start, middleware stack, response builder, error mapping, SSE, and submodule reference.
- **Property-based testing**: 8 rapid-based property tests for `foldUser` invariants.
- **Benchmarks**: 5 benchmarks for `foldUser`, `ReadModel.Handle`, `BeginRegistration`, `BeginLogin`.
- **Fuzz tests**: 3 fuzz tests for WebAuthn ceremony inputs and credential ID decoding.
- **Coverage tests**: 15 tests covering error paths, schema version, RegisterCommands, DefaultEventSourcedSetup, writeJSON, HTTP handlers.

- **AccountLockout wired into WebAuthn login**: `BeginLogin` checks lockout status; `FinishLogin` records failures and resets on success.
- **Credential management HTTP endpoints**: `GET /auth/credentials` (list), `DELETE /auth/credentials/{id}` (remove by base64url ID).
- **WebAuthn session proactive eviction**: Background goroutine (`webauthnEvictionInterval`) periodically removes expired challenge sessions. `Service.Stop()` terminates the goroutine.
- **CasbinProjection subscribes to CredentialAdded/Removed**: Ensures projection ordering without affecting policies.
- **`Service.Stop()` method**: Gracefully shuts down background resources (WebAuthn eviction goroutine). Safe to call multiple times.
- **`Service.Authz()` accessor**: Returns the underlying `*Authz` for direct policy queries.
- **Service source propagation**: `Config.ServiceName` (string) and `App.EventOptions(ctx)` inject `event.WithSource` into event options. `EventOptionsFromContextWithSource(ctx, name)` is the free-function equivalent for callers that don't hold an `*App`.
- **Typed-query examples in root**: `ExampleApp_Query_typedRegister` and `ExampleApp_Query_typedDispatch` demonstrate the `query.RegisterTyped[T]` / `query.DispatchTyped[T]` pattern crossing module boundaries.
- **`ExampleBroadcaster_BroadcastOnSuccessFunc`**: Documents the dynamic-event `AfterDispatchHook` factory.
- **`ExampleConfig_BeforeDispatch` / `ExampleChain` (tracing)**: Show timing/span integration via the lifecycle hooks; no OpenTelemetry dependency required.
- **Per-module nix apps**: `nix run .#test-root`, `nix run .#test-usermgmt`, `nix run .#test-integration`, `nix run .#build-datastar-demo` set `GOWORK=off` automatically — no more per-module `cd` dance.
- **UpdateTodo command in datastar-demo**: `UpdateTodoCmd` + `TodoUpdatedPayload` + `handleUpdateTodo` + `POST /api/todos/update` route + inline edit input in the UI. The read-model `Projector.Apply` now handles `TodoUpdated` and the broadcaster renders it as `todo_updated`.
- **`GetStatsQry` typed query in datastar-demo**: `Stats` struct + `GetStatsQry` + `renderStatsFromQuery` helper. Wired into `handleListTodos` and `handleEventStream` so the stats read goes through the typed query dispatcher (caching/authorization-friendly).
- **SSE replay endpoint in datastar-demo**: `GET /api/events/replay` reads `Last-Event-ID` and replays missed events from the in-memory event store.
- **`FuzzEventOptionsFromContext`**: Fuzzes `EventOptionsFromContext` with arbitrary combinations of (deadline, cancel, user ID, correlation ID, request ID) — verifies the bounded-options invariant (≤ 4 options).
- **Real-server integration tests**:
  - `TestRateLimiter_RealServer_AllowsThenBlocks` + `TestRateLimiter_RealServer_ConcurrentRequests`: end-to-end rate limiting over `httptest.NewServer` with a real `http.Client`.
  - `TestSSE_RealServer_ReconnectionWithLastEventID` + `TestSSE_RealServer_ReconnectionNoLastID`: end-to-end SSE reconnection with real `Last-Event-ID` header.
  - `TestTypedQueryDispatch_CrossModule` + `TestCrossModule_PaginationFlow`: cross-module CQRS + pagination.
- **Benchmarks**:
  - `BenchmarkDecodePagination` (8 URL shapes: no params, with page, with size, with both, with extras, invalid, zero, huge).
  - `BenchmarkCommandRegisterTypedVsRegister` (typed ≈ manual: 57.4 vs 56.9 ns/op).
  - `BenchmarkQueryDispatchTypedVsDispatch` (typed ≈ manual: 85.7 vs 81.0 ns/op).
- **usermgmt service transient-error context**: `withUserIDContext` helper annotates every `event.NewTransient` in the service with `user_id` context (no PII — only the user ID, not email/display name). New `transientErr(userID, msg, cause)` helper keeps service methods concise.
- **ADR 0005**: Documents the go-cqrs-lite v2.3.0 adoption decisions (IsZero validation, FromContext deadline propagation, RegisterTyped/DispatchTyped, replace-directive removal).

### Changed

- **Eliminated production panics**: `marshalPayload` and `aggIDFromUser` now return errors instead of panicking. All callers updated to handle errors gracefully.
- **RegisterCommands returns error**: Silently ignored command registration failures now propagate to `NewService`.
- **bus.Subscribe failures logged**: Event bridge subscription errors now logged at warn level instead of silently ignored.
- **Projection runner errors logged**: Background projection goroutine errors now logged at error level.
- **WebAuthn HTTP refactor**: Finish endpoints use query params (`?user_id=X`) instead of fragile double body read pattern.
- **Coverage**: 86.2% usermgmt, 96%+ root, zero 0% functions.

### Fixed

- **Goroutine leak**: All WebAuthn test services now call `t.Cleanup(svc.Stop)` to terminate eviction goroutines.
- **Stale password references**: Removed all `"password":"secret12"` from test helpers (passwordless migration).
- **gosec G101**: Credential event/command type constants now have proper nolint annotations.
- **wrapcheck**: `marshalPayload` and `Dispatch` errors are now properly wrapped.

## [2.3.0] - 2026-06-13

### Changed

- **go-cqrs-lite upgraded to v2.3.0** across all modules (root, usermgmt, integration_test, datastar-demo). Per-module tags now published — no go.work replace directives needed.
- **Empty command/query type validation**: `App.Command("")` and `App.Query("")` panic at handler registration time using `command.Type.IsZero()` / `query.Type.IsZero()` — fail-fast instead of silent empty dispatch.
- **Deadline propagation**: `EventOptionsFromContext` now propagates context deadline to event options via `event.FromContext(ctx)` when present. Downstream events inherit request timeouts.
- **Type-safe handlers in datastar-demo**: Refactored all 3 command handlers from manual type assertions to `command.RegisterTyped[T]` — eliminates `cmd.(*CreateTodoCmd)` boilerplate.
- **Local MustParse wrappers**: Reimplemented `MustParseUserID`, `MustParseCorrelationID`, `MustParseRequestID` locally (go-cqrs-lite removed `id.MustParse[T]`). Preserves API compatibility for consumers.
- **query.MustNew removed upstream**: Updated `datastar-demo` to use `query.New()` + error check.
- **Code deduplication**: Eliminated all clone groups at threshold 50 (industry standard). At threshold 30, remaining groups are all 2–7 line spans in test code. −104 lines net across 14 source files.
- **goconst warnings eliminated**: Extracted test constants in `sse_test.go` (`eventTodoCreated`, `eventUpdate`, `eventItem`, `dataFirst`) and `coverage_test.go` (`aliceName` usage). Added `goconst` exclusion for `example_test.go` (self-contained examples should not reference test constants).
- **nestif warning fixed**: Extracted `parseWSHeaders` helper from `ParseWSMessageInto` in `ws.go`, reducing nesting complexity from 7 to 1.
- **Test deduplication**: Deleted 6 duplicate ClientIP tests from `coverage_test.go` (already covered by `httputil_test.go`). Merged 3 `sanitizeRedirectURL` DescribeTables into one. Extracted `queryNamedResultHandler` helper for query-result fixtures.
- **`ws_test.go` coverage improved**: `ParseWSMessageInto[T]` from 86.7% → 93.3% by exercising the type-assertion failure path and non-string header values.
- **`usermgmt.UpdateRoles` refactor**: Extracted `transientErr` helper to keep the function under the 60-line `funlen` limit. Replaces inline `withUserIDContext(event.NewTransient(...).WithCause(err), userID)` boilerplate.

## [2.1.0] - 2026-06-08

### Added

- **SSE (Server-Sent Events) support**: Full SSE implementation for the HTMX SSE extension (`hx-ext="sse"`). `SSEStream` manages a single connection with correct headers, flush, and context-aware lifecycle. `SSEEvent` struct with `Event`, `Data`, `ID`, `Retry` fields. `WriteSSEEvent` for writing events to any `io.Writer`.
- **SSE Broadcaster**: Thread-safe fan-out `Broadcaster` with O(1) unsubscribe via channel identity, buffered channels (64), and non-blocking broadcast (drops to slow consumers). `SubscriberCount()` for monitoring.
- **SSE reconnection**: `LastEventIDFromRequest(r)` extracts `Last-Event-ID`. `SSEEventStore` interface and `ReplayEvents(stream, store, lastID)` for full SSE spec reconnection support.
- **SSE + CQRS bridge**: `BroadcastOnSuccess(event, data)` and `BroadcastOnSuccessFunc(fn)` — `AfterDispatchHook` factories that broadcast SSE events on successful command dispatch.
- **WebSocket message parser**: `ParseWSMessage(data)` parses HTMX WebSocket JSON into `WSMessage` with separated `Headers` and `Body` fields. `StringBody(key)` for typed string access.
- **Typed WebSocket parser**: `ParseWSMessageInto[T](data)` — generic typed parser that deserializes body into struct T while separating HEADERS. Compile-time safe.
- **WebSocket OOB HTML**: `WSOOBHTML(id, html, strategy)` wraps HTML with `hx-swap-oob` attributes for out-of-band swaps. Uses `SwapStrategy` type.
- **Pagination**: `DecodePagination(r)` extracts page/page_size from query params, delegates to `query.NewPagination` for defaults (page=1, page_size=20, max=100). `RenderPaginatedJSON[T]()` renders `query.PaginatedResult[T]` as JSON 200.
- **Embedded HTMX JavaScript**: `HTMXScriptHandler()` serves embedded HTMX v2.0.9 (minified, ~49KB) with `Content-Type`, long-lived `Cache-Control` (1 year, immutable), `ETag`, and `If-None-Match` support for 304 responses. `HTMXScriptTag(path)` generates `<script>` tags. `HTMXVersion()` returns `"2.0.9"`. Opt-in, zero CDN dependency.

### Changed

- **go-cqrs-lite upgraded to v2.2.0** across all modules. Adopts `query.Pagination` and `query.PaginatedResult[T]` from upstream.
- **httputil upgraded to v0.1.0** (→ v0.1.1 pending).
- Modernized Go idioms and normalized table formatting across codebase.

## [2.0.0] - 2026-05-27

### Changed

- **CSRF: replaced `gorilla/csrf` with `justinas/nosurf`**: Simpler API, no secret management required, no HMAC/securecookie. Token generation uses `crypto/rand` with origin validation instead. `CSRFConfig.Secret` field removed. `RotateCSRFToken` removed (nosurf rotates automatically). Custom header/field translation via `translateCSRFHeaders`.
- **Error handling: replaced `cockroachdb/errors` with `go-error-family`**: Error classification now uses `github.com/larsartmann/go-error-family` via `go-cqrs-lite/core/event`. `sync.Once` registers sentinels with `event.RegisterClassification`.
- **`ClientIP` delegated to `larsartmann/httputil`**: `httputil.ClientIP(r)` replaces the local implementation.
- **go-cqrs-lite/core upgraded to v1.5.1**: `CatalogEntry` → `HandlerMeta` for API compatibility.

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
