# Book Insights vs. cqrs-htmx: Applied, Missing, and Avoid

> Analysis mapping insights from seven architecture books to the cqrs-htmx codebase.
>
> **Books covered:**
> - Designing Data-Intensive Applications (Kleppmann)
> - Deciphering Data Architectures (Serra)
> - Exploring CQRS and Event Sourcing
> - Implementing DDD, CQRS and Event Sourcing
> - Designing Event-Driven Systems (Stopford)
> - Patterns of Distributed Systems (Joshi)
> - Service Design Patterns (Daigneau)
>
> **Date:** 2026-07-23

---

## Table of Contents

- [What We Already Apply Well](#what-we-already-apply-well)
  - [From DDIA (Kleppmann)](#from-ddia-kleppmann)
  - [From DDD/CQRS/ES Books (Vernon, Exploring CQRS)](#from-dddcqrses-books-vernon-exploring-cqrs)
  - [From Designing Event-Driven Systems (Stopford)](#from-designing-event-driven-systems-stopford)
  - [From Patterns of Distributed Systems (Joshi)](#from-patterns-of-distributed-systems-joshi)
  - [From Service Design Patterns (Daigneau)](#from-service-design-patterns-daigneau)
  - [From Deciphering Data Architectures (Serra)](#from-deciphering-data-architectures-serra)
- [What We Should Apply (Gaps Worth Closing)](#what-we-should-apply-gaps-worth-closing)
  - [High Impact](#high-impact)
  - [Medium Impact](#medium-impact)
- [What We Should NOT Do](#what-we-should-not-do)
- [Summary Matrix](#summary-matrix)

---

## What We Already Apply Well

### From DDIA (Kleppmann)

| Principle | Where in cqrs-htmx | Evidence |
|---|---|---|
| **Append-only log as universal primitive** | Event store = the WAL pattern | `SQLEventStore` delegates to go-cqrs-lite; `event.Store` is append-only; aggregates are rebuilt by replay |
| **Derive state, don't dual-write** | Projections are materialized views | `UserReadModel`, `CasbinProjection`, `MembershipReadModel` are all derived from the single event stream. The event store is the only write target. |
| **Read-your-writes consistency** | `waitForDrain` in `es_projection_setup.go:131` | Blocks `StartProjections` until all projections drain the journal — guarantees queries work immediately after startup |
| **Idempotency over exactly-once** | Three layers: checkpoints, idempotency store, optimistic concurrency | `projectionhost.Host` uses per-projection checkpoints keyed by `Name()`. `IdempotencyStore.CheckAndRecord` is atomic (avoids TOCTOU). Event store expected-version prevents duplicate events. |
| **Strong types make impossible states unrepresentable** | `authMode` enum, branded IDs, `HTTPStatusCarrier` | `authNone`/`authRequired`/`authAuthorized` is a three-state type, not a bool. `brandid.ID` prevents mixing UserID with TenantID. `StructuredError` has status/code/detail that can't be partially set. |
| **Schema evolution is non-optional** | Upcaster registry (ADR-0013) | `UpcasterRegistry` chains `v0->v1->v2` transformations. Every payload has `schema_version`. Old events replay against current code. Codec-agnostic (works for JSON and CBOR). |

### From DDD/CQRS/ES Books (Vernon, Exploring CQRS)

| Principle | Where in cqrs-htmx | Evidence |
|---|---|---|
| **Bounded contexts** | 12 separate Go modules with enforced isolation | `scripts/check-module_isolation.sh` builds each module standalone with `GOWORK=off`. Root <- usermgmt is zero imports. Auth strategies use structural typing. |
| **Small aggregates, references by ID** | User, Membership, Tenant, Bot — 4 small aggregates | Membership references `UserID`, not a `*User` object. Each aggregate has 2-12 event types — no god aggregates. |
| **Decider pattern: aggregate decides events** | `UserDecider()` in `es_setup.go:102` | Command handlers load state -> call decider function -> decider validates invariants -> returns events. The command handler never decides what events to emit. |
| **Past-tense domain events** | All event types named correctly | `UserRegistered`, `MemberAdded`, `TenantSuspended`, `BotDeleted` — never `RegisterUser` or `CreateBot`. |
| **Value objects preferred** | Branded IDs (ADR-0028), `authMode`, `ActorID` kind-discriminated struct | `type UserID = id.UserID` — compile-time type safety, no string mixing. |
| **Context mapping: Anti-Corruption Layer** | Root module duck-types usermgmt concepts | Root defines `Enforcer` interface (satisfied by casbin without importing it), `TemplComponent` (satisfied by templ without importing it). Auth strategy modules satisfy usermgmt interfaces without importing usermgmt. |
| **Saga-free choreography** | Projections react independently | No central coordinator. `CasbinProjection` derives policies from events without any orchestration logic. |

### From Designing Event-Driven Systems (Stopford)

| Principle | Where in cqrs-htmx | Evidence |
|---|---|---|
| **Dead Letter Queue** | `projectionhost.WithDeadLetterStore` | Poison events move to DLQ after exhausting retries. Crash auto-restart with backoff. |
| **Event-carried state transfer** | Event payloads include full data | `UserRegistered` carries email, displayName — not just a reference. Projections don't need to look up the aggregate to build read models. |
| **Idempotent receivers** | Checkpoint-based replay | Projections resume from last checkpoint. Duplicate events at the same version are rejected by the store. |

### From Patterns of Distributed Systems (Joshi)

| Pattern | Where in cqrs-htmx |
|---|---|
| **Write-Ahead Log** | The event store IS a WAL |
| **WAL + Snapshotting** | `SnapshotConfig` (ADR-0041) — `Store` + `Codec` + `Strategy`. `LoadFromVersion` reads only the tail after snapshot. |
| **Segmented Log** | Delegated to go-cqrs-lite SQL storage |
| **Idempotent Receiver** | Per-projection checkpoints |
| **High-Water Mark** | Checkpoint keys per projection `Name()` |

### From Service Design Patterns (Daigneau)

| Principle | Where in cqrs-htmx |
|---|---|
| **Non-CRUDy services** | `RegisterUser`, `AddMember`, `SuspendTenant` — not `CreateUser`, `UpdateUser` |
| **Service facade** | `App` struct hides dispatcher/enforcer wiring |
| **Request/Response mapping** | Three error handlers (default, JSON, Problem Details RFC 7807) — separate concern from business logic |

### From Deciphering Data Architectures (Serra)

| Principle | Where in cqrs-htmx |
|---|---|
| **Data as a product** | Each module (root, usermgmt, adminui, loginpage) is an independently versioned Go module with its own `go.mod` — the library equivalent of domain-owned data products |
| **Event log as data product** | The event store is a first-class, consumable artifact. Any consumer can subscribe via `EventBus` or replay via `SeekableJournal` to build their own projections. |
| **Lakehouse concept: one platform, multiple access patterns** | The single event stream feeds OLTP (command side), read models (query side), Casbin policies (authz side), and audit log — all derived from one source, accessed differently. This is the CQRS analog of "one lake, many query engines." |

---

## What We Should Apply (Gaps Worth Closing)

### High Impact

**1. No projection lag observability** *(DDIA: "measure replication lag")*

There's no built-in metric for "how far behind is this projection from the event log head?" `Server-Timing` headers exist for HTTP but not for projection health. Consumers flying blind on projection catch-up status.

**Recommendation:** Expose `projectionHost.Status()` as a Prometheus-compatible metric or at minimum a `/health/projections` endpoint that reports each projection's last-processed event version vs. the journal head.

**2. No published event schema contract** *(DDD: Published Language; DDIA: Encoding & Evolution)*

The `openapi/` package documents HTTP endpoints — but events are ALSO part of the public API surface for any consumer building custom projections. There's no `EventCatalog` or schema documentation for the 21 event types across 4 aggregates.

**Recommendation:** Generate an event catalog (similar to `OpenAPISpecHandler` but for events). Could be as simple as a Go struct registry that produces a JSON schema or markdown table of all event types + their payload shapes + current schema versions. Consumers building projections need this.

**3. No event store compaction/retention guidance** *(DDIA: Log Compaction; Joshi: Segmented Log cleanup)*

Events accumulate forever. Snapshots help read performance but don't reduce storage. For long-running production systems, this is a growth problem that consumers will hit and not know how to solve.

**Recommendation:** Document retention/compaction strategies. Even if the library doesn't implement it (the SQL backend handles its own storage), provide guidance: "After snapshotting, events older than the snapshot version can be archived to cold storage." Or provide a `Compact(aggregateID, upToVersion)` method that delegates to the store.

**4. No circuit breaker for auth providers** *(DDIA: Transient faults; Stopford: fault tolerance)*

OAuth2, WebAuthn, TOTP call external systems. If Google's OAuth endpoint is down, what happens? The library returns a Transient error (503) — but there's no circuit breaker to fail fast instead of queuing retries against a dead provider.

**Recommendation:** Consider an opt-in circuit breaker wrapper for auth provider calls. Even documentation saying "wrap your auth provider calls with [gobreaker/sony/gobreaker]" would help. The library correctly doesn't force this, but consumers need to know it's their responsibility.

### Medium Impact

**5. No monotonic read guarantee documented** *(DDIA: Replication lag problems)*

Read-your-writes is solved via `waitForDrain` at startup. But there's no mechanism preventing a user from seeing a newer state, then an older state (e.g., if projections are rebuilt or if there's a race). For a single-process system this is unlikely, but the guarantee should be documented.

**Recommendation:** Document the exact consistency model: "This library provides read-your-writes consistency at startup via drain. During steady-state operation, all projections process events in order per aggregate, ensuring causal consistency within a single process."

**6. No event replay tooling** *(DDIA: Event sourcing benefit — temporal queries)*

The whole point of event sourcing is the ability to replay events to rebuild state. The library has the machinery (`SeekableJournal`, checkpoints, projections), but no `ReplayFrom(timestamp)` or `RebuildProjection(name)` API for consumers.

**Recommendation:** Expose a `RebuildProjection(ctx, projectionName, fromVersion)` method on `projectionhost.Host`. This is the killer feature of event sourcing that's currently implicit in the machinery.

**7. `usermgmt` god-package decomposition deferred** *(DDD: module boundaries)*

ADR-0019 (Blocked) and ADR-0038 (Proposed, deferred to v5) acknowledge that usermgmt is a god-package. The aggregates are clean, but they all live in one module with shared infrastructure. This violates the bounded context principle — User, Membership, Tenant, Bot could be separate contexts.

**Recommendation:** This is correctly deferred to v5. The current state works. But it should stay on the roadmap as an acknowledged debt.

---

## What We Should NOT Do

### Architectural Anti-Patterns for This Library

**1. Do NOT add distributed consensus (Raft/Paxos)**

The library runs against a single database instance. Adding leader election or multi-leader replication would massively increase complexity for zero benefit to a library. This is the consumer's database's job. The event store interface correctly abstracts storage — let Postgres/SQLite handle their own replication.

**2. Do NOT build a message broker / Kafka competitor**

The in-process `EventBus` (Watermill) is sufficient. Adding cross-process messaging, partitioning, or consumer groups would turn this into infrastructure, not a library. If consumers need Kafka, they should use Kafka and adapt the `event.Store`/`event.Bus` interfaces.

**3. Do NOT force a schema registry (Avro/Protobuf)**

The upcaster approach is pragmatic and Go-idiomatic. Forcing Avro would add a JVM dependency. Forcing Protobuf would add codegen. JSON with `schema_version` + upcasters handles 95% of cases. The library correctly uses `encoding/json/v2` and stays schema-format-agnostic.

**4. Do NOT implement saga orchestration**

Choreography (projections react to events independently) is the right model for user management. The domain doesn't have complex multi-step workflows (like order processing with payment + shipping + inventory). Adding a saga/process manager engine would be over-engineering. The `MigrateRolesToMemberships` function shows the pattern for cross-aggregate coordination without a saga.

**5. Do NOT build a data lake / lakehouse / data mesh**

This is an OLTP system. The read models are already denormalized projections. Don't add analytics pipelines, ETL, or data products. If consumers need analytics, they should subscribe to the event stream and build their own analytics pipeline.

**6. Do NOT implement 2PC or distributed transactions**

Eventually consistent is correct for projections. Forcing 2PC across aggregates or projections would kill availability and add enormous complexity. The expected-version optimistic concurrency on the event store is the right consistency boundary.

**7. Do NOT add built-in log compaction in the library**

The SQL backend handles its own storage management. Postgres has VACUUM, SQLite has its own mechanisms. Adding compaction logic in the library would conflict with the database's own storage management. Document strategies instead.

**8. Do NOT implement CAP-theorem partition handling**

The library assumes a single connected database. Network partition handling is the database's (or the consumer's deployment's) responsibility. Don't add partition detection, stale-read prevention, or quorum logic.

**9. Do NOT add a built-in HTTP router**

The library is framework-agnostic (works with net/http, Gin, Chi). ADRs and ROADMAP.md correctly reject this. Adding routing would couple consumers to a specific framework.

**10. Do NOT over-abstract aggregates into a generic framework**

The current approach — explicit `UserState`, `MembershipState`, etc. with hand-written `foldX` functions — is more readable than a generic `Aggregate[T]` framework. The decider pattern provides enough abstraction without hiding domain logic behind generics. Don't refactor toward `Aggregate[T]` or `EventSourcedRepository[T, ID]` unless there's a concrete pain point.

---

## Summary Matrix

| Book Insight | Status | Action |
|---|---|---|
| Append-only log as truth | Applied | — |
| Projections as derived state | Applied | — |
| Idempotency (checkpoints + store) | Applied | — |
| Schema evolution (upcasters) | Applied | — |
| Bounded contexts (modules) | Applied | Decompose usermgmt in v5 |
| Small aggregates | Applied | — |
| Read-your-writes | Applied | Document steady-state guarantees |
| DLQ + retry | Applied | — |
| Snapshotting | Applied | — |
| Projection lag observability | Missing | Add health/metrics endpoint |
| Published event schemas | Missing | Generate event catalog |
| Compaction/retention guidance | Missing | Document strategies |
| Circuit breaker for auth | Missing | Document consumer responsibility |
| Replay tooling | Missing | Expose `RebuildProjection` |
| Distributed consensus | Don't | Consumer's DB handles this |
| Message broker | Don't | Out of scope for a library |
| Schema registry | Don't | Upcasters are sufficient |
| Saga orchestration | Don't | Choreography is correct here |
| Data lake/analytics | Don't | OLTP, not OLAP |
| 2PC / distributed transactions | Don't | Eventually consistent is right |
| Built-in router | Don't | Framework-agnostic is correct |
| Generic aggregate framework | Don't | Explicit > abstract for domain logic |
