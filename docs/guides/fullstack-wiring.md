# Full-Stack Wiring Guide

> How to compose all cqrs-htmx sub-modules into a single application using the `setup/v4` SDK module,
> and optionally integrate go-health-dashboard and samber-do-auditlog.

## The One-Call SDK

The `setup/v4` module eliminates all wiring boilerplate. It creates shared stores, constructs the
`usermgmt.Service`, builds the admin/dashboard/login UI panels, and mounts everything with the
correct middleware ordering — in one import.

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/larsartmann/cqrs-htmx/setup/v4"
    totp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    bundle, err := setup.New(setup.Config{
        Title:   "My App",
        TOTP:    totp.New(totp.Config{Issuer: "MyApp"}),
    })
    if err != nil { log.Fatal(err) }

    // Mount, serve with safe timeouts, drain gracefully, close.
    if err := bundle.Run(ctx, ":8080"); err != nil { log.Fatal(err) }
}
```

`Bundle.Run` applies `ReadHeaderTimeout` + `IdleTimeout` but no `WriteTimeout` —
the dashboard's SSE streams outlive any fixed write deadline. To add your own
routes next to the bundle's, compose a mux and use `Bundle.RunHandler`:

```go
mux := http.NewServeMux()
mux.HandleFunc("POST /orders", myOrdersHandler)
err := bundle.RunHandler(ctx, ":8080", bundle.Handler(mux))
```

### What you get

| Route          | Panel                                         | Auth                      |
| -------------- | --------------------------------------------- | ------------------------- |
| `/auth/*`      | Registration, login (TOTP/WebAuthn/OAuth2)    | Public (registration)     |
| `/admin/*`     | Admin dashboard (users, tenants, memberships) | Session + CSRF (401 else) |
| `/dashboard/*` | CQRS/ES observability (events, projections)   | Session gate (401 else)   |
| `/health`      | Readiness check (verifies projection health)  | Public                    |
| `/`            | Login page                                    | Public                    |

Custom `AdminPath`/`DashboardPath` values are passed to the panels as their
`BasePath`, so every internal link and HTMX target matches the mount location.
Paths are validated at `New`: colliding paths (or `/`, which belongs to the
login page) are rejected with a descriptive error instead of panicking at
Mount time.

### Customization

Every sub-component is exposed on the `Bundle` struct:

```go
bundle, _ := setup.New(setup.Config{Title: "My App", DisableDashboard: true})
defer bundle.Close()

// Access the Service for domain operations
user, err := bundle.Service.Register(ctx, email)

// Access the Auth handler to add custom routes
bundle.Auth.RegisterRoutes(mux)

// Add your own CQRS endpoints with shared middleware
app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp})
mux.Handle("POST /orders",
    bundle.SessionMiddleware()(
        app.Command("CreateOrder", cqrshtmx.DecodeJSON(mapOrder)),
    ),
)
```

### Feature flags

Go zero-value (false) = **all panels enabled**. Set to `true` to disable:

```go
setup.Config{
    Title:            "API Only",
    DisableAdmin:     true,  // no admin panel
    DisableDashboard: true,  // no CQRS dashboard
    DisableLogin:     true,  // use your own login page
}
```

### Persistence

Defaults to in-memory (lost on restart). Provide your own stores for production:

```go
setup.Config{
    EventStore:  myEventStore,  // must implement event.SeekableJournal
    EventBus:    myEventBus,
    ReadModelDB: sqlDB,         // optional: enables SQL-backed read models
}
```

### Service source precedence (who builds the `usermgmt.Service`)

`setup.New` builds the service from exactly ONE of three sources. The
precedence is strict and conflicts fail fast at `New` — nothing is silently
ignored:

1. **`Config.Service`** — adopt a service YOU built (custom stores,
   SecurityHooks, MaxUsers, snapshotting, custom `AuditLog`). The bundle
   sources its shared infrastructure from the adopted service
   (`svc.Journal()` / `svc.EventBus()`), and lifecycle ownership stays with
   you: `Bundle.Close` does NOT close an adopted service.
2. **`Config.ServiceConfig`** — the escape hatch: the bundle still builds
   and owns the service, but from a full `*usermgmt.ServiceConfig` you pass
   verbatim (MaxUsers, TokenPepper, SecurityHooks, CheckpointStore,
   SnapshotConfig, SessionStore, Lockout, ...). One default is applied on
   top: a nil `ServiceConfig.AuditLog` gets the in-memory audit log.
   `Bundle.Close` closes a service built this way.
3. **Flattened fields** — the convenience path (`EventStore`, the auth
   provider fields, `SessionTTL`, `Logger`, ...). Use this until you hit a
   knob it cannot express, then switch to `ServiceConfig`.

Mutual exclusion: `Service` and `ServiceConfig` are mutually exclusive, and
either conflicts with the flattened service-construction fields
(`EventStore`, `EventBus`, `ReadModelDB`, `TOTP`, `WebAuthn`, `OAuth2`,
`SessionTTL`, `Logger`, `AsyncStartup`, `OnProjectionFailed`) — set those
inside `ServiceConfig` instead. UI-level fields (paths, colors,
`DashboardReadOnly`, `SSEPath`, ...) stay on `setup.Config` in all modes.

Because `ServiceConfig` is passed verbatim, any FUTURE
`usermgmt.ServiceConfig` capability reaches setup with zero setup-side
changes — do not file "expose X in setup.Config" requests; pass a
`ServiceConfig`.

## Manual Wiring (when you need full control)

If `setup/v4` doesn't fit your needs, wire modules individually. This is the full manual path:

```go
// 1. Shared stores
store := memorystorage.NewMemoryStore()
bus := watermill.NewEventBus()

// 2. User management
svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
    EventStore: store,
    EventBus:   bus,
    AuditLog:   usermgmt.NewAuditLog(),
    TOTP:       totp.New(totp.Config{Issuer: "MyApp"}),
})

// 3. CQRS App
app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp})

// 4. UI panels
admin, _ := adminui.New(adminui.Config{Service: svc, Title: "Admin"})
dash, _ := dashboardui.New(dashboardui.Config{
    EventSource: store, Journal: store.(event.Journal),
    EventBus: bus, ProjectionHost: svc.ProjectionHost(),
})
login, _ := loginpage.New(loginpage.Config{Service: svc, Title: "Sign In"})

// 5. Routes + middleware
mux := http.NewServeMux()
usermgmt.NewAuthHandler(svc).RegisterRoutes(mux)
login.Mount(mux, "/")
admin.Mount(mux, "/admin/")
dash.Mount(mux, "/dashboard/")

sessionMW := usermgmt.NewSessionMiddleware(svc, "session")
csrfMW := httputil.CSRFMiddleware(httputil.CSRFConfig{})
http.ListenAndServe(":8080", cqrshtmx.Chain(
    cqrshtmx.RecommendedSecurityMiddleware(),
)(mux))
```

### Middleware ordering rules

```
OUTER → INNER:
RecommendedSecurityMiddleware()    — security headers, per-request CSP nonce, panic recovery
  SessionMiddleware                — authenticate session cookie, inject *usermgmt.User
    CSRFMiddleware (mutations only) — validate CSRF token on POST/PUT/DELETE
      HTMXMiddleware                — detect HTMX requests
        your handler
```

Non-negotiable rules:

1. **Session OUTSIDE CSRF** — authenticate before checking CSRF.
2. **CSRF on mutations only** — GET (SSE, views) skips CSRF.
3. **Security OUTERMOST** — recovery catches panics from everything.

## External Integrations (Optional)

### go-health + go-health-dashboard (the `health/v4` module)

Real-time health checks and dashboard with one check per projection
(live/stopped pass, drain transient, failed infrastructure — same semantics as
`cqrshtmx.ProjectionReadinessCheck`). Separate module so consumers who don't
need it pay zero dep cost. Available since the v4.8.0 family release.

```go
import (
    "github.com/larsartmann/cqrs-htmx/health/v4"
    gohealth "github.com/larsartmann/go-health"
    healthdashboard "github.com/larsartmann/go-health-dashboard"
)

probe, err := health.NewProbe(bundle.Service,
    gohealth.WithCriticalServices("user-read-model", "casbin-projection"),
    gohealth.WithRefreshInterval(5*time.Second), // drives the probe cache
)
if err != nil { log.Fatal(err) }
if err := probe.Start(ctx); err != nil { log.Fatal(err) } // starts the refresh cache
defer probe.Stop()

// go-health-dashboard UI (HTML / SSE / JSON via Accept negotiation).
// The dashboard serves the probe's CACHE — Start + a refresh interval are
// required for it to show live data.
dash := health.NewDashboard(probe, healthdashboard.WithTitle("Health"))
mux.Handle("/health-dashboard/", http.StripPrefix("/health-dashboard", dash))
```

Already inside a samber/do injector? Merge projection checks with your own
service checks: `gohealth.New(injector, gohealth.WithHealthRecorder(health.Recorder(svc)))` —
projection names win on collision.

### samber-do-auditlog (the `auditlog/v4` module)

DI lifecycle audit logging with a self-contained live HTML viewer (SSE updates,
JSON report API). Available since the v4.8.0 family release.

```go
import (
    cqrsauditlog "github.com/larsartmann/cqrs-htmx/auditlog/v4"
    auditlog "github.com/larsartmann/samber-do-auditlog"
    "github.com/larsartmann/samber-do-auditlog/live"
    "github.com/samber/do/v2"
)

setup, err := cqrsauditlog.WithAuditLog(
    auditlog.Config{MaxEvents: 10_000},
    live.Config{Prefix: "/auditlog"},
)
if err != nil { log.Fatal(err) }

injector := do.NewWithOpts(setup.Opts)
defer func() { _ = injector.Shutdown() }()

mux.Handle("/auditlog/", setup.Viewer) // live dashboard + JSON/SSE API
// setup.Plugin exposes reports/exports for programmatic access.
```

## See Also

- [Architecture review: module integration & composability](../architecture-understanding/2026-08-09_05-36_module-integration-composability.html)
- [Production readiness checklist](production-readiness.md)
- [Middleware ordering rules](dispatch-middleware-ordering.md)
- [samber/do integration guide](leveraging-samber-do.md)
- Examples: [admin-demo](../../examples/admin-demo/), [dashboard-demo](../../examples/dashboard-demo/)
