---
name: cqrs-htmx
description: Build Go web apps with the cqrs-htmx library — CQRS command/query HTTP handlers, HTMX responses, event-sourced user management (WebAuthn/OAuth2/TOTP), and a ready-made admin UI. Use this skill whenever integrating, wiring, mounting, or extending cqrs-htmx, go-cqrs-lite with HTMX, the usermgmt or adminui submodules, or building CQRS/HTMX/SSE/WebSocket/auth features on top of this library — even when the user does not name the library explicitly (e.g. "add a command endpoint", "wire up passkey login", "add an admin panel", "serve HTMX", "broadcast over SSE", "set up CQRS dispatch").
user-invocable: true
---

# Using cqrs-htmx superbly

cqrs-htmx is a **Go library** (not an app) that wires [go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite) CQRS dispatch to `net/http` with first-class HTMX support, plus an event-sourced user-management submodule and a ready-made admin dashboard. Consumers import it; there is no "main app" to run except the examples.

It is framework-agnostic: it works with `net/http`, Chi, Gin, etc. and never picks your router for you.

> **Related skill:** the [`go-cqrs-lite`](https://github.com/LarsArtmann/go-cqrs-lite) skill covers the underlying CQRS/ES building blocks in depth — deciders, event stores, projections, read models, snapshots, schema evolution, signing, encryption. Consult it when a question is about the CQRS core rather than the HTTP/HTMX layer.

## The modules

An app typically composes some subset of these. They are **independent Go modules** with `/v4` suffixes.

| Module                | Import path                                              | Provides                                                                                                                                                                                                                           |
| --------------------- | -------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **root**              | `github.com/larsartmann/cqrs-htmx/v4` (alias `cqrshtmx`) | The `App` builder, `HandlerOption`s, middleware (CSRF/HTMX/recovery/security/rate-limit), context IDs, error→HTTP mapping, SSE + WebSocket, embedded HTMX JS                                                                       |
| **usermgmt**          | `github.com/larsartmann/cqrs-htmx/usermgmt/v4`           | Event-sourced `Service` (register/login/logout/me), roles/tenants/bots, authz via Casbin, session middleware, SQL + in-memory stores. Auth strategies (WebAuthn/OAuth2/TOTP) are optional sub-modules — import only what you need. |
| **usermgmt/totp**     | `github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4`      | TOTP MFA (pquerna/otp). Inject `totp.New(...)` as `ServiceConfig.TOTP`.                                                                                                                                                            |
| **usermgmt/webauthn** | `github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4`  | WebAuthn passkeys (go-webauthn). Inject `webauthn.New(...)` as `ServiceConfig.WebAuthn`.                                                                                                                                           |
| **usermgmt/oauth2**   | `github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4`    | OAuth2/OIDC login (oauth2+oidc). Inject `oauth2.New(...)` as `ServiceConfig.OAuth2`.                                                                                                                                               |
| **adminui**           | `github.com/larsartmann/cqrs-htmx/adminui/v4`            | One-call admin dashboard (templ + HTMX). Depends on root + usermgmt                                                                                                                                                                |

The core CQRS building blocks come from **go-cqrs-lite**, imported per-package:

```go
"github.com/larsartmann/go-cqrs-lite/command/v3"
"github.com/larsartmann/go-cqrs-lite/query/v3"
"github.com/larsartmann/go-cqrs-lite/event/v3"
"github.com/larsartmann/go-cqrs-lite/id/v3"
```

## Step 1: Pick the integration path

Match the user's need to a path. Most real apps end up on path B or C.

```
Does the app need user accounts / authentication?
├─ NO  →  PATH A: root App only (CQRS + HTMX endpoints)
└─ YES →  Does the user want a ready-made admin dashboard?
          ├─ NO  →  PATH B: root App + usermgmt Service + AuthHandler
          └─ YES →  PATH C: usermgmt Service + adminui panel (+ root App for custom endpoints)

Does the app need realtime updates (SSE or WebSocket)?
└─ Use the root Broadcaster / SSEStream / WSBroadcaster on ANY path. See references/realtime.md.

Does the app need persistence (survive restarts)?
└─ Pass a *sql.DB to usermgmt.ServiceConfig{ReadModelDB: db} and SQL stores for the event store,
   OR use NewSQLiteEventSourcedSetup / NewPostgresEventSourcedSetup for a one-call stack.
   See references/usermgmt.md.
```

- **Path A** needs only the root module.
- **Path B/C** need usermgmt (+ adminui for C). adminui also pulls in root (it reuses `cqrshtmx.HTMXScriptHandler`).
- For deeper detail on each path, read the matching `references/*.md`.

## Path A — CQRS + HTMX endpoints (root only)

The whole API is one `App`, configured once, that produces `http.HandlerFunc`s.

```go
package main

import (
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

func main() {
	cmdDisp := command.NewDispatcher()
	qryDisp := query.NewDispatcher()
	// register handlers with cmdDisp.Register(command.Type("X"), func(ctx, cmd) error {...})
	// and    qryDisp.Register(query.Type("Y"),   func(ctx, q) (any, error) {...})

	app := cqrshtmx.MustNew(cqrshtmx.Config{
		Commands: cmdDisp,
		Queries:  qryDisp,
		// UserIDExtractor: func(r *http.Request) (cqrshtmx.UserID, error) { ... }, // if you have auth
	})

	mux := http.NewServeMux()
	mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler()) // serve embedded htmx 2.x

	// A command endpoint: decode JSON → dispatch → push URL on success.
	mux.Handle("POST /items", app.Command("CreateItem",
		cqrshtmx.DecodeJSON(func(req createItemRequest) (command.Command, error) {
			return &createItemCmd{
				typ: command.Type("CreateItem"), aggID: id.NewAggregateID(), cmdID: id.NewCommandID(),
				Name: req.Name,
			}, nil
		}),
		cqrshtmx.WithSuccessStatus(201),
		cqrshtmx.PushURL("/items"), // HTMX: update the address bar
	))

	// A query endpoint: render the result as JSON (or templ, or paginated JSON).
	mux.Handle("GET /items", app.Query("ListItems",
		cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) { return &listItemsQuery{}, nil }),
		cqrshtmx.RenderJSON[[]item](),
	))

	http.ListenAndServe(":8080", cqrshtmx.Chain(
		cqrshtmx.RecoveryMiddleware,
		cqrshtmx.SecurityHeadersMiddleware,
		cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}), // protects mutations; opt-in
		cqrshtmx.HTMXMiddleware,                       // detects HTMX requests
		app.Middleware(),                               // enriches context (user/correlation/request IDs)
	)(mux))
}
```

> See `examples/basic/` in the repo for a complete runnable version (server-rendered HTML + SSE).

## Path B — add user accounts (root + usermgmt)

usermgmt gives you a fully event-sourced `Service` and an `AuthHandler` that mounts all auth routes onto your mux. Authentication is **passwordless** (WebAuthn passkeys by default; OAuth2/OIDC and TOTP optional).

```go
package main

import (
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func main() {
	// 1. Build the user-management service. Zero-config = in-memory everything.
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		// WebAuthn: passkeyProvider, // import usermgmt/webauthn, inject as ServiceConfig.WebAuthn
		// AuditLog:       usermgmt.NewAuditLog(),
	})
	if err != nil {
		panic(err)
	}
	defer svc.Close()

	// 2. Mount the built-in auth endpoints (/auth/register, /auth/webauthn/*, /auth/logout, /auth/me, ...).
	auth := usermgmt.NewAuthHandler(svc)
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux) // mutates YOUR mux — you own routing

	// 3. (Optional) your own CQRS app for domain endpoints.
	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher(), Queries: query.NewDispatcher()})
	mux.Handle("POST /widgets", app.Command("CreateWidget", cqrshtmx.DecodeJSON(/*...*/)))

	// 4. Wrap with session auth + CSRF. Session goes OUTSIDE CSRF.
	sessionMW := usermgmt.NewSessionMiddleware(svc, "session")
	http.ListenAndServe(":8080", cqrshtmx.Chain(
		cqrshtmx.RecoveryMiddleware,
		cqrshtmx.SecurityHeadersMiddleware,
		sessionMW,                                    // authenticate the cookie first
		cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
		cqrshtmx.HTMXMiddleware,
		app.Middleware(),
	)(mux))
}
```

Read `references/usermgmt.md` for: the full setup matrix (in-memory / SQLite / Postgres / event signing), enabling WebAuthn/OAuth2/TOTP/email-verification, role & tenant management, and the `Service` write/read API.

## Path C — add the ready-made admin dashboard (adminui)

One call builds a templ + HTMX dashboard for managing users, tenants, members, and the audit log.

```go
import "github.com/larsartmann/cqrs-htmx/adminui/v4"

panel, err := adminui.New(adminui.Config{
	Service:     svc,             // *usermgmt.Service — required
	Title:       "MyApp Admin",
	AccentColor: "#0ea5e9",
	LogoutURL:   "/logout",
	SSEURL:      "/admin/-/events", // enables honest-UI sync indicator (optional)
})
// Two equivalent ways to mount:
panel.Mount(mux, "/admin/")              // option 1: registers on your mux at a prefix
mux.Handle("/admin/", http.StripPrefix("/admin", panel.Handler())) // option 2: manual

// Recommended wrap: session auth (OUTSIDE CSRF) → CSRF → the panel's own recovery+security-headers.
mux.Handle("/admin/", sessionMW(csrfMW(panel.Middleware()(http.StripPrefix("/admin", panel.Handler())))))
```

`adminui.Config` fields that matter: `Mode` (`ModeSuperAdmin` = global users/tenants/audit; `ModeTenantAdmin` = scoped to one `TenantID`), `Authorizer` (default = role-based; override with `RequireAnyRole(...)`/`RequireAuthenticated()`), `BasePath`. The panel reads the authenticated `*usermgmt.User` from the request context — your session middleware must put it there (`usermgmt.NewSessionMiddleware` already does).

> See `examples/admin-demo/` in the repo: a full runnable showcase (`go run .`, open `:8097`).

## The composition model (the heart of the library)

Everything in Path A is **`App.Command` / `App.Query` + a list of `HandlerOption`s**. The options run as a pipeline. Order matters for two of them.

```go
mux.Handle("POST /widgets", app.Command("CreateWidget",
	cqrshtmx.Authorize("widgets", "create"),       // (1) authz — gate first
	cqrshtmx.DecodeJSON(mapToCommand),              // (2) decode body → command.Command
	cqrshtmx.ValidateCommand(func(c) error { ... }),// (3) validate — MUST come after the decoder
	cqrshtmx.WithSuccessStatus(201),                // (4) config
	cqrshtmx.NotifySuccess("Widget created"),       // (5) side effects / response
	cqrshtmx.PushURL("/widgets"),
))
```

**Why order matters:** `ValidateCommand`/`ValidateQuery` read the decoded command/query off the context, so the decoder must run first. Put authz first (fail fast before touching the body), then decode, then validate, then render/side-effects. See `references/core-api.md` for the full `HandlerOption` catalogue.

**Middleware order** (outer → inner): recovery & security headers first; then your **session** middleware; then **CSRF**; then **HTMX**; then `app.Middleware()` (context enrichment); then the handler. The non-negotiable part is the cqrs-htmx trio `CSRF → HTMX → enrichment`, because enrichment depends on the HTMX context and CSRF must wrap the mutations.

## Serving htmx.js

The root module **embeds htmx 2.0.10** (~51KB). Serve it yourself (no CDN needed):

```go
mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())
```

Add to your HTML with `cqrshtmx.HTMXScriptTag("/htmx.js")` (templ-safe) or `cqrshtmx.HTMXCDNScriptTag("")` if you prefer a CDN. `cqrshtmx.HTMXVersion()` returns `"2.0.10"` for cache-busting query strings.

### Embedded HTMX extensions

The root module also embeds the 3 extensions that pair with cqrs-htmx's server-side building blocks:

| Extension                        | Version | Pairs with                                        |
| -------------------------------- | ------- | ------------------------------------------------- |
| `HTMXExtSSE` ("sse")             | 2.2.4   | `SSEStream`, `Broadcaster`, `JournalSSEStore`     |
| `HTMXExtWS` ("ws")               | 2.0.4   | `WSMessage`, `WSBroadcaster`, `DispatchWSCommand` |
| `HTMXExtIdiomorph` ("idiomorph") | 0.7.4   | SSE partial updates (morph swap)                  |

Serve individually or as a single-request bundle:

```go
// Individual
mux.Handle("GET /ext/sse.js", cqrshtmx.HTMXExtensionHandler(cqrshtmx.HTMXExtSSE))

// Bundle (one HTTP request for all three)
mux.Handle("GET /ext/bundle.js",
    cqrshtmx.HTMXExtensionsHandler(cqrshtmx.HTMXExtSSE, cqrshtmx.HTMXExtWS, cqrshtmx.HTMXExtIdiomorph))
```

Both set `Content-Type`, `ETag`, and `Cache-Control: 1yr immutable` with 304 support — same caching as `HTMXScriptHandler`. Add `<script>` tags in your layout **after** htmx core:

```html
<script src="/htmx.js"></script>
<script src="/ext/bundle.js"></script>
```

Other helpers: `cqrshtmx.HTMXExtensionCDNScriptTag(name)` generates a CDN `<script>` tag (uses embedded version), `cqrshtmx.HTMXExtensionVersion(name)` returns the version string, `cqrshtmx.HTMXExtensionNames()` lists all available extensions.

## Realtime (SSE / WebSocket) — on any path

Realtime is **building blocks, not a server**: you own the HTTP handler, the library gives you the stream + fan-out.

```go
broadcaster := cqrshtmx.NewBroadcaster()
// anywhere a command succeeds:
appCfg := cqrshtmx.Config{ /* ... */, AfterDispatch: broadcaster.BroadcastOnSuccess("itemCreated", data) }

mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
	stream := cqrshtmx.NewSSEStream(w, r)
	defer stream.Close()
	ch := broadcaster.Subscribe()
	defer broadcaster.Unsubscribe(ch)
	for {
		select {
		case <-stream.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok || stream.Send(evt) != nil { return }
		}
	}
})
```

WebSockets mirror this: `NewWSBroadcaster()`, `WSOOBHTML(...)`, `ParseWSMessageInto[T]`, and `app.DispatchWSCommand(...)` to bridge WS → CQRS. Read `references/realtime.md` for the ACK protocol, idempotency store, reconnection/replay, and heartbeat.

## Top gotchas (the ones that bite)

These are the highest-frequency mistakes. Read `references/gotchas.md` for the full list.

1. **Module suffixes are mandatory.** v2+ Go modules require the major-version suffix: `github.com/larsartmann/cqrs-htmx/v4`, `.../usermgmt/v4`, `.../adminui/v4`. Forgetting the suffix gives confusing "module not found" or wrong-version errors.
2. **Two different UserID types.** Root `cqrshtmx.UserID` is ULID-backed (via `go-cqrs-lite/id`); usermgmt `usermgmt.UserID` is string-backed (`go-branded-id`). They are **not** assignable to each other. Bridge with `.Get()` (raw value), never `.String()` (brand-prefixed). When wiring session context into root handlers, convert explicitly.
3. **Validation comes AFTER the decoder** in the `HandlerOption` list. The validator reads the decoded value off the context — reversing the order silently skips validation.
4. **Middleware trio order is `CSRF → HTMX → enrichment`.** Put your session middleware _outside_ (before) CSRF. Getting this wrong causes silent CSRF gaps or missing HTMX context.
5. **No stdlib error constructors** (root + usermgmt + adminui). `errors.New`/`fmt.Errorf`/`errors.Join` are banned in these modules (enforced by `branching-flow errorfamily`). Use `event.New*/Wrap*/Wrapf/Newf` from `go-cqrs-lite/event/v3`. Auth sub-modules (totp/webauthn/oauth2) are intentionally exempt — their errors are wrapped at the Service boundary. When you only build a _message string_ (not an error object), `fmt.Sprintf` is fine.
6. **App.Command("") panics.** Empty command/query type strings are rejected at registration — use real `command.Type("...")`/`query.Type("...")` values.
7. **Serve htmx.js yourself on Path A/B.** Register `cqrshtmx.HTMXScriptHandler()` on your mux and reference it in your layout. HTMX extensions (SSE/WS/idiomorph) are also embedded — use `cqrshtmx.HTMXExtensionHandler(name)` or `cqrshtmx.HTMXExtensionsHandler(names...)` for a bundle. **Exception:** on Path C (adminui) the panel serves htmx.js internally (`adminui/assets.go`) — do NOT register it yourself or you'll create a duplicate route.

## Where to look

- **`references/core-api.md`** — full `Config`, every `HandlerOption`, middleware catalogue, context IDs, error→HTTP mapping, the `Response` builder.
- **`references/usermgmt.md`** — Service setup matrix, auth endpoints, WebAuthn/OAuth2/TOTP, roles/tenants/bots, SQL persistence, lifecycle (`Stop`/`Close`/`GracefulClose`).
- **`references/realtime.md`** — SSE + WebSocket, broadcaster, ACK protocol, idempotency, reconnection/replay, heartbeat.
- **`references/gotchas.md`** — the complete consumer gotcha list with fixes.
- **Repo examples**: `examples/basic/` (minimal CQRS+HTMX+SSE), `examples/admin-demo/` (full admin showcase), `examples/datastar-demo/` (go-cqrs-lite + Datastar SSE), `examples/catalog-demo/` (doc server).
- **ADR docs**: `docs/adr/` — the _why_ behind each design (event sourcing, SSE/WS, OAuth2, event signing, idempotency, honest UI).

When the user asks for something not covered inline here, **read the matching reference file before improvising** — it almost always contains the exact API and the ordering constraints.
