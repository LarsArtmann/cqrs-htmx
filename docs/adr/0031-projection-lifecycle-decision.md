# ADR 0031: Projection Lifecycle — StartProjections vs projectionhost vs CatchUpSubscriber

**Status:** Accepted — 2026-06-29 (updated: checkpoint-based replay shipped in StartProjections; CatchUpSubscriber migration deferred)
**Related:** [ADR 0016](0016-go-cqrs-lite-v3-migration.md) (projection rewrite), [ADR 0032](0032-basic-command-embedding.md) (command embedding), [go-cqrs-lite projectionhost/v3](https://github.com/LarsArtmann/go-cqrs-lite), [go-cqrs-lite watermill/v3 CatchUpSubscriber](https://github.com/LarsArtmann/go-cqrs-lite)

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

**Checkpoint-based replay shipped in v3.3.0. CatchUpSubscriber migration deferred to a future release.**

`StartProjections` now accepts an optional `event.CheckpointStore`. When provided AND the journal implements `event.SeekableJournal`, replay resumes from the last checkpoint via `ReadFrom(afterEventID, 0)` instead of `ReadAll()`. This eliminates the O(n) full-journal replay on every restart — the primary performance concern that motivated this ADR.

The synchronous replay model is preserved: read-your-writes consistency during startup is maintained because replay completes before `StartProjections` returns. The checkpoint is saved after each replayed event, so restarts resume from the exact position.

### The Core Tradeoff

| Feature                        | StartProjections (current)          | projectionhost   | CatchUpSubscriber |
| ------------------------------ | ----------------------------------- | ---------------- | ----------------- |
| Read-your-writes on startup    | **Yes** (synchronous)               | No (async)       | No (async)        |
| Checkpoint persistence         | **Yes** (when cpStore provided)     | **Yes**          | **Yes**           |
| Crash auto-restart             | No                                  | **Yes**          | No                |
| Dead-letter queue              | No                                  | **Yes**          | No                |
| Code complexity                | 155 LOC (hand-rolled)               | ~0 LOC (library) | ~10 LOC (library) |
| Full journal replay on restart | Only when cpStore is nil (backward) | No (O(delta))    | No (O(delta))     |

### Recommendation (for future release)

**CatchUpSubscriber is the better fit** for cqrs-htmx:

- Lighter weight than projectionhost (no worker goroutines, no DLQ complexity)
- Preserves the bus subscriber model that `BlockPublishUntilSubscriberAck` relies on
- Cursor-based replay (`ReadFrom`) is more efficient than `ReadAll` for large journals

The read-your-writes-during-startup gap can be solved with a **sync-wait wrapper**: after starting CatchUpSubscriber, block until the first live event is received (signaling replay is complete). This adds ~5 LOC.

**projectionhost is overkill** for cqrs-htmx's single-process deployment model. Its DLQ and crash-restart features shine in multi-process deployments where a projection handler might OOM. For a library that consumers embed in their own binary, the simpler CatchUpSubscriber is sufficient.

## Consequences

- **v3.3.0 (shipped):** Checkpoint-based replay is available via `EventSourcedConfig.CheckpointStore`. When nil, full journal replay is used (backward compatible). When set, replay resumes from the last checkpoint. Checkpoint saved after each replayed event.
- **Future (deferred):** Migrate from hand-rolled `StartProjections` to `watermill.CatchUpSubscriber` to eliminate the 155 LOC of replay logic entirely. Requires a sync-wait wrapper to preserve read-your-writes during startup. Blocked on evaluating whether CatchUpSubscriber's async replay model can be safely wrapped.
- **Long-term:** If a production deployment hits projection failure storms, upgrade to `projectionhost.Host` for DLQ + crash auto-restart.
