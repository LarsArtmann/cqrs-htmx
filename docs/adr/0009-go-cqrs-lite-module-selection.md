# ADR 0009: go-cqrs-lite Module Selection

**Date:** 2026-06-17
**Status:** ACCEPTED

> **Updates since this ADR:**
>
> - **v4 migration:** All module paths moved from `/v2` to `/v4`.
> - **`projection/v2` deleted** in go-cqrs-lite v3.0.0 — replaced by manual
>   journal replay + `bus.SubscribeAll` (see ADR-0016).
> - **`memory` bus replaced** — `MemoryBus` replaced by `watermill.EventBus`
>   (GoChannel backend). `MemoryEventStore` is still used as the in-memory default.
> - **`catalog` merged upstream** — the `cqrs-htmx/catalog/` sub-package was
>   deleted and merged into `go-cqrs-lite/catalog/v4` (see ADR-0020).
> - **`watermill` now used directly** — the event bus is `watermill.EventBus`
>   with GoChannel backend.
> - **`storage` adopted** — usermgmt now uses `storage.SQLViewStore` for SQL-backed
>   read models and `storage.SQLEventStore` for the event store.
>
> The tables below reflect the original v2-era module set for historical context.

## Context

go-cqrs-lite consists of 24 library modules. cqrs-htmx uses 8 directly, 3 indirectly, and 13 are unused. This ADR documents the deliberate selection rationale for each module — what we use, what we don't, and why.

## Modules Used Directly (8)

| Module       | Purpose in cqrs-htmx                                                 |
| ------------ | -------------------------------------------------------------------- |
| `command/v4` | Command dispatch, `command.New()`, typed handlers                    |
| `event/v4`   | Event types, bus, context propagation, error re-export               |
| `query/v4`   | Query dispatch, `PaginatedResult`, pagination                        |
| `id/v4`      | `UserID`, `CorrelationID`, `RequestID`, `AggregateID` branded types  |
| `memory/v4`  | `MemoryEventStore` for event-sourced usermgmt (bus is now watermill) |
| `decider/v4` | Decider pattern, `Repository` for usermgmt aggregate                 |
| `codec/v4`   | Event serialization                                                  |
| `stack/v4`   | `Materialize`, SQL stack presets (replaces deleted `projection/v2`)  |

## Modules Used Indirectly (3)

| Module          | Pulled in by         |
| --------------- | -------------------- |
| `dispatcher/v4` | command/v4, query/v4 |
| `otel/v4`       | memory/v4, stack/v4  |
| `snapshot/v4`   | memory/v4            |

## Modules NOT Used (13)

### Storage & Transport Backends (correctly excluded)

| Module      | Why Not                                                                                   |
| ----------- | ----------------------------------------------------------------------------------------- |
| `storage`   | Persistence layer — consumer's choice (Postgres/SQLite). cqrs-htmx is transport-agnostic. |
| `pebble`    | Alternative event store backend. Consumer's choice.                                       |
| `turso`     | Turso/libSQL backend. Consumer's choice.                                                  |
| `kv`        | Key-value store abstraction. Consumer's choice.                                           |
| `watermill` | Event bus transport. **Now used** — `watermill.EventBus` (GoChannel) is the event bus.    |

**Rationale:** cqrs-htmx is an HTTP integration layer, not a persistence framework. Offering storage modules would force consumers into a persistence decision they should make themselves. The library principle: don't drag in infrastructure the consumer might disagree with.

### Security & Integrity (correctly excluded)

| Module       | Why Not                                                                                      |
| ------------ | -------------------------------------------------------------------------------------------- |
| `encryption` | Event payload encryption (AES-GCM/XChaCha20). Data-at-rest concern, invisible to HTTP layer. |
| `signing`    | Event signing (HMAC/Ed25519). Event-integrity layer, orthogonal to HTTP handling.            |

**Rationale:** These are data protection concerns at the event store layer. Consumers who need encryption or signing configure it at their event store, not at the HTTP integration layer.

### Application-Level Concerns (correctly excluded)

| Module       | Why Not                                                                                                                                                                                                                                                                           |
| ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `middleware` | CQRS dispatch middleware (retry, circuit breaker, metrics, tracing). Different layer than cqrs-htmx's HTTP middleware. Documented as a recommended integration in [`docs/integrations/go-cqrs-lite-middleware.md`](../integrations/go-cqrs-lite-middleware.md), not a dependency. |
| `schema`     | Event schema registry + upcasters. Event-store-layer concern; cqrs-htmx never touches the store read path.                                                                                                                                                                        |
| `snapshot`   | Aggregate snapshot strategies. Aggregate-load optimization; below cqrs-htmx's layer.                                                                                                                                                                                              |
| `listing`    | Aggregate listing read model with tombstone detection. Conflicts with existing `UserReadModel` projection pattern. Different read path (event store vs projection table).                                                                                                         |

### Tooling & Testing (correctly excluded)

| Module     | Why Not                                                                               |
| ---------- | ------------------------------------------------------------------------------------- |
| `testutil` | Test helpers for go-cqrs-lite's own test suite. Not useful for consumers.             |
| `catalog`  | API doc generation. **Merged upstream** into `go-cqrs-lite/catalog/v4`. See ADR-0020. |

## Decision

The 8/24 direct usage ratio is **intentional and correct**. It reflects clean separation of concerns:

- cqrs-htmx consumes the **CQRS core** (command/event/query/id) + **event-sourcing toolchain** (decider/memory/projection/codec)
- Everything below (storage, crypto, backends) is the consumer's choice
- Everything beside (middleware, catalog) is opt-in via documentation or separate modules

## Revisit Triggers

- If go-cqrs-lite adds a dispatcher enumeration API → consider auto-discovery for catalog
- If usermgmt adds admin "list users" → evaluate listing module vs extending UserReadModel
- If consumers request built-in retry/circuit-breaker → consider documenting middleware wiring more prominently
