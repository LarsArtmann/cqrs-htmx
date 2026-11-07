---
name: cqrs-htmx
description: Build Go web apps with the cqrs-htmx library — CQRS command/query HTTP handlers, HTMX responses, event-sourced user management (WebAuthn/OAuth2/TOTP), and a ready-made admin UI. Use this skill whenever integrating, wiring, mounting, or extending cqrs-htmx, go-cqrs-lite with HTMX, the usermgmt or adminui submodules, or building CQRS/HTMX/SSE/auth features on top of this library — even when the user does not name the library explicitly (e.g. "add a command endpoint", "wire up passkey login", "add an admin panel", "serve HTMX", "broadcast over SSE", "set up CQRS dispatch"). WebSocket transport was removed in v5 — see ADR 0046; the library is SSE-only now.
user-invocable: true
---

# Using cqrs-htmx superbly

cqrs-htmx is a **Go library** (not an app) that wires [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) CQRS dispatch to `net/http` with first-class HTMX support, plus an event-sourced user-management submodule and a ready-made admin dashboard. Consumers import it; there is no "main app" to run except the examples.

It is framework-agnostic: it works with `net/http`, Chi, Gin, etc. and never picks your router for you.

> **Related skill:** the [`go-cqrs-lite`](https://github.com/LarsArtmann/go-cqrs-lite) skill covers the underlying CQRS/ES building blocks in depth — deciders, event stores, projections, read models, snapshots, schema evolution, signing, encryption. Consult it when a question is about the CQRS core rather than the HTTP/HTMX layer.

## Cheat sheet (the 5 most common operations)

```go
// 1. Serve embedded htmx.js (no CDN)
mux.Handle("GET /htmx.js", htmx.ScriptHandler())

// 2. Create a command endpoint (JSON POST -> dispatch -> 201)
mux.Handle("POST /items", app.Command("CreateItem",
    cqrshtmx.DecodeJSON(mapToCmd), cqrshtmx.WithSuccessStatus(201)))

// 3. SSE: stream events to browsers
broadcaster := cqrshtmx.NewBroadcaster()
broadcaster.Broadcast(evt) // -> all connected clients

// 4. Chain middleware (outer -> inner)
cqrshtmx.Chain(cqrshtmx.RecommendedSecurityMiddleware(), httputil.CSRFMiddleware(httputil.CSRFConfig{}), app.Middleware())(mux)

// 5. Map errors -> HTTP status (one function)
status := httperr.MapError(err) // Rejection->400, Conflict->409, Transient->503, etc.

// 6. Wire dispatch middleware (retry, circuit breaker, tracing, recovery)
cmdDisp.Use(middleware.CommandRecovery(), middleware.CommandRetry(middleware.DefaultRetryConfig()))
// See docs/guides/leveraging-go-cqrs-lite.md §1 for the full middleware recipe
```

## The modules

An app typically composes some subset of these. They are **independent Go modules** with `/v4` suffixes.

| Module                | Import path                                              | Provides                                                                                                                                                                                                                                                                                                                                                                 |
| --------------------- | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **root**              | `github.com/larsartmann/cqrs-htmx/v4` (alias `cqrshtmx`) | The `App` builder, `HandlerOption` values, middleware (HTMX/recovery/security; CSRF/rate-limit are deprecated httputil re-exports), context IDs, error->HTTP mapping, SSE + the `transport/` domain-event surface + ADR-0056 frontend sync (`SyncPullHandler`/`App.SyncPushHandler`), OpenAPI 3.1 builder, EventCatalog, embedded HTMX JS (WebSocket removed — ADR 0046) |
| **usermgmt**          | `github.com/larsartmann/cqrs-htmx/usermgmt/v4`           | Event-sourced `Service` (register/login/logout/me), roles/tenants/bots, authz via Casbin, session middleware, SQL + in-memory stores. Auth strategies (WebAuthn/OAuth2/TOTP) are optional sub-modules.                                                                                                                                                                   |
| **usermgmt/totp**     | `github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4`      | TOTP MFA (pquerna/otp). Inject `totp.New(...)` as `ServiceConfig.TOTP`.                                                                                                                                                                                                                                                                                                  |
| **usermgmt/webauthn** | `github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4`  | WebAuthn passkeys (go-webauthn). Inject `webauthn.New(...)` as `ServiceConfig.WebAuthn`.                                                                                                                                                                                                                                                                                 |
| **usermgmt/oauth2**   | `github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4`    | OAuth2/OIDC login (oauth2+oidc). Inject `oauth2.New(...)` as `ServiceConfig.OAuth2`.                                                                                                                                                                                                                                                                                     |
| **adminui**           | `github.com/larsartmann/cqrs-htmx/adminui/v4`            | Ready-made admin dashboard (templ + HTMX): **identity operations** (users/tenants/members/audit over a `*usermgmt.Service`; write actions, role-gated). Depends on root + usermgmt                                                                                                                                                                                       |
| **dashboardui**       | `github.com/larsartmann/cqrs-htmx/dashboardui/v4`        | Ready-made **event-store observability** dashboard (projection health, event catalog, live SSE; read-only by default). NOT user administration — see adminui. Depends on root + go-cqrs-lite only (NO usermgmt dep — verified against dashboardui/go.mod 2026-10-25)                                                                                                     |
| **loginpage**         | `github.com/larsartmann/cqrs-htmx/loginpage/v4`          | Ready-made passwordless login page (templ-components based: `layout.Base`, Tailwind-only styling, CSP-nonce + cookie-theme hooks). Third sibling UI panel.                                                                                                                                                                                                               |
| **setup**             | `github.com/larsartmann/cqrs-htmx/setup/v4`              | One-call composition root: builds shared stores + `usermgmt.Service` + all three UI panels; mounts all routes with correct middleware ordering.                                                                                                                                                                                                                          |
| **datastar**          | `github.com/larsartmann/cqrs-htmx/datastar/v4`           | Optional DataStar adapter: script serving, signal decoding, SSE response builder, EventBridge. Depends only on go-datastar + go-sse (no root dep).                                                                                                                                                                                                                       |

| **huma**              | `github.com/larsartmann/cqrs-htmx/huma/v4`                | Optional [Huma v2](https://huma.rocks/) adapter: `RegisterCommand`/`RegisterQuery` run the same authorize → validate → guard → dispatch stages inside typed OpenAPI operations and emit the same RFC 9457 error document. Huma keeps decode, status codes and the generated schema. Pins root v4.16.0, zero root change.                                                                                                                                                                                    |

Optional bridges also exist: **health** (go-health projection probes), **auditlog** (samber-do-auditlog), **systemadapter** (go-cqrs-lite `system.New` + metaengine), **huma** (typed OpenAPI operations). See `docs/guides/fullstack-wiring.md`.

### Subpackages inside the root module (2026-11-02)

The root module is no longer one flat package. Four packages sit beside `cqrshtmx`, and
each is independently importable — useful when you want one concern without the whole
handler builder:

| Package (root module) | Import path                                        | Use it when you want                                                                                                        |
| --------------------- | -------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `htmx`                | `github.com/larsartmann/cqrs-htmx/v4/htmx`         | Only the HTMX vocabulary: `Request`, `ParseRequest`, `Target`, `ScriptHandler`, `ExtensionHandler`, the `Header*` constants |
| `httperr`             | `github.com/larsartmann/cqrs-htmx/v4/httperr`      | Only error→HTTP mapping: `MapError`, `SafeDetail`, `StructuredError`, the ready-made handlers                               |
| `eventcatalog`        | `github.com/larsartmann/cqrs-htmx/v4/eventcatalog` | Only the published event language: `New`, `Register`, `Handler`                                                             |
| `internal/hx`         | not importable (internal)                          | —                                                                                                                           |

Two rules that matter:

1. **The subpackage is the surface (ADR-0062, 2026-11-06).** The three `*_reexport.go`
   files still alias every subpackage symbol so existing v4 code compiles unchanged, but
   each alias carries `// Deprecated:` naming the direct import and is deleted in v5.
   Write `htmx.ScriptHandler`, not `cqrshtmx.HTMXScriptHandler`. `nix run
.#check-reexport-policy -- --strict` refuses a new alias file or alias growth; the
   policy, the exemption classes and the one-breaking-window rule are in
   `docs/adr/0062-direct-imports-only.md` in the repo.
2. **The layering is one-way and gated.** `internal/hx` imports nothing in-module;
   `htmx`/`eventcatalog` import only `internal/hx`; `httperr` imports `htmx`+`internal/hx`;
   root imports all. `nix run .#check-module-layers` fails on any back-edge — so a
   subpackage may never import the root package.

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

For the v4→v5 cutover (WebSocket removal), see `docs/migrations/v4-to-v5.md`.

## Step 1: Pick the integration path

```
Do you need CQRS dispatch (command/query endpoints)?
+- NO  ->  PATH 0: Building blocks only (middleware, SSE, errors, HTMX -- no App)
+- YES ->  Does the app need user accounts / authentication?
          +- NO  ->  PATH A: root App only (CQRS + HTMX endpoints)
          +- YES ->  Does the user want a ready-made admin dashboard?
                    +- NO  ->  PATH B: root App + usermgmt Service + AuthHandler
                    +- YES ->  PATH C: usermgmt Service + adminui panel (+ root App for custom endpoints)

OpenAPI-first JSON API (Huma v2)? -> huma/v4 adapter over YOUR dispatchers (Path A shape):
                                    same pipeline stages inside typed operations, Huma owns the schema.
Realtime (SSE)? -> Use Broadcaster/SSEStream on ANY path. (WebSocket was removed in v5 — see ADR 0046.)
Persistence? -> Pass *sql.DB or use NewSQLiteEventSourcedSetup on Path B/C.
```

## Path 0 -- Building blocks only (no App)

Many consumers use cqrs-htmx purely for its middleware, SSE building blocks, and error mapping -- without the `App` command/query pipeline.

```go
mux := http.NewServeMux()
mux.Handle("GET /htmx.js", htmx.ScriptHandler())

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
    cqrshtmx.RecommendedSecurityMiddleware(),
    httputil.RateLimiterMiddleware(httputil.DefaultRateLimiterConfig()),
    cqrshtmx.RequestLoggingSlog(logger),
)(mux))
```

Available without `App`: `Chain`, `RecoveryMiddleware`, `RequestLoggingSlog`, `NewSSEStream`/`Broadcaster`/`JournalSSEStore`, `WriteJSON`, `NewResponse(w, r)`, plus from the subpackages: `htmx.ScriptHandler`/`htmx.ExtensionHandler`, `httperr.MapError`/`httperr.JSONErrorHandler`, `htmx.IsRequest`/`htmx.RenderPartial`. **Deprecated re-exports/aliases (import `github.com/larsartmann/httputil` instead):** `CSRFMiddleware`/`CSRFConfig`, `RateLimiterMiddleware`/`DefaultRateLimiterConfig`, `ServerTimingMiddleware`/`MeasureServerTiming`, `SecurityHeadersMiddleware`/`SecurityHeadersConfig`/`SecurityHeaderSkip`/`RecommendedHSTS`/`RecommendedCSP`. All remain as backward-compat aliases but are deprecated and will be removed in v5. Use `httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig())` for security headers.

## Path A -- CQRS + HTMX endpoints (root only)

The whole API is one `App`, configured once, that produces `http.HandlerFunc` values.

```go
cmdDisp := command.NewDispatcher()
qryDisp := query.NewDispatcher()
// Register handlers BEFORE building endpoints:
cmdDisp.Register("CreateItem", func(ctx context.Context, cmd command.Command) error { /* ... */ })
qryDisp.Register("ListItems", func(ctx context.Context, q query.Query) (any, error) { /* ... */ })

app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp, Queries: qryDisp})
```

**What is a `command.Command`?** It is an interface from go-cqrs-lite with three methods: `Type() command.Type`, `StreamID() id.StreamID`, `ID() id.CommandID`. Two shapes satisfy it:

```go
// Mapper path (DecodeJSON/DecodeForm): you construct the command — embed
// the provided core instead of hand-rolling the three methods.
type createItemCmd struct {
    *command.BasicCommand // provides Type(), StreamID(), ID()
    Name                  string
}

func newCreateItemCmd(name string) (*createItemCmd, error) {
    core, err := command.New("CreateItem", id.NewStreamID())
    if err != nil {
        return nil, err // only when the kind is empty
    }
    return &createItemCmd{BasicCommand: core, Name: name}, nil
}

// Typed path (DecodeJSONTyped): the decoder builds a zero value, so an
// embedded pointer would be nil — implement the methods by hand.
type typedItemCmd struct {
    aggID id.StreamID
    cmdID id.CommandID
    name  string
}

func (c *typedItemCmd) Type() command.Type    { return "CreateItem" }
func (c *typedItemCmd) StreamID() id.StreamID { return c.aggID }
func (c *typedItemCmd) ID() id.CommandID      { return c.cmdID }
```

`query.Query` is the same pattern: just `Type() query.Type`.

### Command endpoint: POST JSON -> dispatch -> response

```go
mux.Handle("POST /items", app.Command("CreateItem",
    cqrshtmx.DecodeJSON(func(req createItemRequest) (command.Command, error) {
        return newCreateItemCmd(req.Name), nil
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

### HTTP QUERY method (RFC 10008): body-carrying safe reads

`App.Query` is method-agnostic — the same handler serves native `QUERY`
(safe + idempotent + request body; complex filters never squeezed into URLs)
and the POST fallback from one `QUERY` pattern via one middleware:

```go
mux.Handle("QUERY /contacts", app.Query("SearchContacts",
    cqrshtmx.DecodeJSONQuery(func(req searchRequest) (query.Query, error) {
        return &searchContactsQuery{Filter: req.Filter}, nil
    }),
    cqrshtmx.RenderJSON[[]contact](),
))

// Mount INSIDE the security middleware (CSRF must see the wire POST):
srv := cqrshtmx.Chain(app.Middleware())(cqrshtmx.MethodOverrideMiddleware(mux))
```

- `MethodOverrideMiddleware` rewrites POST + `X-HTTP-Method-Override: QUERY`
  (legacy `X-Http-Method` honored) into QUERY before routing; QUERY is the
  only override target — an override to DELETE is ignored by design.
- `DecodeJSONQuery`/`DecodeJSONQueryTyped` advertise `Accept-Query:
  application/json` themselves (error responses too); form decoders advertise
  nothing. `AcceptQuery(...)` appends consumer types.
- `RequireAnyMethod("QUERY", http.MethodPost)` guards multi-method mounts
  (405 + `Allow` header); `RequireContentType("application/json")` opts a
  route into the RFC media-type ladder (missing type + body → 400, wrong type
  → 415 via `httperr.ErrUnsupportedMediaType`).
- Client recipes (htmx 2.x `hx-post` + override header, htmx 4 `hx-query`,
  DataStar, CSRF/CORS posture): `docs/guides/http-query-method.md`.

### Form-based (SSR / HTMX): POST form -> dispatch -> redirect

```go
mux.Handle("POST /items", app.Command("CreateItem",
    cqrshtmx.DecodeForm(func(req createItemForm) (command.Command, error) {
        return newCreateItemCmd(req.Name), nil
    }),
    cqrshtmx.PushURL("/items"),
    cqrshtmx.NotifySuccess("Item created"), // HX-Trigger toast
))
```

For HTML rendering: `cqrshtmx.RenderTempl(component)` (templ), `cqrshtmx.RenderTemplResult[T](mapper)` (result -> templ), or `cqrshtmx.RenderHTML("<div>...</div>")` (static HTML).

### Typed endpoints (no mapper, no type assertion)

`CommandTyped[Q]` and `QueryTyped[Q, R]` eliminate the mapper function and manual type assertions. The type `Q` itself implements `command.Command` or `query.Query`, and the decoder populates it directly from the request body:

```go
//doc:api-tour
// 1. Define a type that implements command.Command directly.
type greetCmd struct {
    aggID id.StreamID
    cmdID id.CommandID
    Name  string `json:"name"`
}
func (c *greetCmd) Type() command.Type    { return "Greet" }
func (c *greetCmd) StreamID() id.StreamID { return c.aggID }
func (c *greetCmd) ID() id.CommandID      { return c.cmdID }

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
//doc:api-tour
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
cqrshtmx.RenderIf(func(r *http.Request) bool { return htmx.Target(r) == "#avatar" }, avatarRender, fullRender)
```

All use `htmx.RenderPartial(r)` internally (history-restore requests render full page). HTML-specific helpers set `Content-Type: text/html`; generic `RenderIf`/`RenderPartialOrFullFunc` do not (see ADR-0037).

**OOB swap wrapping:** `cqrshtmx.OOBHTML(id, html, swapStrategy...)` wraps HTML with `hx-swap-oob` attributes.

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
        if !ownsGame(extractPlayerID(r), gc.GameID) { return httperr.ErrForbidden }
        return nil
    }),
))
```

`RequestGuard` runs **after decode, before dispatch** -- has access to the decoded command/query.

> **Auth error sentinels -> HTTP status:** `httperr.ErrUnauthorized` -> 401, `httperr.ErrForbidden` -> 403, `httperr.ErrValidationFailed` -> 400, `httperr.ErrRequestTooLarge` -> 413, `httperr.ErrMethodNotAllowed` -> 405.

### Authorization: the contract, and the one durable way to grant

Two rules carry almost all of it. **Grant through the event-sourced path** (a grant written into the enforcer is not an event and vanishes on restart), and **ask one question type** so the answer carries a reason.

```go
// 1. Grant (durable: dispatches the membership commands, replayed by the projection).
err := svc.GrantRoles(ctx, usermgmt.ActorIDFromUser(userID),
    identitymodel.NewTenantID(identitymodel.WildcardDomain),
    []identitymodel.Role{identitymodel.RoleSuperAdmin})

// 2. Decide (any cqrshtmx.PermissionChecker; *identitymodel.Authz satisfies it structurally).
decision, err := cqrshtmx.Decide(svc.Authz(), cqrshtmx.Request{
    Subject:  userID.String(),
    Domain:   tenantID,
    Resource: cqrshtmx.Resource{Type: "orders", ID: orderID},
    Action:   "read",
})

// 3. Branch on the REASON, not just the boolean.
switch decision.Reason {
case cqrshtmx.ReasonNotAuthenticated: // 401
case cqrshtmx.ReasonNoMatchingPolicy: // 403 -- the grant is missing
case cqrshtmx.ReasonDeniedByPolicy:   // 403 -- a rule refused
case cqrshtmx.ReasonAllowed:          // proceed
}
```

- `Service.GrantRoles` / `RevokeRoles` / `HasAdminGrant` are the durable surface. `Authz.AddGroupPolicy` is **deprecated** (ephemeral) -- `Authz.ApplyGroupPolicy` exists only for projections.
- Handler options: `Authorize(resource, action)` asks in the wildcard domain, `AuthorizeIn(domain, resource, action)` scopes to a tenant, `AuthorizeRequest(resolver)` derives the whole question from the request.
- `cqrshtmx.IsForbidden(err)` / `IsUnauthorized(err)` recognise a denial from ANY module (code-based, walks the unwrap chain). `errors.Is` against a single sentinel only ever matches one of them.
- `setup.New` logs a boot warning when the admin panel is mounted but nobody holds `admin`/`super_admin` in `*` -- that warning means step 1 is missing.
- Gates: `nix run .#check-permission-surface` (five anti-patterns, including a raw registration beside an installed dispatch gate) and `nix run .#check-permission-maturity` (`docs/PERMISSIONS.md` L0–L5 level ratchet).
- Proof kit: `authztest.Suite` proves route postures (401/403/200 + undeclared routes); `authztest.CommandSuite` proves the same for dispatches with no route; `authztest.GrantSuite` proves tenant scoping, revoke, and restart durability. All fail-closed. See `authztest/README.md`.

### Gating the dispatch boundary (L5)

`Authorize`/`AuthorizeIn` live in the **handler config**, which the dispatcher never
sees. So a command refused on its route is still admitted when a service method, a
seed, a job, or an admin CLI dispatches it. The dispatch gate closes that hole —
install it on the dispatcher, declare a posture per type, and the same command is
refused no matter who dispatches it.

```go
gate, err := cqrshtmx.NewDispatchGate(svc.Authz())
if err != nil {
    panic(err)
}

// Registering is declaring: "public"/"authenticated" skip the engine,
// "role-gated" asks for command/<Name>.
err = cqrshtmx.RegisterCommand(gate, cmdDisp, "CreateItem",
    cqrshtmx.PostureRoleGated, "creates an item in the caller's tenant",
    func(ctx context.Context, cmd *createItemCmd) error { return handleItem(ctx, cmd) })

cmdDisp.Use(gate.CommandMiddleware()) // outermost: a denial must not reach retry
err = gate.Validate()                 // boot check: no declarations / no reason / bad posture
```

- **Objects are `command/<Name>` and `query/<Name>`**; the actions are `execute`
  and `read`. The delimiter is a slash on purpose — under Casbin's `keyMatch` only
  `*` is special, so `command/*` is a usable prefix, while a colon would become a
  `keyMatch2` wildcard and a dot a regexp any-character. One policy row per repo
  is usually enough: `p, role_user, *, command/*, execute, allow`.
- **Undeclared means denied.** The dispatcher exposes no enumeration API, so a
  handler registered behind the gate's back is refused at dispatch time and named:
  register through `RegisterCommand`/`RegisterQuery` (or `DispatchGate.Declare`).
  `check-permission-surface`'s `undeclared-dispatch` rule flags the raw form.
- **`PolicyContext`** lets a command describe its own question (tenant, invoice id,
  `"refund"`) instead of the derived `command/<Name>`. Parameters reach policy only
  through that closed interface — never as a map of fields.
- **`cqrshtmx.AuditOnly()`** measures instead of enforcing (log + `OnDecision`,
  dispatch proceeds). One wave, then enforce.
- **Proof:** `authztest.CommandSuite` dispatches each declared type as anonymous,
  unprivileged, and privileged, and requires the error class to match the
  declaration. See `authztest/README.md`.
- Full rationale and the three-layer table (route / dispatch / decider):
  `docs/guides/authorization-system.md` §5 and ADR 0063.

### Middleware + serving htmx.js

```go
mux.Handle("GET /htmx.js", htmx.ScriptHandler()) // embedded htmx 2.0.10, ETag + 1yr cache
// htmx 4 (native hx-query) rides alongside: mux.Handle("GET /htmx4.js", htmx.ScriptHandler4())

http.ListenAndServe(":8080", cqrshtmx.Chain(
    cqrshtmx.RecommendedSecurityMiddleware(),
    httputil.CSRFMiddleware(httputil.CSRFConfig{}), // protects mutations; opt-in
    cqrshtmx.HTMXMiddleware,                        // detects HTMX requests
    app.Middleware(),                                // enriches context (user/correlation/request IDs)
)(mux))
```

> **templ-components note:** `templ-components`'s `layout.Base` may generate CDN `<script>` tags for htmx. If you self-host via `HTMXScriptHandler`, override `PageProps.HTMXCDN` or remove the CDN tag -- don't load htmx twice.

## OpenAPI-first JSON APIs (Huma v2)

Huma registers typed operations and owns decode, validation, status codes and
the OpenAPI document. An `http.HandlerFunc` from `app.Command`/`app.Query`
cannot join that contract, so the optional **huma/v4** module runs the same
stages inside a Huma handler: authorize → build → validate → guard → dispatch.

```go
pipe, err := cqrsadapter.New(cqrsadapter.Config{
    Commands: cmdDisp,     // the dispatchers your endpoints already use
    Queries:  qryDisp,
    Checker:  svc.Authz(), // the same engine Authorize/AuthorizeIn ask
})
if err != nil {
    panic(err)
}
```

```go
//doc:api-tour
// RegisterCommand(api, pipe, op, build, opts...)
//   build: func(ctx, *I) (command.Command, error)
//   opts:  Authorize(resource, action) | AuthorizeIn(domain, r, a)
//          AuthorizeOn[I](func(ctx, *I) (cqrshtmx.Request, error))
//          RequireAuth() | ValidateCommand(fn) | Guard(fn) | WithTimeout(d)
// RegisterQuery(api, pipe, op, build, respond, opts...)
//   respond: func(ctx, *I, result any) (*O, error)
// MapError(ctx, err) -> *ProblemError (application/problem+json)
```

- Stage order lives in the adapter, not your handlers: an anonymous caller
  answers 401, a policy denial 403, a validator failure 400 with code
  `cqrshtmx.validate.failed`, a conflict 409, and a broken policy engine 503
  (never a silent denial).
- Mount `app.Middleware()` (or your own enrichment) around the Huma handler so
  the pipeline sees the caller: user/actor/correlation/request IDs ride the
  request context exactly as they do on the routes.
- The error body is the SAME document the route-based handler writes
  (`httperr.StructuredError` with `code`, family, detail, request-ID instance),
  so clients branch on one vocabulary on both transports.
- Deliberate gaps, both documented: no decode/render/HTMX options (Huma owns
  those) and no `BeforeDispatch`/`AfterDispatch` hooks (they take an
  `*http.Request` a Huma handler never sees; use Huma middleware there).
- Modules: `huma/README.md`; guide: `docs/guides/fullstack-wiring.md`
  § OpenAPI-first APIs.

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
    cqrshtmx.RecommendedSecurityMiddleware(),
    sessionMW,                                    // authenticate the cookie first
    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
    cqrshtmx.HTMXMiddleware,
    app.Middleware(),
)(mux))
```

Read `references/usermgmt.md` for: the full setup matrix (in-memory / SQLite / Postgres / event signing), enabling WebAuthn/OAuth2/TOTP/email-verification, role & tenant management, and the `Service` write/read API.

**Per-module command posture (who owns the dispatcher?):**

- **root**: YOU build the dispatcher and hand it to `Config.Commands` — `dispatcher.Use(...)` your middleware before `MustNew` (ordering: `docs/guides/dispatch-middleware-ordering.md`).
- **usermgmt**: `NewService` owns the dispatcher and ALREADY wires the audit chain (actor + causation + correlation on every event, zero config). Your seam is `ServiceConfig.CommandMiddleware` (retry, breaker, idempotency, validation via the shipped `usermgmt.ValidateCommand`, metrics). Do NOT reach for the dispatcher — it is deliberately not exposed.
- **setup**: same seam via `setup.Config.CommandMiddleware` (flattens to the service).
  Full recipes + measured chain cost: `docs/guides/leveraging-go-cqrs-lite.md` §1.

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

> See `examples/admin-demo/` in the repo: a full runnable showcase (`go run .`, open `:18930`).

## The composition model (the heart of the library)

`App.Command` / `App.Query` + a list of `HandlerOption` values. The options run as a pipeline. **Order matters:**

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
status := httperr.MapError(err) // Rejection->400, Conflict->409, Transient->503, Corruption->500, Infrastructure->503; sentinels: 401/403/405/413
```

