# Full-Stack Wiring Guide

> How to compose all cqrs-htmx sub-modules into a single application — and optionally integrate
> go-health-dashboard and samber-do-auditlog.

This guide covers the **composition gap**: individual modules are superbly decoupled, but wiring
them together requires understanding each module's Config struct and the correct middleware
ordering. See the [architecture review](../architecture-understanding/2026-08-09_05-36_module-integration-composability.html)
for the full analysis.

## Prerequisites

- Go 1.26.5+ with `GOEXPERIMENT=jsonv2`
- All modules use `/v4` suffix: `github.com/larsartmann/cqrs-htmx/v4`, `.../usermgmt/v4`, etc.

## Integration Paths

```
Do you need CQRS dispatch (command/query endpoints)?
+- NO  ->  PATH 0: Building blocks only (middleware, SSE, errors, HTMX)
+- YES ->  Need user accounts / authentication?
          +- NO  ->  PATH A: root App only
          +- YES ->  Need a ready-made admin dashboard + CQRS observability?
                    +- NO  ->  PATH B: root App + usermgmt Service + AuthHandler
                    +- YES ->  PATH C: EVERYTHING (this guide)
```

**This guide covers PATH C.** For simpler paths, see the [skill cheat sheet](../../.agents/skills/cqrs-htmx/SKILL.md).

---

## The Full Stack (Manual Wiring)

The full stack composes 6 layers. Each layer connects to the next via a single injected
dependency. Here is the complete wiring, annotated.

### Step 1: Shared infrastructure (stores, bus, projection host)

**Key insight:** `*usermgmt.Service` creates its own internal stores and does not expose them.
`dashboardui.Config` needs the raw stores (Journal, EventSource, EventBus, ProjectionHost, etc.).
**For full-stack wiring, you construct the shared infrastructure first**, then pass it to both
the Service setup and the Dashboard.

```go
// Shared event store — both the Service and the Dashboard read from this.
store := memorystorage.NewMemoryStore()
cmdStore := memorystorage.NewMemoryCommandStore()
queryStore := memorystorage.NewMemoryQueryStore()
snapStore := memorystorage.NewMemorySnapshotStore()
bus := eventtest.NewFakeBus() // or event.NewBus() for production

// Shared projection host — the Dashboard's projection panel reads from this.
projHost, err := projectionhost.New(store, memorystorage.NewMemoryCheckpointStore())
if err != nil { log.Fatal(err) }
```

> **For SQL-backed setups**, replace the memory stores with SQL stores and pass the same `*sql.DB`
> to both the usermgmt SQL setup constructors and the dashboard store constructors.

### Step 2: User management Service

```go
totpProvider := totp.New(totp.Config{Issuer: "MyApp"})
// webauthnProvider, err := webauthn.New(webauthn.Config{RPID: "myapp.com", ...})
// oauth2Provider, err := oauth2.New(oauth2.Config{Providers: map[string]oauth2.ProviderConfig{...}})

svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
    AuditLog: usermgmt.NewAuditLog(),
    TOTP:     totpProvider,
    // WebAuthn: webauthnProvider,
    // OAuth2:   oauth2Provider,
})
if err != nil { log.Fatal(err) }
defer svc.Close()
```

> **Note:** For full SQL-backed setups with shared stores, use `usermgmt.NewSQLiteEventSourcedSetup`
> or `NewPostgresEventSourcedSetup` instead of `NewService`. These accept a `*sql.DB` and create
> SQL-backed stores internally. The Dashboard can read from the same database via its own SQL
> store constructors.

### Step 3: CQRS App (your domain endpoints)

```go
cmdDisp := command.NewDispatcher()
qryDisp := query.NewDispatcher()

// Register your domain handlers BEFORE building endpoints:
cmdDisp.Register("CreateOrder", orderHandler)
qryDisp.Register("ListOrders", listOrdersHandler)

app := cqrshtmx.MustNew(cqrshtmx.Config{
    Commands: cmdDisp,
    Queries:  qryDisp,
})
```

### Step 4: UI panels

