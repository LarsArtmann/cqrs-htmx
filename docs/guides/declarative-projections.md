# Declarative Projections: from ProjectionLayer to system.New()

> Audience: consumers of `systemadapter/v4` who build read models over the
> identity domain. `ProjectionLayer` is **deprecated** and retires in the v5
> removal bundle (`docs/guides/v5-removal-inventory.md`). This guide is the
> migration path.

## The two paths

### Deprecated: `NewProjectionLayer(sys)` (separate host)

```go
sys, _ := system.New(ctx, systemadapter.DomainConfig(), deployment)

projLayer, _ := systemadapter.NewProjectionLayer(sys) // NOT started yet
defer projLayer.Stop()

projLayer.Start(ctx) // must be before sys.Start, if you start the system at all
```

You own a second `projectionhost.Host`: manual `Start`/`Stop`, a separate
drain primitive (`WaitForDrain`), and in-object read models
(`projLayer.User.FindByID(...)`, `projLayer.AuditLog.Entries()`, ...).

### Current: declarative projections (one host)

```go
sys, _ := system.New(ctx, systemadapter.DomainConfig(), deployment)
defer sys.Close()

sys.Start(ctx) // starts the INTERNAL declarative projection host — nothing else to wire
```

`DomainConfig()` already registers every projection (user/tenant/membership/bot
read models, authz policies, audit log) as metaengine fold declarations.
Reads go through typed query functions on the package:

```go
user, err := systemadapter.FindUserByID(ctx, sys, userID.String())
tenants, err := systemadapter.AllTenants(ctx, sys)
allowed, err := systemadapter.Enforce(ctx, sys, subject, domain, "manage")
entries, err := systemadapter.AuditEntriesFor(ctx, sys, aggregateID)
```

## API mapping

| ProjectionLayer                              | Declarative replacement                                                        |
| -------------------------------------------- | ------------------------------------------------------------------------------ |
| `NewProjectionLayer(sys)` + `Start`/`Stop`   | `sys.Start(ctx)` / `sys.Close()` (host is owned by the system)                 |
| `pl.WaitForDrain(5 * time.Second)`           | poll the query until it succeeds (see below)                                   |
| `pl.User.FindByID(id)`                       | `systemadapter.FindUserByID(ctx, sys, id.String())` → `(UserView, error)`      |
| `pl.User.FindByEmail(email)`                 | `systemadapter.FindUserByEmail(ctx, sys, email)`                               |
| `pl.Tenant.FindByID(id)`                     | `systemadapter.FindTenantByID(ctx, sys, id.String())`                          |
| `pl.Membership.FindByAggregateID(id)`        | `systemadapter.FindMembershipByID(ctx, sys, id.String())`                      |
| `pl.Membership.FindByTenant(tenantID)`       | `systemadapter.FindMembershipsByTenant(ctx, sys, tenantID.String())`           |
| `pl.Bot.*`                                   | `systemadapter.FindBotByID` / `FindBotsByOwner` / `FindBotByTokenHash`         |
| Casbin enforcer from the layer               | `systemadapter.Enforce(ctx, sys, subject, domain, action)` / `FindPolicies`    |
| `pl.AuditLog.Entries()`                      | `systemadapter.AuditEntries(ctx, sys)` / `AuditEntriesFor(ctx, sys, aggID)`    |
| `(view, bool)` return                        | `(View, error)` — missing keys return `system.ErrNotFound`, never zero-values  |

The full query surface lives in `systemadapter/queries.go`.

## Drain and readiness semantics

The declarative host exposes `sys.ProjectionHost().Status()` (per-worker
states). There is no `WaitForDrain` on the system path; with the default
GoChannel bus (`BlockPublishUntilSubscriberAck`), a `Dispatch` returns only
after the projection worker's event loop acked the event, so queries usually
see the write immediately. For robustness (first dispatches racing host
startup, cross-instance setups), poll:

```go
func waitForView[V any](read func() (V, error)) (V, bool) {
	var zero V
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		view, err := read()
		if err == nil {
			return view, true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return zero, false
}
```

Runnable proof: `examples/system-demo/` (migrated off ProjectionLayer) and the
test helpers in `systemadapter/declarative_test.go`
(`waitForProjectionsReady` + `eventually`).

## Custom checkpoint / dead-letter stores

`NewProjectionLayer` takes `WithCheckpointStore`/`WithDeadLetterStore` options.
The declarative path does not expose custom-store injection yet — the system's
internal projection host currently uses an in-memory checkpoint store (full
journal replay on restart) and its own dead-letter handling. If you need
persistent checkpoints today, stay on ProjectionLayer until go-cqrs-lite's
`system.New` grows the option (tracked in
`docs/guides/leveraging-system-metaengine.md`); the equivalence test keeps
both paths honest in the meantime.

## Engine support

The declarative folds run on every metaengine driver — memory **and** SQL
engines (SQLite proven by `TestDeclarative_SQLite_UserLifecycle` /
`TestDeclarative_SQLite_AuthzPolicies`; negative paths by
`TestDeclarative_MissingLookups`).

## Equivalence guarantee

`TestDeclarative_EquivalenceWithProjectionLayer` (systemadapter) dispatches
identical commands through both paths and asserts the read models agree on
every compared field. If you find a divergence, that test is the bug report.

## Migration checklist

1. Delete `NewProjectionLayer`/`Start`/`Stop` wiring; call `sys.Start(ctx)`.
2. Replace `pl.<ReadModel>.<Finder>` calls with the `systemadapter` query
   functions (mapping table above); handle `system.ErrNotFound`.
3. Replace `WaitForDrain` with query polling (helper above).
4. Keep behavior: `ErrNotFound` on missing keys replaces `(zero, false)`.
