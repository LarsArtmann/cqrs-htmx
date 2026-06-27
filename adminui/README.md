# adminui — Admin Dashboard for cqrs-htmx

A ready-made, good-looking **Admin Dashboard** for apps built on
[`cqrs-htmx/usermgmt`](../usermgmt). Mount it in one call and get a complete
HTMX-driven UI: a dashboard with live metrics, user management, tenant
management, tenant members, and an audit log.

- **One-call mount** behind your existing session middleware.
- **Two scopes**: a global _Super Admin_ panel, or a per-tenant _Tenant Admin_ panel.
- **Modern look** out of the box — a self-contained stylesheet with automatic
  light/dark theming and a configurable accent color. No Tailwind, no build step.
- **HTMX interactivity** — live search, inline actions, toast notifications.
- **templ-powered** — type-safe HTML components. The generated Go is committed, so
  consumers never run a code generator.

## Quick start

```go
import (
    "net/http"

    "github.com/larsartmann/cqrs-htmx/adminui/v3"
    "github.com/larsartmann/cqrs-htmx/usermgmt/v3"
)

func main() {
    svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})

    panel, _ := adminui.New(adminui.Config{
        Service:     svc,
        Title:       "Acme Admin",
        AccentColor: "#0ea5e9",
        LogoutURL:   "/logout",
    })

    mux := http.NewServeMux()
    // Sit the panel behind your session middleware so *usermgmt.User is in context.
    mux.Handle("/admin/", usermgmt.NewSessionMiddleware(svc, "session")(panel.Handler()))
    // panel.Mount(mux, "/admin/") is the no-middleware shorthand.
    http.ListenAndServe(":8080", mux)
}
```

Open `/admin/` — done.

## How auth works

The panel is **auth-agnostic**: it reads the authenticated `*usermgmt.User` from
the request context (placed there by [`usermgmt.NewSessionMiddleware`](../usermgmt/middleware.go)
or your own middleware). Requests without a user get `401`; users that fail
[`Config.Authorizer`](config.go) get `403`.

> **Security:** like authentication, **CSRF protection is the consumer's
> responsibility**. The panel issues state-changing `POST`s (delete user,
> create/suspend/delete tenant). Wrap it with [`cqrshtmx.CSRFMiddleware`](../csrf_middleware.go)
> (or your own) in production. The showcase demo omits it for simplicity.

The default authorizer checks roles:

| Mode              | Default check                                            |
| ----------------- | -------------------------------------------------------- |
| `ModeSuperAdmin`  | `super_admin` or `admin` in the global (`*`) domain      |
| `ModeTenantAdmin` | `admin` or `owner` within [`Config.TenantID`](config.go) |

Override with your own `Config.Authorizer`, or use the helpers
[`RequireAnyRole`](authz.go) and [`RequireAuthenticated`](authz.go).

## Configuration

| Field         | Purpose                                        | Default    |
| ------------- | ---------------------------------------------- | ---------- |
| `Service`     | The backing `*usermgmt.Service`. **Required.** | —          |
| `Title`       | Brand text in the sidebar / tab.               | `"Admin"`  |
| `BasePath`    | URL prefix the panel is mounted under.         | `"/admin"` |
| `Mode`        | `ModeSuperAdmin` or `ModeTenantAdmin`.         | SuperAdmin |
| `TenantID`    | Scopes a tenant-admin panel.                   | —          |
| `AccentColor` | Highlight color (any CSS color).               | indigo     |
| `Authorizer`  | Access-control function.                       | role-based |
| `LogoutURL`   | "Sign out" link target. Empty hides the link.  | —          |

## What you get

- **Dashboard** — user/tenant/audit counts + recent activity.
- **Users** — searchable list, per-user detail (credentials, MFA, roles across
  tenants), delete.
- **Tenants** — list, create, suspend, reactivate, delete, and view members.
- **Members** — add a user by email + role, or remove a member, on any tenant
  (super-admin) or your scoped tenant (tenant-admin).
- **Audit log** — the recorded user/tenant events.

## Run the demo

```bash
nix run .#build-admin-demo        # build the showcase binary
# or:
cd examples/admin-demo && go run .
```

Then open <http://localhost:8097/> — it signs you in as the demo admin and shows
the panel with seeded data.

## Project layout

```
adminui/
├── config.go / authz.go    # Config, modes, authorization
├── handler.go              # Handler, Mount(), routing, auth guard
├── render.go               # page/partial render + toasts + redirects
├── assets/                 # embedded CSS + JS; reuses root's htmx.js
├── *.templ / *_templ.go    # templ components (generated files committed)
├── handler_*.go            # per-section HTTP handlers
└── *_test.go               # render + handler tests
```

The panel is a leaf module: it depends on `cqrs-htmx/v3` (root) and
`cqrs-htmx/usermgmt/v3`, and nothing depends on it.
