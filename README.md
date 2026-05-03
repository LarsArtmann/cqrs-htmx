# cqrs-htmx

A Go library that makes it **very easy** to use [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) with [HTMX](https://htmx.org) and [Casbin](https://casbin.org) authorization.

## Why

Combining CQRS, HTMX, and authorization requires repetitive wiring:

- HTTP requests → CQRS command/query dispatch
- User identity from auth → event metadata
- Casbin policy checks before dispatch
- CQRS errors → HTTP status codes (HTMX-aware)
- HTMX partial vs full-page rendering
- HTMX response headers (triggers, redirects, push URL)

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
        cqrshtmx.Trigger("userCreated"),
        cqrshtmx.PushURL("/users"),
    ))

    // Query dispatch with render
    mux.Handle("GET /users", app.Query("ListUsers",
        cqrshtmx.Authorize("users", "read"),
        cqrshtmx.DecodeJSONQuery(func(req ListUsersRequest) (query.Query, error) {
            return &ListUsersQuery{}, nil
        }),
        cqrshtmx.Render(func(w http.ResponseWriter, r *http.Request, result any) error {
            return renderUserList(w, r, result)
        }),
    ))

    // Apply context enrichment middleware once
    http.ListenAndServe(":8080", app.Middleware()(mux))
}
```

## Handler Options

### Authorization

| Option | Description |
|--------|-------------|
| `Authorize(resource, action)` | Check Casbin policy before dispatch |
| `RequireAuth()` | Require authenticated user (no specific permission) |

### Request Decoding

| Option | Description |
|--------|-------------|
| `DecodeJSON[T](mapper)` | Decode JSON body into a command via mapper function |
| `DecodeJSONQuery[T](mapper)` | Decode JSON body into a query via mapper function |
| `DecodeForm[T](mapper)` | Decode form data into a command via mapper function |

### Response

| Option | Description |
|--------|-------------|
| `Render(fn)` | Render query result with custom function |
| `Redirect(url)` | Redirect after success (HX-Redirect for HTMX, HTTP redirect otherwise) |
| `Trigger(event)` | Fire HTMX client-side event on success |
| `TriggerWithDetail(event, detail)` | Fire HTMX event with JSON detail data |
| `PushURL(url)` | Push URL into browser history |

## HTMX Response Builder

Build HTMX-aware responses with fluent chaining:

```go
resp := cqrshtmx.NewResponse(w, r)
resp.Trigger("userCreated").
    PushURL("/users/1").
    Retarget("#user-list").
    Reswap(cqrshtmx.SwapInnerHTML).
    Apply()
```

### Methods

| Method | Header Set |
|--------|-----------|
| `PushURL(url)` | `HX-Push-Url` |
| `ReplaceURL(url)` | `HX-Replace-Url` |
| `Redirect(url)` | `HX-Redirect` (HTMX) or HTTP redirect |
| `Refresh()` | `HX-Refresh` |
| `Reswap(strategy)` | `HX-Reswap` |
| `Retarget(selector)` | `HX-Retarget` |
| `Reselect(selector)` | `HX-Reselect` |
| `Trigger(event)` | `HX-Trigger` |
| `TriggerAfterSwap(event)` | `HX-Trigger-After-Swap` |
| `TriggerAfterSettle(event)` | `HX-Trigger-After-Settle` |
| `TriggerWithDetail(name, detail)` | `HX-Trigger` (JSON) |

### Swap Strategies

`SwapInnerHTML`, `SwapOuterHTML`, `SwapBeforeBegin`, `SwapAfterBegin`, `SwapBeforeEnd`, `SwapAfterEnd`, `SwapDelete`, `SwapNone`

## Request Detection

```go
if cqrshtmx.IsHTMXRequest(r) {
    // Return partial HTML
} else {
    // Return full page
}

target := cqrshtmx.HTMXTarget(r)     // Target element ID
trigger := cqrshtmx.HTMXTrigger(r)   // Trigger element ID
prompt := cqrshtmx.HTMXPrompt(r)     // hx-prompt response
url := cqrshtmx.HTMXCurrentURL(r)    // Current page URL
```

## Context Propagation

User identity flows automatically from HTTP → CQRS metadata:

```go
// Set by App.Middleware() or manually
ctx := cqrshtmx.WithUserID(r.Context(), "user-123")

// Retrieve in CQRS handlers
userID := cqrshtmx.UserIDFromContext(ctx)

// Build event options from context
opts := cqrshtmx.EventOptionsFromContext(ctx)
evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, payload, opts...)
```

## Error Mapping

CQRS error families automatically map to HTTP status codes:

| CQRS Family | HTTP Status |
|-------------|-------------|
| Rejection | 400 Bad Request |
| Conflict | 409 Conflict |
| Corruption | 422 Unprocessable Entity |
| Transient | 503 Service Unavailable |
| Infrastructure | 500 Internal Server Error |

Auth errors map specially:
- `ErrUnauthorized` → 401
- `ErrForbidden` → 403

For HTMX requests with auth errors, `DefaultErrorHandler` sets `HX-Redirect: /login` instead of returning an error body.

## Middleware

```go
// Context enrichment (applied once to your router)
mux := app.Middleware()(router)

// Standalone Casbin authorization middleware
mux.Handle("/admin", cqrshtmx.AuthorizeMiddleware(
    enforcer, "admin", "access",
    userIDExtractor,
)(handler))

// Chain multiple middleware
chained := cqrshtmx.Chain(mw1, mw2, mw3)(finalHandler)
```

## License

MIT
