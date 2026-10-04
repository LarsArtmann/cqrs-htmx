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

| Route          | Panel           | Access                                                             |
| -------------- | --------------- | ------------------------------------------------------------------ |
| `/auth/*`      | Auth API        | mixed — see the route table below (ceremonies gated, subset session-wrapped) |
| `/admin/*`     | Admin panel     | session + CSRF (401 without)                                       |
| `/dashboard/*` | CQRS dashboard  | session-gated (401 without)                                        |
| `/sse`         | Shared SSE feed | session-gated (opt-in)                                             |
| `/health`      | Readiness check | public (503 while draining)                                        |
| `/`            | Login page      | public                                                             |

### Auth route posture

The `/auth/*` surface is mixed, and the library enforces it:

| Route group                          | Posture                                                                       |
| ------------------------------------ | ----------------------------------------------------------------------------- |
| `POST /auth/register`                | public — sets the session cookie (first-user bootstrap)                       |
| WebAuthn **login** ceremonies        | public — unauthenticated by nature                                            |
| WebAuthn **enrollment** ceremonies   | **owner-session-gated**: 401 without a session, 403 when the target `user_id` is not the session user |
| `GET /auth/me`, credentials, TOTP, email-verify/send, import/export, OAuth2 unlink | session required (401 without; self-wrapped with an enrich-only session pass) |

The enrollment gate is why `POST /auth/register` issues the session cookie
before any ceremony: the login page's register → begin → finish flow rides
that cookie. Headless or administrative enrollment uses the service-level
API (`Service.BeginRegistration`) — there is no HTTP opt-out. See ADR-0055.

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

## Styling

The **admin panel and CQRS dashboard need nothing from you** — both serve their
own embedded Tailwind bundles (`/-/admin-tw.css`, `/-/dashboard-tw.css`).

