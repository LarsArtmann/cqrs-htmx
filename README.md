# cqrs-htmx

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/cqrs-htmx.svg)](https://pkg.go.dev/github.com/larsartmann/cqrs-htmx)
[![CI](https://github.com/LarsArtmann/cqrs-htmx/actions/workflows/ci.yml/badge.svg)](https://github.com/LarsArtmann/cqrs-htmx/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 1.26+](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)

A Go library that makes it **very easy** to use [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) with [HTMX](https://htmx.org), [templ](https://templ.guide), and [Casbin](https://casbin.org) authorization.

**Framework-agnostic** — works with `net/http`, [Chi](https://github.com/go-chi/chi), [Gin](https://github.com/gin-gonic/gin), or any `http.Handler`-compatible router.

## Features at a Glance

- **Command & Query dispatch** — `app.Command()` / `app.Query()` build `http.HandlerFunc` that decode, authorize, dispatch, and respond
- **Casbin authorization** — `Authorize(resource, action)` checks policies before dispatch; `RequireAuth()` for auth-only guards
- **HTMX-aware responses** — fluent response builder with `HX-Trigger`, `HX-Redirect`, `HX-Push-Url`, swap strategies, and notifications
- **templ integration** — duck-typed `TemplComponent` renders templ components without importing templ
- **User identity propagation** — HTTP request → context → CQRS event metadata, with strongly-typed `UserID` (ULID-backed)
- **Correlation IDs** — automatic `X-Correlation-ID` extraction and context propagation
- **Request validation** — `ValidateCommand`/`ValidateQuery` wrap decoders with validation logic
- **Dispatch timeout** — `Config.Timeout` wraps dispatch with `context.WithTimeout`
- **Lifecycle hooks** — `BeforeDispatchHook`/`AfterDispatchHook` for tracing, logging, metrics
- **Error classification** — CQRS error families automatically map to HTTP status codes
- **JSON or plain-text error handlers** — pluggable `ErrorHandler` with HTMX-aware auth redirects
- **Notification system** — `NotifySuccess`/`NotifyError`/`NotifyWarning`/`NotifyInfo` with standard `{level, message}` payload; custom event names via `NotifyWithEvent`

## Why

Combining CQRS, HTMX, and authorization requires repetitive wiring:

- HTTP requests → CQRS command/query dispatch
- User identity from auth → event metadata
- Casbin policy checks before dispatch
- CQRS errors → HTTP status codes (HTMX-aware)
- HTMX partial vs full-page rendering
- HTMX response headers (triggers, redirects, push URL)
- templ component rendering for query results

This library handles all of it automatically.

## Install

```bash
go get github.com/larsartmann/cqrs-htmx
```

## Quick Start

```go
package main

import (
    "net/http"

    cqrshtmx "github.com/larsartmann/cqrs-htmx"
    "github.com/larsartmann/go-cqrs-lite/core/command"
    "github.com/larsartmann/go-cqrs-lite/core/query"
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
        UserIDExtractor: func(r *http.Request) string {
            return r.Header.Get("X-User-ID")
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

    // Apply middleware once to your router
    handler := cqrshtmx.Chain(
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
    UserIDExtractor: extractFunc,      // func(r *http.Request) string
    LoginRedirect:   "/auth/signin",   // default: "/login"
    Timeout:         5 * time.Second,  // wraps dispatch only; 0 = no timeout
    ErrorHandler:    myErrorHandler,   // custom ErrorHandler; default uses plain text + LoginRedirect
    BeforeDispatch:  beforeHook,       // func(ctx, r) context.Context — tracing, request IDs
    AfterDispatch:   afterHook,        // func(ctx, r, err) — logging, metrics
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

User identity and correlation IDs flow automatically from HTTP → CQRS metadata:

```go
// User ID (strongly-typed, ULID-backed)
userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
ctx := cqrshtmx.WithUserID(r.Context(), userID)
retrieved := cqrshtmx.UserIDFromContext(ctx)

// Correlation ID (strongly-typed, ULID-backed, auto-extracted from X-Correlation-ID header)
cid := cqrshtmx.MustParseCorrelationID("01HK154ANGZHV2ZW0X3SKSNEN2")
ctx = cqrshtmx.WithCorrelationID(ctx, cid)
corrID := cqrshtmx.CorrelationIDFromContext(ctx)

// Build event options from context (includes user ID + correlation ID)
opts := cqrshtmx.EventOptionsFromContext(ctx)
evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, payload, opts...)

// Generate new random IDs
newID  := cqrshtmx.NewUserID()
newCID := cqrshtmx.NewCorrelationID()
```

### Why ULID?

Both `UserID` and `CorrelationID` are strongly-typed wrappers around [ULID](https://github.com/oklog/ulid). This guarantees:

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

| Handler                           | Format     | Login Redirect                              |
| --------------------------------- | ---------- | ------------------------------------------- |
| `DefaultErrorHandler`             | Plain text | `/login`                                    |
| `DefaultErrorHandlerWithRedirect` | Plain text | Custom                                      |
| `JSONErrorHandler`                | JSON       | `/login`                                    |
| `JSONErrorHandlerWithRedirect`    | JSON       | Custom                                      |
| `MapError(err)`                   | —          | Returns HTTP status code for any CQRS error |

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

// Standalone Casbin authorization middleware
mux.Handle("/admin", cqrshtmx.AuthorizeMiddleware(
    enforcer, "admin", "access",
    userIDExtractor,
)(handler))

// Chain multiple middleware
chained := cqrshtmx.Chain(
    cqrshtmx.HTMXMiddleware,
    app.Middleware(),
)(mux)
```

### Request Logging

Log every request with method, path, status, duration, and optional correlation/user IDs:

```go
handler := cqrshtmx.RequestLogging(nil, func(line string) {
    log.Println(line)
})(mux)
```

Use `JSONLogFormatter` for structured JSON output:

```go
handler := cqrshtmx.RequestLogging(cqrshtmx.JSONLogFormatter, func(line string) {
    log.Println(line)
})(mux)
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

**Per-IP rate limiting** — use `KeyExtractorFromRemoteAddr`. **Warning:** behind reverse proxies, `RemoteAddr` is the proxy's IP. Parse `X-Forwarded-For` or similar headers if your deployment uses proxies.

**Exempt requests** — return `""` from `KeyExtractor` to skip rate limiting for that request (useful for health checks or internal IPs).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for build/test/lint commands, code style, and PR checklist.

## License

MIT
