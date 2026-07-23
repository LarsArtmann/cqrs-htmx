# Book Insights Gap Closure Plan

> **Source:** `docs/reviews/book-insights-vs-cqrs-htmx.md`
> **Date:** 2026-07-23
> **Total tasks:** 27 (all <= 12 min each)
> **Estimated total:** ~4h 46m
> **Method:** Pareto breakdown (1% -> 51%, 4% -> 64%, 20% -> 80%), then sorted by impact/customer-value/effort within each tier.

---

## Scoring Methodology

Each task scored on three axes (1-5):

| Axis | Definition |
|------|-----------|
| **Impact** | How much does completing this improve the library? |
| **Customer Value** | How directly does this benefit a consumer building on cqrs-htmx? |
| **Effort** | Implementation time (5 = hardest). Lower effort + high impact = quick win. |

**Priority** = `(Impact * 0.35) + (Customer Value * 0.40) - (Effort * 0.25)`

Higher priority = do first.

---

## Pareto Tiers

### Tier 1: The 1% that delivers 51% (Documentation Quick Wins)

Pure documentation tasks. Zero code risk. Immediate consumer value. These prevent the most common consumer mistakes: false consistency assumptions, unhandled auth outages, unbounded storage growth, and not knowing replay already works.

| # | Task | Gap Area | Impact | Cust.Val | Effort | Priority | Est |
|---|------|----------|--------|----------|--------|----------|-----|
| 1 | Write consistency model doc: read-your-writes at startup (waitForDrain) | Consistency Model | 4 | 5 | 1 | **2.85** | 10m |
| 2 | Write consistency model doc: steady-state causal consistency per aggregate | Consistency Model | 4 | 5 | 1 | **2.85** | 10m |
| 3 | Write consistency model doc: what is NOT guaranteed (cross-projection atomicity, multi-instance) | Consistency Model | 4 | 5 | 1 | **2.85** | 8m |
| 4 | Document `projectionhost.Host.Reset()` as the rebuild/replay mechanism | Event Replay | 4 | 4 | 1 | **2.55** | 10m |
| 5 | Write auth provider fault tolerance guide (transient vs permanent, retry strategy) | Circuit Breaker | 3 | 4 | 2 | **2.15** | 12m |
| 6 | Write gobreaker integration example for wrapping OAuth2/WebAuthn providers | Circuit Breaker | 3 | 4 | 2 | **2.15** | 10m |
| 7 | Write Postgres event store compaction strategies (VACUUM, partitioning, archival) | Compaction | 3 | 4 | 2 | **2.15** | 12m |
| 8 | Write SQLite event store compaction strategies (VACUUM INTO, WAL checkpoint, size monitor) | Compaction | 3 | 4 | 2 | **2.15** | 10m |
| 9 | Write snapshot-then-archive workflow (snapshot aggregate, archive old events, restore procedure) | Compaction | 3 | 4 | 2 | **2.15** | 12m |
| 10 | Write retention patterns and storage growth monitoring guidance | Compaction | 2 | 3 | 1 | **1.85** | 10m |

### Tier 2: The 4% that delivers 64% (High-Impact Implementation)

Two features that consumers genuinely need: an event catalog (to build custom projections) and projection health monitoring (for production readiness). The projectionhost.Host already exposes everything needed — we just need HTTP handlers.

| # | Task | Gap Area | Impact | Cust.Val | Effort | Priority | Est |
|---|------|----------|--------|----------|--------|----------|-----|
| 11 | Create `EventCatalog` type in root module (generic registry of event types + metadata) | Event Catalog | 5 | 5 | 3 | **2.80** | 12m |
| 12 | Add `EventCatalog.Register()` + `EventCatalog.JSON()` methods | Event Catalog | 5 | 5 | 2 | **3.05** | 12m |
| 13 | Create `EventCatalogHandler(catalog)` with FNV-1a ETag + immutable cache + 304 | Event Catalog | 5 | 5 | 3 | **2.80** | 12m |
| 14 | Test EventCatalog register + JSON serialization | Event Catalog | 3 | 3 | 1 | **2.00** | 10m |
| 15 | Test EventCatalogHandler HTTP serving (200 body, 304 If-None-Match, ETag match) | Event Catalog | 3 | 3 | 1 | **2.00** | 10m |
| 16 | Register all 12 User events in EventCatalog (type, aggregate, payload schema, schema version) | Event Catalog | 4 | 5 | 2 | **2.70** | 12m |
| 17 | Register all 9 Membership/Tenant/Bot events + wire EventCatalog to Service/Setup | Event Catalog | 4 | 5 | 2 | **2.70** | 12m |
| 18 | Create `ProjectionStatusHandler` in root module (serves `[]WorkerState` as JSON) | Projection Health | 5 | 5 | 2 | **3.05** | 12m |
| 19 | Test ProjectionStatusHandler (empty, running, failed states) | Projection Health | 3 | 3 | 1 | **2.00** | 10m |
| 20 | Wire projection status accessor to EventSourcedSetup and Service | Projection Health | 4 | 5 | 2 | **2.70** | 10m |
| 21 | Document event catalog usage guide (what it is, how to serve, how to extend) | Event Catalog | 3 | 4 | 1 | **2.35** | 10m |
| 22 | Document projection health monitoring guide (endpoint, interpreting lag, alerting) | Projection Health | 3 | 4 | 1 | **2.35** | 10m |

