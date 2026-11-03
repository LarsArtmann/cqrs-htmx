# Permissions: the maturity system

> One page every service owner reads before touching authorization.
> Inventory snapshot: **2026-11-02** (read-only audit of 17 repositories).
> Machine-checked baseline: [`PERMISSIONS_BASELINE`](PERMISSIONS_BASELINE) —
> checked by `nix run .#check-permission-maturity`.

## Why this exists

The 2026-11-02 consumer audit asked 17 downstream services how they check
permissions. The answer was: they do not.

- **0 of 12** services in the dependency graph call `Enforce`, `EnforceEx`,
  `EnforceAny`, `Service.Authorize`, or the root HTTP authorization options.
- **0** write an object/action policy. Real usage is role membership only.
- Two write role grants imperatively into an in-memory enforcer, so the grant is
  gone after every restart; one reconciles it after every drain; one mounts an
  admin panel nobody can reach.
- Three different forbidden errors existed, with three different codes, so no
  shared classifier could recognise a denial.

That is not a missing feature. It is a missing **system**: no write path, no
contract, no proof, no inventory. This document is the inventory; the other
three are in the library.

## The five levels

Each level has exit criteria that can be checked by someone who did not write
the service.

### L0 — undeclared

Permission-relevant routes exist with no gate and no declaration of intent.
Either everything is public on purpose (and nobody wrote that down) or nobody
knows.

**Exit:** enumerate the routes. Every one gets a posture, even if the posture is
"public".

### L1 — authenticated

Every non-public route requires a session (or a documented alternative such as
basic auth or loopback-only binding). The route set is written down.

**Exit:** no route accepts an anonymous request unless it is declared public.

### L2 — durable roles

Roles are granted through the event-sourced path
(`usermgmt.Service.GrantRoles`, which dispatches `AddMember` /
`UpdateMemberRoles`). The grant is in the journal, so it survives a restart, a
projection rebuild, and a second instance. Role values are validated before
dispatch. No grant is written straight into the enforcer.

**Exit:** restart the process and the role is still there. If a service needs a
bootstrap admin, it grants one on every start (idempotently), not once behind a
"users already exist" guard.

### L3 — enforced and proven

Object/action decisions go through the contract
(`cqrshtmx.Decide` with a `PermissionChecker`, or `AuthorizeIn` /
`AuthorizeRequest` on a handler) rather than a bare boolean check. A test suite
proves all four answers: 401 without credentials, 403 without the role, 200 with
it, and 403 in a tenant the actor does not belong to.

**Exit:** deleting the authorization call makes a test fail.

### L4 — ratcheted

The service commits a route-posture manifest, and a gate fails when a route
appears without a declared posture or when a declared level drops. Posture is a
build artifact, not tribal knowledge.

**Exit:** adding an unprotected route fails CI before review.

## The three write paths, in order of preference

| Path                                            | Durable                              | Use it                                                      |
| ----------------------------------------------- | ------------------------------------ | ----------------------------------------------------------- |
| `Service.GrantRoles(ctx, actor, tenant, roles)` | Yes — dispatches membership commands | Always, for grants                                          |
| `Service.AddMember` / `UpdateMemberRoles`       | Yes — same commands, one level lower | When you already know the membership is new                 |
| `Authz.AddGroupPolicy`                          | **No** — in-memory enforcer only     | Never in application code (deprecated in v4, removed in v5) |

`Authz.ApplyGroupPolicy` is the projection's write path: it carries no warning
because its caller derives the grant from the journal.

## The contract

```go
decision, err := cqrshtmx.Decide(checker, cqrshtmx.Request{
    Subject:  userID.String(),
    Domain:   tenantID,             // "" or "*" = installation-wide
    Resource: cqrshtmx.Resource{Type: "orders", ID: orderID},
    Action:   "read",
})
```

`decision.Reason` distinguishes the answers a bare boolean cannot: `allowed`,
`denied_by_policy` (a rule said no), `no_matching_policy` (nobody spoke — for a
default-deny engine this means the grant is missing), `not_authenticated`.
Engine failures come back as errors, never as silent denials.

