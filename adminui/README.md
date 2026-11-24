# adminui — Admin Dashboard for cqrs-htmx

A ready-made, good-looking **Admin Dashboard** for apps built on
[`cqrs-htmx/usermgmt`](../usermgmt). Mount it in one call and get a complete
HTMX-driven UI: a dashboard with live metrics, user management, tenant
management, tenant members, and an audit log.

- **One-call mount** behind your existing session middleware.
- **Two scopes**: a global _Super Admin_ panel, or a per-tenant _Tenant Admin_ panel.
- **Modern look** out of the box: a self-contained stylesheet (a committed
  Tailwind v4 build, embedded at compile time) with automatic light/dark theming
  and a configurable accent color. Consumers run no CSS build step.
- **HTMX interactivity** — live search, inline actions, toast notifications.
- **templ-powered** — type-safe HTML components. The generated Go is committed, so
  consumers never run a code generator.

## adminui or dashboardui?

cqrs-htmx ships two dashboards, and the names alone do not say which is which:

- **adminui (this module) manages your users.** Identity operations over a
  `*usermgmt.Service`: accounts, tenants, members, roles, and the audit log,
  with write actions, gated by roles.
- **[dashboardui](../dashboardui/README.md) introspects your event store.**
  CQRS/event-sourcing observability: journal, aggregates, projections, dead
  letters, snapshots, time-travel. Read-only by default.

They are complementary, not alternatives: most apps mount both.
[setup/v4](../setup/README.md) wires admin panel, dashboard, and
[loginpage](../loginpage/README.md) in one call.

## Quick start