### Tier 3: The 20% that delivers 80% (Wiring, Tests, Remaining)

Convenience methods, runbooks, and acknowledged debt tracking. These polish the rough edges and set up future work.

| # | Task | Gap Area | Impact | Cust.Val | Effort | Priority | Est |
|---|------|----------|--------|----------|--------|----------|-----|
| 23 | Add `RebuildProjection(name)` convenience method to Service wrapping `host.Reset()` | Event Replay | 4 | 4 | 2 | **2.40** | 12m |
| 24 | Test projection rebuild workflow (reset checkpoint, replay, verify read model) | Event Replay | 3 | 3 | 2 | **1.75** | 12m |
| 25 | Write rebuild-projection runbook (when to rebuild, how, verification steps) | Event Replay | 3 | 4 | 1 | **2.15** | 10m |
| 26 | Update ROADMAP.md with v5 usermgmt decomposition plan (module boundaries) | usermgmt Split | 2 | 2 | 1 | **1.25** | 10m |
| 27 | Document dependency tree analysis for extraction trigger (when to split modules) | usermgmt Split | 1 | 1 | 1 | **0.50** | 10m |

---

## Execution Order (Sorted by Priority, Then Pareto Tier)

Tasks below are sorted by priority score (highest first), which naturally surfaces quick wins before heavier implementation tasks.

| Seq | # | Task | Type | Tier | Priority | Est |
|-----|---|------|------|------|----------|-----|
| 1 | 12 | Add EventCatalog.Register() + JSON() methods | Impl | 2 | 3.05 | 12m |
| 2 | 18 | Create ProjectionStatusHandler in root module | Impl | 2 | 3.05 | 12m |
| 3 | 1 | Doc: read-your-writes consistency at startup | Doc | 1 | 2.85 | 10m |
| 4 | 2 | Doc: steady-state causal consistency | Doc | 1 | 2.85 | 10m |
| 5 | 3 | Doc: what is NOT guaranteed | Doc | 1 | 2.85 | 8m |
| 6 | 11 | Create EventCatalog type in root module | Impl | 2 | 2.80 | 12m |
| 7 | 13 | Create EventCatalogHandler with ETag + cache | Impl | 2 | 2.80 | 12m |
| 8 | 16 | Register all 12 User events in catalog | Wire | 2 | 2.70 | 12m |
| 9 | 17 | Register 9 Membership/Tenant/Bot events + wire to Service | Wire | 2 | 2.70 | 12m |
| 10 | 20 | Wire projection status accessor to Setup/Service | Wire | 2 | 2.70 | 10m |
| 11 | 4 | Document projectionhost.Host.Reset() as rebuild mechanism | Doc | 1 | 2.55 | 10m |
| 12 | 21 | Document event catalog usage guide | Doc | 2 | 2.35 | 10m |
| 13 | 22 | Document projection health monitoring guide | Doc | 2 | 2.35 | 10m |
| 14 | 23 | Add RebuildProjection(name) convenience method | Impl | 3 | 2.40 | 12m |
| 15 | 5 | Write auth provider fault tolerance guide | Doc | 1 | 2.15 | 12m |
| 16 | 6 | Write gobreaker integration example for auth providers | Doc | 1 | 2.15 | 10m |
| 17 | 7 | Write Postgres compaction strategies | Doc | 1 | 2.15 | 12m |
| 18 | 8 | Write SQLite compaction strategies | Doc | 1 | 2.15 | 10m |
| 19 | 9 | Write snapshot-then-archive workflow | Doc | 1 | 2.15 | 12m |
| 20 | 25 | Write rebuild-projection runbook | Doc | 3 | 2.15 | 10m |
| 21 | 14 | Test EventCatalog register + JSON serialization | Test | 2 | 2.00 | 10m |
| 22 | 15 | Test EventCatalogHandler HTTP serving | Test | 2 | 2.00 | 10m |
| 23 | 19 | Test ProjectionStatusHandler | Test | 2 | 2.00 | 10m |
| 24 | 10 | Write retention patterns + monitoring guidance | Doc | 1 | 1.85 | 10m |
| 25 | 24 | Test projection rebuild workflow | Test | 3 | 1.75 | 12m |
| 26 | 26 | Update ROADMAP with v5 usermgmt decomposition plan | Doc | 3 | 1.25 | 10m |
> **Total** | **27 tasks** | **~286m (~4h 46m)** |
| 28 | 27 | Document dep-tree analysis for module extraction trigger | Doc | 3 | 0.50 | 10m |

