# Features — cqrs-htmx

**Updated:** 2026-05-27 | **Source:** All .go files analyzed

## Root Module

### Core

| #   | Feature          | Status           | Description                                                                                                                                                 |
| --- | ---------------- | ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | App Builder      | FULLY_FUNCTIONAL | `New(Config)` creates App with command/query dispatchers, enforcer, error handler. `MustNew(cfg)` panics on error. Validates at least one dispatcher.       |
| 2   | Command Dispatch | FULLY_FUNCTIONAL | `app.Command(type, opts...)` → HTTP handler. Decodes, dispatches, applies HTMX response. 204 on success by default.                                         |
| 3   | Query Dispatch   | FULLY_FUNCTIONAL | `app.Query(type, opts...)` → HTTP handler. Decodes, dispatches, renders result. 204 when no renderer.                                                       |
| 4   | Handler Options  | FULLY_FUNCTIONAL | Decoders, renderers, auth, validation, notifications, `Redirect`, `Trigger`, `PushURL`, `RequireMethod`, `WithSuccessStatus`, `OnError`, `WithMaxBodySize`. |
| 5   | Health Check     | FULLY_FUNCTIONAL | `App.HealthHandler()` → 200/503 JSON for load balancer probes.                                                                                              |

### Decoding

| #   | Feature          | Status           | Description                                                                                 |
| --- | ---------------- | ---------------- | ------------------------------------------------------------------------------------------- |
| 6   | JSON Decoding    | FULLY_FUNCTIONAL | `DecodeJSON[T]` / `DecodeJSONQuery[T]` with mapper. Invalid JSON → 400.                     |
| 7   | Form Decoding    | FULLY_FUNCTIONAL | `DecodeForm[T]` / `DecodeFormQuery[T]` via JSON round-trip. Parse errors → 400.             |
| 8   | Body Size Limits | FULLY_FUNCTIONAL | `DefaultMaxBodySize` (10 MB). Per-App `Config.MaxBodySize` + per-handler `WithMaxBodySize`. |

### Rendering

| #   | Feature           | Status           | Description                                                                                                 |
| --- | ----------------- | ---------------- | ----------------------------------------------------------------------------------------------------------- |
| 9   | Custom Render     | FULLY_FUNCTIONAL | `Render(fn)` HandlerOption for arbitrary response writing.                                                  |
| 10  | Templ Integration | FULLY_FUNCTIONAL | `RenderTempl(component)` and `RenderTemplResult[T](mapper)`. Duck-typed `TemplComponent` — no templ import. |
| 11  | JSON Rendering    | FULLY_FUNCTIONAL | `RenderJSON[T]()` renders query result as JSON 200. `RenderJSONStatus[T](status)` for custom status.        |

### HTMX

| #   | Feature               | Status           | Description                                                                                                                           |
| --- | --------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| 12  | HTMX Request Context  | FULLY_FUNCTIONAL | `HTMXMiddleware` parses all `HX-*` headers into context. `HTMXRequest` struct with all accessors.                                     |
| 13  | HTMX Response Builder | FULLY_FUNCTIONAL | Fluent `Response`: `PushURL`, `ReplaceURL`, `Redirect`, `Refresh`, `Reswap`, `Retarget`, `Reselect`, `Trigger*`. HTMX-aware redirect. |
| 14  | Notifications         | FULLY_FUNCTIONAL | `NotifySuccess/Error/Warning/Info` as HandlerOptions and Response methods. `NotifyWithEvent` builder.                                 |
| 15  | Swap Strategies       | FULLY_FUNCTIONAL | All 8 HTMX swap strategies as typed constants.                                                                                        |
| 16  | Header Constants      | FULLY_FUNCTIONAL | All HTMX headers are unexported constants. `HeaderTrue` exported for consumers.                                                       |

### Auth & Security