Set `Config.ErrorHandler: httperr.JSONErrorHandler` to auto-map errors to JSON responses:

```json
{ "error": "battle already exists", "status": 409, "code": "battle.exists" }
```

The `"code"` field carries the machine-readable error code from the innermost errorfamily error. For RFC 7807 (`application/problem+json`), use `httperr.ProblemDetailsErrorHandler`. For text/plain with HTMX redirects, use `DefaultErrorHandlerWithRedirect`.

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

## Realtime (SSE) -- on any path

Realtime is **building blocks, not a server**: you own the HTTP handler, the library gives you the stream + fan-out.

> **Note:** The WebSocket transport (`WSBroadcaster`, `WSMessage`, `DispatchWSCommand`, etc.) was removed (ADR 0046) — the current tree is already WS-free; the v5 train is the consumer-facing cutover and `docs/migrations/v4-to-v5.md` has per-symbol recipes. SSE covers the same use cases with simpler reconnect semantics and no upgrade/auth dance. Consumers who need bi-directional transport should use SSE for server→client and POST endpoints for client→server, or integrate a dedicated WebSocket library directly (the library intentionally avoids pulling in `gorilla/websocket` or similar).

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

**Domain events → SSE:** for event-sourced streams use the canonical `transport/` surface — `transport.ServeDomainEvents` + the `DomainEventToSSE` envelope + `WithSSEFilter`/`WithSSEMaxReplay` (envelope bytes are golden-pinned). ADR-0056 adds the cursor-based frontend sync protocol on top: see `docs/guides/frontend-sync.md`.