---

## Dependency Graph

```
Tier 1 (Docs — no dependencies, do in any order)
├── Tasks 1-3:  Consistency model docs
├── Task 4:     Replay/Reset documentation
├── Tasks 5-6:  Circuit breaker guide
├── Tasks 7-10: Compaction/retention docs
│
Tier 2 (Implementation — sequential within feature)
├── Event Catalog:
│   ├── 11 (type) → 12 (methods) → 13 (handler)
│   ├── 14, 15 (tests — after 12, 13)
│   ├── 16, 17 (registration — after 12)
│   └── 21 (doc — after 13, 17)
│
├── Projection Health:
│   ├── 18 (handler) → 19 (test)
│   ├── 20 (wiring — after 18)
│   └── 22 (doc — after 20)
│
Tier 3 (Polish — depends on Tier 2)
├── 23 (RebuildProjection — after understanding host.Reset)
├── 24 (test — after 23)
├── 25 (runbook — after 23, 24)
├── 26, 27 (roadmap — no deps)
└── 26, 27 (roadmap — no deps)
```

---

## Parallelization Strategy

Tasks with no dependencies can be done in parallel:

| Parallel Group | Tasks | Can Run Together |
|----------------|-------|-----------------|
| A | 1, 2, 3, 4 | All consistency/replay docs |
| B | 5, 6 | Circuit breaker docs |
| C | 7, 8, 9, 10 | Compaction docs |
| D | 11 + 18 | EventCatalog type + ProjectionStatusHandler (different files) |
| E | 14 + 19 | EventCatalog test + ProjectionStatus test (different files) |
| F | 16 + 17 | Sequential (same module, but different aggregate types) |
| G | 26, 27 | All roadmap updates |

---

## What This Plan Does NOT Include (Explicitly Out of Scope)

From `docs/reviews/book-insights-vs-cqrs-htmx.md` "What We Should NOT Do":

- Distributed consensus (Raft/Paxos) — consumer's DB job
- Message broker / Kafka — out of scope for a library
- Schema registry (Avro/Protobuf) — upcasters are sufficient
- Saga orchestration — choreography is correct for user management
- Data lake / analytics pipeline — OLTP, not OLAP
- 2PC / distributed transactions — eventually consistent is correct
- Built-in log compaction in Go code — DB handles its own storage
- CAP partition handling — consumer deployment concern
- Built-in HTTP router — framework-agnostic by design
- Generic Aggregate[T] framework — explicit domain code > over-abstraction

---

## Gap-to-Task Traceability

| Gap (from book insights review) | Tasks | Total Est |
|--------------------------------|-------|-----------|
| Projection lag observability | 18, 19, 20, 22 | 42m |
| Published event schema contract | 11, 12, 13, 14, 15, 16, 17, 21 | 88m |
| Event store compaction/retention | 7, 8, 9, 10 | 44m |
| Circuit breaker for auth | 5, 6 | 22m |
| Monotonic read / consistency model | 1, 2, 3 | 28m |
| Event replay tooling | 4, 23, 24, 25 | 42m |
| usermgmt decomposition (roadmap) | 26, 27 | 20m |
| **Total** | **27 tasks** | **~286m (~4h 46m)** |
