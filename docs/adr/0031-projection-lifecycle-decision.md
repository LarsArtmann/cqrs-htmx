# ADR 0031: Projection Lifecycle — StartProjections vs projectionhost vs CatchUpSubscriber

**Status:** Superseded — 2026-07-22 (projectionhost/v4 adopted; CatchUpSubscriber evaluation closed as Not Needed)
**Related:** [ADR 0016](0016-go-cqrs-lite-v3-migration.md) (projection rewrite), [ADR 0032](0032-basic-command-embedding.md) (command embedding), [go-cqrs-lite projectionhost/v4](https://github.com/LarsArtmann/go-cqrs-lite), [go-cqrs-lite watermill/v4 CatchUpSubscriber](https://github.com/LarsArtmann/go-cqrs-lite)

## Context

cqrs-htmx's `usermgmt.StartProjections` (`es_projection_setup.go`) was a 155-line hand-rolled implementation that:

1. Reads ALL events from the journal via `journal.ReadAll()` (synchronous)
2. Dispatches each event to every registered projection (synchronous)
3. Subscribes to live events via `bus.SubscribeAll()` with a dedup map

Combined with the synchronous event bus (`watermill.EventBus` with `BlockPublishUntilSubscriberAck: true`), this provides **read-your-writes consistency**: after `Execute()` returns, the read model already reflects the change — no timing-based sleeps.

go-cqrs-lite v4 ships two alternatives that could replace this code:

### Option A: `projectionhost.Host`

- Per-projection goroutines with crash auto-restart + exponential backoff
- Checkpoint persistence per projection (no full journal replay on restart)
- Poison-message dead-letter queue
- `MetricsRecorder` interface for operational visibility
- `WithSubscriber` option enables live tailing after journal drain

### Option B: `watermill.CatchUpSubscriber`

- Replay from journal via `ReadFrom(afterEventID, limit)` (cursor-based)
- Checkpoint persistence per topic
- Lighter than projectionhost — preserves the bus subscriber model
- Replay-to-live dedup via event ID set
- Requires `message.Message` adapter (Watermill model, not `event.Event`)

## Decision

**projectionhost/v4 adopted. CatchUpSubscriber evaluation closed as Not Needed.**

### Timeline

1. **v3.3.0 (interim):** Checkpoint-based replay shipped in the hand-rolled `StartProjections` via optional `event.CheckpointStore`. Eliminated O(n) full-journal replay on restart. The 155 LOC of hand-rolled replay+dedup+live-handler logic remained.

2. **v4.x (this ADR update):** `StartProjections` rewritten to use `projectionhost.Host` internally. The hand-rolled replay, dedup ring, and live handler are deleted — replaced by ~60 LOC of projectionhost wiring + a `waitForDrain` sync-wait wrapper that preserves read-your-writes.

### Why projectionhost over CatchUpSubscriber

The original ADR recommended CatchUpSubscriber as "the better fit" and called projectionhost "overkill." That assessment was reversed:

- **CatchUpSubscriber requires a `message.Message` adapter** — it operates in Watermill's message model, not go-cqrs-lite's `event.Event` model. This adds adapter complexity rather than reducing it.
- **projectionhost already includes the replay→live handoff** via `WithSubscriber(event.Subscriber)`. Each worker drains the journal (checkpoint-based `ReadFrom`), then transitions to `SubscribeAll` with dedup at the boundary. This is exactly what CatchUpSubscriber does, but integrated into the projection lifecycle.
- **The DLQ and crash-restart features are not "overkill" for a library** — consumers embed cqrs-htmx in production binaries where projection handlers can fail on poison events. Without DLQ, a single bad event blocks all projections indefinitely.
- **Per-projection checkpoints** (keyed by projection `Name()`) are more granular than CatchUpSubscriber's per-topic model. Each projection resumes from its own position, not a shared cursor.

### The sync-wait wrapper

The read-your-writes concern (the original reason projectionhost was rejected) is solved by `waitForDrain` in `es_projection_setup.go`: after `host.Start()`, the function polls `host.Status()` until all workers reach `WorkerLive` or `WorkerStopped` (the watermill bus returns from `SubscribeAll` immediately after registering the handler). This preserves the synchronous startup guarantee.

## Consequences

- **projectionhost/v4 is a new dependency** of the `usermgmt` module. It was already in `go.work` replaces; now it's in `usermgmt/go.mod` require.
- **Per-projection checkpoint keys** replace the former single `"usermgmt:start_projections"` key. Existing checkpoint data is incompatible — one-time full replay on first run after upgrade.
- **DLQ is enabled by default** (in-memory `MemoryDeadLetterStore`). Poison events are captured, not fatal. Consumers can inspect/replay via `host.ReplayDeadLetters()`.
- **`StartProjections` now returns `(*projectionhost.Host, error)`** instead of just `error`. Callers must `defer host.Stop()`. All setup structs (`EventSourcedSetup`, `SQLiteEventSourcedSetup`, `PostgresEventSourcedSetup`, `Service`) hold the host and stop it in `Close()`.
- **CatchUpSubscriber is not needed.** The replay→live handoff it provides is already built into projectionhost via `WithSubscriber`. Adopting CatchUpSubscriber would add the message-model adapter overhead without any benefit.

## Comparison (final)

| Feature                        | Former StartProjections (v3.3.0)   | projectionhost (adopted) | CatchUpSubscriber (not adopted) |
| ------------------------------ | ----------------------------------- | ------------------------ | ------------------------------- |
| Read-your-writes on startup    | **Yes** (synchronous)               | **Yes** (waitForDrain)   | Would need wrapper              |
| Checkpoint persistence         | Single key                          | **Per-projection**       | Per-topic                       |
| Crash auto-restart             | No                                  | **Yes**                  | No                              |
| Dead-letter queue              | No                                  | **Yes**                  | No                              |
| Code complexity                | 155 LOC (hand-rolled)               | ~60 LOC (wiring+wrapper) | ~10 LOC + message adapter       |
| Full journal replay on restart | Only when cpStore is nil            | No (O(delta))             | No (O(delta))                   |
| Message model                  | `event.Event`                       | `event.Event`             | `message.Message` (adapter req) |
