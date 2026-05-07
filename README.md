# cqrs-htmx

A Go library that makes it **very easy** to use [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) with [HTMX](https://htmx.org), [templ](https://templ.guide), and [Casbin](https://casbin.org) authorization.

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

All use the configurable `NotificationEvent` (default: `"showMessage"`). Client-side JS:

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

User identity flows automatically from HTTP → CQRS metadata:

```go
// Set by App.Middleware() or manually
userID := cqrshtmx.MustParseUserID("01HK1549P84T9XF8R94E960633")
ctx := cqrshtmx.WithUserID(r.Context(), userID)

// Retrieve in CQRS handlers
retrieved := cqrshtmx.UserIDFromContext(ctx)

// Build event options from context
opts := cqrshtmx.EventOptionsFromContext(ctx)
evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, payload, opts...)
```

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

For HTMX requests with auth errors, `DefaultErrorHandler` sets `HX-Redirect` to the configured login path (default: `/login`) instead of returning an error body. Configure per-App:

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

## License

MIT
