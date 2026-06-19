# cqrs-htmx

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/cqrs-htmx.svg)](https://pkg.go.dev/github.com/larsartmann/cqrs-htmx)
[![CI](https://github.com/LarsArtmann/cqrs-htmx/actions/workflows/ci.yml/badge.svg)](https://github.com/LarsArtmann/cqrs-htmx/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 1.26+](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)

A Go library that makes it **very easy** to use [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) with [HTMX](https://htmx.org), [templ](https://templ.guide), and [Casbin](https://casbin.org) authorization.

**Framework-agnostic** — works with `net/http`, [Chi](https://github.com/go-chi/chi), [Gin](https://github.com/gin-gonic/gin), or any `http.Handler`-compatible router.

## Features at a Glance

- **Command & Query dispatch** — `app.Command()` / `app.Query()` return `http.HandlerFunc` that decode, authorize, dispatch, and respond
- **Casbin authorization** — `Authorize(resource, action)` checks policies before dispatch; `RequireAuth()` for auth-only guards
- **HTMX-aware responses** — fluent response builder with `HX-Trigger`, `HX-Redirect`, `HX-Push-Url`, swap strategies, and notifications
- **templ integration** — duck-typed `TemplComponent` renders templ components without importing templ
- **User identity propagation** — HTTP request → context → CQRS event metadata, with strongly-typed `UserID` and `CorrelationID` (ULID-backed)
- **Request validation** — `ValidateCommand`/`ValidateQuery` wrap decoders with validation logic
- **Dispatch timeout** — `Config.Timeout` wraps dispatch with `context.WithTimeout`
- **Lifecycle hooks** — `BeforeDispatchHook`/`AfterDispatchHook` for tracing, logging, metrics
- **Error classification** — CQRS error families automatically map to HTTP status codes
- **JSON or plain-text error handlers** — pluggable `ErrorHandler` with HTMX-aware auth redirects
- **Notification system** — `NotifySuccess`/`NotifyError`/`NotifyWarning`/`NotifyInfo` with standard `{level, message}` payload; custom event names via `NotifyWithEvent`
- **CSRF protection** — double-submit cookie pattern with HTMX-aware `X-CSRF-Token` header validation
- **Rate limiting** — per-key token-bucket with min-heap eviction, configurable burst, and hook callbacks
- **Security headers** — automatic `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, plus optional CSP/HSTS/Permissions-Policy
- **Request logging** — plain-text or structured JSON logging with status, duration, and context IDs
- **SSE streaming** — `SSEStream`, `Broadcaster` (thread-safe fan-out), `SSEEventStore` for reconnection replay, CQRS bridge via `BroadcastOnSuccess`/`BroadcastOnError`, `Heartbeat` for proxy keepalive
- **WebSocket helpers** — `ParseWSMessage`, `ParseWSMessageInto[T]` (typed), `WSOOBHTML` for OOB swaps, `WSBroadcaster` fan-out, `DispatchWSCommand`/`DispatchWSQuery` CQRS bridge
- **Pagination** — `DecodePagination(r)` + `RenderPaginatedJSON[T]()` with go-cqrs-lite v2.6.0
- **Embedded HTMX JS** — `HTMXScriptHandler()` serves embedded HTMX v2.0.9 (minified) with ETag/caching. Opt-in, zero CDN dependency
- **User management** — optional [`usermgmt`](#user-management-usermgmt) submodule with RBAC, sessions, account lockout, and HTTP auth handlers

## Why

Combining CQRS, HTMX, and authorization requires repetitive wiring:

- HTTP requests → CQRS command/query dispatch
- User identity from auth → event metadata
- Casbin policy checks before dispatch
- CQRS errors → HTTP status codes (HTMX-aware)
- HTMX partial vs full-page rendering
- HTMX response headers (triggers, redirects, push URL)
- templ component rendering for query results
- CSRF protection, rate limiting, security headers

This library handles all of it automatically.

## Install

```bash
go get github.com/larsartmann/cqrs-htmx
```

For the user management submodule:

```bash
go get github.com/larsartmann/cqrs-htmx/usermgmt
```

## Quick Start

```go
package main

import (
    "net/http"

    cqrshtmx "github.com/larsartmann/cqrs-htmx"
    "github.com/larsartmann/go-cqrs-lite/command/v2"
    "github.com/larsartmann/go-cqrs-lite/query/v2"
    "github.com/casbin/casbin/v3"
)

func main() {
    cmdDisp := command.NewDispatcher()
    qryDisp := query.NewDispatcher()
    enforcer, _ := casbin.NewEnforcer("model.conf", "policy.csv")

    app, _ := cqrshtmx.New(cqrshtmx.Config{
        Commands:        cmdDisp,
        Queries:         qryDisp,
        Enforcer:        enforcer,
        UserIDExtractor: func(r *http.Request) (cqrshtmx.UserID, error) {
            return cqrshtmx.ParseUserID(r.Header.Get("X-User-ID"))
        },
    })

    mux := http.NewServeMux()

    // Command dispatch with auth + HTMX
    mux.Handle("POST /users", app.Command("CreateUser",
        cqrshtmx.Authorize("users", "create"),
        cqrshtmx.DecodeJSON(func(req CreateUserRequest) (command.Command, error) {
            return NewCreateUserCmd(req.Email, req.Name), nil
        }),
        cqrshtmx.NotifySuccess("User created"),
        cqrshtmx.PushURL("/users"),
    ))

    // Query dispatch with templ rendering
    mux.Handle("GET /users", app.Query("ListUsers",
        cqrshtmx.Authorize("users", "read"),
        cqrshtmx.DecodeJSONQuery(func(req ListUsersRequest) (query.Query, error) {
            return &ListUsersQuery{}, nil
        }),
        cqrshtmx.RenderTemplResult(func(result []*User) cqrshtmx.TemplComponent {
            return userListPage(result)
        }),
    ))

    // Apply middleware stack
    handler := cqrshtmx.Chain(
        cqrshtmx.SecurityHeadersMiddleware,
        cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
        cqrshtmx.HTMXMiddleware,
        app.Middleware(),
    )(mux)

    http.ListenAndServe(":8080", handler)
}
```

## Config

`New(Config)` creates an `App`. `Commands` or `Queries` must be non-nil:

```go
app, err := cqrshtmx.New(cqrshtmx.Config{
    Commands:        cmdDisp,          // *command.Dispatcher
    Queries:         qryDisp,          // *query.Dispatcher
    Enforcer:        enforcer,         // Enforcer interface (Casbin or custom)
    UserIDExtractor: extractFunc,      // func(r *http.Request) (UserID, error)
    LoginRedirect:   "/auth/signin",   // default: "/login"
    Timeout:         5 * time.Second,  // wraps dispatch only; 0 = no timeout
    ErrorHandler:    myErrorHandler,   // custom ErrorHandler; default uses plain text + LoginRedirect
    BeforeDispatch:           beforeHook,       // func(ctx, r) context.Context — tracing, request IDs
    AfterDispatch:            afterHook,        // func(ctx, r, err) — logging, metrics
    IncludeRequestIDInErrors: true,             // prefix errors with request_id for log correlation
    MaxBodySize:              10 << 20,         // max request body (default 10 MB)
    ServiceName:              "my-service",     // identifies the service in event metadata
})
```

## Handler Options

### Authorization

| Option                        | Description                                         |
| ----------------------------- | --------------------------------------------------- |
| `Authorize(resource, action)` | Check Casbin policy before dispatch                 |
| `RequireAuth()`               | Require authenticated user (no specific permission) |

### Request Decoding

| Option                       | Description                                         |
| ---------------------------- | --------------------------------------------------- |
| `DecodeJSON[T](mapper)`      | Decode JSON body into a command via mapper function |
| `DecodeJSONQuery[T](mapper)` | Decode JSON body into a query via mapper function   |
| `DecodeForm[T](mapper)`      | Decode form data into a command via mapper function |
| `DecodeFormQuery[T](mapper)` | Decode form data into a query via mapper function   |

All decoders enforce `Config.MaxBodySize` when set (returns `ErrRequestTooLarge` / 413 when exceeded).

### Validation

Wrap your decoder with validation. Must be applied **after** the decoder in the option list:

```go
app.Command("CreateUser",
    cqrshtmx.DecodeJSON(createUserMapper),
    cqrshtmx.ValidateCommand(func(cmd command.Command) error {
        return validateStruct(cmd)
    }),
)
```

| Option                       | Description                                        |
| ---------------------------- | -------------------------------------------------- |
| `ValidateCommand(validator)` | Validate decoded command; errors → 400 Bad Request |
| `ValidateQuery(validator)`   | Validate decoded query; errors → 400 Bad Request   |

### Response

| Option                             | Description                                                            |
| ---------------------------------- | ---------------------------------------------------------------------- |
| `Render(fn)`                       | Render query result with custom function                               |
| `RenderTempl(component)`           | Render a fixed templ.Component (duck-typed)                            |
| `RenderTemplResult[T](mapper)`     | Map query result to a templ.Component and render                       |
| `RenderJSON[T]()`                  | Render query result as JSON (200 OK)                                   |
| `RenderJSONStatus[T](status)`      | Render query result as JSON with custom status                         |
| `Redirect(url)`                    | Redirect after success (HX-Redirect for HTMX, HTTP redirect otherwise) |
| `Trigger(event)`                   | Fire HTMX client-side event on success                                 |
| `TriggerWithDetail(event, detail)` | Fire HTMX event with JSON detail data                                  |
| `PushURL(url)`                     | Push URL into browser history                                          |

### Notifications

Convenience wrappers around `TriggerWithDetail` with a standard `{level, message}` payload:

| Option                   | Description          |
| ------------------------ | -------------------- |
| `NotifySuccess(message)` | Success notification |
| `NotifyError(message)`   | Error notification   |
| `NotifyWarning(message)` | Warning notification |
| `NotifyInfo(message)`    | Info notification    |

All use the default `"showMessage"` event. For custom event names, use the `NotifyWithEvent` builder:

```go
app.Command("CreateUser",
    cqrshtmx.NotifyWithEvent("showToast").Success("User created"),
)
```

Client-side JS:

```js
document.body.addEventListener("showMessage", function (evt) {
  showNotification(evt.detail.level, evt.detail.message);
});
```

### Per-Handler Timeout

Override the global `Config.Timeout` per handler:

```go
app.Command("SlowOperation",
    cqrshtmx.WithTimeout(30*time.Second),
    cqrshtmx.DecodeJSON(mapper),
)
```

### Per-Handler Options

| Option                    | Description                                            |
| ------------------------- | ------------------------------------------------------ |
| `WithMaxBodySize(n)`      | Override `Config.MaxBodySize` per handler              |
| `WithSuccessStatus(code)` | Custom success status (default 204; e.g., 201 Created) |
| `RequireMethod(method)`   | Reject wrong HTTP methods with 405                     |
| `OnError(fn)`             | Per-handler error callback after App-level handler     |

### Convenience Helpers

| Helper                                   | Description                                  |
| ---------------------------------------- | -------------------------------------------- |
| `App.HealthHandler()`                    | 200/503 JSON health check for load balancers |
| `App.HasCommands()` / `App.HasQueries()` | Report dispatcher availability               |
| `MustNew(cfg)`                           | Panics on error — for init-time setup        |
| `IsAuthenticated(r)`                     | Checks for non-zero UserID in context        |

## HTMX Request Context

`HTMXMiddleware` parses all HTMX headers once and stores them in context:

```go
handler := cqrshtmx.Chain(
    cqrshtmx.HTMXMiddleware,
    app.Middleware(),
)(mux)
```

### HTMXRequest Struct

```go
h := cqrshtmx.HTMXFromContext(r.Context())
h.IsHTMX           // bool
h.IsBoosted        // bool
h.IsHistoryRestore // bool
h.Target           // string
h.TriggerID        // string
h.TriggerName      // string
h.Prompt           // string
h.CurrentURL       // string
h.RenderPartial()  // bool — IsHTMX && !IsHistoryRestore
```

### Standalone Accessors (work with or without middleware)

```go
cqrshtmx.IsHTMXRequest(r)    // bool
cqrshtmx.IsBoosted(r)        // bool
cqrshtmx.IsHistoryRestore(r) // bool
cqrshtmx.RenderPartial(r)    // bool — partial vs full page
cqrshtmx.HTMXTarget(r)       // string
cqrshtmx.HTMXTrigger(r)      // string
cqrshtmx.HTMXTriggerName(r)  // string
cqrshtmx.HTMXPrompt(r)       // string
cqrshtmx.HTMXCurrentURL(r)   // string
```

## HTMX Response Builder

Build HTMX-aware responses with fluent chaining:

```go
resp := cqrshtmx.NewResponse(w, r)
resp.Trigger("userCreated").
    PushURL("/users/1").
    Retarget("#user-list").
    Reswap(cqrshtmx.SwapInnerHTML).
    NotifySuccess("User created").
    Apply()
```

### Methods

| Method                            | Header Set                            |
| --------------------------------- | ------------------------------------- |
| `PushURL(url)`                    | `HX-Push-Url`                         |
| `ReplaceURL(url)`                 | `HX-Replace-Url`                      |
| `Redirect(url)`                   | `HX-Redirect` (HTMX) or HTTP redirect |
| `Refresh()`                       | `HX-Refresh`                          |
| `Location(url)`                   | `HX-Location`                         |
| `Reswap(strategy)`                | `HX-Reswap`                           |
| `Retarget(selector)`              | `HX-Retarget`                         |
| `Reselect(selector)`              | `HX-Reselect`                         |
| `Trigger(event)`                  | `HX-Trigger`                          |
| `TriggerAfterSwap(event)`         | `HX-Trigger-After-Swap`               |
| `TriggerAfterSettle(event)`       | `HX-Trigger-After-Settle`             |
| `TriggerWithDetail(name, detail)` | `HX-Trigger` (JSON)                   |
| `NotifySuccess(message)`          | `HX-Trigger` (JSON notification)      |
| `NotifyError(message)`            | `HX-Trigger` (JSON notification)      |
| `NotifyWarning(message)`          | `HX-Trigger` (JSON notification)      |
| `NotifyInfo(message)`             | `HX-Trigger` (JSON notification)      |

### Swap Strategies

`SwapInnerHTML`, `SwapOuterHTML`, `SwapBeforeBegin`, `SwapAfterBegin`, `SwapBeforeEnd`, `SwapAfterEnd`, `SwapDelete`, `SwapNone`

## SSE (Server-Sent Events)

First-class SSE support for the HTMX SSE extension (`hx-ext="sse"`):

```go
// Start an SSE stream
stream := cqrshtmx.NewSSEStream(w, r)
_ = stream.Send(cqrshtmx.SSEEvent{
    Event: "todoUpdated",
    Data:  "<div id='todos'><ul><li>Buy milk</li></ul></div>",
})

// Fan-out to multiple clients
broadcaster := cqrshtmx.NewBroadcaster()
ch := broadcaster.Subscribe()
broadcaster.Broadcast(cqrshtmx.SSEEvent{Event: "update", Data: html})

// Bridge to CQRS: broadcast SSE on successful command dispatch
app, _ := cqrshtmx.New(cqrshtmx.Config{
    AfterDispatch: broadcaster.BroadcastOnSuccess("itemUpdated", ""),
})

// Broadcast StructuredError (RFC 7807) on dispatch failure
errApp, _ := cqrshtmx.New(cqrshtmx.Config{
    AfterDispatch: broadcaster.BroadcastOnError("commandError"),
})

// Prevent reverse proxies from killing idle connections
go stream.Heartbeat(stream.Context(), 15*time.Second)

// Reconnection support (SSE spec)
lastID := cqrshtmx.LastEventIDFromRequest(r)
cqrshtmx.ReplayEvents(stream, eventStore, lastID)
```

Client-side:

```html
<div hx-ext="sse" sse-connect="/events" sse-swap="todoUpdated">
  <!-- content swapped here -->
</div>
```

### SSE API

|| Type / Function | Description |
|| -------------------------------------- | ---------------------------------------------------------------------------------- |
|| `SSEEvent` | Event struct: `Event`, `Data`, `ID`, `Retry` |
|| `WriteSSEEvent(w, event)` | Write a single SSE event to any `io.Writer` |
|| `NewSSEStream(w, r)` | Create a managed SSE connection (correct headers, flush, context-aware lifecycle) |
|| `stream.Send(event)` | Send event to connected client |
|| `stream.SendHTML(name, html)` | Shorthand for HTML content events |
|| `stream.LastEventID()` | Client's last event ID (for reconnection) |
|| `stream.Close()` | Graceful shutdown |
| `stream.Heartbeat(ctx, interval)` | Send SSE comment-frame pings to prevent proxy idle kills |
| `stream.OnDisconnect(fn)` | Register cleanup callback fired on `Close()` |
|| `NewBroadcaster()` | Thread-safe fan-out hub |
|| `broadcaster.Subscribe()` | Get a receiver channel |
|| `broadcaster.Unsubscribe(ch)` | O(1) unsubscribe via channel identity |
|| `broadcaster.Broadcast(event)` | Non-blocking send to all subscribers (drops to slow consumers) |
|| `broadcaster.SubscriberCount()` | Active subscriber count |
|| `BroadcastOnSuccess(event, data)` | `AfterDispatchHook` that broadcasts on successful dispatch |
|| `BroadcastOnSuccessFunc(fn)` | `AfterDispatchHook` with dynamic event generation |
| `BroadcastOnError(eventName)` | `AfterDispatchHook` that broadcasts StructuredError on dispatch failure |
| `BroadcastOnErrorFunc(fn)` | `AfterDispatchHook` with dynamic error event generation |
| `NewStructuredError(err, r)` | RFC 7807 error payload with type/title/status/detail/instance. `.JSON()` for SSE/WS |
|| `LastEventIDFromRequest(r)` | Extract `Last-Event-ID` from request |
|| `SSEEventStore` | Interface for reconnection replay (`EventsSince(id)`) |
|| `ReplayEvents(stream, store, lastID)` | Replay missed events to reconnecting client |

## WebSocket Helpers

Protocol helpers for the HTMX WebSocket extension (`hx-ext="ws"`). Library consumers choose their own WebSocket library (gorilla, coder, etc.):

```go
// Parse incoming HTMX WS message
msg, err := cqrshtmx.ParseWSMessage(rawJSON)
msg.Headers    // map[string]string — HTMX headers
msg.Body       // map[string]any — form field values
msg.StringBody("field_name")  // typed string access

// Typed parsing (generic)
type ChatMsg struct { Room string; Message string }
msg, headers, err := cqrshtmx.ParseWSMessageInto[ChatMsg](rawJSON)

// Encode outbound messages (counterpart to Parse)
err := cqrshtmx.WriteWSMessage(conn, msg)
err := cqrshtmx.WriteWSMessageInto(conn, ChatMsg{Room: "dev"}, headers)

// Out-of-band HTML swap
html := cqrshtmx.WSOOBHTML("notifications", "<div>3 new items</div>", cqrshtmx.SwapInnerHTML)

// Fan-out to multiple WS clients
wsBroadcaster := cqrshtmx.NewWSBroadcaster()
ch := wsBroadcaster.Subscribe()
defer wsBroadcaster.Unsubscribe(ch)
wsBroadcaster.Broadcast(html)

// Bridge to CQRS: dispatch WS message as command/query
err := app.DispatchWSCommand(r, "CreateTask", decoder, rawMessage)
result, err := app.DispatchWSQuery(r, "GetTasks", queryDecoder, rawMessage)
```

Client-side:

```html
<div hx-ext="ws" ws-connect="/ws">
  <form ws-send>
    <input name="message" />
    <button>Send</button>
  </form>
</div>
```

### WebSocket API

| Type / Function                             | Description                                                                       |
| ------------------------------------------- | --------------------------------------------------------------------------------- |
| `WSMessage`                                 | Incoming HTMX WS message (`Headers`, `Body`). `StringBody(key)` for typed access. |
| `ParseWSMessage(data)`                      | Parse incoming WS JSON into `WSMessage`. Separates HEADERS from body.             |
| `ParseWSMessageInto[T](data)`               | Generic typed parser — deserializes body into struct T. Compile-time safe.        |
| `WriteWSMessage(w, msg)`                    | Encode outbound `WSMessage` to HTMX WS JSON format.                               |
| `WriteWSMessageInto[T](w, body, headers)`   | Encode typed body struct + headers to HTMX WS JSON.                               |
| `WSOOBHTML(id, html, strategy...)`          | Wrap HTML with hx-swap-oob for OOB swap.                                          |
| `NewWSBroadcaster()`                        | Thread-safe fan-out hub for WS messages.                                          |
| `wsBroadcaster.Subscribe()`                 | Get a receiver channel (`<-chan string`).                                         |
| `wsBroadcaster.Unsubscribe(ch)`             | O(1) unsubscribe via channel identity.                                            |
| `wsBroadcaster.Broadcast(msg)`              | Non-blocking send to all subscribers.                                             |
| `wsBroadcaster.BroadcastHTML(id, html)`     | Convenience: wraps in OOB then broadcasts.                                        |
| `BroadcastOnSuccessWS(msg)`                 | `AfterDispatchHook` — broadcast on dispatch success.                              |
| `BroadcastOnSuccessWSFunc(fn)`              | `AfterDispatchHook` — dynamic message generation on success.                      |
| `BroadcastOnErrorWS()`                      | `AfterDispatchHook` — broadcast StructuredError on failure.                       |
| `BroadcastOnErrorWSFunc(fn)`                | `AfterDispatchHook` — dynamic error message generation on failure.                |
| `DispatchWSCommand(r, type, decoder, data)` | Decode WS message → dispatch command. Returns error.                              |
| `DispatchWSQuery(r, type, decoder, data)`   | Decode WS message → dispatch query. Returns `(result, error)`.                    |
| `DecodeWSJSON[T](mapper)`                   | Create `WSCommandDecoder` from JSON → T → command mapper.                         |
| `DecodeWSJSONQuery[T](mapper)`              | Create `WSQueryDecoder` from JSON → T → query mapper.                             |

## Embedded HTMX JavaScript

Embed HTMX v2.0.9 (minified, ~49KB) directly in your binary — no CDN dependency:

```go
// Serve embedded HTMX JS
mux.Handle("/static/htmx.js", cqrshtmx.HTMXScriptHandler())

// Generate script tag for templates
cqrshtmx.HTMXScriptTag("/static/htmx.js")
// => <script src="/static/htmx.js"></script>

// Check version
cqrshtmx.HTMXVersion() // "2.0.9"
```

`HTMXScriptHandler` sets `Content-Type: text/javascript`, long-lived `Cache-Control` (1 year, immutable), and `ETag` with `If-None-Match` support for 304 responses.

## templ Integration

First-class support for [templ](https://templ.guide) without importing it as a dependency.
Uses duck-typing — any type with `Render(ctx context.Context, w io.Writer) error` works.

```go
// Render a fixed component
app.Query("GetPage",
    cqrshtmx.RenderTempl(myPageComponent()),
)

// Map query result to a component
app.Query("ListUsers",
    cqrshtmx.DecodeJSONQuery(mapper),
    cqrshtmx.RenderTemplResult(func(result []*User) cqrshtmx.TemplComponent {
        return userListView(result)
    }),
)
```

Works seamlessly with [templ-components](https://github.com/larsartmann/templ-components).

## Context Propagation

User identity, correlation IDs, and request IDs flow automatically from HTTP → CQRS metadata.
All ID types are **strongly-typed** wrappers around [ULID](https://github.com/oklog/ulid) — collision-free, lexicographically sortable, and impossible to confuse with raw strings.

```go
// User ID
userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
ctx := cqrshtmx.WithUserID(r.Context(), userID)
retrieved := cqrshtmx.UserIDFromContext(ctx)

// Correlation ID (auto-extracted from X-Correlation-ID header)
cid := cqrshtmx.MustParseCorrelationID("01HK154ANGZHV2ZW0X3SKSNEN2")
ctx = cqrshtmx.WithCorrelationID(ctx, cid)
corrID := cqrshtmx.CorrelationIDFromContext(ctx)

// Request ID
reqID := cqrshtmx.NewRequestID()
ctx = cqrshtmx.WithRequestID(ctx, reqID)

// Build event options from context (includes user ID + correlation ID)
opts := cqrshtmx.EventOptionsFromContext(ctx)
evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, payload, opts...)

// Generate new random IDs
newID  := cqrshtmx.NewUserID()
newCID := cqrshtmx.NewCorrelationID()
newRID := cqrshtmx.NewRequestID()
```

### Why ULID?

- **Collision-free keys** — 48-bit timestamp + 80-bit randomness
- **Lexicographic sortability** — time-ordered without a central coordinator
- **Type safety** — impossible to pass an arbitrary string where a valid ID is required
- **Consistency** — event metadata, context values, and Casbin subjects all use the same representation

`ContextEnrichmentMiddleware` silently drops non-ULID values from `X-Correlation-ID` headers (and `UserIDExtractor` outputs that fail parsing). Use `NewUserID()` / `NewCorrelationID()` to generate valid IDs, or `ParseUserID()` / `ParseCorrelationID()` for safe parsing from external input.

## Error Mapping

CQRS error families automatically map to HTTP status codes:

| CQRS Family    | HTTP Status               |
| -------------- | ------------------------- |
| Rejection      | 400 Bad Request           |
| Conflict       | 409 Conflict              |
| Corruption     | 422 Unprocessable Entity  |
| Transient      | 503 Service Unavailable   |
| Infrastructure | 500 Internal Server Error |

Auth errors map specially:

- `ErrUnauthorized` → 401
- `ErrForbidden` → 403

### Error Handlers

The default error handler writes plain text and redirects HTMX auth errors to the login page. For JSON APIs:

```go
app, _ := cqrshtmx.New(cqrshtmx.Config{
    ErrorHandler: cqrshtmx.JSONErrorHandlerWithRedirect,
    LoginRedirect: "/auth/signin",
})
```

| Handler                                       | Format     | Login Redirect                              |
| --------------------------------------------- | ---------- | ------------------------------------------- |
| `DefaultErrorHandler`                         | Plain text | `/login`                                    |
| `DefaultErrorHandlerWithRedirect`             | Plain text | Custom                                      |
| `DefaultErrorHandlerWithRequestID`            | Plain text | `/login` (includes request_id when present) |
| `DefaultErrorHandlerWithRedirectAndRequestID` | Plain text | Custom (includes request_id when present)   |
| `JSONErrorHandler`                            | JSON       | `/login`                                    |
| `JSONErrorHandlerWithRedirect`                | JSON       | Custom                                      |
| `MapError(err)`                               | —          | Returns HTTP status code for any CQRS error |

### Login Redirect

Configure per-App. For HTMX requests with auth errors, the handler sets `HX-Redirect` instead of returning an error body:

```go
app, _ := cqrshtmx.New(cqrshtmx.Config{
    LoginRedirect: "/auth/signin",
})
```

## Middleware

```go
// Context enrichment (applied once to your router)
mux := app.Middleware()(router)

// HTMX header parsing (applied once, stores in context)
mux := cqrshtmx.HTMXMiddleware(mux)

// Panic recovery (catches panics, logs stack trace, writes 500)
mux := cqrshtmx.RecoveryMiddleware(mux)
// Or use App.RecoverHandler() for App-configured error handling:
mux := app.RecoverHandler()(mux)

// Standalone Casbin authorization middleware
mux.Handle("/admin", cqrshtmx.AuthorizeMiddleware(
    enforcer, "admin", "access",
    userIDExtractor,
)(handler))

// Chain multiple middleware
chained := cqrshtmx.Chain(
    cqrshtmx.SecurityHeadersMiddleware,
    cqrshtmx.RecoveryMiddleware,
    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
    cqrshtmx.HTMXMiddleware,
    app.Middleware(),
)(mux)
```

### Request Logging

Log every request with method, path, status, duration, and context IDs:

```go
// Plain text or custom format
handler := cqrshtmx.RequestLogging(nil, func(line string) {
    log.Println(line)
})(mux)

// Structured JSON
handler := cqrshtmx.RequestLogging(cqrshtmx.JSONLogFormatter, func(line string) {
    log.Println(line)
})(mux)

// slog integration (recommended for production)
handler := cqrshtmx.RequestLoggingSlog(slog.Default())(mux)
```

### CSRF Protection

Double-submit cookie CSRF protection with HTMX awareness. Uses [`justinas/nosurf`](https://github.com/justinas/nosurf) internally for token generation and origin validation. Validates `X-CSRF-Token` header (or form field) on state-changing methods:

```go
handler := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{
    CookieName: "csrf_token",
    HeaderName: "X-CSRF-Token",
    MaxAge:     24 * time.Hour,
    Secure:     true,
    SameSite:   http.SameSiteLaxMode,
})
```

**CSRFConfig fields:**

| Field            | Default          | Description                            |
| ---------------- | ---------------- | -------------------------------------- |
| `CookieName`     | `"csrf_token"`   | Cookie name                            |
| `HeaderName`     | `"X-CSRF-Token"` | Header/field name to validate          |
| `FieldName`      | `"csrf_token"`   | Form field name                        |
| `MaxAge`         | 24h              | Token lifetime                         |
| `Secure`         | false            | Set `true` in production (HTTPS only)  |
| `SameSite`       | `Lax`            | Cross-site cookie policy               |
| `Domain`         | `""`             | Cookie domain (host-only by default)   |
| `Path`           | `"/"`            | Cookie path                            |
| `TrustedOrigins` | `nil`            | Additional trusted origins             |
| `ErrorHandler`   | 403 plain text   | Custom error handler for CSRF failures |

**HTMX Integration:**

```html
<!-- Set token globally for all HTMX requests -->
<body hx-headers='{"X-CSRF-Token":"{{ .CSRFToken }}"}'></body>
```

```go
// Pass token to templates from handler
token := cqrshtmx.CSRFTokenFromContext(r.Context())

// Or use Response builder
resp := cqrshtmx.NewResponse(w, r)
resp.CSRFToken(token).Apply()
```

**Template Helpers** (for HTML/templ):

```go
data := map[string]any{
    "CSRFMeta":      cqrshtmx.CSRFTokenHTMLMeta(r),     // <meta name="csrf-token" content="...">
    "CSRFHXHeaders": cqrshtmx.CSRFTokenHXHeaders(r),    // hx-headers='{"X-CSRF-Token":"..."}'
    "CSRFFormField": cqrshtmx.CSRFTokenFormField(r),    // <input type="hidden" name="..." value="...">
    "CSRFToken":     cqrshtmx.CSRFTokenFromContext(r.Context()),
}
```

**Auto-inject token** (no handler changes needed):

```go
handler := cqrshtmx.Chain(
    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
    cqrshtmx.CSRFResponseHeaderMiddleware, // auto-sets X-CSRF-Token on every response
    cqrshtmx.HTMXMiddleware,
    app.Middleware(),
)(mux)
```

**Per-handler CSRF** (instead of global middleware):

```go
app.Command("CreateUser",
    cqrshtmx.CSRFProtect(cqrshtmx.CSRFConfig{}),
    cqrshtmx.DecodeJSON(...),
)
```

### Rate Limiting

Token-bucket rate limiter per key. Global rate limit when `KeyExtractor` is nil:

```go
handler := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
    Limit:        100,          // requests per Window
    Window:       time.Minute,
    Burst:        10,           // max burst (zero defaults to Limit)
    KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
})(mux)
```

**RateLimiterConfig fields:**

| Field              | Default       | Description                                              |
| ------------------ | ------------- | -------------------------------------------------------- |
| `Limit`            | 100           | Requests per `Window`                                    |
| `Window`           | 1 minute      | Rate window                                              |
| `Burst`            | = `Limit`     | Max burst size                                           |
| `KeyExtractor`     | nil (global)  | `func(*http.Request) string`; nil = single global bucket |
| `TTL`              | 10 minutes    | Idle limiter eviction time                               |
| `MaxKeys`          | 0 (unbounded) | Cap on tracked keys; oldest evicted via min-heap         |
| `OnAllowed`        | nil           | Hook called on allowed requests                          |
| `OnRejected`       | nil           | Hook called on rejected requests (includes retry-after)  |
| `RejectionHandler` | default 429   | Custom handler for rejected requests                     |

**Per-IP rate limiting** — use `KeyExtractorFromRemoteAddr`. Behind reverse proxies, use `ClientIP(r)` which respects `X-Forwarded-For` and `X-Real-IP`.

**Exempt requests** — return `""` from `KeyExtractor` to skip rate limiting for that request.

### Security Headers

Defense-in-depth headers on every response. Recommended for all production deployments:

```go
handler := cqrshtmx.SecurityHeadersMiddleware(mux)
```

Defaults: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`.

For custom headers:

```go
handler := cqrshtmx.SecurityHeadersMiddlewareWithConfig(cqrshtmx.SecurityHeadersConfig{
    ContentSecurityPolicy:   "default-src 'self'",
    StrictTransportSecurity: "max-age=31536000; includeSubDomains",
    PermissionsPolicy:       "camera=(), microphone=(), geolocation=()",
    Custom: map[string]string{
        "X-Custom-Header": "value",
    },
})(mux)
```

## Utilities

### WriteJSON

Encode and write JSON responses in one call:

```go
if err := cqrshtmx.WriteJSON(w, http.StatusOK, user); err != nil {
    // handle encode error
}
```

### ClientIP

Extract client IP from `X-Forwarded-For` → `X-Real-IP` → `RemoteAddr`:

```go
ip := cqrshtmx.ClientIP(r)
```

**Warning:** trusts proxy headers without validation. Only use behind a trusted reverse proxy that strips/overwrites these headers.

## User Management (`usermgmt`)

An independent submodule with **passwordless** authentication (WebAuthn/Passkeys), event-sourced CQRS, RBAC via Casbin, and session management:

```bash
go get github.com/larsartmann/cqrs-htmx/usermgmt
```

### Setup

```go
import (
    "log/slog"
    "time"

    "github.com/larsartmann/cqrs-htmx/usermgmt"
)

// Create service with WebAuthn configuration
svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
    WebAuthnConfig: &usermgmt.WebAuthnConfig{
        RPID:          "example.com",
        RPDisplayName: "My App",
        RPOrigins:     []string{"https://example.com"},
    },
    SessionTTL: 24 * time.Hour,
    Logger:     slog.Default(),
    Lockout:    usermgmt.NewAccountLockout(usermgmt.LockoutConfig{
        MaxAttempts: 5,
        Duration:    15 * time.Minute,
    }),
    EventHandler: func(userID usermgmt.UserID, evt any) {
        slog.Info("user event", "user_id", userID, "event", evt)
    },
})
defer svc.Stop()

// Create HTTP handlers
authHandler := usermgmt.NewAuthHandler(svc, usermgmt.HandlerConfig{
    SessionMaxAge: 86400,
})

// Register routes
mux := http.NewServeMux()
authHandler.RegisterRoutes(mux)
// Routes: POST /auth/register, POST /auth/webauthn/register/begin|finish,
//         POST /auth/webauthn/login/begin|finish, POST /auth/logout, GET /auth/me,
//         GET /auth/credentials, DELETE /auth/credentials/{id}

// Session middleware (validates cookie, loads user into context)
handler := usermgmt.NewSessionMiddleware(svc, "session_token")(mux)
```

### Registration & Login Flow

Users register with **email only** (no password). Authentication is via WebAuthn/Passkeys:

```
1. POST /auth/register              → create account (email only), get session
2. POST /auth/webauthn/register/begin  → get credential creation challenge
3. POST /auth/webauthn/register/finish → verify attestation, persist credential
4. POST /auth/webauthn/login/begin     → get assertion challenge
5. POST /auth/webauthn/login/finish    → verify assertion, create session
```

Finish endpoints read `user_id` from the URL query param (`?user_id=...`), since the request body contains the WebAuthn attestation/assertion response.

### Service Methods

| Method                                                                   | Description                                                    |
| ------------------------------------------------------------------------ | -------------------------------------------------------------- |
| `Register(ctx, RegisterRequest)` → `(*RegisterResponse, error)`          | Create user (email only), assign default roles, create session |
| `BeginRegistration(ctx, userID)` → `(*BeginRegistrationResponse, error)` | Start WebAuthn credential registration                         |
| `FinishRegistration(ctx, userID, r, credentialName)` → `error`           | Complete credential registration                               |
| `BeginLogin(ctx, email)` → `(*BeginLoginResponse, error)`                | Start WebAuthn login ceremony                                  |
| `FinishLogin(ctx, userID, r)` → `(*FinishLoginResponse, error)`          | Complete login, create session                                 |
| `Logout(ctx, token)` → `error`                                           | Delete session                                                 |
| `Authenticate(ctx, token)` → `(*User, error)`                            | Validate session, return user                                  |
| `Authorize(ctx, sub, dom, obj, act)` → `error`                           | Check RBAC policy                                              |
| `GetUser(ctx, id)` → `(*User, error)`                                    | Find user by ID                                                |
| `UpdateRoles(ctx, userID, roles, domain)` → `error`                      | Atomically update user's roles                                 |
| `ChangeEmail(ctx, userID, newEmail)` → `error`                           | Change user's email                                            |
| `ChangeDisplayName(ctx, userID, newName)` → `error`                      | Change user's display name                                     |
| `DeleteUser(ctx, userID, reason)` → `error`                              | Soft-delete user (tombstone), revoke sessions                  |
| `AddCredential(ctx, userID, cred)` → `error`                             | Add WebAuthn credential                                        |
| `RemoveCredential(ctx, userID, credID)` → `error`                        | Remove WebAuthn credential                                     |
| `Authz()` → `*Authz`                                                     | Access underlying RBAC engine                                  |
| `ReadModel()` → `*UserReadModel`                                         | Access read model directly                                     |
| `Stop()`                                                                 | Gracefully shutdown background resources                       |

### Input Validation

`RegisterRequest` has a `Validate()` method:

```go
req := usermgmt.RegisterRequest{
    ID:          usermgmt.NewUserID(ulid),
    Email:       "user@example.com",
    DisplayName: "Alice",
}
if err := req.Validate(); err != nil {
    // handle validation error
}
```

Checks: valid email, required ID, display name length ≤ 100. Trims whitespace on `Email` and `DisplayName` in-place.

### Domain Events

Optional `EventHandler` callback receives events for audit logging, analytics, or side effects:

```go
usermgmt.EventHandler(func(userID usermgmt.UserID, evt any) {
    switch e := evt.(type) {
    case usermgmt.UserRegisteredEvent:
        slog.Info("user registered", "email", e.Email)
    case usermgmt.RolesUpdatedEvent:
        slog.Info("roles updated", "roles", e.Roles, "domain", e.Domain)
    }
})
```

Events are panic-safe — handler errors are logged but never propagate to the caller.

### Authz (RBAC Engine)

Domain-aware Casbin RBAC with typed constants:

```go
authz, _ := usermgmt.NewAuthz()

// Check permission
allowed, _ := authz.Enforce(userID.String(), "acme", "documents", "read")

// Query roles
roles, _ := authz.RolesForUser(userID.String(), "acme")

// Atomic policy update
authz.Apply(usermgmt.PolicyUpdate{
    AddPolicies:    []usermgmt.Policy{{...}},
    RemovePolicies: []usermgmt.Policy{{...}},
    AddGroups:      []usermgmt.GroupPolicy{{...}},
    RemoveGroups:   []usermgmt.GroupPolicy{{...}},
})

// Bridge to parent cqrs-htmx Enforcer interface
enforcer := authz.AsEnforcer()
```

### Predefined Constants

| Type     | Values                                             |
| -------- | -------------------------------------------------- |
| `Action` | `ActionExecute`, `ActionRead`, `ActionAll`         |
| `Effect` | `EffectAllow`, `EffectDeny`                        |
| `Role`   | `RoleAdmin`, `RoleUser`, `RoleViewer`, `RoleOwner` |

### Account Lockout

Configurable failed-login tracking with automatic expiry:

```go
lockout := usermgmt.NewAccountLockout(usermgmt.LockoutConfig{
    MaxAttempts: 5,
    Duration:    15 * time.Minute,
})

// Integrated into Service via ServiceConfig.Lockout
// Automatic cleanup: lockout.EvictStale() removes expired entries
```

### Session Store

```go
type SessionStore interface {
    Create(ctx context.Context, userID UserID, ttl time.Duration) (*Session, error)
    Find(ctx context.Context, token string) (*Session, error)
    Delete(ctx context.Context, token string) error
    DeleteByUserID(ctx context.Context, userID UserID) error
}
```

Default: `InMemorySessionStore` (with `EvictExpired()` cleanup). Implement this interface for Redis, SQL, or other backends.

### CSRF & Rate Limiting on Auth Endpoints

Wire `cqrs-htmx` middleware around the usermgmt auth handler:

```go
handler := cqrshtmx.Chain(
    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{Secure: true}),
    cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
        Limit:        10,
        Window:       time.Minute,
        Burst:        3,
        KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
    }),
)(mux)
```

This protects WebAuthn ceremony endpoints from CSRF attacks and credential stuffing.

### Strongly-Typed UserID

All user IDs in the submodule use `usermgmt.UserID` — a branded type via [`go-branded-id`](https://github.com/larsartmann/go-branded-id) (`brandid.ID[userBrand, string]`):

```go
uid := usermgmt.NewUserID("01HK1549P84T9XF8R94E960633")
uid.String() // "User:01HK1549P84T9XF8R94E960633" (brand-prefixed)
uid.Get()    // "01HK1549P84T9XF8R94E960633"       (raw ULID, for cqrs-htmx bridge)
```

### User Context

```go
// Set user in context (done by SessionMiddleware)
ctx := usermgmt.WithUser(r.Context(), user)

// Retrieve
user, ok := usermgmt.UserFromContext(ctx)
user = usermgmt.UserFromContextOr(ctx, fallback)

// Bridge to cqrs-htmx UserIDExtractor
func extractor(r *http.Request) (cqrshtmx.UserID, error) {
    return cqrshtmx.MustParseUserID(usermgmt.UserIDFromRequest(r)), nil // reads from context, returns string
}
```

## Architecture

```
cqrs-htmx/
├── app.go              # App builder, Config, Command(), Query(), lifecycle hooks
├── handler.go          # Command/query dispatch handlers
├── options_types.go    # HandlerOption type, TemplComponent interface
├── options_decode.go   # DecodeJSON[T], DecodeJSONQuery[T], DecodeForm[T] decoders
├── options_htmx.go     # HTMX-aware error handling, response application
├── options_json.go     # RenderJSON[T], RenderJSONStatus[T], pagination, RequireMethod
├── options_render.go   # Render, RenderTempl, Redirect, Trigger, PushURL
├── options_validate.go # ValidateCommand, ValidateQuery, WithTimeout, WithMaxBodySize
├── response.go         # HTMX response builder (fluent API), ContentType constants
├── authz.go            # Enforcer interface, Authorize, RequireAuth, AuthorizeMiddleware
├── context.go          # UserID, CorrelationID, RequestID — strongly-typed context helpers
├── errors.go           # CQRS error → HTTP status mapping, sentinels, error handlers
├── htmx.go             # HTMXRequest struct, accessors, swap strategies
├── htmx_embed.go       # Embedded HTMX v2.0.9 JS (minified)
├── htmx_serve.go       # HTMXScriptHandler, HTMXScriptTag, HTMXVersion
├── notify.go           # Notification HandlerOptions, NotifyWithEvent builder
├── middleware.go        # ContextEnrichmentMiddleware, HTMXMiddleware, Chain
├── csrf_config.go      # CSRFConfig, defaults, Validate()
├── csrf_context.go     # Token context helpers (get/set)
├── csrf_middleware.go  # CSRFMiddleware, double-submit validation
├── csrf_handler.go     # CSRFProtect (per-handler CSRF)
├── csrf_helpers.go     # CSRFTokenHTMLMeta, CSRFTokenHXHeaders, CSRFTokenFormField
├── decoder.go          # Body reading, form/JSON decoding, MaxBodySize
├── httputil.go         # WriteJSON, ClientIP (delegates to larsartmann/httputil)
├── logging.go          # RequestLogging, RequestLoggingSlog, StatusRecorder
├── ratelimit_config.go     # RateLimiterConfig
├── ratelimit_middleware.go # RateLimiterMiddleware, token bucket, min-heap eviction
├── security.go         # SecurityHeadersMiddleware, SecurityHeadersConfig
├── recovery.go         # RecoveryMiddleware (package-level), App.RecoverHandler()
├── sse_broadcaster.go  # Thread-safe fan-out Broadcaster (O(1) unsubscribe)
├── sse_event.go        # SSEEvent protocol type, WriteSSEEvent, splitSSELines
├── sse_store.go        # SSEEventStore interface for reconnection replay
├── sse_stream.go       # SSEStream, BroadcastOnSuccess CQRS bridge
├── ws.go               # WebSocket message parser, OOB HTML, typed generic parser
├── usermgmt/           # User management submodule (independent Go module)
│   ├── id.go               # Branded UserID type (go-branded-id)
│   ├── authz_types.go      # Authz wrapper, AsEnforcer bridge
│   ├── authz_policies.go   # Apply, AddGroupPolicy, AddPolicy
│   ├── authz_roles.go      # RolesForUser, ImplicitRolesForUser
│   ├── es_constants.go     # Event-sourced aggregate + event + command type constants
│   ├── es_events.go        # 7 event payload structs
│   ├── es_commands.go      # 7 command structs
│   ├── es_state.go         # UserState + foldUser() pure function
│   ├── es_decide.go        # 7 pure decide functions
│   ├── es_dispatch.go      # RegisterCommands — wires commands to decider.Repository
│   ├── es_setup.go         # DefaultEventSourcedSetup, UserDecider
│   ├── es_readmodel.go     # UserReadModel projection + email index
│   ├── es_casbin_projection.go  # CasbinProjection — derives policies from events
│   ├── es_projection_setup.go   # StartProjections orchestration
│   ├── service_core.go     # Service struct, ServiceConfig, NewService
│   ├── service_register.go # Service.Register
│   ├── service_login.go    # Service.Logout/Authenticate/Authorize
│   ├── service_misc.go     # GetUser, UpdateRoles, ChangeEmail, etc.
│   ├── credential.go       # WebAuthnCredential type
│   ├── credential_http.go  # Credential listing/removal HTTP handlers
│   ├── webauthn_service.go # BeginRegistration/FinishRegistration/BeginLogin/FinishLogin
│   ├── webauthn_http.go    # HTTP handlers for WebAuthn ceremony endpoints
│   ├── webauthn_adapter.go # Adapts domain User → webauthn.User interface
│   ├── webauthn_session.go # WebAuthnConfig + in-memory challenge store
│   ├── user.go             # Immutable User read model
│   ├── store.go            # SessionStore interface + InMemorySessionStore
│   ├── events.go           # EventHandler callback + notification event structs
│   ├── http.go             # AuthHandler (register, logout, me, webauthn endpoints)
│   ├── middleware.go       # NewSessionMiddleware, user context helpers
│   ├── lockout.go          # AccountLockout (configurable attempts + duration)
│   └── errors.go           # Sentinel errors
├── catalog/             # API documentation generation (opt-in, 5th Go module)
├── integration_test/   # Cross-module integration tests (independent Go module)
└── examples/
    └── datastar-demo/  # Standalone datastar + go-cqrs-lite SSE example
```

## Optional Sub-Packages

### catalog — API Documentation Generation

Generate OpenAPI, AsyncAPI, D2 diagrams, and EventCatalog docs from your Go CQRS types:

```bash
go get github.com/larsartmann/cqrs-htmx/catalog/v2
```

```go
b := cataloghtmx.New("User Service", "1.0.0")
cataloghtmx.Command[RegisterUserCmd](b, "register-user")
cat := b.Build()
mux.Handle("/docs/openapi.json", cataloghtmx.OpenAPIHandler(cat))
```

See [catalog/README.md](catalog/README.md) for full documentation.

## Dependencies

| Dependency             | Purpose                                    |
| ---------------------- | ------------------------------------------ |
| go-cqrs-lite v2.6.0    | CQRS command/query dispatch, pagination    |
| casbin/casbin/v3       | Authorization                              |
| go-error-family v0.4.0 | Error classification                       |
| justinas/nosurf        | CSRF protection                            |
| larsartmann/httputil   | ClientIP extraction                        |
| golang.org/x/time      | Token-bucket rate limiting                 |
| go-branded-id          | Branded types (usermgmt)                   |
| go-webauthn v0.17.4    | WebAuthn/Passkey authentication (usermgmt) |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for build/test/lint commands, code style, and PR checklist.

## License

MIT
