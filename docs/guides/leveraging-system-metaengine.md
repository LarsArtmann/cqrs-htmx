# Leveraging go-cqrs-lite system/ and metaengine/

> How to use go-cqrs-lite's `system.New()` composition root and `metaengine` storage planner with cqrs-htmx's identity-model domain.

> **Declarative-first (ADR-0051):** `system.New()` wires the projections
> itself — `DomainConfig()` carries the declarations. `NewProjectionLayer`
> is the deprecated pre-declarative path, kept only for migration and
> scheduled for the v5 removal bundle. The declarative how-to lives in
> [`declarative-projections.md`](declarative-projections.md); this guide
> keeps the ProjectionLayer sections for migration reference.

## Overview

go-cqrs-lite provides two powerful modules that cqrs-htmx consumers can benefit from:

- **`system/`** — A deployer-driven composition root. `system.New(ctx, domain, deployment)` auto-wires event store, command/query dispatchers, event bus, snapshot store, projection host, and lifecycle management. Separates domain code (`DomainConfig`) from infrastructure decisions (`DeploymentConfig`).
- **`metaengine/`** — A cost-based storage planner. You declare queries + fold functions; the engine auto-plans indexes, ADTs (Map/Set/Counter/Graph/Log), and engine assignments. The `projectionadapter` bridges events into typed projection stores.

The **`systemadapter/`** submodule bridges cqrs-htmx's identity-model domain (4 aggregates, 20 commands, 21 events) into the `system.New()` API with three exports:

| Export                     | Purpose                                                                                                         |
| -------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `DomainConfig()`           | Pre-wires all deciders + commands + TypeDecoder into a `system.DomainConfig`                                    |
| `EventTypeDecoder()`       | Maps all 21 event types to their payload structs for `projectionadapter`                                        |
| `DomainConfig(opts...)`    | Pre-wires deciders + commands + TypeDecoder + declarative projections; options carry checkpoint/DLQ/host tuning |
| `Recommended*Deployment()` | Memory / SQLite / split-SQLite deployment presets                                                               |
| `EventTypeDecoder()`       | Maps all 21 event types to their payload structs for `projectionadapter`                                        |
| `NewProjectionLayer(sys)`  | **Deprecated** (pre-declarative): read models, Casbin authz, audit log on a dedicated host                      |

## Quick Start

```go
import (
    systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
    "github.com/larsartmann/go-cqrs-lite/system/v4"
)

ctx := context.Background()

// 1. Pick a deployment preset (or hand-write system.DeploymentConfig)
deployment := systemadapter.RecommendedSQLiteDeployment("file:app.db")

// 2. Get pre-wired domain config (all 20 commands, 4 deciders, 21 event
//    decodings, AND the declarative projections — read models, authz, audit)
domain := systemadapter.DomainConfig(
    systemadapter.WithCheckpointStore(durableCheckpoints), // optional
)

// 3. Create + start the system — projections come up with it, no extra step
sys, err := system.New(ctx, domain, deployment)
if err != nil { log.Fatal(err) }
defer sys.Close()
if err := sys.Start(ctx); err != nil { log.Fatal(err) }

// 4. Dispatch commands
disp := sys.CommandDispatcher()
disp.Dispatch(ctx, identitymodel.NewRegisterUserCmd(
    id.NewStreamID(), "alice@example.com", "Alice", nil,
))

// 5. Query the declarative read models through the typed helpers
user, err := systemadapter.FindUserByEmail(ctx, sys, "alice@example.com")
if err != nil { log.Fatal(err) }
fmt.Println(user.DisplayName)
```

## What DomainConfig() Wires

The `DomainConfig()` function returns a `system.DomainConfig` that registers:

### 4 Deciders (via `system.RegisterDecider`)

| Aggregate  | Stream Type    | Decider                        | State Type        |
| ---------- | -------------- | ------------------------------ | ----------------- |
| User       | `"User"`       | `usermgmt.UserDecider()`       | `UserState`       |
| Membership | `"Membership"` | `usermgmt.MembershipDecider()` | `MembershipState` |
| Tenant     | `"Tenant"`     | `usermgmt.TenantDecider()`     | `TenantState`     |
| Bot        | `"Bot"`        | `usermgmt.BotDecider()`        | `BotState`        |

### 20 Commands (via `system.RegisterCommand`)

All commands from identity-model: 11 User, 3 Membership, 4 Tenant, 2 Bot. Each command handler returns a `system.Op[State]` via `system.Execute()`, which the system routes to the appropriate decider repository.

### 21 Event Type Decodings (via `EventTypeDecoder()`)

A `projectionadapter.TypeDecoder` mapping every event type string to its payload struct, wrapped in `EventWithID[P]` for stream ID access in fold handlers.

## What NewProjectionLayer Provides (deprecated)

The `ProjectionLayer` creates a dedicated `projectionhost.Host` backed by the system's event infrastructure:

