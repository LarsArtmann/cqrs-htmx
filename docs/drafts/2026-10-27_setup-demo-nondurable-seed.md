## Problem

`examples/setup-demo` seeds the super admin with an out-of-band casbin write:

- `examples/setup-demo/main.go:182` — `bundle.Service.Authz().AddGroupPolicy(...)`
- `examples/admin-demo/main.go:207` — same pattern

`Authz().AddGroupPolicy` writes the policy straight into the enforcer's
in-memory casbin state. It is not journalled. The `CasbinProjection`
"derives Casbin policies from User events" (`usermgmt/es_casbin_projection.go:11`)
— membership grants reach casbin only through journalled
`MemberAdded`/`MemberRolesChanged`/`MemberRemoved` events
(`es_casbin_projection.go:108`). So in any persistent setup, the demo's
seed grant evaporates on the first projection rebuild (fresh boot with a
dropped projection table, replay, snapshot restore): the admin is 401
again, with no error anywhere.

Reproduced 2026-10-26 against this tree: seeded via `AddGroupPolicy`,
the grant survives a restart only because the in-memory enforcer stays
warm; re-deriving the projection from the journal loses it. Seeding via
the event-sourced path survives the same restart.

## The durable path already exists

`Service.AddMember` (`usermgmt/service_membership.go:10`) dispatches an
`AddMember` command; the journal entry is what the projection replays:

```go
actorID := identitymodel.ActorIDFromUser(ownerID)
_ = bundle.Service.AddMember(ctx, actorID,
    identitymodel.NewTenantID("*"),
    []identitymodel.Role{identitymodel.RoleSuperAdmin})
```

## Ask

Change both examples to seed via `Service.AddMember` (or keep the
`AddGroupPolicy` line with a loud comment that it is ephemeral and dies
with the next replay). The example is the first thing integrators copy —
mine taught me the wrong path before I diffed the projection.

> [!NOTE]
> Drafted with Crush (prose + verification assist); the repro and
> diagnosis are verified against `2dcaa884`.

💘 Generated with Crush

Assisted-by: Crush:glm-5.3-flash
