# setup — One-Call Composition Root for cqrs-htmx

Wire a full-stack cqrs-htmx application — event-sourced user management, admin
panel, CQRS observability dashboard, login page, health endpoint, middleware —
with a single `setup.New` call and one line to serve.

## What it does

- Creates the shared event infrastructure (in-memory by default, bring your own store/bus)
- Constructs the `usermgmt.Service` with all auth strategies you inject (WebAuthn/TOTP/OAuth2)
- Builds and mounts the admin panel, CQRS dashboard, and login page with correct middleware ordering
- Mounts a `/health` readiness endpoint that tracks projection drain state
- Provides `Bundle.Run` — serve with safe timeouts, graceful shutdown, and cleanup in one call

## Quick start

```go
package main

import (
    "context"
    "log"

    totp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
    "github.com/larsartmann/cqrs-htmx/setup/v4"
)

func main() {
    bundle, err := setup.New(setup.Config{
        Title: "My App",
        TOTP:  totp.New(totp.Config{Issuer: "MyApp"}),
    })
    if err != nil {
        log.Fatal(err)
    }

    // Mount, serve with safe timeouts, drain gracefully on SIGINT.
    if err := bundle.Run(context.Background(), ":8080"); err != nil {
        log.Fatal(err)
    }
}
```

## What you get (default routes)

| Route          | Panel           | Access                       |
| -------------- | --------------- | ---------------------------- |
| `/auth/*`      | Auth API        | public                       |
| `/admin/*`     | Admin panel     | session + CSRF (401 without) |
| `/dashboard/*` | CQRS dashboard  | session-gated (401 without)  |
| `/sse`         | Shared SSE feed | session-gated (opt-in)       |
| `/health`      | Readiness check | public (503 while draining)  |
| `/`            | Login page      | public                       |

## Serving options

```go
// Option 1 — everything in one call (recommended):
if err := bundle.Run(ctx, ":8080"); err != nil { log.Fatal(err) }

// Option 2 — your own mux with extra routes:
mux := http.NewServeMux()
mux.Handle("POST /orders", bundle.SessionMiddleware()(myOrdersHandler))
if err := bundle.Handler(mux).Serve(...); err != nil { ... } // or serve mux yourself

// Option 3 — full control:
mux := http.NewServeMux()
bundle.Mount(mux)
http.ListenAndServe(":8080", bundle.Middleware()(mux))
```

`Run` sets `ReadHeaderTimeout` and `IdleTimeout` but deliberately no
`WriteTimeout` — the dashboard serves SSE streams that outlive any fixed
deadline.

## Configuration

Everything is optional; zero-value `Config{}` gives a working in-memory app.