| Component       | Type                   | Purpose                                    |
| --------------- | ---------------------- | ------------------------------------------ |
| `pl.User`       | `*UserReadModel`       | Query users by ID, email, external account |
| `pl.Membership` | `*MembershipReadModel` | Query memberships by tenant/actor          |
| `pl.Tenant`     | `*TenantReadModel`     | Query tenants by ID, name                  |
| `pl.Bot`        | `*BotReadModel`        | Query bots by owner                        |
| `pl.Casbin`     | `*CasbinProjection`    | Authorization policy projection            |
| `pl.Authz`      | `*Authz`               | Casbin enforcer for authz checks           |
| `pl.AuditLog`   | `*AuditLog`            | Append-only audit trail                    |

The projection host uses checkpoint-based catch-up (survives restarts with persistent checkpoint stores) and supports dead-letter queues for poison messages.

## Deployment Config

systemadapter ships the three common shapes as presets (the SQLite presets
set `journal_mode=wal`), so hand-writing the struct is only needed for
genuinely custom topologies:

```go
// Memory (dev/test) — one engine, everything in-process, dies with it:
systemadapter.RecommendedMemoryDeployment()

// SQLite (single-file persistence) — one WAL file holds every role:
systemadapter.RecommendedSQLiteDeployment("file:app.db")

// SQLite + separate projection engine — journal and projections on
// different files so replays never contend with the write path:
systemadapter.RecommendedSplitSQLiteDeployment("file:events.db", "file:projections.db")
```

Each returns a plain `system.DeploymentConfig` — inspect it, tweak it, or
pass it straight to `system.New`.

## System Introspection

```go
// Topology
topology, _ := sys.Snapshot(ctx)

// Health for k8s probes
if err := sys.HealthCheck(ctx); err != nil { /* unhealthy */ }

// Human-readable explanation
fmt.Println(sys.Explain(ctx))

// Per-engine health
health := sys.HealthCheckDetailed(ctx)

// Shutdown order
fmt.Println(sys.ShutdownOrder())
```

## Safety Checks (SCREAM Store)

The system runs safety checks before construction:

```go
report, _ := system.CheckSafety(ctx, deployment)
if report.HasErrors() {
    for _, d := range report.Diagnostics {
        fmt.Println(d.Tier, d.Rule, d.Detail)
    }
}
```

Rules:

- `volatile-source-of-truth` — warns if using memory driver with non-relaxed durability
- `durability-downgrade` — warns if source-of-truth durability is Relaxed

## Lifecycle

Declarative path: `sys.Start(ctx)` starts everything (projections included);
`sys.GracefulClose(ctx)` drains and closes. The legacy layer adds its own two
steps around the system:

```go
sys, _ := system.New(ctx, domain, deployment)
projLayer, _ := systemadapter.NewProjectionLayer(sys)

projLayer.Start(ctx)     // start projection workers
// ... use the system ...
projLayer.Stop()          // stop projections (graceful)
sys.Close()               // close system (stops engines, joins errors)
```

For Kubernetes graceful shutdown:

```go
sys.GracefulClose(ctx)    // drain in-flight work, then close
```

## Metaengine Projections (Advanced)

For custom projections on top of the same plumbing, the pattern is
`metaengine.Query[Input, View]` carrying `OnRecordTyped` folds over
`projectionadapter.EventWithID[P]` — the decoded payload plus the event's
journal ID, so one fold can key by ID, email, or any derived value. This is
exactly how `DeclarativeProjections()` builds every identity view:

```go
tenantEvents := metaengine.Query[system.LookupInput[string], TenantView]("tenant_by_id",
    metaengine.OnRecordTyped(
        string(identitymodel.EventTenantCreated),
        projectionadapter.EventWithID[identitymodel.TenantCreatedPayload]{},
        func(_ record.Record, e projectionadapter.EventWithID[identitymodel.TenantCreatedPayload]) (string, TenantView) {
            return e.ID, TenantView{ID: e.ID, Name: e.Payload.Name}
        },
    ),
    // ...further folds for Suspended/Reactivated/Deleted
)

decl := system.RawQuery(tenantEvents) // hand the declaration to system.New
```

`DomainConfig().ProjectionTypeDecoder` (from `EventTypeDecoder()`) is what
turns journal records into those typed payloads before the folds run — register
your custom event types the same way if you extend the domain.

## See Also

- **Runnable example**: `examples/system-demo/` — full working demo
- **go-cqrs-lite system/ docs**: `https://github.com/larsartmann/go-cqrs-lite/tree/master/system`
- **go-cqrs-lite metaengine/ docs**: `https://github.com/larsartmann/go-cqrs-lite/tree/master/metaengine`
- **Leveraging go-cqrs-lite guide**: `docs/guides/leveraging-go-cqrs-lite.md`
- **Full-stack wiring guide**: `docs/guides/fullstack-wiring.md`
