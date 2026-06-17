# ADR 0009: go-cqrs-lite Module Selection

**Date:** 2026-06-17
**Status:** ACCEPTED

## Context

go-cqrs-lite consists of 24 library modules. cqrs-htmx uses 8 directly, 3 indirectly, and 13 are unused. This ADR documents the deliberate selection rationale for each module — what we use, what we don't, and why.

## Modules Used Directly (8)

| Module          | Purpose in cqrs-htmx                                                |
| --------------- | ------------------------------------------------------------------- |
| `command/v2`    | Command dispatch, `command.New()`, typed handlers                   |
| `event/v2`      | Event types, bus, context propagation, error re-export              |
| `query/v2`      | Query dispatch, `PaginatedResult`, pagination                       |
| `id/v2`         | `UserID`, `CorrelationID`, `RequestID`, `AggregateID` branded types |
| `memory/v2`     | `MemoryBus`, `MemoryEventStore` for event-sourced usermgmt          |
| `decider/v2`    | Decider pattern, `Repository` for usermgmt aggregate                |
| `codec/v2`      | Event serialization                                                 |
| `projection/v2` | `Runner` for CasbinProjection + UserReadModel                       |

## Modules Used Indirectly (3)

| Module          | Pulled in by             |
| --------------- | ------------------------ |
| `dispatcher/v2` | command/v2, query/v2     |
| `otel/v2`       | memory/v2, projection/v2 |
| `snapshot/v2`   | memory/v2                |

## Modules NOT Used (13)

### Storage & Transport Backends (correctly excluded)

| Module      | Why Not                                                                                   |
| ----------- | ----------------------------------------------------------------------------------------- |
| `storage`   | Persistence layer — consumer's choice (Postgres/SQLite). cqrs-htmx is transport-agnostic. |
| `pebble`    | Alternative event store backend. Consumer's choice.                                       |
| `turso`     | Turso/libSQL backend. Consumer's choice.                                                  |
| `kv`        | Key-value store abstraction. Consumer's choice.                                           |
| `watermill` | Watermill message broker bridge. Consumer's choice.                                       |

**Rationale:** cqrs-htmx is an HTTP integration layer, not a persistence framework. Offering storage modules would force consumers into a persistence decision they should make themselves. The library principle: don't drag in infrastructure the consumer might disagree with.

### Security & Integrity (correctly excluded)

| Module       | Why Not                                                                                      |
| ------------ | -------------------------------------------------------------------------------------------- |
| `encryption` | Event payload encryption (AES-GCM/XChaCha20). Data-at-rest concern, invisible to HTTP layer. |
| `signing`    | Event signing (HMAC/Ed25519). Event-integrity layer, orthogonal to HTTP handling.            |

**Rationale:** These are data protection concerns at the event store layer. Consumers who need encryption or signing configure it at their event store, not at the HTTP integration layer.

### Application-Level Concerns (correctly excluded)

| Module       | Why Not                                                                                                                                                                           |
| ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `middleware` | CQRS dispatch middleware (retry, circuit breaker, metrics, tracing). Different layer than cqrs-htmx's HTTP middleware. Documented as a recommended integration, not a dependency. |
| `schema`     | Event schema registry + upcasters. Event-store-layer concern; cqrs-htmx never touches the store read path.                                                                        |
| `snapshot`   | Aggregate snapshot strategies. Aggregate-load optimization; below cqrs-htmx's layer.                                                                                              |
| `listing`    | Aggregate listing read model with tombstone detection. Conflicts with existing `UserReadModel` projection pattern. Different read path (event store vs projection table).         |

### Tooling & Testing (correctly excluded)

| Module     | Why Not                                                                                             |
| ---------- | --------------------------------------------------------------------------------------------------- |
| `testutil` | Test helpers for go-cqrs-lite's own test suite. Not useful for consumers.                           |
| `catalog`  | API doc generation. **NOW INTEGRATED** as a separate opt-in sub-package (`catalog/`). See ADR 0008. |

## Decision

The 8/24 direct usage ratio is **intentional and correct**. It reflects clean separation of concerns:

- cqrs-htmx consumes the **CQRS core** (command/event/query/id) + **event-sourcing toolchain** (decider/memory/projection/codec)
- Everything below (storage, crypto, backends) is the consumer's choice
- Everything beside (middleware, catalog) is opt-in via documentation or separate modules

## Revisit Triggers

- If go-cqrs-lite adds a dispatcher enumeration API → consider auto-discovery for catalog
- If usermgmt adds admin "list users" → evaluate listing module vs extending UserReadModel
- If consumers request built-in retry/circuit-breaker → consider documenting middleware wiring more prominently
