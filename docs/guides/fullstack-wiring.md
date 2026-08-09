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
    "net/http"

    "github.com/larsartmann/cqrs-htmx/setup/v4"
    totp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
)

func main() {
    bundle, err := setup.New(setup.Config{
        Title:   "My App",
        TOTP:    totp.New(totp.Config{Issuer: "MyApp"}),
    })
    if err != nil { panic(err) }
    defer bundle.Close()

    mux := http.NewServeMux()
    http.ListenAndServe(":8080", bundle.Handler(mux))
}
```

### What you get

| Route          | Panel                                         | Auth                  |
| -------------- | --------------------------------------------- | --------------------- |
| `/auth/*`      | Registration, login (TOTP/WebAuthn/OAuth2)    | Public (registration) |
| `/admin/*`     | Admin dashboard (users, tenants, memberships) | Session + CSRF        |
| `/dashboard/*` | CQRS/ES observability (events, projections)   | Session               |
| `/health`      | Readiness check (verifies projection health)  | Public                |
| `/`            | Login page                                    | Public                |

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

## External Integrations (Optional, future)

### go-health-dashboard (proposed `health/v4` module)

Real-time health dashboard with Kubernetes probes (`/healthz`, `/readyz`, `/startupz`).
Separate module so consumers who don't need it pay zero dep cost.

```go
// FUTURE — proposed, not yet built
probe := healthint.NewProbe(bundle.Service)
dash := healthint.NewDashboard(probe, healthint.WithTitle("Health"))
dash.RegisterRoutes(mux) // /health, /healthz, /readyz, /startupz
```

### samber-do-auditlog (proposed `auditlog/v4` module)

DI lifecycle audit logging with self-contained HTML visualization.

```go
// FUTURE — proposed, not yet built
hooks := auditint.WithAuditLog(auditlog.Config{Enabled: true})
injector := do.New(hooks...)
auditint.MountReport(mux, report, "/debug/di/")
```

## See Also

- [Architecture review: module integration & composability](../architecture-understanding/2026-08-09_05-36_module-integration-composability.html)
- [Production readiness checklist](production-readiness.md)
- [Middleware ordering rules](dispatch-middleware-ordering.md)
- [samber/do integration guide](leveraging-samber-do.md)
- Examples: [admin-demo](../../examples/admin-demo/), [dashboard-demo](../../examples/dashboard-demo/)