```go
// Admin panel (user management dashboard)
adminPanel, err := adminui.New(adminui.Config{
    Service:     svc,             // required: *usermgmt.Service
    Title:       "MyApp Admin",
    AccentColor: "#0ea5e9",
    LogoutURL:   "/logout",
    SSEURL:      "/admin/-/events", // optional: enables sync indicator
})
if err != nil { log.Fatal(err) }

// CQRS/ES observability dashboard
reader := listing.NewInMemoryStreamReader(store)
dash, err := dashboardui.New(dashboardui.Config{
    Title:          "MyApp CQRS Dashboard",
    EventSource:    store,       // shared from Step 1
    Journal:        store,       // shared from Step 1
    StreamReader:   reader,
    CommandJournal: cmdStore,    // shared from Step 1
    QueryJournal:   queryStore,  // shared from Step 1
    SnapshotStore:  snapStore,   // shared from Step 1
    ProjectionHost: projHost,    // shared from Step 1
    EventBus:       bus,         // shared from Step 1
    PageSize:       25,
})
if err != nil { log.Fatal(err) }

// Login page
loginHandler, err := loginpage.New(loginpage.Config{
    Service: svc,               // required: *usermgmt.Service
    Title:   "Sign In",
    Brand:   "MyApp",
    Redirect: "/admin/",
})
if err != nil { log.Fatal(err) }
```

### Step 5: Auth routes + session middleware

```go
auth := usermgmt.NewAuthHandler(svc)

sessionMW := usermgmt.NewSessionMiddleware(svc, "session")
csrfMW := httputil.CSRFMiddleware(httputil.CSRFConfig{})
```

### Step 6: Mount everything + middleware chain

```go
mux := http.NewServeMux()

// Auth routes (register, webauthn, logout, me) — outside CSRF (registration is a POST)
auth.RegisterRoutes(mux)

// Login page — outside session middleware (it IS the login page)
mux.Handle("/", loginHandler.Handler())

// Admin panel — behind session + CSRF + panel security middleware
mux.Handle("/admin/", sessionMW(csrfMW(adminPanel.Middleware()(
    http.StripPrefix("/admin", adminPanel.Handler()),
))))

// Admin SSE — behind session only (GET, no CSRF needed)
mux.Handle("/admin/-/events", sessionMW(adminSSEHandler(broadcaster)))

// CQRS Dashboard — behind session (read-only observability, auth required)
mux.Handle("/dashboard/", sessionMW(dash.Middleware()(
    http.StripPrefix("/dashboard", dash.Handler()),
)))

// Your domain endpoints — behind session + CSRF + App middleware
mux.Handle("POST /orders", sessionMW(csrfMW(app.Middleware()(
    app.Command("CreateOrder", cqrshtmx.DecodeJSON(mapCreateOrder)),
))))

// HTMX script (admin panel serves its own; register for your custom endpoints)
mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())

// Serve
http.ListenAndServe(":8080", cqrshtmx.Chain(
    cqrshtmx.RecommendedSecurityMiddleware(), // security headers + nonce + recovery
)(mux))
```

### Middleware ordering rules

```
OUTER → INNER:

RecommendedSecurityMiddleware()    — security headers, per-request CSP nonce, panic recovery
  sessionMW                        — authenticate the session cookie, inject *usermgmt.User
    csrfMW (mutations only)         — validate CSRF token on POST/PUT/DELETE
      HTMXMiddleware                — detect HTMX requests (use app.Middleware() which wraps this)
        your handler                — the actual endpoint
```

The non-negotiable rules:
1. **Session middleware OUTSIDE CSRF** — authenticate before checking CSRF.
2. **CSRF only on mutations** — GET requests (SSE, dashboard views) skip CSRF.
3. **Security middleware OUTERMOST** — recovery must catch panics from everything.

---

## The Wiring Problem (and the proposed solution)

The manual wiring above is ~50 lines of boilerplate with several friction points:

1. **dashboardui.Config has 11+ fields** — all of which must match the stores you passed to the
   Service setup. No compile-time check catches a mismatch.
2. **No "mount everything" helper** — each panel is mounted independently with its own middleware
   chain, and a typo in the ordering silently breaks auth.
