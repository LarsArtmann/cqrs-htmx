# ADR-0041: Aggregate Snapshot Integration

## Status

ACCEPTED — 2026-07-19

## Context

cqrs-htmx's usermgmt module is fully event-sourced: every Load of an aggregate
(User, Membership, Tenant, Bot) replays its full event journal from the event
store. For aggregates with a small number of events this is fast, but as an
aggregate accumulates thousands of events (>10K), every Load pays the full
replay cost — which degrades read latency for long-lived users and busy
tenants.

go-cqrs-lite v4 ships snapshot support in the `decider` package
(`WithSnapshotStore`, `WithCodec`, `WithSnapshotStrategy`) and a `snapshot`
package (`SnapshotStore` interface, `EveryNEvents` and `ReadPressure`
strategies). The Repository restores the most recent snapshot on Load and
replays only the events appended since — turning O(n) replay into O(1) restore

- O(delta) replay.

## Decision

**Wire go-cqrs-lite's snapshot support into usermgmt as an opt-in configuration.**

- Add `SnapshotConfig` (Store, Codec, Strategy) embedded in `ServiceConfig`,
  `EventSourcedConfig`, `SQLiteSetupConfig`, and `PostgresSetupConfig`.
- `snapshotOptions[State]` translates the config into the typed
  `decider.RepositoryOption` list, threaded through `buildStackRepositories`
  and `buildDeciderRepositories`.
- Ship `MemorySnapshotStore` for dev/test (deep-copies state bytes; safe for
  concurrent use).
- Zero-value `SnapshotConfig` (nil Store) leaves repositories in full-replay
  mode: zero behavior change for existing consumers.

## Consequences

- **Positive:** High-event-volume aggregates load in constant time after a
  snapshot. Snapshots are best-effort (encode/save errors are logged and
  swallowed on the write path), so a snapshot failure never blocks a command.
- **Positive:** Fully opt-in and backward-compatible — no consumer is forced to
  adopt snapshots, and unconfigured setups see no change.
- **Negative:** Production deployments that want snapshot durability across
  restarts must supply a persistent SnapshotStore (SQL/pebble); the in-memory
  store is dev/test only.
- **Trade-off:** `EveryNEvents(n)` snapshots on a write cadence; for hot-read
  aggregates use `NewReadPressure(threshold)` instead. The choice is the
  consumer's, per `SnapshotConfig.Strategy`.

## Usage

    strategy, _ := snapshot.EveryNEvents(500)
    svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
        SnapshotConfig: usermgmt.SnapshotConfig{
            Store:    usermgmt.NewMemorySnapshotStore(),
            Codec:    codec.JSONCodec{},
            Strategy: strategy,
        },
    })

Addresses TODO_LIST "Snapshot integration for high-event-volume aggregates".