```go
import (
    "net/http"

    "github.com/larsartmann/cqrs-htmx/adminui/v4"
    "github.com/larsartmann/cqrs-htmx/usermgmt/v4"
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

> **Security:** the panel ships no CSRF middleware of its own — protection comes from
> the middleware you wrap it with. The [Security](#security) section below has the
> verified CSRF, rate-limit, and session-cookie recipes. The showcase demo omits CSRF
> for simplicity.

The default authorizer checks roles:

| Mode              | Default check                                            |
| ----------------- | -------------------------------------------------------- |
| `ModeSuperAdmin`  | `super_admin` or `admin` in the global (`*`) domain      |
| `ModeTenantAdmin` | `admin` or `owner` within [`Config.TenantID`](config.go) |

Override with your own `Config.Authorizer`, or use the helpers
[`RequireAnyRole`](authz.go) and [`RequireAuthenticated`](authz.go).

## Routes

Every path the panel answers comes from one table in [`routes.go`](routes.go): the mux
registration and the [`Handler.Routes()`](handler.go) manifest are both generated from
those rows, so a route cannot exist outside the table. The source table has **61 rows**;
a running instance serves the subset for its mode — **56 rows in `ModeSuperAdmin`**
(every row below except the five tenant-admin Members rows) and **18 rows in
`ModeTenantAdmin`** (the shared rows, Members, and the fallbacks; the Users and Tenants
panels do not exist in that mode).

Gate legend:

- **public** — answered without a session (static assets, the method-refusal fallback);
  `Routes()` reports these rows with posture `public`.
- **role-gated** — behind the panel's guard: `401` without an authenticated user, `403`
  when [`Config.Authorizer`](config.go) refuses (default checks in
  [How auth works](#how-auth-works)). `Routes()` reports these rows with posture `role`
  under the default authorizer; a custom authorizer can re-declare them `authenticated`
  via `Config.AuthorizerPosture` (the `public`/`ungated` postures are rejected at `New`).
  State-changing rows are marked *(write)*.

| Method | Path                                        | Panel     | Mode         | Gate       | Purpose                                                                                            |
| ------ | ------------------------------------------- | --------- | ------------ | ---------- | -------------------------------------------------------------------------------------------------- |
| GET    | /-/admin-tw.css                             | Assets    | both         | public     | Embedded stylesheet (committed Tailwind v4 build); static, no data                                 |
| GET    | /-/admin.js                                 | Assets    | both         | public     | Panel runtime JS — CSRF header injection, sidebar, toasts, theme toggle; static, no data           |
| GET    | /-/htmx.js                                  | Assets    | both         | public     | Vendored HTMX; static, no data                                                                     |
| GET    | /-/sync-worker.js                           | Assets    | both         | public     | Offline-sync SharedWorker; static, no data                                                         |
| GET    | /-/sync-client.js                           | Assets    | both         | public     | Offline-sync client; static, no data                                                               |
| GET    | /-/ext/cqrs-query.js                        | Assets    | both         | public     | Vendored HTMX QUERY extension; static, no data                                                     |
| GET    | /                                           | Dashboard | both         | role-gated | The dashboard: counts, adoption ratios, recent activity — summarizes every identity in the system  |
| GET    | /partials/stats                             | Dashboard | both         | role-gated | Dashboard stats partial (polled every 30s for the live counters)                                   |
| GET    | /audit                                      | Audit log | both         | role-gated | The audit trail — names every actor and command; searchable and filterable                         |
| GET    | /users                                      | Users     | super-admin  | role-gated | The user directory across every tenant: search, filter chips, access column                        |
| QUERY  | /users/search                               | Users     | super-admin  | role-gated | Body-carrying search (RFC 10008 QUERY): a safe read whose criteria travel as a request body        |
| POST   | /users/search                               | Users     | super-admin  | role-gated | The search fallback twin: byte-identical responses for clients that cannot send QUERY              |
| GET    | /users/new                                  | Users     | super-admin  | role-gated | The create-user form                                                                               |
| POST   | /users                                      | Users     | super-admin  | role-gated | Create a user *(write)*                                                                            |
| GET    | /users/import                               | Users     | super-admin  | role-gated | The bulk-import form                                                                               |
| POST   | /users/import                               | Users     | super-admin  | role-gated | Bulk CSV/JSON import: create accounts from the uploaded file *(write)*                             |
| GET    | /users/export.csv                           | Users     | super-admin  | role-gated | Stream the whole directory as CSV (every user's public profile)                                    |
| GET    | /users/export.json                          | Users     | super-admin  | role-gated | Stream the whole directory as JSON                                                                 |
| GET    | /users/{id}                                 | Users     | super-admin  | role-gated | User detail: profile, credentials metadata, tenant access, per-user timeline                       |
| POST   | /users/{id}/delete                          | Users     | super-admin  | role-gated | Delete a user *(write)*                                                                            |
| POST   | /users/{id}/display-name                    | Users     | super-admin  | role-gated | Edit the display name *(write)*                                                                    |
| POST   | /users/{id}/email                           | Users     | super-admin  | role-gated | Change the sign-in email *(write)*                                                                 |
| POST   | /users/{id}/external/{provider}/unlink      | Users     | super-admin  | role-gated | Unlink an external account *(write)*                                                               |
| POST   | /users/{id}/roles                           | Users     | super-admin  | role-gated | Grant platform roles — they decide who can reach this panel *(write)*                              |
| POST   | /users/{id}/roles/revoke                    | Users     | super-admin  | role-gated | Revoke platform roles *(write)*                                                                    |
| POST   | /users/{id}/credentials/{credential}/delete | Users     | super-admin  | role-gated | Remove a passkey/credential *(write)*                                                              |
| POST   | /users/{id}/credentials/restore             | Users     | super-admin  | role-gated | Restore a passkey from uploaded backup JSON *(write)*                                              |
| POST   | /users/{id}/resend-verification             | Users     | super-admin  | role-gated | Resend the verification email *(write)*                                                            |
| POST   | /users/{id}/tenants                         | Users     | super-admin  | role-gated | Grant the user access to a tenant *(write)*                                                        |
| POST   | /users/{id}/tenants/{tenant}                | Users     | super-admin  | role-gated | Change the user's role in one tenant *(write)*                                                     |
| POST   | /users/{id}/tenants/{tenant}/delete         | Users     | super-admin  | role-gated | Revoke the user's access to one tenant *(write)*                                                   |
| GET    | /tenants                                    | Tenants   | super-admin  | role-gated | The tenant roster — the platform roster, with lifecycle filter chips and member counts             |
| GET    | /tenants/new                                | Tenants   | super-admin  | role-gated | The create-tenant form                                                                             |
| POST   | /tenants                                    | Tenants   | super-admin  | role-gated | Create a tenant *(write)*                                                                          |
| GET    | /tenants/{id}                               | Tenants   | super-admin  | role-gated | Tenant detail: identity, status, member roster                                                     |
| GET    | /tenants/{id}/roster.csv                    | Tenants   | super-admin  | role-gated | Export the tenant's FULL roster as CSV                                                             |
| POST   | /tenants/{id}/transfer                      | Tenants   | super-admin  | role-gated | Transfer ownership: grant owner to one member, demote the prior owner *(write)*                    |
| POST   | /tenants/{id}/suspend                       | Tenants   | super-admin  | role-gated | Suspend a tenant — disables every member of it *(write)*                                           |
| POST   | /tenants/{id}/reactivate                    | Tenants   | super-admin  | role-gated | Reverse a platform suspension *(write)*                                                            |
| POST   | /tenants/{id}/delete                        | Tenants   | super-admin  | role-gated | Delete a tenant — destroys its history *(write)*                                                   |
| POST   | /tenants/{id}/members                       | Tenants   | super-admin  | role-gated | Add a member from the tenant detail view *(write)*                                                 |
| POST   | /tenants/{id}/members/{actor}/delete        | Tenants   | super-admin  | role-gated | Remove a member *(write)*                                                                          |
| POST   | /tenants/{id}/members/{actor}               | Tenants   | super-admin  | role-gated | Change a member's role inside the tenant *(write)*                                                 |
| GET    | /members                                    | Members   | super-admin  | role-gated | Global membership directory — every membership across every tenant                                 |
| GET    | /members                                    | Members   | tenant-admin | role-gated | The member roster of `Config.TenantID` — tenant-internal                                           |
| GET    | /members/roster.csv                         | Members   | tenant-admin | role-gated | Export the scoped tenant's FULL roster as CSV                                                      |
| POST   | /members                                    | Members   | tenant-admin | role-gated | Add a member to the scoped tenant *(write)*                                                        |
| POST   | /members/{actor}/delete                     | Members   | tenant-admin | role-gated | Remove a member *(write)*                                                                          |
| POST   | /members/{actor}                            | Members   | tenant-admin | role-gated | Change a member's role *(write)*                                                                   |
| GET    | /*                                          | Fallback  | both         | role-gated | Unknown pages stay behind the gate: the guarded 404 — the 404 page itself reveals the panel exists |
| *      | /*                                          | Fallback  | both         | public     | Any unmatched method answers 405 regardless of caller — nothing to protect                         |

Notes:

- `QUERY` is RFC 10008's safe, body-carrying method. The same handler also answers a
  wire `POST` carrying `X-HTTP-Method-Override: QUERY` — the rewrite runs inside the
  panel's own stack, i.e. after the consumer's CSRF layer has validated the wire POST,
  and only `QUERY` is an override target, never an unsafe verb.
- `GET /` registers as an exact match (`GET /{$}`) so the dashboard root cannot shadow
  any other route; the two fallbacks register as subtree roots.
- The user-side tenant-access writes (`/users/{id}/tenants…`) and the tenant-side
  membership writes (`/tenants/{id}/members…`, `/members…`) dispatch the same durable
  commands — the admin starts wherever they are.
- Ownership transfer (`POST /tenants/{id}/transfer`) is TWO durable commands under one
  confirm — grant the owner role to the target, then demote the prior owner — sequenced
  by the handler with mid-sequence failure honesty: a failed demotion leaves TWO owners
  (the safe direction), and the toast says exactly what landed, naming both sides and
  the failure reason. Design: docs/planning/2026-11-24_T22-ownership-transfer-design.md.
- A DELETED tenant's detail route renders a read-only tombstone — identity plus
  deleted-at/by resolved from the tenant's journal stream (the audit-log projection
  subscribes only to user/auth events), with an honest "not recorded" when the actor
  chain predates attribution. The live view never renders for a deleted tenant.
- The roster paginates beyond `MaxListRows` like every other list (one page of rows,
  full-roster counts for the captions and confirms), and both roster views export the
  FULL roster as CSV (`/tenants/{id}/roster.csv` super-admin, `/members/roster.csv`
  tenant-admin) — never just the visible page.
- The create-tenant form auto-suggests the slug from the display name as you type
  (editable field; a manual edit is never overwritten), the role checkboxes carry
  per-role descriptions, and the tenant switcher gains an in-dropdown filter once the
  live-tenant list is long enough to need it.

**Freshness:** this table is pinned by [`routes_manifest_test.go`](routes_manifest_test.go) —
`TestRoutes_ManifestGolden` asserts the full super-admin manifest (method, path,
posture, panel) row-for-row, `TestRoutes_TenantModeManifest` pins the tenant-mode shape
(Users/Tenants absent; Dashboard/Members/Audit present), and
`TestRoutes_FallbackSemantics` pins the guarded 404, the methodless 405, the exact-match
root, and the public assets. A new route fails the golden test — update this README
table in the same commit. Check from `adminui/` with `go test -run TestRoutes .`.

## Security

The panel is auth-agnostic and ships no security middleware of its own — the recipes
below are what wraps it. Under the recommended
[setup/v4](../setup/README.md) stack, [`Bundle.Mount`](../setup/mount.go) composes the
panel as `session(csrf(panel))` — session outermost, CSRF inside it, the panel
innermost — and the outer security layer (headers, CSP nonce, panic recovery) comes
from `Bundle.Middleware`. Self-hosting consumers replicate that ordering by hand.

### Read-only mode

Set `ReadOnly: true` on `adminui.Config` to mount the panel without its write
routes. Every route-table row marked `write: true` (creation, deletion, role
grants, tenant lifecycle, merges) is skipped **at registration** — the write
handlers never mount, so a mutating request falls through to the method
catch-all and answers `405`. The refusal is structural: there is no route to
reach and no handler check that can be forgotten. Reads (lists, detail pages,
exports, the audit trail, and the safe `POST /users/search` fallback twin)
are unaffected.

Three surfaces reflect the same truth:

- `Routes()` omits the write rows, so a reverse-proxy allowlist or
  `authztest` matrix built from the manifest never names a route this
  instance refuses.
- The sidebar footer renders a "Read-only" notice chip instead of implying
  the creation/deletion forms are live. (The read-only *forms* themselves
  still render — posting to one is what 405s.)
- Nothing else changes: the default (`ReadOnly: false`) mounts the full
  surface, and existing consumers are unaffected.

A minimal recipe:

```go
panel, err := adminui.New(adminui.Config{
    Service:  svc,
    ReadOnly: true, // browse + audit only; every write answers 405
})
```

For a review-only deployment, pair it with the auth recipes above —
`ReadOnly` removes the write *surface*; it does not change who can *read*.
The demo (`examples/admin-demo`) runs fully interactive and does not set the
knob; to try read-only there, add `ReadOnly: true` to its
`adminui.Config{...}` literals in `main.go` (two: the super-admin panel and
the tenant panel).

Pinned by `readonly_test.go` (per-write-route 405 probes derived from the
full manifest, safe-POST-still-mounted precision, manifest-minus-write-rows
truth, notice chip render + markup golden) and `security_posture_test.go`.

### CSRF

The panel's state changes ride HTMX `POST`s (delete user, create/suspend/delete tenant,
role grants, …), so CSRF protection belongs in the middleware wrapped around it. The
knob is [`setup.Config.CSRF`](../setup/config.go) (`*httputil.CSRFConfig`), passed to
[`httputil.CSRFMiddleware`](https://pkg.go.dev/github.com/larsartmann/httputil#CSRFMiddleware)
verbatim and validated at mount time — every httputil knob is reachable
(`TrustedOrigins`, `TrustedProxies`, cookie/header/field names, `SameSite`, `MaxAge`,
`Domain`, `Path`). The default (`nil`) is the zero config: the CSRF cookie is issued
without the `Secure` flag and setup logs a warning — fine for local HTTP dev only. On
HTTPS set:

```go
setup.Config{CSRF: &httputil.CSRFConfig{Secure: true}}
```

The token dance inside the panel is automatic: page renders read the token from the
request context (`httputil.CSRFTokenHTMLMeta`) and the layout emits
`<meta name="csrf-token">` in the head when a CSRF middleware is active; the embedded
`admin.js` hooks `htmx:configRequest` and attaches that token as `X-CSRF-Token` to
every HTMX request (the double-submit pattern). Without an upstream CSRF middleware
there is no meta tag and the script no-ops.

### Rate limiting

The panel does not rate-limit itself; wrap it with httputil's keyed limiter. (The root
package's `cqrshtmx.RateLimiterMiddleware` and friends are deprecated aliases of these
symbols — import `httputil` directly.) Limiter outermost, so the cheapest check runs
before session lookup and CSRF validation:

```go
import (
    "net/http"
    "time"

    "github.com/larsartmann/httputil"
)

