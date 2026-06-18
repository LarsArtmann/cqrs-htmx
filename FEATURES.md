# Features — cqrs-htmx

**Updated:** 2026-06-18 | **Source:** All .go files analyzed

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

| #   | Feature               | Status           | Description                                                                                                                            |
| --- | --------------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| 12  | HTMX Request Context  | FULLY_FUNCTIONAL | `HTMXMiddleware` parses all `HX-*` headers into context. `HTMXRequest` struct with all accessors.                                      |
| 13  | HTMX Response Builder | FULLY_FUNCTIONAL | Fluent `Response`: `PushURL`, `ReplaceURL`, `Redirect`, `Refresh`, `Reswap`, `Retarget`, `Reselect`, `Trigger*`. HTMX-aware redirect.  |
| 14  | Notifications         | FULLY_FUNCTIONAL | `NotifySuccess/Error/Warning/Info` as HandlerOptions and Response methods. `NotifyWithEvent` builder.                                  |
| 15  | Swap Strategies       | FULLY_FUNCTIONAL | All 8 HTMX swap strategies as typed constants.                                                                                         |
| 16  | Header Constants      | FULLY_FUNCTIONAL | All HTMX headers are unexported constants. `HeaderTrue` exported for consumers.                                                        |
| 16b | Embedded HTMX JS      | FULLY_FUNCTIONAL | `HTMXScriptHandler()` serves embedded HTMX v2.0.9 (minified, ~49KB) with ETag/caching. `HTMXVersion()`, `HTMXScriptTag(path)`. Opt-in. |

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

| #   | Feature               | Status           | Description                                                                                                                                         |
| --- | --------------------- | ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| 37  | Event-Sourced User    | FULLY_FUNCTIONAL | 7 events, 7 commands, Decider pattern via go-cqrs-lite. Pure decide functions + foldUser. No CRUD UserStore. Read-your-writes via MemoryBus.        |
| 37b | WebAuthn Passwordless | FULLY_FUNCTIONAL | go-webauthn v0.17.4. BeginRegistration/FinishRegistration/BeginLogin/FinishLogin. In-memory challenge store with proactive eviction.                |
| 37c | Credential Mgmt HTTP  | FULLY_FUNCTIONAL | GET /auth/credentials (list), DELETE /auth/credentials/{id} (remove by base64url ID). Sanitized credential summaries exclude sensitive fields.      |
| 37d | Account Lockout       | FULLY_FUNCTIONAL | Wired into BeginLogin (check) and FinishLogin (record/reset). Configurable max attempts + duration. `ErrAccountLocked` → 429.                       |
| 37e | Session Eviction      | FULLY_FUNCTIONAL | Background goroutine proactively removes expired WebAuthn sessions. `Service.Stop()` for cleanup.                                                   |
| 38  | Branded UserID        | FULLY_FUNCTIONAL | `UserID = brandid.ID[userBrand, string]` via go-branded-id. `.Get()` for cross-module conversion. `NewUserID(s)` constructor.                       |
| 39  | RBAC Authorization    | FULLY_FUNCTIONAL | Casbin RBAC with domains. CasbinProjection derives policies from events. `AsEnforcer()` bridge to parent `Enforcer` interface.                      |
| 40  | SessionStore          | FULLY_FUNCTIONAL | `SessionStore` interface + `InMemorySessionStore`. TTL-based expiry. `DeleteByUserID` for user deletion revocation.                                 |
| 41  | HTTP Handlers         | FULLY_FUNCTIONAL | `AuthHandler` with session cookies, WebAuthn endpoints, credential management. `SessionMiddleware`. Configurable timeout, `*bool` Secure.           |
| 42  | Input Validation      | FULLY_FUNCTIONAL | `RegisterRequest.Validate()`. Email format, required fields. Passwordless — no password validation needed.                                          |
| 42b | Email Verification    | FULLY_FUNCTIONAL | Token-based email confirmation. `EmailVerified` event. Optional SMTP callback. Email change resets verification. Single-use tokens with TTL.        |
| 42c | TOTP MFA              | FULLY_FUNCTIONAL | RFC 6238 TOTP from scratch (no external deps). Two-phase setup. `otpauth://` URIs for QR codes. Event-sourced secret (TOTPEnabled/Disabled).        |
| 42d | User Import/Export    | FULLY_FUNCTIONAL | JSON/CSV batch import (skips existing emails). JSON/CSV export with public profile only. Flexible CSV header detection.                             |
| 42e | SQL Event Store       | FULLY_FUNCTIONAL | Postgres/SQLite/MySQL event store with optimistic concurrency. Auto-migrates schema. Parameterized queries per dialect.                             |
| 42f | Audit Log             | FULLY_FUNCTIONAL | Event-sourced audit log projection. Queryable by user, recent N, total count. Optional via `ServiceConfig.AuditLog`.                                |
| 42g | Rate-Limited Reg      | FULLY_FUNCTIONAL | Per-IP fixed-window rate limiting on registration endpoint. Configurable via `HandlerConfig.RegistrationRateLimit`.                                 |
| 42h | Session Rotation      | FULLY_FUNCTIONAL | `UpdateRoles` deletes all user sessions after privilege change, forcing re-authentication. Non-blocking.                                            |
| 42i | WebAuthn Rate Limit   | FULLY_FUNCTIONAL | Per-IP rate limiting on all 4 WebAuthn endpoints. `HandlerConfig.WebAuthnRateLimit` uses the shared `RateLimitConfig`. Same pattern as TOTP/import. |

