# The authorization system

> How to gate a route, where roles come from, and how to prove it works.
> Companion to [`docs/PERMISSIONS.md`](../PERMISSIONS.md) (the maturity system
> and the per-service inventory) and
> [ADR-0057](../adr/0057-authorization-contract.md) (the decision record).

## The four moves

1. **Grant** a role — durable, event-sourced.
2. **Declare** the route's posture — who may call it.
3. **Decide** through the contract — one question type, four answer kinds.
4. **Prove** it — a test that fails when the gate is removed.

Most services skip straight to step 3 with a boolean and discover months later
that the grant was never durable. Do them in order.

## 1. Grant

```go
actor  := usermgmt.ActorIDFromUser(userID)
domain := identitymodel.NewTenantID(identitymodel.WildcardDomain)

if err := svc.GrantRoles(ctx, actor, domain,
    []identitymodel.Role{identitymodel.RoleSuperAdmin}); err != nil {
    return err
}
```

`GrantRoles` dispatches the membership commands, so the grant is an event: it is
in the journal, it survives a restart, a projection rebuild, and a second
instance, and the Casbin projection reconstructs it without any consumer code.

It **converges**: a second call replaces the role set instead of failing with a
conflict, so a bootstrap grant can run on every start.

For a tenant-scoped grant, pass the tenant ID instead of the wildcard:

```go
if err := svc.GrantRoles(ctx, actor, tenantID,
    []identitymodel.Role{identitymodel.RoleAdmin}); err != nil {
    return err
}
```

Revoke with `svc.RevokeRoles(ctx, actor, domain)`.

### What not to do

```go
// Wrong: writes into the in-memory enforcer, not the journal.
_ = svc.Authz().AddGroupPolicy(identitymodel.GroupPolicy{
    Subject: userID.String(), Role: identitymodel.RoleAdmin, Domain: "*",
})
```

That grant is gone the moment the process stops. It is deprecated in v4 and
removed in v5, and `check-permission-surface` fails the build on it.

### Bootstrap admin on a fresh install

Grant on every start, unconditionally. The call converges, so the "only if there
are no users yet" guard is unnecessary — and it is what made one audited service
lose its admin on the second boot.

```go
func ensureAdmin(ctx context.Context, svc *usermgmt.Service, userID identitymodel.UserID) error {
    return svc.GrantRoles(ctx,
        usermgmt.ActorIDFromUser(userID),
        identitymodel.NewTenantID(identitymodel.WildcardDomain),
        []identitymodel.Role{identitymodel.RoleSuperAdmin})
}
```

`setup.New` logs a boot warning when the admin panel is mounted but nobody holds
`admin` or `super_admin` in the wildcard domain — that warning is the signal that
this step is missing.

## 2. Declare the posture

Write down what each route requires. The maturity system (level L4) asks for this
list in the repository, and the level does not count as reached without it.

| Route                   | Posture                               |
| ----------------------- | ------------------------------------- |
| `GET /health`           | public                                |
| `GET /login`            | public                                |
| `POST /auth/login`      | public                                |
| `GET /admin/*`          | role: `super_admin` or `admin` in `*` |
| `POST /api/orders/{id}` | role: `user` in the order's tenant    |

A posture is a decision, not a description: "public" on a route that mutates data
is a finding, not a default.

## 3. Decide

### On a CQRS handler

```go
app.Command("DeleteOrder",
    cqrshtmx.DecodeJSONWithRequest(decodeDeleteOrder),
    cqrshtmx.AuthorizeRequest(func(r *http.Request) (cqrshtmx.Request, error) {
        return cqrshtmx.Request{
            Domain:   chi.URLParam(r, "tenant"),
            Resource: cqrshtmx.Resource{Type: "orders", ID: chi.URLParam(r, "id")},
            Action:   "delete",
        }, nil
    }),
)
```

Leave `Subject` empty to use the authenticated user; set it to authorize a bot or
service identity. A static question is shorter:

```go
app.Query("ListTenants", cqrshtmx.Authorize("tenants", "read"))
```