`PermissionChecker` is primitive-shaped on purpose: `identitymodel.Authz`
satisfies it structurally, so the root package never imports Casbin and the
domain model never imports HTTP.

## Inventory

Levels are from the 2026-11-02 audit. `?` means "not inspected beyond dependency
presence" — an unverified row, not a clean bill.

| Service                  | Level | Posture at audit                                                                           | Evidence                                                                                                                                                                                             | Next step                                                |
| ------------------------ | ----- | ------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| `cqrs-htmx`              | L4    | this library: contract + durable grants + scanner gate + committed route-posture manifests | `authz_contract.go`, `service_grants.go`, `check-permission-surface.sh`, `check-route-postures.sh` (in check-modules + CI), `Handler.Routes` / `Dashboard.Routes` / `Bundle.RoutePostures` manifests | keep every new route inside the tables                   |
| `browser-history`        | L1    | admin panel gated by `adminui.RequireAnyRole`, role re-granted on every drain              | `api/admin_panel.go` reconcile loop; `NonceFunc` bridge                                                                                                                                              | L2: switch to `GrantRoles`, delete the reconcile loop    |
| `games/SwettySwipperWeb` | L1    | `adminui.RequireAnyRole` on API routes                                                     | one-shot `AddGroupPolicy` behind a "users exist" guard                                                                                                                                               | L2: grant on every boot, idempotently                    |
| `crm`                    | L0    | `setup.New` with the default admin panel                                                   | zero role grants anywhere; `MaxUsers: 1`                                                                                                                                                             | L1: either seed an admin or `DisableAdmin`               |
| `sperrmuell-direct`      | L1    | basic auth; all panels disabled                                                            | `internal/identity/mount.go` disables admin/dashboard/login                                                                                                                                          | L2 if roles are wanted; otherwise record the posture     |
| `webphone`               | L1    | 44 `RequireSession` gates, own session service                                             | no role model                                                                                                                                                                                        | L2 if roles are wanted; otherwise record the posture     |
| `github-local-sync`      | L1    | admin panel via default authorizer                                                         | no role grant                                                                                                                                                                                        | L2: grant a global admin at boot                         |
| `InboxClean`             | L1    | session-gated CQRS dashboard                                                               | `sseDeadlineFree` wrapper around the SSE route                                                                                                                                                       | delete the wrapper (the library now clears the deadline) |
| `bank-sync`              | L1    | session-gated dashboard                                                                    | no role model                                                                                                                                                                                        | record the posture                                       |
| `crush-daily`            | L1    | session-gated dashboard + SSE                                                              | no role model                                                                                                                                                                                        | record the posture                                       |
| `file-and-image-renamer` | L1    | dashboard with a custom authorizer                                                         | `pkg/injector/dashboardui_providers.go`                                                                                                                                                              | record the posture                                       |
| `journal`                | L1    | loopback-only store panel                                                                  | `internal/server/server.go`                                                                                                                                                                          | record the posture                                       |
| `dnsblockd`              | L1    | adapter only, no panel                                                                     | —                                                                                                                                                                                                    | record the posture                                       |
| `quotaboard`             | ?     | not inspected                                                                              | —                                                                                                                                                                                                    | inspect                                                  |
| `darknet-monitor`        | ?     | not inspected                                                                              | —                                                                                                                                                                                                    | inspect                                                  |
| `KeyHolderAI`            | ?     | not inspected                                                                              | —                                                                                                                                                                                                    | inspect                                                  |
| `Code-To-CV-Agent`       | ?     | not inspected                                                                              | —                                                                                                                                                                                                    | inspect                                                  |
| `go-appkit/integration`  | ?     | not inspected                                                                              | —                                                                                                                                                                                                    | inspect                                                  |

## Proving it: `authztest`

`github.com/larsartmann/cqrs-htmx/v4/authztest` is the matrix you import into
your own `_test.go`. Declare the surface, supply three actors, and it proves
every posture:

```go
authztest.Suite{
    Handler: mux, // fully wired, middleware included
    Routes: []authztest.Route{
        {Method: "GET", Path: "/health", Posture: authztest.Public, Why: "liveness probe"},
        {Method: "GET", Path: "/orders", Posture: authztest.Authenticated, Why: "any signed-in user"},
        {Method: "GET", Path: "/admin/users", Posture: authztest.RoleGated, Why: "admin only"},
    },
    Anonymous:    withoutCredentials,
    Unprivileged: asUser("member"),
    Privileged:   asUser("admin"),
}.Run(t)
```

It is **fail-closed by construction**: a role-gated declaration with no
unprivileged actor, or a gated route with no anonymous actor, is a verification
error rather than a skipped case. A skipped case reports green, which is the
failure mode this exists to prevent.

`authztest.Undeclared(declared, registered)` closes the other direction: a route
that nobody declared is invisible to the matrix, and invisible is how a gate goes
missing.

### Proving the grants behind it: `authztest.GrantSuite`

The posture matrix answers "does this route refuse the wrong caller?". It cannot
see the defect that costs the most: the grant itself. `GrantSuite` proves the
three properties that matter, each case self-contained:

```go
suite := authztest.GrantSuite{
    Checker:       svc.Authz(),          // cqrshtmx.PermissionChecker
    Governor:      governor{svc: svc},   // GrantRoles + RevokeRoles, string-shaped
    Subject:       "user-1",
    Tenant:        "acme",
    ForeignTenant: "globex",             // required: isolation needs a comparison
    Roles:         []string{"staff"},
    ResourceType:  "orders",
    Action:        "read",
    Fresh:         func(t *testing.T) cqrshtmx.PermissionChecker { return reopen(t).Authz() },
}

suite.ProveTenantIsolation(t) // allowed in acme, denied in globex
suite.ProveRevocation(t)      // grant → allowed → revoke → denied
suite.ProveDurability(t)      // grant → fresh process → still allowed
```

`ProveRevocation` asserts the grant authorized _before_ revoking, so a deny
afterwards cannot be an accident of a grant that never worked. `ProveDurability`
builds the second checker **after** the grant, from durable state alone — that is
the 2026-11-02 defect mechanized (a grant written into an in-memory enforcer
vanishes on restart). Run either suite after the projections have drained: under
`AsyncStartup` poll readiness first, or a green result means "replay had not
arrived yet".

## Onboarding a service (5 steps)

1. **Enumerate.** List every route with method, path, and who may call it. Public
   routes get `public`; everything else gets `authenticated` or `role`.
2. **Grant durably.** Replace every `AddGroupPolicy` with
   `Service.GrantRoles`. For a bootstrap admin, grant on every start — the call
   converges, so a restart is a no-op.
3. **Decide through the contract.** Replace boolean checks with
   `cqrshtmx.Decide` / `AuthorizeIn` / `AuthorizeRequest`. Add `Domain` — a
   wildcard check answers only global policies.
4. **Prove it.** Add 401 / 403 / 200 tests plus a tenant-isolation case. A test
   that passes when the authorization call is deleted proves nothing.
   `authztest.Suite` + `authztest.GrantSuite` are that test — see
   [`authztest/README.md`](../authztest/README.md) for the full surface.
5. **Ratchet it.** Commit the posture list and add the service's row here; the
   maturity gate fails on a dropped level.

## The gate

```bash
nix run .#check-permission-maturity              # advisory report
nix run .#check-permission-maturity -- --ratchet # fail on a dropped level
nix run .#check-permission-surface               # anti-pattern scanner
```

The scanner fails on four defects: ephemeral grants, untyped engine calls, raw
engine calls, and re-coded denial sentinels. Suppressions are inline
(`permission-surface-allow` with a reason) and are counted in the output, so
they stay visible in review.

## Non-goals

- **No network policy server.** Authorization is in-process. A remote PDP adds a
  failure mode (the PDP is down, so everything is denied or everything is
  allowed) with no compensating benefit at this scale.
- **No relationship graph yet.** Demand is role-in-domain. `Resource{Type, ID}`
  reserves the shape a ReBAC engine needs; the trigger for building one is
  recorded in ADR-0058.
- **Not a role-vocabulary redesign.** `super_admin > admin > owner > user >
viewer` is what the shipped model and every consumer already use.