| #   | Feature              | Status           | Description                                                                                                                                                                            |
| --- | -------------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 17  | Casbin Authorization | FULLY_FUNCTIONAL | `Authorize(resource, action)`, `RequireAuth()`, `Enforce()`, `AuthorizeMiddleware`. `Enforcer` interface enables mocks.                                                                |
| 18  | CSRF Protection      | FULLY_FUNCTIONAL | `CSRFMiddleware` using justinas/nosurf. `CSRFProtect` per-handler. Template helpers: `CSRFTokenHTMLMeta`, `CSRFTokenHXHeaders`, `CSRFTokenFormField`. Custom header/field translation. |
| 19  | Security Headers     | FULLY_FUNCTIONAL | `SecurityHeadersMiddleware` with configurable CSP, HSTS, X-Frame-Options, etc. `RecommendedCSP`/`RecommendedHSTS` constants.                                                           |
| 20  | Rate Limiting        | FULLY_FUNCTIONAL | `RateLimiterMiddleware` with token-bucket per key. Min-heap O(log n) eviction. `MaxKeys` cap. `Retry-After` header. `ActiveKeys()` monitoring.                                         |
| 21  | Panic Recovery       | FULLY_FUNCTIONAL | `RecoveryMiddleware` recovers panics, logs stack trace, writes 500. `App.RecoverHandler()` uses App's ErrorHandler. Re-raises `http.ErrAbortHandler`.                                  |

### Context & Identity

| #   | Feature        | Status           | Description                                                                                                                                                                        |
| --- | -------------- | ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 22  | User Identity  | FULLY_FUNCTIONAL | `UserIDExtractor` → context → event metadata. `WithUserID`/`UserIDFromContext` (branded `id.UserID`). `IsAuthenticated(r)` helper. Dedup: handlers skip if middleware already set. |
| 23  | Correlation ID | FULLY_FUNCTIONAL | `WithCorrelationID`/`CorrelationIDFromContext` (branded `id.CorrelationID`). Auto-extracted from `X-Correlation-ID`. Propagated to event metadata.                                 |
| 24  | Request ID     | FULLY_FUNCTIONAL | `RequestID` (branded `id.RequestID`). `NewRequestID`/`ParseRequestID`/`MustParseRequestID`. Propagated to event metadata and `X-Request-ID` response header.                       |

### Error Handling

| #   | Feature               | Status           | Description                                                                                                      |
| --- | --------------------- | ---------------- | ---------------------------------------------------------------------------------------------------------------- |
| 25  | Error Classification  | FULLY_FUNCTIONAL | go-error-family registers sentinels. `MapError` maps families → HTTP status. Custom `ErrorHandler` support.      |
| 26  | Default Error Handler | FULLY_FUNCTIONAL | Plain text. HTMX auth → HX-Redirect. Per-App `LoginRedirect`. `text/plain` prevents XSS.                         |
| 27  | JSON Error Handler    | FULLY_FUNCTIONAL | `JSONErrorHandlerWithRedirect` writes `{error, status}`. Includes `request_id` when available.                   |
| 28  | Request ID in Errors  | FULLY_FUNCTIONAL | `Config.IncludeRequestIDInErrors` auto-selects request-ID-aware handler. Plain-text prefix: `[request_id: RID]`. |

### Middleware & Observability

| #   | Feature            | Status           | Description                                                                                                                      |
| --- | ------------------ | ---------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| 29  | Middleware Chain   | FULLY_FUNCTIONAL | `Chain(mw1, mw2, ...)` composes left-to-right.                                                                                   |
| 30  | Context Enrichment | FULLY_FUNCTIONAL | `ContextEnrichmentMiddleware` extracts User ID, Correlation ID, Request ID from headers. Sets `X-Request-ID` response header.    |
| 31  | Request Logging    | FULLY_FUNCTIONAL | `RequestLogging(formatter, writer)`. `DefaultLogFormatter` and `JSONLogFormatter`. Captures method, path, status, duration, IDs. |
| 32  | Lifecycle Hooks    | FULLY_FUNCTIONAL | `BeforeDispatchHook(ctx, r)` / `AfterDispatchHook(ctx, r, err)` on Config.                                                       |
| 33  | Timeout            | FULLY_FUNCTIONAL | `Config.Timeout` wraps dispatch with `context.WithTimeout`. Dispatch-only, not decode/auth.                                      |

### Convenience

| #   | Feature                | Status           | Description                                                                                           |
| --- | ---------------------- | ---------------- | ----------------------------------------------------------------------------------------------------- |
| 34  | ~~Catalog Entries~~    | REMOVED          | Removed in go-cqrs-lite v2.0.0 — `dispatcher.HandlerMeta`/`CatalogDispatcher` deleted upstream.       |
| 35  | HasCommands/HasQueries | FULLY_FUNCTIONAL | Report dispatcher availability.                                                                       |
| 36  | Request Validation     | FULLY_FUNCTIONAL | `ValidateCommand(validator)` / `ValidateQuery(validator)` wrap decoders. `ErrValidationFailed` → 400. |