`Authorize` asks in the **wildcard** domain — only global policies answer it. Use
`AuthorizeIn(domain, resource, action)` or `AuthorizeRequest` when the answer
depends on tenant membership.

### Anywhere else

```go
decision, err := cqrshtmx.Decide(svc.Authz(), cqrshtmx.Request{
    Subject:  userID.String(),
    Domain:   tenantID,
    Resource: cqrshtmx.Resource{Type: "invoices", ID: invoiceID},
    Action:   "read",
})
if err != nil {
    return err // engine failure: 503, never a silent denial
}

if !decision.Allowed {
    switch decision.Reason {
    case cqrshtmx.ReasonNotAuthenticated:
        // 401 — no subject to ask about.
    case cqrshtmx.ReasonNoMatchingPolicy:
        // 403 — nobody granted this. Usually a missing groove, not a refusal.
    default:
        // 403 — a rule said no.
    }
}
```

`*identitymodel.Authz` satisfies `cqrshtmx.PermissionChecker` structurally, so
passing it to `Decide` needs no adapter and no import of the root package in your
domain layer.

### Recognising a denial you did not create

```go
if cqrshtmx.IsForbidden(err) {
    // any module's forbidden error, at any depth of the wrap chain
}
```

Use this instead of comparing against one sentinel: the same denial exists as
several values across modules, and `errors.Is` only ever matches one.

## 4. Prove

Four tests, every gated route:

| Case                                                    | Expect                              |
| ------------------------------------------------------- | ----------------------------------- |
| no credentials                                          | 401                                 |
| authenticated, without the role                         | 403                                 |
| authenticated, with the role                            | 200 (or whatever the route returns) |
| with the role, in a tenant the actor is not a member of | 403                                 |

The fourth is the one that gets skipped and the one that finds real bugs: a
wildcard check passes it, a domain check does not.

A test that still passes when you delete the `Authorize` option proves nothing
about authorization. Make the removal break the suite before you merge.

### The kit that writes those four tests for you

`authztest.Suite` proves the posture table above; `authztest.GrantSuite` proves
the grant behind it (tenant scoping, revoke, durability across a restart). Both
are fail-closed — a declaration that cannot be tested is an error, not a skip.
[`authztest/README.md`](../../authztest/README.md) is the full surface; the short
version:

```go
authztest.Suite{Handler: mux, Routes: routes, Anonymous: anon,
    Unprivileged: asUser("member"), Privileged: asUser("admin")}.Run(t)

grants := authztest.GrantSuite{Checker: svc.Authz(), Governor: gov,
    Subject: "user-1", Tenant: "acme", ForeignTenant: "globex",
    Roles: []string{"staff"}, ResourceType: "orders", Action: "read",
    Fresh: reopen}
grants.ProveTenantIsolation(t)
grants.ProveRevocation(t)
grants.ProveDurability(t)
```

`GrantSuite.ProveDurability` is the mechanized form of the defect that started
this work: a grant written into an in-memory enforcer reads back as allowed
in-process and as denied after a restart. It builds the second checker from
durable state **after** granting, so it cannot pass on a cached engine.

## Streaming and CSP notes

Two consumer workarounds were symptoms of library gaps that are now closed:

- **SSE and server write timeouts.** `transport.ServeDomainEvents` and
  `Broadcaster.ServeSSE` clear the connection's write deadline, so a route-wide
  `http.Server.WriteTimeout` no longer kills a live stream. Custom SSE handlers
  call `transport.ClearWriteDeadline(w)` first. Do not set the server timeout to
  zero globally — that also removes the deadline from ordinary routes.
- **Panel CSP nonce.** `setup.Config.NonceFunc` (and `adminui`/`dashboardui`
  `Config.NonceFunc`) resolves the nonce for the panels' inline scripts. Set it
  when your nonce does not come from `httputil.Nonce`; return exactly what your
  CSP header advertises.

## Moving up a level

See [`docs/PERMISSIONS.md`](../PERMISSIONS.md) for the L0–L4 exit criteria and the
per-service inventory. When a service moves up, update its row in that file —
`check-permission-maturity` fails a dropped level, so the inventory cannot
silently rot.