The **login page** is different: it renders
[templ-components](https://github.com/larsartmann/templ-components) Tailwind v4
utility classes, which only exist after YOU compile a stylesheet that scans the
loginpage package, and serve it at the login page's `CSSPath`
(default `/app.css`, configurable via `LoginCSSPath`). Skip this and the page
renders structurally correct but unstyled HTML.

Minimal consumer stylesheet (Tailwind v4 CSS-first config):

```css
@import "tailwindcss";
@custom-variant dark (&:where(.dark, .dark *));

/* Scan your own templates too, if they use Tailwind classes: */
@source "./**/*.{templ,go,js}";
```

The `@source` scan must include the loginpage module directory
(`page.templ` and `assets/login.js` — the embedded WebAuthn JS injects
`lp-spinner`/`animate-spin` classes at runtime) and the templ-components
packages it imports (`layout`, `forms`, `display`, `feedback`, `utils`). Two
proven ways to do that:

1. **Copy the demo's build app** — `nix run .#build-setup-demo-css` compiles
   `examples/setup-demo/tailwind.css` → `assets/app.css` by resolving the
   module directories with `go list -m` and copying only `.templ` sources into
   a scan dir (scanning the module cache directly can exhaust RAM; and the
   variant class maps live in the library's `*_go.go` files, not `.templ`).
2. **Point `@source` at your module cache paths** (`$(go env GOMODCACHE)/github.com/larsartmann/cqrs-htmx/loginpage/v4@<version>/...`)
   — simpler, but re-pin the path on every dependency bump.

Then serve the compiled file at `/app.css` (or any URL you pass as
`LoginCSSPath`). `examples/setup-demo` is the working reference: it embeds
`assets/app.css` and registers `GET /app.css` next to the bundle's routes.

## Configuration

Everything is optional; zero-value `Config{}` gives a working in-memory app.

| Field                                                     | Type                                | Default                                 | Description                                                                                                                                                                                                                                                                                                         |
| --------------------------------------------------------- | ----------------------------------- | --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `TOTP` / `WebAuthn` / `OAuth2`                            | provider interfaces                 | none                                    | Auth strategies; import the sub-modules and inject                                                                                                                                                                                                                                                                  |
| `EventStore` / `EventBus`                                 | `event.Store` / `event.Bus`         | in-memory                               | Shared infrastructure; store must be a `SeekableJournal`                                                                                                                                                                                                                                                            |
| `ReadModelDB`                                             | `*sql.DB`                           | nil (in-memory)                         | SQL-backed read models that survive restarts; flattened path is SQLite-only (dialect-probed at `New`) — Postgres/MySQL go through `ServiceConfig.ReadModelDialect`                                                                                                                                                  |
| `Title` / `AccentColor`                                   | `string`                            | `"cqrs-htmx"` / sky                     | Branding across all panels                                                                                                                                                                                                                                                                                          |
| `AdminPath`                                               | `string`                            | `"/admin/"`                             | Trailing slash auto-normalized; links follow the mount                                                                                                                                                                                                                                                              |
| `DashboardPath`                                           | `string`                            | `"/dashboard/"`                         | Trailing slash auto-normalized                                                                                                                                                                                                                                                                                      |
| `LoginRedirect`                                           | `string`                            | `"/admin/"`                             | Post-login destination                                                                                                                                                                                                                                                                                              |
| `HealthPath`                                              | `string`                            | `"/health"` (`"-"` = off)               | Readiness endpoint; must not collide with other paths                                                                                                                                                                                                                                                               |
| `LivePath`                                                | `string`                            | off                                     | Opt-in public liveness probe (always-200 while serving) — point k8s `livenessProbe` here so journal drain never restarts the pod                                                                                                                                                                                    |
| `EventCatalogPath` / `ProjectionStatusPath` / `DebugPath` | `string`                            | off                                     | Opt-in session-gated JSON endpoints: 21-event catalog (immutable cache+ETag), live projection statuses (no-cache), build metadata                                                                                                                                                                                   |
| `SSEPath`                                                 | `string`                            | off                                     | Session-gated shared SSE feed of all committed events                                                                                                                                                                                                                                                               |
| `SSEScriptPath`                                           | `string`                            | `"/sse.js"` (with `SSEPath`; `-` = off) | HTMX SSE extension script served beside the feed (DataStarScriptPath symmetry)                                                                                                                                                                                                                                      |
| `SSEHeartbeatInterval`                                    | `time.Duration`                     | `15s` (0 = off)                         | Keep-alive comment frames on `/sse`                                                                                                                                                                                                                                                                                 |
| `SSEMaxReplay`                                            | `int`                               | `0` (transport default 1000)            | Cap on first-connect journal backfill for `/sse`                                                                                                                                                                                                                                                                    |
| `SSEFilter`                                               | `func(sse.Event) bool`              | nil (full feed)                         | Scopes the shared `/sse` feed — both the live stream AND journal replay (replay filters fail-closed); see `docs/guides/sse-and-datastar.md` §Scoped Feeds (setup ≥ v4.12.0)                                                                                                                                         |
| `DataStarPath` / `DataStarScriptPath`                     | `string`                            | off / `"/datastar.js"`                  | Session-gated DataStar SSE feed on the shared hub + embedded SDK script (ADR-0050; setup ≥ v4.10.0)                                                                                                                                                                                                                 |
| `Service`                                                 | `*usermgmt.Service`                 | built by `New`                          | Adopt your own service; panels wire on top of it                                                                                                                                                                                                                                                                    |
| `ServiceConfig`                                           | `*usermgmt.ServiceConfig`           | nil                                     | Escape hatch: full `usermgmt.ServiceConfig` override                                                                                                                                                                                                                                                                |
| `Observability`                                           | `*middleware.OTelBundle`            | nil (no OTel middleware)                | Wire a go-cqrs-lite OTel bundle into the service's event bus (publish + handle spans); conflicts with `Service`/`ServiceConfig` — those users set `SecurityHooks` directly                                                                                                                                          |
| `CookieName` / `SessionTTL`                               | `string` / `time.Duration`          | `"session"` / 24h                       | Session cookie configuration                                                                                                                                                                                                                                                                                        |
| `AuthHandlerConfig`                                       | `*usermgmt.HandlerConfig`           | nil                                     | HTTP-layer auth hardening: the six per-group rate limiters (`WebAuthnRateLimit`, ...), cookie `Secure`/`SessionMaxAge`, handler `Timeout`, OAuth2 redirect URLs, import/export authorizer; empty `CookieName` inherits `CookieName`                                                                                 |
| `Metrics`                                                 | `*appkit.MetricsConfig`             | nil                                     | Prometheus `/metrics` + request metrics on the `RunWithAppkit` path only; Basic-Auth-gated by default                                                                                                                                                                                                               |
| `Version`                                                 | `string`                            | `""`                                    | `GET /version` JSON build stamp on the `RunWithAppkit` path only; also labels `appkit_build_info`                                                                                                                                                                                                                   |
| `CSRF`                                                    | `*httputil.CSRFConfig`              | nil (= zero config)                     | Admin CSRF; set `{Secure: true}` for HTTPS (silences the plain-HTTP cookie warning)                                                                                                                                                                                                                                 |
| `Logger`                                                  | `*slog.Logger`                      | `slog.Default()`                        | Structured auth event logging                                                                                                                                                                                                                                                                                       |
| `LogoutURL` / `SSEURL`                                    | `string`                            | hidden / off                            | Logout link; admin panel real-time sync indicator                                                                                                                                                                                                                                                                   |
| `AdminMode` / `TenantID`                                  | `adminui.Mode` / `TenantID`         | super-admin                             | Tenant-scoped admin panel (TenantID required in that mode)                                                                                                                                                                                                                                                          |
| `AdminAuthorizer`                                         | `func(*usermgmt.User) error`        | role-based                              | Custom admin access control                                                                                                                                                                                                                                                                                         |
| `DashboardAuthorizer`                                     | `func(*http.Request) error`         | none                                    | Extra dashboard gate (runs after the session gate)                                                                                                                                                                                                                                                                  |
| `DashboardLayout`                                         | `dashboardui.LayoutFunc`            | none (built-in shell)                   | Replace the dashboard's own sidebar/header shell with a consumer-owned document — pages render inside your app chrome (see dashboardui README § Integration Modes)                                                                                                                                                  |
| `OnProjectionFailed`                                      | `func(name, lastErr string)`        | none                                    | Alerting hook when a projection exhausts restarts                                                                                                                                                                                                                                                                   |
| `AsyncStartup`                                            | `bool`                              | `false`                                 | Bind immediately; `/health` gates readiness during drain                                                                                                                                                                                                                                                            |
| `DashboardReadOnly`                                       | `*bool`                             | `true`                                  | Set `false` at your own risk (enables reset/DLQ replay)                                                                                                                                                                                                                                                             |
| `DashboardPageSize`                                       | `int`                               | 50                                      | Rows per dashboard table page (max 200)                                                                                                                                                                                                                                                                             |
| `LoginNoRegistration`                                     | `bool`                              | `false`                                 | Hide the registration section                                                                                                                                                                                                                                                                                       |
| `LoginCSSPath`                                            | `string`                            | `"/app.css"`                            | URL of the compiled Tailwind stylesheet the login page loads — the consumer MUST compile one scanning the loginpage package (see [Styling](#styling)); the admin/dashboard panels need nothing (self-contained bundles)                                                                                               |
| `DisableAdmin` / `DisableDashboard` / `DisableLogin`      | `bool`                              | `false`                                 | Feature flags to shrink the route surface                                                                                                                                                                                                                                                                           |
| `DisableAuth`                                             | `bool`                              | `false`                                 | Build no auth handler: `Bundle.Auth` is nil, no `/auth/*` routes mount, the service and panels stay (own-login-endpoint mode). Rejects `DisableLogin=false` and `AuthHandlerConfig` at `New` (ADR-0054)                                                                                                             |
| `DisableService`                                          | `bool`                              | `false`                                 | The identity-external shell (ADR-0054): no usermgmt.Service, no auth, no panels; `Stores` from `EventStore`/`EventBus` (memory + watermill defaults). Session-gated surfaces (feeds, machine endpoints) are rejected at `New` — their gate would dereference the missing service; health = your `HealthChecks` only |
| `ExtraMiddleware`                                         | `[]func(http.Handler) http.Handler` | nil                                     | Your HTTP middleware, composed INSIDE the built-in security stack (first entry outermost of the extras); applies to every serve path (`Handler`, `Run`, `RunHandler`, `RunWithAppkit`)                                                                                                                              |
| `DisableSecurityMiddleware`                               | `bool`                              | `false`                                 | Remove the built-in security layer entirely — you own the whole outer chain; rebuild it in `ExtraMiddleware` (e.g. `cqrshtmx.RecommendedSecurityMiddleware()` plus your CORS) if you still want it                                                                                                                  |
| `HealthChecks`                                            | `[]cqrshtmx.NamedCheck`             | nil                                     | Your readiness checks appended to the built-ins (`projections`, `sse-hub`) on the mounted `/health` — one probe surface instead of a second port; names validated for uniqueness/collisions at `New`; ignored when `HealthPath` is `-`                                                                              |

Invalid configs fail fast at `New` with descriptive errors: paths must start
with `/`, must not be `/` (reserved for the login page), and must be pairwise
distinct — misconfiguration surfaces before `Mount` can panic.

### Owning the middleware chain and health surface

The bundle's chain is `request-logging (if set) → security → routes`.
Two seams change who owns it (both apply to every serve path — `Handler`,
`Run`, `RunHandler`, `RunWithAppkit`):

```go
bundle, _ := setup.New(setup.Config{
    Title: "My App",
    // Your middleware, INSIDE the built-in security stack (so CSP nonces,
    // HSTS, etc. still wrap your layer). First entry is outermost.
    ExtraMiddleware: []func(http.Handler) http.Handler{
        myCORSMiddleware, myRequestIDMiddleware,
    },
    // ...or take the whole outer chain: the built-in security layer is
    // removed and you rebuild it (if you want it) in ExtraMiddleware.
    // DisableSecurityMiddleware: true,
})
```

Zero values keep the historical behavior: no extras, built-in security on,
chain byte-identical to pre-seam releases.

The same one-surface idea applies to readiness: `HealthChecks` appends your
`cqrshtmx.NamedCheck` entries to the built-ins (`projections`, `sse-hub`) on
the mounted `/health`, so a database or downstream dependency shares the
bundle's 503-while-draining contract instead of needing its own probe:

```go
HealthChecks: []cqrshtmx.NamedCheck{
    cqrshtmx.NewNamedCheck("postgres", db.PingContext), // names must be unique
},
```

Names are validated at `New` (non-empty, non-nil checks, no collisions with
each other or the built-ins); the field is ignored when `HealthPath` is `-`.

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

### ServiceConfig override (escape hatch)

If you want the bundle to keep building and owning the service — but need
`usermgmt.ServiceConfig` knobs the flattened fields cannot express (`MaxUsers`,
`TokenPepper`, `SecurityHooks`, `CheckpointStore`, `SnapshotConfig`,
`SessionStore`, `Lockout`, ...) — pass a `ServiceConfig` instead:

```go
bundle, err := setup.New(setup.Config{
    ServiceConfig: &usermgmt.ServiceConfig{
        MaxUsers:    1, // single-user deployment: registration closes after user #1
        TokenPepper: pepper,
    },
})
defer func() { _ = bundle.Close() }() // closes the service it built
```

Precedence is `Service` > `ServiceConfig` > flattened fields. `Service` and
`ServiceConfig` are mutually exclusive, and either conflicts with the flattened
service-construction fields (`EventStore`, `TOTP`, ... `AsyncStartup`) — set
those inside `ServiceConfig` instead. The bundle applies exactly one default on
top of your override: a nil `AuditLog` gets the in-memory audit log, matching
the flattened path.

### Shared SSE endpoint

Set `SSEPath` to mount a session-gated endpoint streaming every event committed
to the event bus as a small JSON envelope (`type`, `streamId`, `version`, ...).
Reconnecting clients resume from their `Last-Event-ID`, and first-time
subscribers receive a journal backfill when the event store implements
`event.Journal` (plain in-memory stores stream live-only). A heartbeat comment
frame is sent every `SSEHeartbeatInterval` (default 15s; `0` or negative
disables) so proxies and load balancers keep the connection open.
`bundle.Broadcaster` is the fan-out hub behind it — subscribe to it (or share
its `Hub()` with a DataStar broadcaster, as `Bundle.DataStarBroadcaster` does)
to push custom real-time payloads through the same connection topology.

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

## Capability floor: what your dependencies unlock

`setup.New` hard-requires `event.SeekableJournal` from a custom `EventStore`
(the default in-memory store implements it) — that single interface is what
projectionhost checkpoints and SSE replay are built on. Everything else
degrades per-interface. When you bring your own store, panels and features
light up according to what it implements:

| Your `EventStore` implements                | Unlocks                                                                                                                                                                                      |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `event.SeekableJournal` (required)          | Projection checkpoints + restart resume; efficient position-based SSE replay (`ReadFrom` cursors); dashboard event log with pagination                                                       |
| `event.Journal` (implied by the above)      | Global event log panel; SSE first-connect backfill via `ReadAll` + filter (the fallback when a cursor-efficient path is absent); aggregate browser via auto-created in-memory `StreamReader` |
| `event.EventSource` (implied by `Store`)    | Aggregate detail view + time-travel in the dashboard                                                                                                                                         |
| also `projectionhost.Host` (wired by `New`) | Projection health panel + DLQ view                                                                                                                                                           |

The machine endpoints (`EventCatalogPath`, `ProjectionStatusPath`, `DebugPath`)
have no store requirements: the catalog comes from
`usermgmt.DefaultEventCatalog()` (the 21 identity events), projection statuses
from the service, and debug metadata from the build.

Practical consequence: a minimal live-only store that cannot replay history
still boots the bundle but silently loses the replay-backed surfaces above —
which is exactly why `New` refuses non-`SeekableJournal` stores rather than
shipping a half-working dashboard. Wiring `dashboardui` by hand lights the
same per-interface surfaces, plus two more the bundle does not wire:
`CommandJournal`/`QueryJournal` (command/query audit panels) and a custom
`StreamReader`; see `dashboardui/README.md`.

## Troubleshooting

| Symptom                                                | Cause and fix                                                                                                                                                                                                                              |
| ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Admin/dashboard routes return **401**                  | Deliberate: both panels are session-gated like the API. Log in through the login page first. To serve a public dashboard, mount `bundle.Dashboard.Handler()` yourself instead of relying on `Mount`.                                       |
| `/health` returns **503**                              | The projection readiness gate is doing its job: at least one projection is still replaying/draining (always the case briefly at startup, and while `AsyncStartup: true` replays in the background). Poll until 200 before routing traffic. |
| `setup.New` rejects the config                         | `New` validates paths: `/` is rejected (the login page owns root), duplicate mounts are rejected, trailing slashes are normalized. Fix the `Config` paths — the error names the offending pair.                                            |
| A second `bundle.Mount(mux)` **panics**                | `Mount` is once-only by design (stdlib mux rejects duplicate patterns). Call it once per mux.                                                                                                                                              |
| SSE clients connect but receive nothing                | A buffering proxy in front of the server (nginx `proxy_buffering`, CDNs). Disable response buffering for the SSE route (`X-Accel-Buffering: no`) and make sure the proxy does not impose short read timeouts.                              |
| TOTP logins fail after upgrading an old SQL read model | Rows written before checkpointed hydration lack the TOTP secret. Run `RebuildProjection(ctx, "user-read-model")` once after upgrading; affected users would otherwise have to re-enroll. See `docs/guides/event-replay-and-rebuild.md`.    |

## Security and TLS posture

`setup` serves plain HTTP and performs **no TLS termination** — put a reverse
proxy (Caddy, nginx, a cloud LB) in front for HTTPS, HSTS, and certificate
handling. The default wrap from [Bundle.Middleware] (applied by
[Bundle.Handler], [Bundle.Run], and [Bundle.RunHandler]) is
`RecommendedSecurityMiddleware`: security headers, a per-request CSP nonce,
and panic recovery. CSRF protection guards the admin panel's mutation
endpoints by default; customize it via `Config.CSRF` (the default config
issues its cookie without the Secure flag — fine for local HTTP dev; set
`Secure: true` behind HTTPS):

```go
handler := bundle.Handler(mux) // default security chain, or compose your own
```

Session cookies are issued by usermgmt; when terminating TLS at a proxy, keep
the proxy-to-app hop on a private network or loopback so the session cookie is
never transported in the clear. Rate limiting and body limits are opt-in via
`httputil` — see `docs/guides/leveraging-httputil.md`.

### Why the `/auth/*` mutations carry no CSRF token (deliberate)

The admin panel's form-based mutations are CSRF-protected by a token; the auth
endpoints (`POST /auth/register`, the WebAuthn/TOTP/OAuth2 ceremonies,
`POST /auth/logout`) intentionally are not. Two reasons:

1. The login/registration ceremonies are **unauthenticated** — they establish
   a session rather than ride one. CSRF is an attack on _ambient_ cookie
   authorization; a cross-site POST to `/auth/webauthn/login/begin` creates
   nothing and leaks nothing.
2. The auth mutations that DO ride the session cookie (`logout`, credential
   deletion) are guarded by the cookie itself: usermgmt issues it with
   `SameSite=Strict`, so a cross-site request never carries it. Defense in
   depth for these endpoints is available via `Config.AuthHandlerConfig`
   (rate limits, timeouts) rather than a second token dance.

If your threat model requires token CSRF on auth mutations anyway (e.g. you
must support browsers that predate strict SameSite), wrap
`bundle.Auth.RegisterRoutes` behind `bundle.CSRFMiddleware()` on your own mux
instead of using `Bundle.Mount` for those routes.

### Credential enrollment requires the owner's session (library-enforced)

`POST /auth/webauthn/register/{begin,finish}` used to accept any `user_id`
from the request, which let an unauthenticated caller enroll a passkey onto an
arbitrary (enumerable, time-ordered ULID) account. The library now enforces
the owner-match rule: **401** without a session, **403** when the requested
`user_id` is not the session user. First-user bootstrap is unaffected because
`POST /auth/register` sets the session cookie before the login page continues
into the ceremony (same-origin fetches carry it), and headless or
administrative enrollment stays possible through the service-level API
(`Service.BeginRegistration` / `FinishRegistration`). There is no HTTP-level
opt-out — this is an authorization invariant, not a default. Decision record:
`docs/adr/0055-owner-session-gated-credential-ceremonies.md`.

For your own session-gated surfaces, setup exports the two gates it uses
internally: `setup.RequireSession` (401, JSON/API convention) and
`setup.RequireSessionRedirect(loginURL)` (303, HTML convention). Mount either
AFTER `bundle.SessionMiddleware()`, which enriches but never blocks.

## See also

- `docs/guides/fullstack-wiring.md` — full wiring guide (SDK vs manual)
- `docs/guides/setup-vs-hand-wiring.md` — the decision tree: when setup is the wrong answer
- `examples/setup-demo/` — runnable demo of the whole bundle
- `docs/guides/async-projection-startup.md` — the readiness model behind `/health`

## Benchmarks

`BenchmarkSpikeBaselineVsAppkit` (`setup/run_appkit_test.go`) compares the request-path cost of the bundle's two serve paths: `RunHandler` (httputil.Server) vs `RunWithAppkit` (appkit.Service, the spike for ADR-001 adoption). Three sub-benchmarks: `baseline-httputil`, `appkit-service`, and `json-roundtrip` (no HTTP — an envelope-shaped JSON encode/decode in isolation, so stack deltas attribute to middleware rather than codec work). The bench runs 5× with benchstat-friendly metrics (ns/op, req/s via `b.ReportMetric`, B/op + allocs/op via `-benchmem`); the spike passes `appkit.LogLevelError` to suppress the per-request INFO line so the comparison isolates stack overhead from observability overhead.

Prefer the gate wrapper, which prints the benchstat table and fails on a median regression vs the machine-pinned baseline:

```sh
# from repo root:
nix run .#bench-spike                       # gate (10% median threshold)
nix run .#bench-spike -- --save-baseline    # re-pin after bench changes

# manual exploration (hermetic, GOWORK=off):
GOEXPERIMENT=jsonv2 GOWORK=off go test \
    -run xxx \
    -bench '^BenchmarkSpikeBaselineVsAppkit$' \
    -benchtime=2s -benchmem -count=5 -timeout=120s \
    ./setup | tee /tmp/before.txt
# edit, repeat into /tmp/after.txt
benchstat /tmp/before.txt /tmp/after.txt
```

Baseline numbers (5× runs, machine-pinned — the gate-comparable artifact is `docs/benchmarks/setup-baseline.raw.txt`; see `docs/benchmarks/README.md` for the raw-vs-markdown split and re-pin flow):

| Case                | ns/op | req/s | B/op | allocs/op |
| ------------------- | ----- | ----- | ---- | --------- |
| `baseline-httputil` | ~21k  | ~47k  | —    | 180       |
| `appkit-service`    | ~24k  | ~40k  | —    | 207       |
| `json-roundtrip`    | ~1.0k | —     | 483  | 6         |

(2026-08-29 pinning, AMD Ryzen AI MAX+ 395; the 2026-08-17 Xeon numbers are not comparable.) The alloc gap is the per-request cost the appkit middleware stack adds; ns/op overhead disappears in any real handler doing I/O, and the ~1µs codec floor shows how little of it is JSON. `LogLevelError` is load-bearing — without it, appkit's per-request formatted INFO line dominates the measurement (see the comparison report, finding 7).
