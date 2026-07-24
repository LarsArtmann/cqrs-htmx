---
name: cqrs-htmx
description: Build Go web apps with the cqrs-htmx library — CQRS command/query HTTP handlers, HTMX responses, event-sourced user management (WebAuthn/OAuth2/TOTP), and a ready-made admin UI. Use this skill whenever integrating, wiring, mounting, or extending cqrs-htmx, go-cqrs-lite with HTMX, the usermgmt or adminui submodules, or building CQRS/HTMX/SSE/WebSocket/auth features on top of this library — even when the user does not name the library explicitly (e.g. "add a command endpoint", "wire up passkey login", "add an admin panel", "serve HTMX", "broadcast over SSE", "set up CQRS dispatch").
user-invocable: true
---

# Using cqrs-htmx superbly

cqrs-htmx is a **Go library** (not an app) that wires [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) CQRS dispatch to `net/http` with first-class HTMX support, plus an event-sourced user-management submodule and a ready-made admin dashboard. Consumers import it; there is no "main app" to run except the examples.

It is framework-agnostic: it works with `net/http`, Chi, Gin, etc. and never picks your router for you.

> **Related skill:** the [`go-cqrs-lite`](https://github.com/LarsArtmann/go-cqrs-lite) skill covers the underlying CQRS/ES building blocks in depth — deciders, event stores, projections, read models, snapshots, schema evolution, signing, encryption. Consult it when a question is about the CQRS core rather than the HTTP/HTMX layer.

## Cheat sheet (the 5 most common operations)

```go
// 1. Serve embedded htmx.js (no CDN)
mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())

// 2. Create a command endpoint (JSON POST -> dispatch -> 201)
mux.Handle("POST /items", app.Command("CreateItem",
    cqrshtmx.DecodeJSON(mapToCmd), cqrshtmx.WithSuccessStatus(201)))

// 3. SSE: stream events to browsers
broadcaster := cqrshtmx.NewBroadcaster()
broadcaster.Broadcast(evt) // -> all connected clients

// 4. Chain middleware (outer -> inner)
cqrshtmx.Chain(cqrshtmx.RecoveryMiddleware, cqrshtmx.SecurityHeadersMiddleware, cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}), app.Middleware())(mux)

// 5. Map errors -> HTTP status (one function)
status := cqrshtmx.MapError(err) // Rejection->400, Conflict->409, Transient->503, etc.
```

## The modules

An app typically composes some subset of these. They are **independent Go modules** with `/v4` suffixes.

| Module                | Import path                                              | Provides                                                                                                                                                                                               |
| --------------------- | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **root**              | `github.com/larsartmann/cqrs-htmx/v4` (alias `cqrshtmx`) | The `App` builder, `HandlerOption`s, middleware (CSRF/HTMX/recovery/security/rate-limit), context IDs, error->HTTP mapping, SSE + WebSocket, embedded HTMX JS                                          |
| **usermgmt**          | `github.com/larsartmann/cqrs-htmx/usermgmt/v4`           | Event-sourced `Service` (register/login/logout/me), roles/tenants/bots, authz via Casbin, session middleware, SQL + in-memory stores. Auth strategies (WebAuthn/OAuth2/TOTP) are optional sub-modules. |
| **usermgmt/totp**     | `github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4`      | TOTP MFA (pquerna/otp). Inject `totp.New(...)` as `ServiceConfig.TOTP`.                                                                                                                                |
| **usermgmt/webauthn** | `github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4`  | WebAuthn passkeys (go-webauthn). Inject `webauthn.New(...)` as `ServiceConfig.WebAuthn`.                                                                                                               |
| **usermgmt/oauth2**   | `github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4`    | OAuth2/OIDC login (oauth2+oidc). Inject `oauth2.New(...)` as `ServiceConfig.OAuth2`.                                                                                                                   |
| **adminui**           | `github.com/larsartmann/cqrs-htmx/adminui/v4`            | One-call admin dashboard (templ + HTMX). Depends on root + usermgmt                                                                                                                                    |

The core CQRS building blocks come from **go-cqrs-lite**, imported per-package:

```go
"github.com/larsartmann/go-cqrs-lite/command/v4"
"github.com/larsartmann/go-cqrs-lite/query/v4"
"github.com/larsartmann/go-cqrs-lite/event/v4"
"github.com/larsartmann/go-cqrs-lite/id/v4"
```

### v3 vs v4: which version should I use?

The skill targets **v4**. If you're on v3, the migration depends on what you use:

- **Root module only (no auth)?** v3->v4 is trivial: change import paths `/v3` -> `/v4`, run `go mod tidy`. The root module API (SSE, middleware, errors, HTMX, `App.Command/Query`) is **unchanged** between v3 and v4.
- **Using usermgmt auth strategies (TOTP/WebAuthn/OAuth2)?** v4 moved these to separate modules behind interfaces. See `docs/migrations/v3-to-v4.md`.

All features documented in this skill have been available since v3.3.0+.

## Step 1: Pick the integration path

```
Do you need CQRS dispatch (command/query endpoints)?
+- NO  ->  PATH 0: Building blocks only (middleware, SSE, errors, HTMX -- no App)
+- YES ->  Does the app need user accounts / authentication?
          +- NO  ->  PATH A: root App only (CQRS + HTMX endpoints)
          +- YES ->  Does the user want a ready-made admin dashboard?
                    +- NO  ->  PATH B: root App + usermgmt Service + AuthHandler
                    +- YES ->  PATH C: usermgmt Service + adminui panel (+ root App for custom endpoints)

Realtime (SSE/WS)? -> Use Broadcaster/SSEStream on ANY path.
Persistence? -> Pass *sql.DB or use NewSQLiteEventSourcedSetup on Path B/C.
```

## Path 0 -- Building blocks only (no App)

Many consumers use cqrs-htmx purely for its middleware, SSE building blocks, and error mapping -- without the `App` command/query pipeline.

```go
mux := http.NewServeMux()
mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())

broadcaster := cqrshtmx.NewBroadcaster()
defer broadcaster.Close() // graceful SSE drain on shutdown

mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
    stream := cqrshtmx.NewSSEStream(w, r)
    defer stream.Close()
    ch := broadcaster.Subscribe()
    defer broadcaster.Unsubscribe(ch)
    stream.Send(cqrshtmx.SSEEvent{Event: cqrshtmx.SSEEventConnected, Data: "ok"})
    for {
        select {
        case <-stream.Context().Done(): return
        case evt, ok := <-ch:
            if !ok || stream.Send(evt) != nil { return }
        }
    }
})

http.ListenAndServe(":8080", cqrshtmx.Chain(
    cqrshtmx.RecoveryMiddleware,
    cqrshtmx.SecurityHeadersMiddleware,
    cqrshtmx.RateLimiterMiddleware(cqrshtmx.DefaultRateLimiterConfig()),
    cqrshtmx.RequestLoggingSlog(logger),
)(mux))
```

Available without `App`: `Chain`, `RecoveryMiddleware`, `SecurityHeadersMiddleware` (+`SecurityHeaderSkip`), `RateLimiterMiddleware` (+`DefaultRateLimiterConfig`), `RequestLoggingSlog`, `CSRFMiddleware`, `HTMXScriptHandler`/`HTMXExtensionHandler`, `NewSSEStream`/`Broadcaster`/`JournalSSEStore`, `MapError`/`JSONErrorHandler`, `ServerTimingMiddleware`/`MeasureServerTiming`, `WriteJSON`, `IsHTMXRequest`/`RenderPartial`, `NewResponse(w, r)`.

## Path A -- CQRS + HTMX endpoints (root only)

The whole API is one `App`, configured once, that produces `http.HandlerFunc`s.

```go
cmdDisp := command.NewDispatcher()
qryDisp := query.NewDispatcher()
// Register handlers BEFORE building endpoints:
cmdDisp.Register("CreateItem", func(ctx context.Context, cmd command.Command) error { /* ... */ })
qryDisp.Register("ListItems", func(ctx context.Context, q query.Query) (any, error) { /* ... */ })

app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp, Queries: qryDisp})
```

**What is a `command.Command`?** It is an interface from go-cqrs-lite with three methods: `Type() command.Type`, `AggregateID() id.AggregateID`, `ID() id.CommandID`. Your command struct must implement all three:

```go
type createItemCmd struct {
    aggID id.AggregateID
    cmdID id.CommandID
    Name  string
}
func (c *createItemCmd) Type() command.Type          { return "CreateItem" }
func (c *createItemCmd) AggregateID() id.AggregateID { return c.aggID }
func (c *createItemCmd) ID() id.CommandID            { return c.cmdID }
```

`query.Query` is the same pattern: just `Type() query.Type`.

### Command endpoint: POST JSON -> dispatch -> response

```go
mux.Handle("POST /items", app.Command("CreateItem",
    cqrshtmx.DecodeJSON(func(req createItemRequest) (command.Command, error) {
        return &createItemCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID(), Name: req.Name}, nil
    }),
    cqrshtmx.WithSuccessStatus(201),
    cqrshtmx.PushURL("/items"), // HTMX: update the address bar
))
```

### Query endpoint: GET -> dispatch -> JSON

```go
mux.Handle("GET /items", app.Query("ListItems",
    cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
        return &listItemsQuery{}, nil // GET with no body -> zero-value T (works!)
    }),
    cqrshtmx.RenderJSON[[]item](),
))
```

For GET endpoints with query params (`?filter=foo&page=2`), use the request-aware variant:

```go
mux.Handle("GET /items", app.Query("ListItems",
    cqrshtmx.DecodeFormQueryWithRequest(func(r *http.Request, _ struct{}) (query.Query, error) {
        return &listItemsQuery{Filter: r.URL.Query().Get("filter")}, nil
    }),
    cqrshtmx.RenderPaginatedJSON[item](), // pair with DecodePagination(r) for page/page_size
))
```

### Form-based (SSR / HTMX): POST form -> dispatch -> redirect

```go
mux.Handle("POST /items", app.Command("CreateItem",
    cqrshtmx.DecodeForm(func(req createItemForm) (command.Command, error) {
        return &createItemCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID(), Name: req.Name}, nil
    }),
    cqrshtmx.PushURL("/items"),
    cqrshtmx.NotifySuccess("Item created"), // HX-Trigger toast
))
```

For HTML rendering: `cqrshtmx.RenderTempl(component)` (templ), `cqrshtmx.RenderTemplResult[T](mapper)` (result -> templ), or `cqrshtmx.RenderHTML("<div>...</div>")` (static HTML).

### Typed endpoints (no mapper, no type assertion)

`CommandTyped[Q]` and `QueryTyped[Q, R]` eliminate the mapper function and manual type assertions. The type `Q` itself implements `command.Command` or `query.Query`, and the decoder populates it directly from the request body:

```go
// 1. Define a type that implements command.Command directly.
type greetCmd struct {
    aggID id.AggregateID
    cmdID id.CommandID
    Name  string `json:"name"`
}
func (c *greetCmd) Type() command.Type          { return "Greet" }
func (c *greetCmd) AggregateID() id.AggregateID { return c.aggID }
func (c *greetCmd) ID() id.CommandID            { return c.cmdID }

// 2. Register with command.RegisterTyped (typed handler, no type assertion).
_ = command.RegisterTyped(cmdDisp, "Greet",
    func(_ context.Context, cmd *greetCmd) error {
        db.add("Hello, " + cmd.Name + "!")
        return nil
    })

// 3. Wire the endpoint with CommandTyped + DecodeJSONTyped.
mux.Handle("POST /api/greet", cqrshtmx.CommandTyped[*greetCmd](
    app, "Greet",
    cqrshtmx.DecodeJSONTyped[*greetCmd](),
    cqrshtmx.WithSuccessStatus(201),
))
```

Queries work the same way — the type implements `query.Query` and the handler returns a concrete result type:

```go
type sumQuery struct {
    A int `json:"a"`
    B int `json:"b"`
}
func (q *sumQuery) Type() query.Type { return "Sum" }

_ = query.RegisterTyped(qryDisp, "Sum",
    func(_ context.Context, q *sumQuery) (int, error) {
        return q.A + q.B, nil
    })

mux.Handle("POST /api/sum", cqrshtmx.QueryTyped[*sumQuery, int](
    app, "Sum",
    cqrshtmx.DecodeJSONQueryTyped[*sumQuery](),
    cqrshtmx.RenderJSON[int](),
))
```

**Available typed decoders:** `DecodeJSONTyped[Q]()` (JSON body), `DecodeFormTyped[Q]()` (form body), `DecodeJSONQueryTyped[Q]()` (JSON query), `DecodeFormQueryTyped[Q]()` (form query).

> **Gotcha:** `CommandTyped`/`QueryTyped` are package-level functions, not methods — Go does not allow generic methods on receiver types.

**Partial-vs-full rendering (HTMX)** — eliminates `if HX-Request { partial } else { full }` boilerplate:

```go
// Generic typed version for templ — auto-selects based on RenderPartial(r)
app.Query("ListUsers", decoder,
    cqrshtmx.RenderPartialOrFull(
        func(users []*User) cqrshtmx.TemplComponent { return userListPartial(users) },
        func(users []*User) cqrshtmx.TemplComponent { return usersPage(users) },
    ),
)

// Non-generic for non-templ renderers (html/template, raw strings)
app.Query("ListUsers", decoder,
    cqrshtmx.RenderPartialOrFullFunc(partialRenderFunc, fullRenderFunc),
)

// Standalone for non-CQRS routes (net/http handlers)
func(w http.ResponseWriter, r *http.Request) {
    _ = cqrshtmx.RenderTemplComponent(w, r, itemsPartial(data), itemsPage(data))
}

// Custom predicate (not just partial-vs-full)
cqrshtmx.RenderIf(func(r *http.Request) bool { return cqrshtmx.HTMXTarget(r) == "#avatar" }, avatarRender, fullRender)
```

All use `cqrshtmx.RenderPartial(r)` internally (history-restore requests render full page). HTML-specific helpers set `Content-Type: text/html`; generic `RenderIf`/`RenderPartialOrFullFunc` do not (see ADR-0037).

**OOB swap wrapping:** `cqrshtmx.OOBHTML(id, html, swapStrategy...)` wraps HTML with `hx-swap-oob` attributes. `WSOOBHTML` delegates to it.

### Custom auth (cookie-based, API keys, ownership checks)

The `App` pipeline defaults to Casbin-based auth (`Authorize`). For custom auth models, use `DecodeJSONWithRequest` + `RequestGuard`:

```go
mux.Handle("POST /games/{id}/delete", app.Command("DeleteGame",
    cqrshtmx.DecodeJSONWithRequest(func(r *http.Request, body deleteGameReq) (command.Command, error) {
        playerID := extractPlayerIDFromCookie(r) // read cookies in the mapper
        return &deleteGameCmd{PlayerID: playerID, GameID: body.GameID}, nil
    }),
    cqrshtmx.RequestGuard(func(r *http.Request, cmd any) error {
        gc := cmd.(*deleteGameCmd)
        if !ownsGame(extractPlayerID(r), gc.GameID) { return cqrshtmx.ErrForbidden }
        return nil
    }),
))
```

`RequestGuard` runs **after decode, before dispatch** -- has access to the decoded command/query.

> **Auth error sentinels -> HTTP status:** `cqrshtmx.ErrUnauthorized` -> 401, `cqrshtmx.ErrForbidden` -> 403, `cqrshtmx.ErrValidation` -> 400, `cqrshtmx.ErrRequestTooLarge` -> 413, `cqrshtmx.ErrMethodNotAllowed` -> 405.

### Middleware + serving htmx.js

```go
mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler()) // embedded htmx 2.0.10, ETag + 1yr cache

http.ListenAndServe(":8080", cqrshtmx.Chain(
    cqrshtmx.RecoveryMiddleware,
    cqrshtmx.SecurityHeadersMiddleware,
    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}), // protects mutations; opt-in
    cqrshtmx.HTMXMiddleware,                        // detects HTMX requests
    app.Middleware(),                                // enriches context (user/correlation/request IDs)
)(mux))
```

> **templ-components note:** `templ-components`'s `layout.Base` may generate CDN `<script>` tags for htmx. If you self-host via `HTMXScriptHandler`, override `PageProps.HTMXCDN` or remove the CDN tag -- don't load htmx twice.

## Path B -- add user accounts (root + usermgmt)

usermgmt gives you a fully event-sourced `Service` and an `AuthHandler` that mounts all auth routes. Authentication is **passwordless** (WebAuthn passkeys by default; OAuth2/OIDC and TOTP optional).

```go
svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
    // WebAuthn: passkeyProvider, // import usermgmt/webauthn, inject as ServiceConfig.WebAuthn
})
if err != nil { panic(err) }
defer svc.Close()

auth := usermgmt.NewAuthHandler(svc)
mux := http.NewServeMux()
auth.RegisterRoutes(mux) // registers /auth/register, /auth/webauthn/*, /auth/logout, /auth/me, ...

// Your own CQRS app (remember to Register handlers on the dispatchers!):
cmdDisp := command.NewDispatcher()
cmdDisp.Register("CreateWidget", widgetHandler) // <- don't forget this
app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp})
mux.Handle("POST /widgets", app.Command("CreateWidget", cqrshtmx.DecodeJSON(/*...*/)))

// Session goes OUTSIDE CSRF:
sessionMW := usermgmt.NewSessionMiddleware(svc, "session")
http.ListenAndServe(":8080", cqrshtmx.Chain(
    cqrshtmx.RecoveryMiddleware,
    cqrshtmx.SecurityHeadersMiddleware,
    sessionMW,                                    // authenticate the cookie first
    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
    cqrshtmx.HTMXMiddleware,
    app.Middleware(),
)(mux))
```

Read `references/usermgmt.md` for: the full setup matrix (in-memory / SQLite / Postgres / event signing), enabling WebAuthn/OAuth2/TOTP/email-verification, role & tenant management, and the `Service` write/read API.

## Path C -- add the ready-made admin dashboard (adminui)

```go
panel, err := adminui.New(adminui.Config{
    Service:     svc,             // *usermgmt.Service -- required
    Title:       "MyApp Admin",
    AccentColor: "#0ea5e9",
    LogoutURL:   "/logout",
    SSEURL:      "/admin/-/events", // enables honest-UI sync indicator (optional)
})
panel.Mount(mux, "/admin/")
// The panel serves htmx.js internally -- do NOT register HTMXScriptHandler yourself on Path C.
```

`adminui.Config` fields that matter: `Mode` (`ModeSuperAdmin` = global; `ModeTenantAdmin` = scoped to one `TenantID`), `Authorizer` (default = role-based), `BasePath`. The panel reads `*usermgmt.User` from context -- your session middleware must put it there.

> See `examples/admin-demo/` in the repo: a full runnable showcase (`go run .`, open `:8097`).

## The composition model (the heart of the library)

`App.Command` / `App.Query` + a list of `HandlerOption`s. The options run as a pipeline. **Order matters:**

```go
mux.Handle("POST /widgets", app.Command("CreateWidget",
    cqrshtmx.Authorize("widgets", "create"),        // (1) authz -- gate first
    cqrshtmx.DecodeJSON(mapToCommand),               // (2) decode body -> command
    cqrshtmx.ValidateCommand(func(c) error { ... }),  // (3) validate -- MUST come after decode
    cqrshtmx.RequestGuard(func(r, cmd) error {...}), // (3b) custom auth -- also after decode
    cqrshtmx.WithSuccessStatus(201),                 // (4) config
    cqrshtmx.NotifySuccess("Widget created"),        // (5) side effects / response
    cqrshtmx.PushURL("/widgets"),
))
```

**The two ordering rules:** (1) `Authorize`/`RequireAuth` first (fail fast before touching the body). (2) `ValidateCommand`/`RequestGuard` MUST come after the decoder -- they read the decoded value from context.

**Middleware order** (outer -> inner): recovery & security headers first; then your **session** middleware; then **CSRF**; then **HTMX**; then `app.Middleware()` (context enrichment). The non-negotiable trio is `CSRF -> HTMX -> enrichment`.

See `references/core-api.md` for the full `HandlerOption` catalogue.

## Errors and the Response builder

### MapError: errors -> HTTP status (one function)

```go
status := cqrshtmx.MapError(err) // Rejection->400, NotFound->404, Conflict->409, Transient->503, Corruption->422, Infrastructure->500
```

Set `Config.ErrorHandler: cqrshtmx.JSONErrorHandler` to auto-map errors to JSON responses:

```json
{ "error": "battle already exists", "status": 409, "code": "battle.exists" }
```

The `"code"` field carries the machine-readable error code from the innermost errorfamily error. For RFC 7807 (`application/problem+json`), use `cqrshtmx.ProblemDetailsErrorHandler`. For text/plain with HTMX redirects, use `DefaultErrorHandlerWithRedirect`.

### Response builder: fluent HTMX headers

For manual handlers (not using `App.Command/Query`):

```go
cqrshtmx.NewResponse(w, r).
    NotifySuccess("Saved!").      // HX-Trigger toast
    Redirect("/dashboard").        // HX-Redirect (sanitized)
    PushURL("/items/42").          // address bar update
    Retarget("#result").           // hx-retarget
    TriggerWithDetail("showError", msg). // custom event
    Apply()                        // writes all headers -- call once
```

### WriteJSON: buffered encoding

```go
cqrshtmx.WriteJSON(w, http.StatusOK, data) // buffers before WriteHeader -- encode failure doesn't commit partial status
```

## Realtime (SSE / WebSocket) -- on any path

Realtime is **building blocks, not a server**: you own the HTTP handler, the library gives you the stream + fan-out.

```go
broadcaster := cqrshtmx.NewBroadcaster()

// Bridge CQRS -> SSE (optional):
app := cqrshtmx.MustNew(cqrshtmx.Config{
    Commands:     cmdDisp,
    AfterDispatch: broadcaster.BroadcastOnSuccess("itemCreated", ""),
})

mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
    stream := cqrshtmx.NewSSEStream(w, r)
    defer stream.Close()
    ch := broadcaster.Subscribe()
    defer broadcaster.Unsubscribe(ch)
    for {
        select {
        case <-stream.Context().Done(): return
        case evt, ok := <-ch:
            if !ok || stream.Send(evt) != nil { return }
        }
    }
})
```

**SSE channel lifecycle:**

- `Subscribe()` returns a buffered channel (64 capacity). `Unsubscribe(ch)` closes the channel -- never send on it afterward.
- `Broadcaster.Close()` closes all subscriber channels and prevents new subscriptions. Call it on server shutdown for graceful SSE drain.
- `Broadcaster.SubscriberCount()` returns the current subscriber count (useful for metrics).
- Broadcasts are non-blocking -- slow consumers silently drop events. Pair with idempotency + ACK for guaranteed delivery.
- Standard event names: `cqrshtmx.SSEEventConnected` (`"connected"`), `cqrshtmx.SSEEventHeartbeat` (`"heartbeat"`).

WebSockets mirror this: `NewWSBroadcaster()`, `WSOOBHTML(...)`, `ParseWSMessageInto[T]`, `app.DispatchWSCommand(...)`. Read `references/realtime.md` for the ACK protocol, idempotency store, reconnection/replay, and heartbeat.

### Embedded HTMX extensions

The root module embeds 3 extensions that pair with cqrs-htmx's server-side building blocks: SSE (2.2.4), WS (2.0.4), idiomorph (0.7.4).

```go
// Individual or bundled:
mux.Handle("GET /ext/sse.js", cqrshtmx.HTMXExtensionHandler(cqrshtmx.HTMXExtSSE))
mux.Handle("GET /ext/bundle.js", cqrshtmx.HTMXExtensionsHandler(cqrshtmx.HTMXExtSSE, cqrshtmx.HTMXExtWS, cqrshtmx.HTMXExtIdiomorph))
```

Same caching as `HTMXScriptHandler` (ETag, `Cache-Control: 1yr immutable`, 304). Add `<script>` tags after htmx core.

## Top gotchas (the ones that bite)

These are the highest-frequency mistakes. Read `references/gotchas.md` for the full list.

1. **Module suffixes are mandatory.** v2+ Go modules require the major-version suffix: `github.com/larsartmann/cqrs-htmx/v4`, `.../usermgmt/v4`, `.../adminui/v4`.
2. **Two different UserID types.** Root `cqrshtmx.UserID` is ULID-backed (via `go-cqrs-lite/id`); usermgmt `usermgmt.UserID` is string-backed (`go-branded-id`). They are **not** assignable to each other. Bridge with `.Get()` (raw value), never `.String()` (brand-prefixed).
3. **Validation comes AFTER the decoder** in the `HandlerOption` list. The validator reads the decoded value off the context -- reversing the order silently skips validation (logs a `slog.Warn`).
4. **Middleware trio order is `CSRF -> HTMX -> enrichment`.** Put your session middleware _outside_ (before) CSRF.
5. **No stdlib error constructors** (root + usermgmt + adminui). `errors.New`/`fmt.Errorf`/`errors.Join` are banned (enforced by `branching-flow errorfamily`). Use `event.New*/Wrap*/Wrapf/Newf` from `go-cqrs-lite/event/v4`.
6. **App.Command("") panics.** Empty command/query type strings are rejected at registration.
7. **Serve htmx.js yourself on Path A/B.** Register `cqrshtmx.HTMXScriptHandler()` on your mux. **Exception:** on Path C (adminui) the panel serves htmx.js internally -- do NOT register it yourself.
8. **Register your command/query handlers BEFORE building endpoints.** `cmdDisp.Register("CreateItem", handler)` must happen before `app.Command("CreateItem", ...)`.

### Discoverability notes (APIs that are easy to miss)

- **`CSRFResponseHeaderMiddleware`** -- sets the CSRF token in a response header (`X-CSRF-Token`) so HTMX can pick it up via `hx-headers`. The middleware trio is: `CSRFMiddleware` (validates) + `CSRFResponseHeaderMiddleware` (exposes token) + `app.Middleware()` (sets context). If you use CSRF with HTMX, you need all three.
- **`CSRFTestToken(mw)`** -- test helper that extracts a valid CSRF token through your middleware chain. nosurf uses token masking (the cookie value is NOT the valid header token); this helper handles the dance.
- **`ContextEnrichmentMiddleware(nil)`** -- nil extractor = auto-generate ULID request IDs. This is the zero-config path; `app.Middleware()` wraps this.
- **`IsHTMXRequest(r)` vs `RenderPartial(r)`** -- `IsHTMXRequest` checks `HX-Request: true` (any HTMX request). `RenderPartial` additionally excludes `HX-History-Restore-Request` (only "real" HTMX navigations that should render a partial).
- **`NewSSEEventID("")`** -- empty string auto-generates a ULID. This is the conventional way to mint event IDs.
- **`HealthHandler()`** -- returns `{"status":"ok"}` (200) or `{"status":"unhealthy","error":"..."}` (503).
- **`SecurityHeaderSkip`** (`"-"`) -- set on `ContentTypeOptions`/`FrameOptions`/`ReferrerPolicy` to suppress that default header entirely (empty string = use the default).
- **`cqrshtmx.Chain` vs `httputil.Chain`** -- `cqrshtmx.Chain` is the standard middleware composition helper for this library. If you also import `go-httputil`, its `Chain` has the same signature; pick one and use it consistently.
- **`cqrshtmx.EventCatalogHandler(catalog)`** -- serves an event schema catalog as immutable JSON (1-year cache, FNV-1a ETag). Use with `usermgmt.DefaultEventCatalog()` or build your own `cqrshtmx.NewEventCatalog()`. Mirrors `OpenAPISpecHandler` pattern. See `docs/guides/event-catalog-guide.md`.
- **`cqrshtmx.ProjectionStatusHandler(provider)`** -- serves live projection health as JSON (`no-cache`, per-request ETag). Pass any `cqrshtmx.ProjectionStatusProvider` (e.g. `*usermgmt.Service`). Returns 503 when provider is nil. See `docs/guides/projection-health-monitoring.md`.
- **`svc.RebuildProjection(ctx, name)`** -- stops the projection host, resets the named projection's checkpoint + read-model, creates a fresh host, and replays the entire journal. Available on both `*usermgmt.Service` and `*usermgmt.EventSourcedSetup`. See `docs/guides/rebuild-projection-runbook.md`.

## Where to look

- **`references/core-api.md`** -- full `Config`, every `HandlerOption`, middleware catalogue, context IDs, error->HTTP mapping.
- **`references/usermgmt.md`** -- Service setup matrix, auth endpoints, WebAuthn/OAuth2/TOTP, roles/tenants/bots, SQL persistence.
- **`references/realtime.md`** -- SSE + WebSocket, broadcaster, ACK protocol, idempotency, reconnection/replay, heartbeat, event filtering patterns.
- **`references/gotchas.md`** -- the complete consumer gotcha list with fixes.
- **Repo examples**: `examples/basic/` (minimal CQRS+HTMX+SSE), `examples/admin-demo/` (full admin showcase).
- **ADR docs**: `docs/adr/` -- the _why_ behind each design.

When the user asks for something not covered inline here, **read the matching reference file before improvising** -- it almost always contains the exact API and the ordering constraints.