---

### Real-Time (SSE & WebSocket)

| #   | Feature                 | Status           | Description                                                                                                                                                                                                                            |
| --- | ----------------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 44  | SSE Event Writer        | FULLY_FUNCTIONAL | `SSEEvent` struct, `WriteSSEEvent(w, event)` — properly formatted SSE events (event/data/id/retry, multi-line, CRLF normalization).                                                                                                    |
| 45  | SSE Stream              | FULLY_FUNCTIONAL | `SSEStream` manages a single SSE connection. Correct headers, flush after send, context-aware lifecycle. `LastEventID()` for reconnection. `Heartbeat(ctx, interval)` prevents proxy idle kills. `OnDisconnect(fn)` cleanup callbacks. |
| 46  | SSE Broadcaster         | FULLY_FUNCTIONAL | `Broadcaster` — thread-safe fan-out. O(1) Unsubscribe via uintptr identity. Buffered channels (64). Non-blocking broadcast drops to slow consumers.                                                                                    |
| 47  | WebSocket Message       | FULLY_FUNCTIONAL | `WSMessage`, `ParseWSMessage` — parses HTMX WebSocket JSON (form fields + HEADERS). `StringBody` helper for typed field access.                                                                                                        |
| 48  | WebSocket OOB HTML      | FULLY_FUNCTIONAL | `WSOOBHTML(id, html, strategy)` — wraps HTML with hx-swap-oob attributes for OOB swap. Uses `SwapStrategy` type. Passthrough for custom markup.                                                                                        |
| 49  | SSE Reconnection        | FULLY_FUNCTIONAL | `LastEventIDFromRequest(r)`, `SSEEventStore` interface, `ReplayEvents(stream, store, lastID)` — full reconnection support per SSE spec.                                                                                                |
| 50  | SSE + CQRS Bridge       | FULLY_FUNCTIONAL | `BroadcastOnSuccess(event, data)` / `BroadcastOnSuccessFunc(fn)` — broadcast on success. `BroadcastOnError(eventName)` / `BroadcastOnErrorFunc(fn)` — broadcast StructuredError on failure. Full dispatch feedback for SSE clients.    |
| 51  | Typed WS Message Parser | FULLY_FUNCTIONAL | `ParseWSMessageInto[T](data)` — generic typed parser that deserializes body into struct T while separating HEADERS. Compile-time safe.                                                                                                 |
| 51a | StructuredError         | FULLY_FUNCTIONAL | RFC 7807-shaped transport-agnostic error payload (`type`, `title`, `status`, `detail`, `instance`). `NewStructuredError(err, r)` maps via `MapError` + request ID. `JSON()` for SSE/WS serialization.                                  |
| 51b | WS Message Encoder      | FULLY_FUNCTIONAL | `WriteWSMessage(w, msg)` / `WriteWSMessageInto[T](w, body, headers)` — outbound WS encoders, counterparts to `ParseWSMessage` / `ParseWSMessageInto[T]`. Round-trip verified.                                                          |
| 51c | WS Broadcaster          | FULLY_FUNCTIONAL | `WSBroadcaster` — thread-safe fan-out for WS messages. Mirrors SSE `Broadcaster` API. O(1) unsubscribe via channel pointer identity. `BroadcastHTML` for OOB swaps.                                                                    |
| 51d | WS + CQRS Bridge        | FULLY_FUNCTIONAL | `BroadcastOnSuccessWS(msg)` / `BroadcastOnErrorWS()` — AfterDispatchHook factories for `WSBroadcaster`. WebSocket equivalents of SSE `BroadcastOnSuccess` / `BroadcastOnError`.                                                        |
| 51e | WS Dispatch Bridge      | FULLY_FUNCTIONAL | `DispatchWSCommand(r, type, decoder, data)` / `DispatchWSQuery(r, type, decoder, data)` — decode WS message → dispatch via App. Runs lifecycle hooks. `DecodeWSJSON[T]` / `DecodeWSJSONQuery[T]` decoder factories.                    |