**SSE channel lifecycle:**

- `Subscribe()` returns a buffered channel (64 capacity). `Unsubscribe(ch)` closes the channel -- never send on it afterward.
- `Broadcaster.Close()` closes all subscriber channels and prevents new subscriptions. Call it on server shutdown for graceful SSE drain.
- `Broadcaster.SubscriberCount()` returns the current subscriber count (useful for metrics).
- Broadcasts are non-blocking -- slow consumers silently drop events. Pair with idempotency + ACK for guaranteed delivery.
- Standard event names: `cqrshtmx.SSEEventConnected` (`"connected"`), `cqrshtmx.SSEEventHeartbeat` (`"heartbeat"`).

### Embedded HTMX extensions

The root module embeds 3 extensions that pair with cqrs-htmx's server-side building blocks: SSE (2.2.4), idiomorph (0.7.4), and cqrs-query (1.0.0, the library's own: `hx-query` upgrades an `hx-post` to native QUERY with automatic 405/501 downgrade to the override fallback — RFC 10008 on htmx 2). The WS extension (htmx-ext-ws 2.0.4) was removed in v5 alongside the WS transport — see ADR 0046.

```go
// Individual or bundled:
mux.Handle("GET /ext/sse.js", htmx.ExtensionHandler(htmx.ExtSSE))
mux.Handle("GET /ext/bundle.js", htmx.ExtensionsHandler(htmx.ExtSSE, htmx.ExtIdiomorph))
```