| Field                                                | Type                         | Default             | Description                                                |
| ---------------------------------------------------- | ---------------------------- | ------------------- | ---------------------------------------------------------- |
| `TOTP` / `WebAuthn` / `OAuth2`                       | provider interfaces          | none                | Auth strategies; import the sub-modules and inject         |
| `EventStore` / `EventBus`                            | `event.Store` / `event.Bus`  | in-memory           | Shared infrastructure; store must be a `SeekableJournal`   |
| `ReadModelDB`                                        | `*sql.DB`                    | nil (in-memory)     | SQL-backed read models that survive restarts               |
| `Title` / `AccentColor`                              | `string`                     | `"cqrs-htmx"` / sky | Branding across all panels                                 |
| `AdminPath`                                          | `string`                     | `"/admin/"`         | Trailing slash auto-normalized; links follow the mount     |
| `DashboardPath`                                      | `string`                     | `"/dashboard/"`     | Trailing slash auto-normalized                             |
| `LoginRedirect`                                      | `string`                     | `"/admin/"`         | Post-login destination                                     |
| `HealthPath`                                         | `string`                     | `"/health"`         | Readiness endpoint; must not collide with other paths      |
| `SSEPath`                                            | `string`                     | off                 | Session-gated shared SSE feed of all committed events      |
| `SSEHeartbeatInterval`                               | `time.Duration`              | `15s` (0 = off)     | Keep-alive comment frames on `/sse`                        |
| `Service`                                            | `*usermgmt.Service`          | built by `New`      | Adopt your own service; panels wire on top of it           |
| `CookieName` / `SessionTTL`                          | `string` / `time.Duration`   | `"session"` / 24h   | Session cookie configuration                               |
| `Logger`                                             | `*slog.Logger`               | `slog.Default()`    | Structured auth event logging                              |
| `LogoutURL` / `SSEURL`                               | `string`                     | hidden / off        | Logout link; admin panel real-time sync indicator          |
| `AdminMode` / `TenantID`                             | `adminui.Mode` / `TenantID`  | super-admin         | Tenant-scoped admin panel (TenantID required in that mode) |
| `AdminAuthorizer`                                    | `func(*usermgmt.User) error` | role-based          | Custom admin access control                                |
| `DashboardAuthorizer`                                | `func(*http.Request) error`  | none                | Extra dashboard gate (runs after the session gate)         |
| `OnProjectionFailed`                                 | `func(name, lastErr string)` | none                | Alerting hook when a projection exhausts restarts          |
| `AsyncStartup`                                       | `bool`                       | `false`             | Bind immediately; `/health` gates readiness during drain   |
| `DashboardReadOnly`                                  | `*bool`                      | `true`              | Set `false` at your own risk (enables reset/DLQ replay)    |
| `DashboardPageSize`                                  | `int`                        | 50                  | Rows per dashboard table page (max 200)                    |
| `LoginNoRegistration`                                | `bool`                       | `false`             | Hide the registration section                              |
| `DisableAdmin` / `DisableDashboard` / `DisableLogin` | `bool`                       | `false`             | Feature flags to shrink the route surface                  |

Invalid configs fail fast at `New` with descriptive errors: paths must start
with `/`, must not be `/` (reserved for the login page), and must be pairwise
distinct — misconfiguration surfaces before `Mount` can panic.

### Bringing your own service

Construct `*usermgmt.Service` yourself (custom `SecurityHooks`, `MaxUsers`,
snapshotting, a custom `AuditLog`, ...) and hand it to the bundle:

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{ /* your advanced config */ })

bundle, _ := setup.New(setup.Config{ Service: svc })

defer func() { _ = bundle.Close() }() // does NOT close the adopted service
defer func() { _ = svc.Close() }()     // you own its lifecycle
```

The bundle sources its shared stores from the service (`svc.Journal()`,
`svc.EventBus()`), so panels observe the exact infrastructure your service
publishes to. Service-construction fields (`EventStore`, `TOTP`, ... `AsyncStartup`)
are rejected as conflicts in this mode — nothing is silently ignored.

### Shared SSE endpoint

Set `SSEPath` to mount a session-gated endpoint streaming every event committed
to the event bus as a small JSON envelope (`type`, `streamId`, `version`, ...).
Reconnecting clients resume from their `Last-Event-ID`, and first-time
subscribers receive a journal backfill when the event store implements
`event.Journal` (plain in-memory stores stream live-only). A heartbeat comment
frame is sent every `SSEHeartbeatInterval` (default 15s; `0` or negative
disables) so proxies and load balancers keep the connection open.
`bundle.Broadcaster` is the fan-out hub behind it — subscribe to it (or share
its `Raw()` hub with a DataStar broadcaster) to push custom real-time payloads
through the same connection topology.

## Customization after construction

Every sub-component is exposed on the `Bundle`:

```go
bundle.Admin.SetAccentColor("#ff0000")
mux.Handle("POST /orders", bundle.SessionMiddleware()(ordersHandler))
store := bundle.Stores.EventStore // reuse for your own projections/SSE
```

## Persistence

Defaults are in-memory (lost on restart). For production, provide your own
`event.Store` (SQL stores live in go-cqrs-lite) and `ReadModelDB`:

```go
bundle, err := setup.New(setup.Config{
    EventStore:  mySQLStore,   // must implement event.SeekableJournal
    ReadModelDB: db,
})
```

## See also

- `docs/guides/fullstack-wiring.md` — full wiring guide (SDK vs manual)
- `examples/setup-demo/` — runnable demo of the whole bundle
- `docs/guides/async-projection-startup.md` — the readiness model behind `/health`