### Pagination (go-cqrs-lite v2.3.0)

| #   | Feature               | Status           | Description                                                                                                       |
| --- | --------------------- | ---------------- | ----------------------------------------------------------------------------------------------------------------- |
| 52  | Pagination Decoder    | FULLY_FUNCTIONAL | `DecodePagination(r)` extracts page/page_size from query params. Delegates to `query.NewPagination` for defaults. |
| 53  | Paginated JSON Render | FULLY_FUNCTIONAL | `RenderPaginatedJSON[T]()` renders `query.PaginatedResult[T]` as JSON 200. Type-safe via generic parameter.       |

---

## catalog Sub-Package (Opt-In)

| #   | Feature             | Status           | Description                                                                                                                         |
| --- | ------------------- | ---------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| 54  | Catalog Builder     | FULLY_FUNCTIONAL | `New(title, version)` wraps `catalog.Builder`. `Command[T]`, `Query[T]`, `Event[T]` register messages with auto-derived schemas.    |
| 55  | OpenAPI Handler     | FULLY_FUNCTIONAL | `OpenAPIHandler(cat)` serves OpenAPI 3.0 JSON/YAML. Auto-generates paths from message IDs and HTTP operations.                      |
| 56  | AsyncAPI Handler    | FULLY_FUNCTIONAL | `AsyncAPIHandler(cat)` serves AsyncAPI 3.0 JSON/YAML. Maps commands→receive, events→send/receive, queries→handle.                   |
| 57  | D2 Diagram Handler  | FULLY_FUNCTIONAL | `D2Handler(cat)` serves D2 architecture diagrams as text/plain. Cross-service event flows, domain grouping, CSS classes.            |
| 58  | EventCatalog Export | FULLY_FUNCTIONAL | `GenerateEventCatalog(cat, dir)` writes MDX file tree for EventCatalog CLI. Auto-derives producers/consumers from event directions. |
| 59  | Schema Reflection   | FULLY_FUNCTIONAL | Auto-derives JSON Schema from Go struct tags (`json`, `doc`, `format`, `enum`, `default`). Results cached via `sync.Map`.           |
| 60  | Catalog Validation  | FULLY_FUNCTIONAL | `Build()` panics on invalid catalogs (duplicate IDs, empty names). `BuildValid()` returns violations for non-panic usage.           |

### Not Planned

| Feature                 | Status      | Reason                                                                                      |
| ----------------------- | ----------- | ------------------------------------------------------------------------------------------- |
| WebSocket upgrade logic | NOT_PLANNED | Consumers choose their own WebSocket library (gorilla, coder, etc.). Protocol helpers only. |

---

## Metrics

| Metric         | Root  | usermgmt | catalog |
| -------------- | ----- | -------- | ------- |
| Coverage       | 96.4% | 88.7%    | 95.3%   |
| Ginkgo specs   | 464+  | —        | —       |
| Lint issues    | 0     | 0        | 0       |
| Prod files     | 23    | 10       | 2       |
| Test files     | 26+   | 12       | 4       |
| Benchmarks     | 20+   | 3        | 0       |
| Godoc examples | 17+   | 1        | 1       |