Same caching as `htmx.ScriptHandler` (ETag, `Cache-Control: 1yr immutable`, 304). Add `<script>` tags after htmx core.

## Top gotchas (the ones that bite)

These are the highest-frequency mistakes. Read `references/gotchas.md` for the full list.

1. **Module suffixes are mandatory.** v2+ Go modules require the major-version suffix: `github.com/larsartmann/cqrs-htmx/v4`, `.../usermgmt/v4`, `.../adminui/v4`.
2. **One `UserID` type everywhere — construct it strictly.** Root `cqrshtmx.UserID`, `identitymodel.UserID`, and `usermgmt.UserID` all alias the same `go-cqrs-lite/id` ULID type (interoperable by design; no bridging needed). The trap is the deprecated `NewUserID(string)`: it silently SHA-256-hashes non-ULID input. Use `ParseUserID` (strict), `GenerateUserID` (fresh ULID), or `SyntheticUserID` (explicit stable hash).
3. **Validation comes AFTER the decoder** in the `HandlerOption` list. The validator reads the decoded value off the context -- reversing the order silently skips validation (logs a `slog.Warn`).
4. **Middleware trio order is `CSRF -> HTMX -> enrichment`.** Put your session middleware _outside_ (before) CSRF.
5. **No stdlib error constructors** (root + usermgmt + adminui). `errors.New`/`fmt.Errorf`/`errors.Join` are banned (enforced by `branching-flow errorfamily`). Use `event.New*/Wrap*/Wrapf/Newf` from `go-cqrs-lite/event/v4`.
6. **App.Command("") panics.** Empty command/query type strings are rejected at registration.
7. **Serve htmx.js yourself on Path A/B.** Register `htmx.ScriptHandler()` on your mux. **Exception:** on Path C (adminui) the panel serves htmx.js internally -- do NOT register it yourself.
8. **Register your command/query handlers BEFORE building endpoints.** `cmdDisp.Register("CreateItem", handler)` must happen before `app.Command("CreateItem", ...)`.
9. **A dispatch gate is not a route gate.** `Authorize*` options live in the handler config; a command dispatched through `Service.Dispatch`, a seed, a job, or a CLI bypasses them entirely. Install `DispatchGate.CommandMiddleware()` / `QueryMiddleware()` on the dispatcher if non-HTTP paths matter (ADR 0063).
10. **Policy objects use a slash, never a colon or dot.** `command:Create` becomes a `keyMatch2` parameter wildcard (`command[^/]+`) and `command.Create` becomes a regexp any-character — both silently grant more than written. `command/Create` stays a literal under `keyMatch`, and `command/*` is the prefix form.
11. **The auth posture is library-enforced (ADR-0055).** WebAuthn _enrollment_ ceremonies answer 401 without a session and 403 when the target `user_id` is not the session user (loginpage's register-issues-cookie flow passes untouched); every `currentUser`-reading `/auth/*` route self-wraps with an enrich-only session pass, so a bare `RegisterRoutes` mount works and fails closed. Headless enrollment: service-level `BeginRegistration`/`FinishRegistration`. For your own routes: `setup.RequireSession` / `setup.RequireSessionRedirect("/login")` after `bundle.SessionMiddleware()`.

### Discoverability notes (APIs that are easy to miss)

- **`CSRFResponseHeaderMiddleware`** -- sets the CSRF token in a response header (`X-CSRF-Token`) so HTMX can pick it up via `hx-headers`. The middleware trio is: `CSRFMiddleware` (validates) + `CSRFResponseHeaderMiddleware` (exposes token) + `app.Middleware()` (sets context). If you use CSRF with HTMX, you need all three.
- **`CSRFTestToken(mw)`** -- test helper that extracts a valid CSRF token through your middleware chain. nosurf uses token masking (the cookie value is NOT the valid header token); this helper handles the dance.
- **`ContextEnrichmentMiddleware(nil)`** -- nil extractor = auto-generate ULID request IDs. This is the zero-config path; `app.Middleware()` wraps this.
- **`IsHTMXRequest(r)` vs `RenderPartial(r)`** -- `IsHTMXRequest` checks `HX-Request: true` (any HTMX request). `RenderPartial` additionally excludes `HX-History-Restore-Request` (only "real" HTMX navigations that should render a partial).
- **`NewSSEEventID("")`** -- empty string auto-generates a ULID. This is the conventional way to mint event IDs.
- **`HealthHandler()`** -- returns `{"status":"ok"}` (200) or `{"status":"unhealthy","error":"..."}` (503).
- **`SecurityHeaderSkip`** (`"-"`) -- set on `ContentTypeOptions`/`FrameOptions`/`ReferrerPolicy` to suppress that default header entirely (empty string = use the default). Now deprecated alias for `httputil.SecurityHeaderSkip`.
- **`cqrshtmx.Chain` vs `httputil.Chain`** -- `cqrshtmx.Chain` is the standard middleware composition helper for this library. If you also import `go-httputil`, its `Chain` has the same signature; pick one and use it consistently.
- **Server-Timing / CSRF / rate-limiting are deprecated re-exports over httputil** -- `CSRFMiddleware`, `CSRFConfig`, `RateLimiterMiddleware`, `ServerTiming`, etc. are now deprecated type/var aliases for the equivalent symbols in `github.com/larsartmann/httputil`. Import httputil directly (these aliases will be removed in v5). The implementation (and deps like `justinas/nosurf`) lives in httputil. See `docs/guides/leveraging-httputil.md` for the migration table.
- **`cqrshtmx.RegisterErrorClassifications()`** -- registers httputil error classifications so errors from httputil middleware classify correctly through `MapError`. `New()` calls this automatically via `sync.Once`. Call it once at startup if you use httputil middleware without creating an App (e.g., a static file server with `httputil.Compression`). Safe to call multiple times.
- **`cqrshtmx.RecommendedSecurityMiddleware()`** -- returns the baseline production middleware chain: `SecurityHeaders` (with `RecommendedPermissionsPolicy`) + per-request CSP `Nonce` + `RecoveryMiddleware`. Both `adminui.Handler.Middleware()` and `dashboardui.Dashboard.Middleware()` delegate to it, so mounting either UI gives the same security posture. Use it directly for your own UIs: `cqrshtmx.Chain(cqrshtmx.RecommendedSecurityMiddleware(), sessionMW, app.Middleware())(mux)`.
- **`eventcatalog.Handler(catalog)`** -- serves an event schema catalog as immutable JSON (1-year cache, FNV-1a ETag). Use with `usermgmt.DefaultEventCatalog()` or build your own `eventcatalog.New()`. Mirrors `OpenAPISpecHandler` pattern. See `docs/guides/event-catalog-guide.md`.
- **`cqrshtmx.ProjectionStatusHandler(provider)`** -- serves live projection health as JSON (`no-cache`, per-request ETag). Pass any `cqrshtmx.ProjectionStatusProvider` (e.g. `*usermgmt.Service`). Returns 503 when provider is nil. See `docs/guides/projection-health-monitoring.md`.
- **`svc.RebuildProjection(ctx, name)`** -- stops the projection host, resets the named projection's checkpoint + read-model, creates a fresh host, and replays the entire journal. Available on both `*usermgmt.Service` and `*usermgmt.EventSourcedSetup`. See `docs/guides/rebuild-projection-runbook.md`.

## Where to look

- **`references/core-api.md`** -- full `Config`, every `HandlerOption`, middleware catalogue, context IDs, error->HTTP mapping.
- **`references/usermgmt.md`** -- Service setup matrix, auth endpoints, WebAuthn/OAuth2/TOTP, roles/tenants/bots, SQL persistence.
- **`references/realtime.md`** -- SSE broadcaster, ACK protocol, idempotency, reconnection/replay, heartbeat, event filtering patterns.
- **`references/gotchas.md`** -- the complete consumer gotcha list with fixes.
- **`docs/guides/leveraging-go-cqrs-lite.md`** -- how to leverage 58 go-cqrs-lite modules from cqrs-htmx (dispatch middleware, OTel/Prometheus, durable scheduling, signing/encryption, catalog docs, scenario testing, sagas, schema evolution).
- **`docs/guides/production-readiness.md`** -- single checklist for taking a cqrs-htmx app to production (middleware stack, observability, security, projection health, performance).
- **`docs/guides/dispatch-middleware-ordering.md`** -- correct ordering rules for dispatch middleware (recovery, circuit breaker, retry, tracing, metrics, logging).
- **`docs/guides/fullstack-wiring.md`** -- the integration guide: one-call `setup/v4` SDK, manual wiring, feature flags, persistence, middleware ordering, and optional health/auditlog integrations.
- **`docs/guides/sse-and-datastar.md`** -- the Broadcaster duality guide: `Hub()` accessor, `NewBroadcasterFromHub` cross-transport hub sharing (both adapters embed the hub; `Raw()`/`NewBroadcasterFromRaw` are deprecated v5-removal aliases), and HTMX vs Datastar transport comparison.
- **`docs/guides/frontend-sync.md`** -- ADR-0056 frontend sync: cursor-based `SyncPullHandler` event pull (ETag + permission filter), `App.SyncPushHandler` batched push, sync assets.
- **`docs/guides/datastar-integration.md`** -- the DataStar adapter: signals, SSE response builder, EventBridge wiring.
- **`huma/README.md`** -- the Huma v2 adapter: operation registration, the pipeline options, the problem document, and what the adapter deliberately leaves to Huma.
- **`docs/guides/authorization-system.md`** -- the authorization system end to end: durable grants, route posture, `Decide`/`AuthorizeIn`/`AuthorizeRequest`, the 401/403/200 proof matrix, SSP/CSP-adjacent notes. `docs/PERMISSIONS.md` is the L0-L4 maturity system + per-service inventory; `docs/adr/0057-authorization-contract.md` and `0058-defer-relationship-based-authorization.md` are the decisions.
- **Repo examples**: `examples/basic/` (minimal CQRS+HTMX+SSE, plus the `/demo/items` partial-render showcase: `RenderPartialOrFull` + `OOBHTML` + `RenderIf`), `examples/admin-demo/` (full admin showcase), `examples/middleware-demo/` (dispatch middleware composition proof), `examples/middleware-showcase/` (all 8 httputil HTTP middleware in a single validated `MiddlewareStack`: recovery, security headers, metrics, CORS, client IP, rate limiting, compression, ETag), `examples/setup-demo/` (one-call full-stack), `examples/sync-demo/` (executable ADR-0056 guide), `examples/datastar-demo/` (DataStar adapter).
- **ADR docs**: `docs/adr/` -- the _why_ behind each design.

When the user asks for something not covered inline here, **read the matching reference file before improvising** -- it almost always contains the exact API and the ordering constraints.

> **Skill-authoring note:** code fences in this SKILL.md and the module READMEs are parse-checked by `check-skill-snippets.sh` (wired into check-modules + CI) -- keep every fence linguistically tagged and compilable-shaped. A fence that is a deliberate API tour (statements + `// -> ...` method listings, not a program) carries `//doc:api-tour` as its literal first line; the gate then skips it. Reference-file fences (`references/*.md`) are skipped wholesale.
