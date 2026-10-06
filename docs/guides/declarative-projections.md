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

| ProjectionLayer                            | Declarative replacement                                                       |
| ------------------------------------------ | ----------------------------------------------------------------------------- |
| `NewProjectionLayer(sys)` + `Start`/`Stop` | `sys.Start(ctx)` / `sys.Close()` (host is owned by the system)                |
| `pl.WaitForDrain(5 * time.Second)`         | poll the query until it succeeds (see below)                                  |
| `pl.User.FindByID(id)`                     | `systemadapter.FindUserByID(ctx, sys, id.String())` → `(UserView, error)`     |
| `pl.User.FindByEmail(email)`               | `systemadapter.FindUserByEmail(ctx, sys, email)`                              |
| `pl.Tenant.FindByID(id)`                   | `systemadapter.FindTenantByID(ctx, sys, id.String())`                         |
| `pl.Membership.FindByAggregateID(id)`      | `systemadapter.FindMembershipByID(ctx, sys, id.String())`                     |
| `pl.Membership.FindByTenant(tenantID)`     | `systemadapter.FindMembershipsByTenant(ctx, sys, tenantID.String())`          |
| `pl.Bot.*`                                 | `systemadapter.FindBotByID` / `FindBotsByOwner` / `FindBotByTokenHash`        |
| Casbin enforcer from the layer             | `systemadapter.Enforce(ctx, sys, subject, domain, action)` / `FindPolicies`   |
| `pl.AuditLog.Entries()`                    | `systemadapter.AuditEntries(ctx, sys)` / `AuditEntriesFor(ctx, sys, aggID)`   |
| `(view, bool)` return                      | `(View, error)` — missing keys return `system.ErrNotFound`, never zero-values |

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

Both paths take the SAME option vocabulary:

```go
sys, err := system.New(ctx, systemadapter.DomainConfig(
    systemadapter.WithCheckpointStore(durableCheckpointStore),
    systemadapter.WithDeadLetterStore(durableDLQStore),       // replaces the in-memory default
    systemadapter.WithHostOptions(                            // appended after curated defaults,
        projectionhost.WithBatchSize(512),                    // per-field consumer options win
    ),
), deployment)
```

- `WithCheckpointStore` becomes `system.DomainConfig.CheckpointStore`. Left
  nil, `system.New` persists checkpoints as entries of a `system_checkpoints`
  Map collection on the deployment-declared engine when one carries the Map
  ADT (ADR-0142); engines without it fall back to in-memory (replay on
  restart). Passing your own store always wins.
- `WithDeadLetterStore` replaces systemadapter's default in-memory DLQ
  (threshold 10) on the declarative path; the legacy
  `NewProjectionLayer(..., WithDeadLetterStore(...))` call is unchanged.
- `WithHostOptions` appends `projectionhost.HostOption`s (batch size,
  restarts, backoff, logger, metrics) after systemadapter's curated defaults —
  the same tuning the legacy layer ships.

Durability semantics worth knowing: a checkpoint records "everything before me
is processed", so after a restart the host resumes instead of replaying. Pair
durable checkpoints with durable read models (or trigger a rebuild) — a fresh
in-memory read model plus an up-to-date checkpoint processes nothing new and
stays empty. The restart wiring itself is pinned by
`TestDeclarative_DurableCheckpointSurvivesRestart`: same SQLite journal, same
checkpoint store, second `system.New` resumes at exactly the saved positions.

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
