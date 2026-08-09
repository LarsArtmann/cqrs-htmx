# Leveraging go-cqrs-lite system/ and metaengine/

> How to use go-cqrs-lite's `system.New()` composition root and `metaengine` storage planner with cqrs-htmx's identity-model domain.

## Overview

go-cqrs-lite provides two powerful modules that cqrs-htmx consumers can benefit from:

- **`system/`** — A deployer-driven composition root. `system.New(ctx, domain, deployment)` auto-wires event store, command/query dispatchers, event bus, snapshot store, projection host, and lifecycle management. Separates domain code (`DomainConfig`) from infrastructure decisions (`DeploymentConfig`).
- **`metaengine/`** — A cost-based storage planner. You declare queries + fold functions; the engine auto-plans indexes, ADTs (Map/Set/Counter/Graph/Log), and engine assignments. The `projectionadapter` bridges events into typed projection stores.

The **`systemadapter/`** submodule bridges cqrs-htmx's identity-model domain (4 aggregates, 20 commands, 21 events) into the `system.New()` API with three exports:

| Export | Purpose |
|--------|---------|
| `DomainConfig()` | Pre-wires all deciders + commands + TypeDecoder into a `system.DomainConfig` |
| `EventTypeDecoder()` | Maps all 21 event types to their payload structs for `projectionadapter` |
| `NewProjectionLayer(sys)` | Creates usermgmt read models, Casbin authz, and audit log backed by the system's event store + bus |

## Quick Start

```go
import (
    systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
    "github.com/larsartmann/go-cqrs-lite/system/v4"
)

ctx := context.Background()

// 1. Define deployment (operator-facing, pure data)
deployment := system.DeploymentConfig{
    Engines: map[string]system.EngineConfig{
        "primary": {Driver: "memory"}, // or "sqlite" for persistence
    },
    Instances: []system.InstanceConfig{
        {Role: system.RoleSourceOfTruth, Engines: []string{"primary"}},
    },
}

// 2. Get pre-wired domain config (all 20 commands, 4 deciders, 21 event decodings)
domain := systemadapter.DomainConfig()

// 3. Create system (auto-wires everything)
sys, err := system.New(ctx, domain, deployment)
if err != nil { log.Fatal(err) }
defer sys.Close()

// 4. Create projections (read models, authz, audit)
projLayer, err := systemadapter.NewProjectionLayer(sys)
if err != nil { log.Fatal(err) }
defer projLayer.Stop()

// 5. Start
projLayer.Start(ctx)

// 6. Dispatch commands
disp := sys.CommandDispatcher()
disp.Dispatch(ctx, identitymodel.NewRegisterUserCmd(
    id.NewStreamID(), "alice@example.com", "Alice", nil,
))

// 7. Query read models
projLayer.WaitForDrain(5 * time.Second)
user, _ := projLayer.User.FindByEmail("alice@example.com")
fmt.Println(user.DisplayName)
```

## What DomainConfig() Wires

The `DomainConfig()` function returns a `system.DomainConfig` that registers:

### 4 Deciders (via `system.RegisterDecider`)

| Aggregate | Stream Type | Decider | State Type |
|-----------|-------------|---------|------------|
| User | `"User"` | `usermgmt.UserDecider()` | `UserState` |
| Membership | `"Membership"` | `usermgmt.MembershipDecider()` | `MembershipState` |
| Tenant | `"Tenant"` | `usermgmt.TenantDecider()` | `TenantState` |
| Bot | `"Bot"` | `usermgmt.BotDecider()` | `BotState` |

### 20 Commands (via `system.RegisterCommand`)

All commands from identity-model: 11 User, 3 Membership, 4 Tenant, 2 Bot. Each command handler returns a `system.Op[State]` via `system.Execute()`, which the system routes to the appropriate decider repository.

### 21 Event Type Decodings (via `EventTypeDecoder()`)

A `projectionadapter.TypeDecoder` mapping every event type string to its payload struct, wrapped in `EventWithID[P]` for stream ID access in fold handlers.

## What NewProjectionLayer Provides

The `ProjectionLayer` creates a dedicated `projectionhost.Host` backed by the system's event infrastructure:

| Component | Type | Purpose |
|-----------|------|---------|
| `pl.User` | `*UserReadModel` | Query users by ID, email, external account |
| `pl.Membership` | `*MembershipReadModel` | Query memberships by tenant/actor |
| `pl.Tenant` | `*TenantReadModel` | Query tenants by ID, name |
| `pl.Bot` | `*BotReadModel` | Query bots by owner |
| `pl.Casbin` | `*CasbinProjection` | Authorization policy projection |
| `pl.Authz` | `*Authz` | Casbin enforcer for authz checks |
| `pl.AuditLog` | `*AuditLog` | Append-only audit trail |

The projection host uses checkpoint-based catch-up (survives restarts with persistent checkpoint stores) and supports dead-letter queues for poison messages.

## Deployment Config

### Memory (dev/test)

```go
system.DeploymentConfig{
    Engines: map[string]system.EngineConfig{
        "primary": {Driver: "memory"},
    },
    Instances: []system.InstanceConfig{
        {Role: system.RoleSourceOfTruth, Engines: []string{"primary"}},
    },
}
```

### SQLite (single-file persistence)

```go
system.DeploymentConfig{
    Engines: map[string]system.EngineConfig{
        "primary": {Driver: "sqlite", DSN: "file:app.db"},
    },
    Instances: []system.InstanceConfig{
        {Role: system.RoleSourceOfTruth, Engines: []string{"primary"}},
    },
}
```

### SQLite + Separate Projection Engine

```go
system.DeploymentConfig{
    Engines: map[string]system.EngineConfig{
        "events": {Driver: "sqlite", DSN: "file:events.db"},
        "projections": {Driver: "sqlite", DSN: "file:projections.db"},
    },
    Instances: []system.InstanceConfig{
        {Role: system.RoleSourceOfTruth, Engines: []string{"events"}},
        {Role: system.RoleProjections, Engines: []string{"projections"}},
    },
}
```

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

For consumers who want to use metaengine's cost-based planner for custom projections, the `EventTypeDecoder()` provides the event type registry. Declare queries with fold functions:

```go
import (
    "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

queries := []metaengine.QueryDecl[any, any]{
    metaengine.Query[UserByEmailQuery, UserView]("user-by-email",
        metaengine.On(identitymodel.UserRegisteredPayload{}, func(p identitymodel.UserRegisteredPayload) (string, string, error) {
            return p.Email, p.Email, nil
        }),
    ),
}

store, _ := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, queries...)
```

The system's `ProjectionTypeDecoder` ensures events are decoded into the right Go types for these fold handlers.

## See Also

- **Runnable example**: `examples/system-demo/` — full working demo
- **go-cqrs-lite system/ docs**: `https://github.com/larsartmann/go-cqrs-lite/tree/main/system`
- **go-cqrs-lite metaengine/ docs**: `https://github.com/larsartmann/go-cqrs-lite/tree/main/metaengine`
- **Leveraging go-cqrs-lite guide**: `docs/guides/leveraging-go-cqrs-lite.md`
- **Full-stack wiring guide**: `docs/guides/fullstack-wiring.md`
