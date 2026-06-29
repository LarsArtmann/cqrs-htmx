# ADR 0031: Projection Lifecycle — StartProjections vs projectionhost vs CatchUpSubscriber

**Status:** PROPOSED — 2026-06-29
**Related:** [ADR 0016](0016-go-cqrs-lite-v3-migration.md) (projection rewrite), [go-cqrs-lite projectionhost/v3](https://github.com/LarsArtmann/go-cqrs-lite), [go-cqrs-lite watermill/v3 CatchUpSubscriber](https://github.com/LarsArtmann/go-cqrs-lite)

## Context

cqrs-htmx's `usermgmt.StartProjections` (`es_projection_setup.go`) is a 155-line hand-rolled implementation that:

1. Reads ALL events from the journal via `journal.ReadAll()` (synchronous)
2. Dispatches each event to every registered projection (synchronous)
3. Subscribes to live events via `bus.SubscribeAll()` with a dedup map

Combined with the synchronous event bus (`watermill.EventBus` with `BlockPublishUntilSubscriberAck: true`), this provides **read-your-writes consistency**: after `Execute()` returns, the read model already reflects the change — no timing-based sleeps.

go-cqrs-lite v3.4.0 ships two alternatives that could replace this code:

### Option A: `projectionhost.Host`

- Per-projection goroutines with crash auto-restart + exponential backoff
- Checkpoint persistence (no full journal replay on restart)
- Poison-message dead-letter queue
- `MetricsRecorder` interface for operational visibility

### Option B: `watermill.CatchUpSubscriber`

- Replay from journal via `ReadFrom(afterEventID, limit)` (cursor-based)
- Checkpoint persistence per topic
- Lighter than projectionhost — preserves the bus subscriber model
- Replay-to-live dedup via event ID set

## Decision

**DEFERRED — no change in v3.3.0.**

The current synchronous `StartProjections` is deliberately simple and provides a critical guarantee: read-your-writes consistency during startup. Both upstream alternatives run replay **asynchronously** in goroutines:

- **projectionhost.Host.Start(ctx)** launches per-projection workers in goroutines. Replay happens in the background. A command dispatched immediately after `NewService()` may not see its own write.
- **CatchUpSubscriber.Subscribe(ctx, topic)** also runs replay in a goroutine. Same gap.

### The Core Tradeoff

| Feature                        | StartProjections (current)     | projectionhost   | CatchUpSubscriber |
| ------------------------------ | ------------------------------ | ---------------- | ----------------- |
| Read-your-writes on startup    | **Yes** (synchronous)          | No (async)       | No (async)        |
| Checkpoint persistence         | No (full replay every restart) | **Yes**          | **Yes**           |
| Crash auto-restart             | No                             | **Yes**          | No                |
| Dead-letter queue              | No                             | **Yes**          | No                |
| Code complexity                | 155 LOC (hand-rolled)          | ~0 LOC (library) | ~10 LOC (library) |
| Full journal replay on restart | **Yes** (O(n) every boot)      | No (O(delta))    | No (O(delta))     |

### Recommendation (for future release)

**CatchUpSubscriber is the better fit** for cqrs-htmx:

- Lighter weight than projectionhost (no worker goroutines, no DLQ complexity)
- Preserves the bus subscriber model that `BlockPublishUntilSubscriberAck` relies on
- Cursor-based replay (`ReadFrom`) is more efficient than `ReadAll` for large journals

The read-your-writes-during-startup gap can be solved with a **sync-wait wrapper**: after starting CatchUpSubscriber, block until the first live event is received (signaling replay is complete). This adds ~5 LOC.

**projectionhost is overkill** for cqrs-htmx's single-process deployment model. Its DLQ and crash-restart features shine in multi-process deployments where a projection handler might OOM. For a library that consumers embed in their own binary, the simpler CatchUpSubscriber is sufficient.

## Consequences

- **v3.3.0**: No change. StartProjections remains the implementation.
- **v3.4.0 (target)**: Adopt CatchUpSubscriber with a sync-wait wrapper. Eliminates 155 LOC of hand-rolled replay. Gains checkpoint persistence (faster restarts). Read-your-writes preserved via wrapper.
- **Future**: If a production deployment hits projection failure storms, upgrade to projectionhost for DLQ + crash restart.
