# systemadapter

Bridges [cqrs-htmx](https://github.com/larsartmann/cqrs-htmx)'s identity domain
(User/Membership/Tenant/Bot — 4 deciders, 20 commands, 21 event types) into
[go-cqrs-lite](https://github.com/larsartmann/go-cqrs-lite)'s `system.New()`
composition root and `metaengine` projection planner.

Use `system.New()` as your infrastructure backbone and get the full identity
domain wired automatically.

## Quick start (declarative-first)

```go
domain := systemadapter.DomainConfig()                            // the whole identity domain
deployment := systemadapter.RecommendedSQLiteDeployment("file:app.db") // infrastructure shape

sys, err := system.New(ctx, domain, deployment)
if err != nil { log.Fatal(err) }
if err := sys.Start(ctx); err != nil { log.Fatal(err) }

// Dispatch commands, query read models, mount the dashboard...
sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewCreateTenantCmd(...))
tenant, _ := systemadapter.FindTenantByID(ctx, sys, tenantID.String())
```

The declarative path is canonical: `DomainConfig` registers the deciders,
commands, event decoder, and the projection declarations (read models, Casbin
authz, audit log) that `system.New` auto-wires onto its projection host.

### Deployment presets

| Preset                                                        | Shape                                     | Use                                                    |
| ------------------------------------------------------------- | ----------------------------------------- | ------------------------------------------------------ |
| `RecommendedMemoryDeployment()`                               | one memory engine, every role             | dev, tests                                             |
| `RecommendedSQLiteDeployment(dsn)`                            | one WAL SQLite file, every role           | single-binary persistence                              |
| `RecommendedSplitSQLiteDeployment(eventsDSN, projectionsDSN)` | journal and projections on separate files | production — replays never contend with the write path |

Each returns a plain `system.DeploymentConfig` — inspect, tweak, or pass
straight through.

### Projection knobs

Options ride the `DomainConfig` call and share one vocabulary with the legacy
`NewProjectionLayer`:

```go
systemadapter.DomainConfig(
    systemadapter.WithCheckpointStore(durableCheckpoints), // restart-safe positions
    systemadapter.WithDeadLetterStore(durableDLQ),         // replaces in-memory default
    systemadapter.WithHostOptions(                         // appended after curated defaults
        projectionhost.WithBatchSize(512),
    ),
)
```

Durable checkpoints skip replay of already-processed events after a restart —
pair them with durable read models (or rebuild on demand); a fresh in-memory
read model with an up-to-date checkpoint stays empty by design.

## Introspection

`system.New` exposes topology snapshots, health checks, and explanations
(`sys.Snapshot(ctx)`, `sys.HealthCheck(ctx)`, `sys.Explain(ctx)`) — feed them
to your probes, or mount `cqrs-htmx/dashboardui` over `sys.EventStore()` /
`sys.Bus()` for the ready-made observability dashboard.

## Legacy: NewProjectionLayer

`NewProjectionLayer(sys, ...)` builds the same read models/authz/audit on a
dedicated host. It predates the declarative path, is deprecated, and is
scheduled for the v5 removal bundle — its equivalence with the declarative
path is pinned by `TestDeclarative_EquivalenceWithProjectionLayer`. Use it
only while migrating.

## Docs

- Declarative projections (canonical how-to): `docs/guides/declarative-projections.md`
- system/metaengine deep dive: `docs/guides/leveraging-system-metaengine.md`
- Runnable proof: `examples/system-demo/`
