# CatchUpSubscriber Evaluation

> Should we replace the manual journal replay in `StartProjections` with
> `watermill.CatchUpSubscriber`?

**Date:** 2026-06-28 · **Status:** Evaluated — Defer

## Current Approach

`es_projection_setup.go` uses manual journal replay + `bus.SubscribeAll` + dedup:

1. On startup, read ALL events from the journal via `journal.ReadAll()`
2. Feed each event to matching projections (with dedup via `id.EventID` map)
3. After replay completes, subscribe to live events via `bus.SubscribeAll`
4. The live handler also dedups (in case the bus had buffered events during replay)

This is ~130 LOC. It works correctly and is well-tested.

## What CatchUpSubscriber Provides

`watermill.CatchUpSubscriber` is a generic subscriber that:
- Reads from a seekable source (journal) from a given offset
- Then subscribes to live events
- Handles the journal→live transition internally

## Evaluation

### Pros of Adoption
- Less custom code (~40-50 LOC saved)
- Battle-tested transition logic
- Built-in offset tracking

### Cons of Adoption
- **Requires watermill Router setup** — adds boilerplate (Router, SubscriberAdapter, MessageHandler)
- **Changes the message model** — watermill uses `message.Message` wrappers; our projections expect `event.Event` directly. Adapter needed.
- **Our read models don't fit `kv.TypedStore`** — the CatchUpSubscriber is designed for `kv.TypedStore`-backed projections. Our read models have complex event handling (12-event switches, external account indexes) that don't fit the declarative pattern.
- **Dedup is already solved** — our `id.EventID` map works correctly. CatchUpSubscriber's offset-based dedup is not better.
- **Testing investment** — our existing tests (3 in es_projection_setup_test.go) would need rewriting.

### Verdict: Defer

The manual replay is correct, tested, and simple. The migration cost (adapters, router setup, test rewrites) exceeds the benefit (~50 LOC saved). Revisit when:
- We need offset-based resumable replay (e.g., after crash)
- We adopt `kv.TypedStore` for read models (ADR 0019 unblocks)
- A production deployment reports replay performance issues

## Related

- [ADR 0016: go-cqrs-lite v3.0.0 Migration](../adr/0016-go-cqrs-lite-v3-migration.md) — documents the original decision
