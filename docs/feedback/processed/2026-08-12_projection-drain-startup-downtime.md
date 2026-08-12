# Feedback: Synchronous projection drain causes multi-minute downtime on every restart

**From:** browser-history project (github.com/larsartmann/browser-history)
**Date:** 2026-08-12
**Perspective:** Consumer deploying browser-history behind a reverse proxy in a homelab
**Tone:** Direct, constructive, with concrete proposals

---

## The problem

Every restart of a cqrs-htmx application causes a **multi-minute outage** because `usermgmt.NewService()` calls `waitForDrain()` synchronously during construction. The HTTP server does not bind until ALL projection workers have finished replaying the full event journal.

In browser-history, this means:

```
t=0s    Old process stops → 502 from Caddy
t=0s    New binary starts, NewService() begins
t=0s    Projection drain starts: 6 projections (User, Membership, Tenant, Bot, Casbin, AuditLog)
t=0-4m  Replaying entire event journal... HTTP server NOT listening, port 8087 closed
t=~4m   Drain complete, NewService() returns, server binds 127.0.0.1:8087
t=~4m   Back online
```

`ProjectionDrainTimeout` is set to **5 minutes** in browser-history. The actual drain takes 2-4 minutes depending on journal size and disk I/O.

**Every deploy triggers this.** Every crash-restart triggers this. Every config change triggers this. In a homelab where `nh os switch` rebuilds the systemd unit, this means a guaranteed 2-4 minute window where `https://history.home.lan` returns 502.

---

## Why this is a design problem

### 1. Readiness and liveness are conflated

The synchronous drain pattern conflates "the process is alive" with "the read models are fully consistent." These are fundamentally different concerns:

- **Liveness** = the HTTP server is accepting connections and can respond
- **Readiness** = all projections have caught up and reads will reflect all writes

A well-designed system starts the HTTP server immediately and gates reads behind a readiness check. During the catch-up window:

- Writes can be accepted (events appended to the journal — projections catch up async)
- Reads can return stale-but-valid data, or return 503 from `/readyz` if strict consistency is required
- Health checks pass (process is alive and serving)

cqrs-htmx already HAS a `ReadinessHandler` (`readiness.go`) and a `ProjectionStatusHandler` (`projection_status_handler.go`) — but they're useless during startup because the HTTP server never starts until drain is complete. The readiness infrastructure exists; it's just never exercised during the startup window where it matters most.

### 2. The checkpoint mechanism exists but is impractical to use

`ServiceConfig.CheckpointStore` would solve this — projections resume from the last checkpoint instead of replaying the full journal. But browser-history explicitly does NOT set it:

```go
// NOTE: CheckpointStore and ReadModelDB are intentionally NOT set here.
```

The reason: the read models are in-memory maps hydrated by projections. If `CheckpointStore` skips past the events that populated those maps, the maps are empty after restart — `FindByUserID` returns "not found."

This is a **catch-22**:

- No checkpoint → full replay every restart → 2-4 min downtime
- Checkpoint set → skip replayed events → in-memory maps empty → broken read models

There is no "hydrate read models from a SQL snapshot, then resume from checkpoint" path. The library offers checkpoints OR full replay, with no middle ground for consumers whose read models are ephemeral (in-memory) but whose journals grow over time.

### 3. The drain timeout is a band-aid, not a solution

`DrainTimeout` (default 30s, browser-history sets 5m) just controls how long to wait before giving up. It doesn't make the drain faster, doesn't reduce the outage window, and doesn't provide a fallback path. If the drain exceeds the timeout, the service **fails to start entirely** — which is worse than starting in a degraded state.

---

## Impact

| Scenario                    | Current behavior                   | Impact                                            |
| --------------------------- | ---------------------------------- | ------------------------------------------------- |
| Deploy (`nh os switch`)     | 2-4 min 502 outage                 | Deploy windows must be planned around downtime    |
| Crash + auto-restart        | 2-4 min outage per restart         | Cascading failures during instability             |
| Large event journal         | Drain time grows linearly          | Gets worse over time; no plateau                  |
| Slow disk I/O (build storm) | Drain takes even longer            | Worst-case outage when system is already stressed |
| Pre-deploy health checks    | Can't reach `/health` during drain | Post-deploy smoke tests fail during drain window  |

---

## Proposed solutions

### Option A: Async projection startup with readiness gate (recommended)

Start the HTTP server immediately. Run projection drain in the background. Gate read endpoints behind a readiness check.

```go
// New config option
type ServiceConfig struct {
    // ...
    // SyncDrain controls whether NewService blocks until projections drain.
    // When false (the new default), projections drain in the background
    // and the HTTP server can start immediately. Use ReadinessHandler
    // to gate traffic until drain completes.
    // When true (legacy behavior), NewService blocks until drain or timeout.
    SyncDrain bool
}
```