3. **The Service doesn't expose its internal stores** — so you must construct shared stores
   yourself and pass them to both sides.

### Proposed: `cqrs-htmx/setup/v4` module

A new optional module (zero external deps) that eliminates the boilerplate:

```go
// FUTURE — not yet implemented. See TODO_LIST.md and ROADMAP.md.
import "github.com/larsartmann/cqrs-htmx/setup/v4"

// One call replaces Steps 1, 4, and 6:
panels, err := setup.MountAll(mux, setup.MountAllConfig{
    Service:     svc,       // *usermgmt.Service
    App:         app,       // *cqrshtmx.App
    Stores:      stores,    // setup.Stores{Journal: store, Bus: bus, ...} (shared infra)
    AdminPath:   "/admin/",
    DashboardPath: "/dashboard/",
    EnableLogin: true,
})
// panels.Admin, panels.Dashboard, panels.Login are available for further customization
```

---

## External Integrations (Optional)

### go-health-dashboard

Adds real-time health dashboard with Kubernetes-grade probes (`/healthz`, `/readyz`, `/startupz`).

**Why a separate module?** go-health-dashboard pulls templ, templ-components, go-datastar, go-sse,
go-health. These should NOT be transitive deps for consumers who only want CQRS dispatch.

```go
// FUTURE — cqrs-htmx/health/v4 module. See ROADMAP.md.
import healthint "github.com/larsartmann/cqrs-htmx/health/v4"

// Bridge usermgmt projection health → go-health checks
probe := healthint.NewProbe(svc,
    healthint.WithCriticalServices("usermgmt"),
)
dash := healthint.NewDashboard(probe,
    healthint.WithTitle("MyApp Health"),
    healthint.WithNonceExtractor(httputil.NonceFromRequest),
)
dash.Start(ctx)
defer dash.Shutdown()
dash.RegisterRoutes(mux) // mounts /health, /healthz, /readyz, /startupz
```

### samber-do-auditlog

Adds DI lifecycle audit logging with self-contained HTML visualization.

```go
// FUTURE — cqrs-htmx/auditlog/v4 module. See ROADMAP.md.
import auditint "github.com/larsartmann/cqrs-htmx/auditlog/v4"

// One-line DI audit logging (if using samber/do)
hooks := auditint.WithAuditLog(auditlog.Config{
    Enabled:     true,
    ContainerID: "myapp",
})
injector := do.New(hooks...)

// Mount the HTML report viewer
report := auditint.BuildReport(injector)
auditint.MountReport(mux, report, "/debug/di/")
```

---

## Checklist: Full-Stack Wiring

- [ ] Shared stores constructed (Step 1)
- [ ] `*usermgmt.Service` created with auth strategies (Step 2)
- [ ] `*cqrshtmx.App` created with registered handlers (Step 3)
- [ ] adminui panel created with `Service: svc` (Step 4)
- [ ] dashboardui created with shared stores (Step 4)
- [ ] loginpage created with `Service: svc` (Step 4)
- [ ] `NewAuthHandler(svc).RegisterRoutes(mux)` — auth routes mounted (Step 5)
- [ ] Session middleware OUTSIDE CSRF (Step 6)
- [ ] CSRF only on mutation endpoints (Step 6)
- [ ] Security middleware OUTERMOST (Step 6)
- [ ] `svc.Close()` deferred (graceful shutdown)
- [ ] `projHost.Stop()` deferred (projection shutdown)
- [ ] HTMX script handler registered (unless admin panel serves it internally)

## See Also

- [Architecture review: module integration & composability](../architecture-understanding/2026-08-09_05-36_module-integration-composability.html)
- [Production readiness checklist](production-readiness.md)
- [Middleware ordering rules](dispatch-middleware-ordering.md)
- [Leveraging go-cqrs-lite](leveraging-go-cqrs-lite.md)
- [Leveraging httputil](leveraging-httputil.md)
- [samber/do integration guide](leveraging-samber-do.md)
- Examples: [admin-demo](../../examples/admin-demo/), [dashboard-demo](../../examples/dashboard-demo/), [samber-do-demo](../../examples/samber-do-demo/)