adminLimiter := httputil.KeyedRateLimiterMiddleware(httputil.KeyedRateLimiterConfig{
    Limit:        100,                                  // requests per window, per key
    Window:       time.Minute,
    KeyExtractor: httputil.KeyExtractorFromClientIP(), // X-Forwarded-For → X-Real-IP → RemoteAddr
})

mux.Handle("/admin/", adminLimiter(sessionMW(csrfMW(http.StripPrefix("/admin", panel.Handler())))))
```

Behavior (httputil v1.9.0): over-limit keys get `429` with `Retry-After` (custom
response via `RejectionHandler`; observers via `OnAllowed`/`OnRejected`); `Burst`
defaults to `Limit`; idle keys are evicted after a 10-minute `TTL`; `MaxKeys` caps
tracked keys (zero = unbounded growth). `KeyExtractorFromClientIP` trusts
`X-Forwarded-For`/`X-Real-IP` — a reverse proxy in front must strip or overwrite those
headers, or a spoofed header buys a client fresh budgets; on a directly exposed
listener use `httputil.KeyExtractorFromRemoteAddr()` instead. Note that setup's
per-group rate limiters (`AuthHandlerConfig`) cover the `/auth/*` login ceremonies
only — the panel needs the recipe above.

### Session cookie flags

The panel sets no cookies — the session cookie is written by your login stack (under
`setup`, usermgmt's auth handler). That stack already does the right things (verified
in [`usermgmt/http.go`](../usermgmt/http.go)): `HttpOnly: true` always, `Secure`
defaulting to **true** (`HandlerConfig.Secure`, `nil` → true; downgrade only for local
plain-HTTP dev), `SameSite=Strict`, and `Max-Age` defaulting to 86400 (24h, via
`SessionMaxAge`).

If you roll your own session cookie:

- **`Secure`** — always in production (HTTPS-only transport).
- **`HttpOnly`** — always; nothing in the panel's JS reads the session cookie (the only
  cookie-adjacent script, `admin.js`, reads the CSRF meta tag, not cookies).
- **`SameSite=Strict`** — the safest default for an admin surface; it is also what lets
  setup's un-tokened `/auth/*` mutations (logout, credential deletion) stay
  CSRF-safe. Use `Lax` only if users must land back in the panel from cross-site
  top-level navigations.

The CSRF cookie (`csrf_token`) is a separate cookie with its own flags (`SameSite=Lax`,
`Secure=false` by default) — set `CSRFConfig.Secure` (and `SameSite` if needed) on
HTTPS deployments.

## Server-Timing

The panel emits no `Server-Timing` header of its own. What exists today:

- **Under `setup` + `RunWithAppkit`:** go-appkit (>= v0.10.0) mounts the W3C
  Server-Timing middleware outermost in its default stack, so every response — panel
  pages, HTMX partials, assets included — carries `Server-Timing: total;dur=<ms>`,
  stamped at response commit and SSE-safe. The knob is
  [`setup.Config.AppkitServerTiming`](../setup/config.go) (`*bool`, tri-state):
  unset/`nil` = header on (appkit default), `&false` = the middleware is omitted
  entirely, `&true` = same as unset. `RunWithAppkit`-only — `Bundle.Run` and
  `Bundle.RunHandler` serve via httputil and never emit the header.
- **Self-hosted:** wrap the panel with
  [`httputil/server_timing`](https://pkg.go.dev/github.com/larsartmann/httputil/server_timing)'s
  `ServerTimingMiddleware()` (or `WrapServerTiming`), and add finer-grained spans in
  your own code with `MeasureServerTiming`.
- The root package's `cqrshtmx.Config.ServerTiming` predicate does **not** affect the
  panel: it gates Server-Timing on `App.Command()`/`App.Query()` handlers you wire
  yourself, and the panel is plain `net/http`.

Reading it: `Server-Timing: total;dur=12` names one metric, `total`, whose `dur` is the
full request duration in **milliseconds**. Browser DevTools shows it under Network →
Timing; `curl -i` shows it on the wire. appkit's default gives you the one `total`
metric — anything finer is spans you add yourself.

### Render performance

The directory pages are the panel's hot path, and they are measured: `BenchmarkUsersPageRender`
(`users_render_bench_test.go`) renders a full 200-row users page against a real in-memory
service and reports ns/op, allocations, and a req/s metric (`b.Loop`, per-request logging
suppressed for the run). Each render builds its authorization facts in ONE
`GroupPolicies()` pass plus one `AllTenants()` pass (hierarchy rows filtered the same way
`usermgmt.HasAdminGrant` filters them, so a seeded role hierarchy cannot manufacture
admins), and the last-seen column's audit-window scan is memoized for 30 seconds — a
burst of renders pays one scan, not one per render. The raw baseline capture lives in
[`docs/benchmarks/adminui-baseline.raw.txt`](../docs/benchmarks/adminui-baseline.raw.txt)
(advisory: captured on a shared machine). Before and after any render-path change, re-run:

```sh
GOEXPERIMENT=jsonv2 go test ./adminui/ -run XXX -bench BenchmarkUsersPageRender -benchtime=2s
```

## Configuration

| Field          | Purpose                                                                                                                                                                                                 | Default    |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| `Service`      | The backing `*usermgmt.Service`. **Required.**                                                                                                                                                          | —          |
| `Title`        | Brand text in the sidebar / tab.                                                                                                                                                                        | `"Admin"`  |
| `BasePath`     | URL prefix the panel is mounted under.                                                                                                                                                                  | `"/admin"` |
| `Mode`         | `ModeSuperAdmin` or `ModeTenantAdmin`.                                                                                                                                                                  | SuperAdmin |
| `TenantID`     | Scopes a tenant-admin panel.                                                                                                                                                                            | —          |
| `AccentColor`  | Highlight color (CSS literal: #hex, rgb()/hsl(), or named; validated at `New`).                                                                                                                         | indigo     |
| `Authorizer`   | Access-control function.                                                                                                                                                                                | role-based |
| `LogoutURL`    | "Sign out" link target. Empty hides the link.                                                                                                                                                           | —          |
| `SSEURL`       | SSE endpoint for the live sync indicator.                                                                                                                                                               | disabled   |
| `SSEHubStats`  | Source for two dashboard cards: live SSE subscribers + lifetime slow-consumer drops (amber when nonzero). Nil adds nothing; `setup` wires it automatically when its shared SSE/DataStar hub is enabled. | —          |
| `ExtraNav`     | Consumer-owned sidebar entries appended after the built-ins. Hrefs render                                                                                                                               | —          |
|                | VERBATIM (no `BasePath` prefix) — point at sibling surfaces (`/dashboard`,                                                                                                                              |            |
|                | `https://docs.example.com`). Empty `Label` or `Href` is rejected at `New`.                                                                                                                              |            |
| `DashboardURL` | Cross-links the sibling [dashboardui](../dashboardui) panel as an "Event                                                                                                                                | hidden     |
|                | store" sidebar entry. Pass its mount path (usually `"/dashboard/"`). For more                                                                                                                           |            |
|                | than one link, prefer `ExtraNav`.                                                                                                                                                                       |            |

## What you get

- **Dashboard** — user/tenant/audit counts, adoption ratios (verified email,
  MFA), a no-admin-grant warning banner, + recent activity.
- **Users** — searchable, filterable list (verified / MFA / external provider /
  administrators chips) with an access column: each row names the platform
  roles the user holds and how many tenants they belong to, so privileged
  accounts are visible without opening every profile. The detail page covers
  the full lifecycle: create, edit display name and email, grant/revoke
  platform roles (durable, event-sourced), manage passkeys and external
  accounts, resend verification, **manage tenant access from the user side**
  (grant a tenant + role, change a role, revoke), delete. Each write escalates
  its confirmation dialog to match its blast radius. The external-accounts
  card identifies each link fully (provider glyph, issuer subject id with the
  full value in its tooltip, locale-shaped linked-at timestamp), the Details
  card answers the verification-token lifetime honestly — it is configured by
  the embedding application and the panel cannot read the configured value —
  and a lockout badge fires when a user holds no sign-in channel at all. On
  your OWN detail page a self-service card renders: edit your display name,
  read back your grants, and follow the **Sign out** link — that link points
  at the endpoint YOU mount via `Config.LogoutURL`; unset, the card says so
  instead of linking to nothing (the panel owns no session routes; there is
  deliberately no credential-restore-from-backup route — a future route task,
  though the published `AddCredential` API makes it straightforward).
- **Tenants** — searchable roster with per-tenant member counts, lifecycle
  filter chips (active / suspended / deleted, with live counts) and a
  Name/Members ordering toggle; create, suspend, reactivate, delete — the
  destructive confirms name the blast radius (how many memberships are
  revoked or cut off). The detail page leads with the tenant's identity
  (id + copy, slug, display name, status, roster size) before its members,
  and super-admin mode carries a **header tenant switcher**: a dropdown of
  live tenants that jumps between tenant dashboards and marks the one being
  viewed (tenant-admin mode renders no switcher — that mode is scoped to one
  tenant by construction).
- **Membership directory** (super-admin only) — one page answering "who has
  access to what" across the whole installation: every membership resolved to
  its tenant and its person (email + display name, linked to the user page),
  with role chips, a joined column, a search over email/name, tenant and role
  filter chips with live counts, a Tenant/Joined ordering toggle, and the
  shared page-size-bounded pager. The same `/members` path serves the
  tenant-admin roster in the other mode — the route table scopes the two
  handlers by mode, so the two pages never mount in the same panel. When no
  memberships exist at all the page says so explicitly instead of rendering
  a blank table.
- **Members** — add a user by email + role, or remove a member, on any tenant
  (super-admin) or your scoped tenant (tenant-admin). Rows resolve the member's
  actor to a person (email + display name, linked to their user page in
  super-admin mode) and show when they joined; the add form autocompletes from
  the directory, offers only accounts that are not members yet, requires an
  explicit role choice, and explains an unknown email inline (preserving what
  was typed) instead of dropping it in a toast.
- **Bulk user import/export** — `GET /users/import` renders the form,
  `POST /users/import` accepts a CSV/JSON upload (size-capped, dispatched to
  the published Service importer, exact imported/skipped counts with a
  bounded per-row error list), and `GET /users/export.csv` /
  `GET /users/export.json` stream the whole directory straight from the
  Service. Super-admin mode only.
- **Audit log** — searchable/filterable event trail with human-readable action
  labels and severity badges; the user detail page carries the per-user
  timeline.
- **Theme toggle** — user-controllable dark mode (v4.12.0): a header button
  persists the choice to `localStorage` and a pre-paint script applies it
  (falling back to the OS preference), so there is no flash of the wrong
  theme. No-JS degrades to light. Rides the templ-components library's
  `layout.ThemeToggle` + `layout.ThemeScript`.

## Run the demo

```bash
nix run .#build-admin-demo        # build the showcase binary
# or:
cd examples/admin-demo && go run .
```

Then open <http://localhost:8097/> — it signs you in as the demo admin and shows
the panel with seeded data.

## i18n: labels, locale, direction

adminui ships a **minimal i18n seam** — deliberately not a translation engine (no locale files, no pluralization, no message-format machinery). Three config fields cover what most deployments actually need:

- **`Config.Labels map[string]string`** overrides display strings by key from the panel's label table (`labels.go`: nav, chrome, confirm dialogs, empty states, page headers, table headers, primary actions — every covered string lives in ONE table that all surfaces read). An override **replaces** the table value for a known key; an **unknown key is rejected at `New`** — an accepted-but-inert override would hide a typo forever. Unknown keys in the table itself render the KEY text, never blank.
- **`Config.Locale string`** (BCP-47 tag, e.g. `"de"`, `"pt-BR"`) sets the `<html lang>` attribute and picks the date shapes. The shape table is tiny by design: the default keeps the historical `en` shapes, `de` ships its dotted day-first shape, and everything else falls back to the unambiguous ISO shape rather than a wrong guess.
- **`Config.Dir string`** (`"ltr"` default, `"rtl"`) sets the `<html dir>` attribute. Note: this is a starting point for right-to-left deployments, not an automatic mirror of the layout.

```go
panel, err := adminui.New(adminui.Config{
    Service: svc,
    Locale:  "de",
    Dir:     "ltr",
    Labels: map[string]string{
        "nav.users":       "Benutzer",
        "empty.users.none.title": "Noch keine Benutzer",
    },
})
```

Scope honesty: if you need full localization (plural rules, per-locale formatting, translated audit-action vocabulary), you need a real i18n framework — this seam is the rename/reword/lang-dir escape hatch, and it stays that size on purpose.

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

The panel is a leaf module: it depends on `cqrs-htmx/v4` (root) and
`cqrs-htmx/usermgmt/v4`, and nothing depends on it.

## Duplicate reconciliation (2026-11-24, cqrs-htmx #46/#47)

The panel ships two report pages and their write surfaces for consolidating
duplicate identities:

- `GET /users/duplicates` — `Service.DuplicateUsers` clusters (shared email,
  shared external account); each row posts `POST /users/{id}/merge` with a
  survivor picker (the path user merges INTO the form survivor).
- `GET /tenants/duplicates` — `Service.DuplicateTenants` clusters (normalized
  name); each row carries a rename form (`POST /tenants/{id}/rename`,
  `display_name`) and a merge form (`POST /tenants/{id}/merge`, survivor).

Every route is super-admin, role-gated, and pinned in the route manifest.
Merging is terminal: the confirm burden sits with the report's clarity
(survivor options exclude the row itself; a row can never merge into
itself).