---

## usermgmt Submodule

| #   | Feature            | Status           | Description                                                                                                                                                     |
| --- | ------------------ | ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 37  | User Service       | FULLY_FUNCTIONAL | Register, Login, Logout, Authenticate, ChangePassword, UpdateRoles. `context.Context` first param. Compensating transactions on Register rollback.              |
| 37b | Domain Model       | FULLY_FUNCTIONAL | Rich `User` entity: `SetRoles`, `ChangePassword`, `SetEmail`, `SetDisplayName`, `AddRole`, `RemoveRole`, `IsPasswordSet`. No direct field mutations in service. |
| 37c | Domain Events      | FULLY_FUNCTIONAL | Optional `EventHandler` callback. Emits `UserRegisteredEvent`, `UserLoggedInEvent`, `PasswordChangedEvent`, `RolesUpdatedEvent`. Panic-safe.                    |
| 38  | Branded UserID     | FULLY_FUNCTIONAL | `UserID = brandid.ID[userBrand, string]` via go-branded-id. `.String()` at Casbin boundaries. `NewUserID(s)` constructor.                                       |
| 39  | RBAC Authorization | FULLY_FUNCTIONAL | Casbin RBAC with domains. `AsEnforcer()` bridge to parent `Enforcer` interface. `ImplicitRoles`, `ImplicitPermissions`, `Policies`.                             |
| 40  | In-Memory Stores   | FULLY_FUNCTIONAL | `InMemoryUserStore` (email index, atomic Create, `Count()`). `InMemorySessionStore` (TTL, `EvictExpired()`, `Count()`). Both accept `context.Context`.          |
| 41  | Account Lockout    | FULLY_FUNCTIONAL | Configurable max attempts + duration. `ErrAccountLocked` → 429. `EvictStale()` for periodic cleanup.                                                            |
| 42  | HTTP Handlers      | FULLY_FUNCTIONAL | `AuthHandlers` with session cookies. `SessionMiddleware` (cookie + bearer). Configurable timeout, `*bool` Secure (nil defaults to true), cookie name.           |
| 43  | Input Validation   | FULLY_FUNCTIONAL | `RegisterRequest.Validate()` and `LoginRequest.Validate()`. Email format, password 8-128 chars, required fields. Pointer receivers persist trimmed values.      |

---

### Real-Time (SSE & WebSocket)

| #   | Feature            | Status           | Description                                                                                                                                        |
| --- | ------------------ | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| 44  | SSE Event Writer   | FULLY_FUNCTIONAL | `SSEEvent` struct, `WriteSSEEvent(w, event)` — properly formatted SSE events (event/data/id/retry, multi-line, CRLF normalization).                |
| 45  | SSE Stream         | FULLY_FUNCTIONAL | `SSEStream` manages a single SSE connection. Correct headers (text/event-stream, no-cache, keep-alive), flush after send, context-aware lifecycle. |
| 46  | SSE Broadcaster    | FULLY_FUNCTIONAL | `Broadcaster` — thread-safe fan-out. Subscribe/Unsubscribe with buffered channels (64). Non-blocking broadcast drops to slow consumers.            |
| 47  | WebSocket Message  | FULLY_FUNCTIONAL | `WSMessage`, `ParseWSMessage` — parses HTMX WebSocket JSON (form fields + HEADERS). `StringBody` helper for typed field access.                    |
| 48  | WebSocket OOB HTML | FULLY_FUNCTIONAL | `WSOOBHTML(id, html, strategy)` — wraps HTML with hx-swap-oob attributes for OOB swap. Uses `SwapStrategy` type. Passthrough for custom markup.    |

### Not Planned

| Feature                 | Status      | Reason                                                                                      |
| ----------------------- | ----------- | ------------------------------------------------------------------------------------------- |
| WebSocket upgrade logic | NOT_PLANNED | Consumers choose their own WebSocket library (gorilla, coder, etc.). Protocol helpers only. |

---

## Metrics

| Metric         | Root  | usermgmt |
| -------------- | ----- | -------- |
| Coverage       | 96.9% | 91.1%    |
| Lint issues    | 0     | 0        |
| Prod files     | 19    | 9        |
| Test files     | 23    | 11       |
| Benchmarks     | 18+   | 3        |
| Godoc examples | 10+   | 1        |
