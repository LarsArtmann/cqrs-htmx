# ADR 0016: go-cqrs-lite v3.0.0 Migration — Projection Rewrite

**Date:** 2026-06-22
**Status:** Accepted
**Supersedes:** Projection startup approach from ADR 0006

## Context

go-cqrs-lite v3.0.0 deleted the `projection/` module (ADR-0030 upstream).
The `projection.Runner` — which handled replay, live tailing, dedup, retry,
and dead-letter queuing — is gone. Our `StartProjections()` function in
`es_projection_setup.go` depended entirely on it.

Three replacement options existed:

1. **`watermill.CatchUpSubscriber` + `stack.Materialize`** — the upstream
   recommended replacement. Requires Watermill Router setup, message adapter
   boilerplate, and rewriting read model internals to `kv.TypedStore`.

2. **`bus.SubscribeAll` only** — skip replay entirely. Only works for fresh
   deployments with empty event stores.

3. **Manual journal replay + `bus.SubscribeAll` + dedup** — hand-rolled
   ~130 LOC that preserves our existing read model architecture.

## Decision

**Option 3: Manual journal replay + `bus.SubscribeAll` + dedup.**

### Rationale

- **`stack.Materialize` is too heavy for a library.** It requires consumers to
  adopt `kv.TypedStore` and rewrites 6 read model internals. The value (typed
  materialized views with tombstone semantics) doesn't justify the cost for
  cqrs-htmx's in-memory default deployment.

- **`event.Projection` survived into v3.** The `Projection` interface
  (`Name() + Handle() + EventTypes()`) is in the `event` package itself. All
  6 read models already satisfy it — zero read model code changes needed.

- **Our replay needs are simple.** We need synchronous replay from the journal
  (read-your-writes), event-type-based routing, and replay→live dedup. That's
  ~130 LOC of straightforward Go.

- **`watermill.EventBus` preserves synchronous delivery.** GoChannel backend
  with `BlockPublishUntilSubscriberAck: true` means Publish blocks until the
  subscriber acks — identical semantics to the old `MemoryBus`.

### What we lost

- **Retry logic** — `projection.Runner` retried failed handlers with backoff.
  Our live handler logs errors but continues. Consumers needing retry can wrap
  their projection's `Handle` method.
- **Dead-letter queue** — failed events went to a DLQ for inspection. Dropped
  in favor of simplicity.
- **Checkpoint persistence** — the old code used `MemoryCheckpointStore`
  (in-memory, so restart-replay was already needed). Our manual replay always
  replays from scratch, which is equivalent for in-memory deployments.

### What we preserved

- **Read-your-writes consistency** — synchronous replay + synchronous bus
- **All 6 read models unchanged** — `event.Projection` interface survived
- **Dedup** — replay event IDs seed a `map[id.EventID]struct{}` checked in
  the live handler
- **Event-type routing** — `shouldDispatch()` checks each projection's
  `EventTypes()` before calling `Handle`

## Consequences

- `StartProjections` is now ~130 LOC of owned code instead of a dependency.
- No external `projection/` dependency — one less module in the graph.
- Future migration to `CatchUpSubscriber` + `Materialize` remains possible
  when persistent read models or multi-process deployment is needed.
- The dedup map is bounded by startup event count (never grows during live).