When `SyncDrain = false`:

1. `NewService()` starts projection workers and returns immediately
2. The consumer's HTTP server binds and starts accepting connections
3. `ReadinessHandler` returns 503 until all projections reach `WorkerLive`
4. Writes work immediately (events are appended to the journal)
5. Reads either return 503 (strict) or stale data (lenient — consumer choice)

**The readiness infrastructure already exists** — `setup.Bundle.healthHandler()` checks `ProjectionStatuses()`. It just needs to also check "are we still draining?" The `ProjectionStatusEntry` already has `Status`, `Processed`, `LagMillis` — add a `Draining bool` field or check if any worker is still in `WorkerStarting` state.

### Option B: SQL-backed read model hydration + checkpoint resume

Provide a path to hydrate in-memory read models from a persistent store on restart, then resume projections from the last checkpoint. This eliminates full journal replay without the empty-maps problem.

```go
type ServiceConfig struct {
    // ...
    // ReadModelHydrator, when set, is called on startup to populate
    // in-memory read models from persistent storage BEFORE projections
    // resume from checkpoint. This avoids full journal replay.
    ReadModelHydrator ReadModelHydrator
}
```

The hydrator reads the current state of each read model from SQL, populates the in-memory maps, then projections resume from the checkpoint. Only events since the last checkpoint are replayed.

This is more complex to implement but provides the fastest restart time (seconds, not minutes) and is the correct long-term architecture for event-sourced systems with large journals.

### Option C: Projection snapshots (materialized read-model state)

Periodically snapshot the materialized read-model state (the in-memory maps). On restart, load the snapshot, then replay only events since the snapshot.

```go
type ServiceConfig struct {
    // ...
    ProjectionSnapshotStore SnapshotStore
    SnapshotInterval        int  // snapshot every N events
}
```

This is the standard event-sourcing pattern. It's between Option A (no persistence, just async) and Option B (full SQL hydration). Snapshots are simpler than full SQL hydration because they're just serialized maps, not live SQL queries.

### Option D: Provide a SQLite CheckpointStore implementation

The library currently provides `memory.NewMemoryCheckpointStore()` as the default. Providing a ready-to-use SQLite implementation would lower the barrier:

```go
// In cqrs-htmx or a sub-package:
cpStore, err := eventstore.NewSQLiteCheckpointStore(db)
```

This alone doesn't solve the empty-maps problem, but paired with Option B or C, it makes the incremental-replay path turnkey.

---

## What the consumer can do NOW (without library changes)

browser-history already has the right pieces — it just doesn't wire them to usermgmt:

1. **`api/checkpoint.go`** — A SQLite checkpoint store already exists for visit-event projections. It could be adapted to implement `event.CheckpointStore` for usermgmt projections.
2. **`api/server_setup.go`** — `replayEvents()` already loads a checkpoint and replays from an offset for visit events.
3. **The missing link** — There's no path to hydrate usermgmt's in-memory read models from SQL on restart. This is the library-level gap.

---

## Comparison to other systems

| System                     | Startup replay                                                     | Outage window |
| -------------------------- | ------------------------------------------------------------------ | ------------- |
| Kafka Streams              | Replays from last committed offset, not from beginning             | Seconds       |
| EventStoreDB (Projections) | Emits emitted events continuously; restart resumes from checkpoint | Seconds       |
| Axon Framework (Java)      | Supports snapshotting + token-based tracking; async by default     | Seconds       |
| cqrs-htmx (current)        | Full journal replay, synchronous, blocks HTTP server               | Minutes       |

The synchronous full-replay pattern is appropriate for **integration tests** and **first-run bootstrap**. It is not appropriate as the only startup mode for production deployments.

---

## Summary

| Aspect                    | Current state                                   | With async startup                               |
| ------------------------- | ----------------------------------------------- | ------------------------------------------------ |
| Deploy downtime           | 2-4 minutes (502 from reverse proxy)            | Seconds (server binds immediately)               |
| Crash recovery            | 2-4 minute outage per restart                   | Seconds (server binds, projections catch up)     |
| Scaling with journal size | Drain time grows linearly forever               | Checkpoint/snapshot bounds replay                |
| Readiness signaling       | Binary (server up = ready)                      | Gradual (`/readyz` reflects projection lag)      |
| Health check accuracy     | `/health` unreachable during drain (looks dead) | `/health` 200 (alive), `/readyz` 503 (not ready) |

The fix is not architectural redesign — it's **decoupling liveness from readiness**. The library already has `ReadinessHandler`, `ProjectionStatusHandler`, and `ProjectionStatuses()`. The missing piece is choosing to use them during the startup window instead of blocking the entire server behind a synchronous drain.
