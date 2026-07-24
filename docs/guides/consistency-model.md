# Consistency Model

> What consistency guarantees cqrs-htmx provides, how they work, and what is explicitly NOT guaranteed.

---

## Overview

cqrs-htmx uses CQRS with event sourcing. The write side (commands) and read side (projections) are separated by an asynchronous event bus. This document explains exactly what consistency properties hold and what consumers must not assume.

---

## Guarantees That Hold

### Read-Your-Writes at Startup

When `StartProjections` returns (called internally by `NewEventSourcedSetup` and `NewService`), all projections have drained the full event journal. This means:

- After `NewService()` returns, every query reflects all historical events.
- There is no startup race where a query returns stale data.
- The `waitForDrain` function (`es_projection_setup.go:131`) blocks until every projection worker reaches `WorkerLive` or `WorkerStopped` state.

**What this means for consumers:** You can start serving HTTP requests immediately after `NewService()` returns. No warmup period is needed.

### Causal Consistency Per Aggregate (Steady State)

During normal operation:

1. A command appends events to the event store (optimistic concurrency via expected-version).
2. Events are published to the in-process event bus.
3. Each projection processes events **in order** (per aggregate stream, via the journal's ReadFrom).
4. A single-process deployment processes all events through one bus — no cross-process reordering.

**What this means:** If command A completes before command B starts (same aggregate), then all projections will process A's events before B's events. This is causal consistency within a single process.

### Idempotent Projection Processing

Projections use per-projection checkpoints (keyed by `projection.Name()`). If a projection crashes and restarts:

1. It resumes from its last checkpoint.
2. Events already processed are not re-applied (checkpoint prevents re-reading).
3. If a duplicate event somehow reaches the projection, the read model's idempotent handlers (e.g., `INSERT OR REPLACE`) handle it safely.

---

## What Is NOT Guaranteed

### Cross-Projection Atomicity

Projections process events independently. If a `UserRegistered` event updates both the `user-read-model` and the `casbin-projection`, there is a brief window where one is updated and the other is not.

**Impact:** A query to the user read model might show a new user before the Casbin policy for that user is ready (or vice versa). This window is sub-millisecond in practice (in-process bus).

**Mitigation:** If you need cross-projection atomicity, you need a single projection that updates both atomically, or you accept the brief inconsistency.

### Multi-Instance Consistency

cqrs-htmx assumes a single process. If you deploy multiple instances:

- Each instance has its own in-process event bus.
- Events written on instance A will NOT automatically reach projections on instance B.
- The event store is shared (SQL), but the bus is not.

**This is by design.** cqrs-htmx is a library, not distributed infrastructure. If you need multi-instance, you need an external message bus (Kafka, NATS, etc.) and a custom `event.Bus` implementation.

### Monotonic Reads Across Reconnects

If a projection is rebuilt (checkpoint cleared), it replays from scratch. During replay, there is no guarantee that a query sees a monotonically advancing state. The projection might be empty, then suddenly show all data after replay completes.

**Mitigation:** Rebuild during maintenance windows. See `docs/guides/rebuild-projection-runbook.md`.

### Strong Consistency Between Command and Query

There is no synchronous command-query cycle. After a command succeeds:

1. Events are appended to the store.
2. Events are published to the bus asynchronously.
3. Projections process them and update read models.

A query immediately after a command might NOT reflect the command's effect. The delay is sub-millisecond in practice (in-process bus, single goroutine per projection), but it is not zero.

**When this matters:** WebAuthn registration flows that immediately query the user's credentials. The `Service` handles this by reading from the write model (aggregate state) when it needs strong consistency, not from the read model.

---

## Summary

| Property                         | Guaranteed?          | Mechanism                            |
| -------------------------------- | -------------------- | ------------------------------------ |
| Read-your-writes at startup      | Yes                  | `waitForDrain` in `StartProjections` |
| Causal consistency per aggregate | Yes (single process) | In-order event processing per stream |
| Idempotent processing            | Yes                  | Checkpoints + idempotent handlers    |
| Cross-projection atomicity       | No                   | Independent projection goroutines    |
| Multi-instance consistency       | No                   | In-process bus only                  |
| Monotonic reads during rebuild   | No                   | Full replay from scratch             |
| Strong command-query consistency | No                   | Async projection lag                 |
