# Status Report — Structural Hardening Session Complete

**Date:** 2026-06-29 06:39 CEST
**Session:** 11 commits — dep upgrade → bug fix → lint → release → structural hardening → checkpoint replay
**Latest tag:** v3.3.0 (more unreleased work since)
**Commits:** 856 total, 11 this session

---

## Executive Summary

What started as "upgrade dependencies" became a full release + structural hardening cycle. Shipped: go-cqrs-lite v3.4.0 upgrade across 8 modules, fixed a critical latent bug (7 commands returning zero cmdID), **structurally eliminated that bug class forever** via BasicCommand embedding, added checkpoint-based projection replay (no full journal replay on restart), cleared all lint debt to 0, tagged v3.3.0, wrote ADR-0031, proved scenario/v3 BDD DSL works for both happy and error paths. All CI gates green.

---

## a) FULLY DONE

| #   | Item                                   | Details                                                                                                                                                                                                                                                                 |
| --- | -------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **go-cqrs-lite v3.4.0 upgrade**        | All 8 modules upgraded to latest tags.                                                                                                                                                                                                                                  |
| 2   | **cmdID bug fix (initial)**            | 7 constructors patched to call `id.NewCommandID()`.                                                                                                                                                                                                                     |
| 3   | **cmdID bug fix (structural)**         | All 20 commands now embed `*command.BasicCommand` — ID/Type/AggregateID are promoted methods, impossible to forget. ~120 lines of boilerplate removed. Constructor signatures unchanged.                                                                                |
| 4   | **cmdID regression test**              | Table-driven test asserting all 20 constructors produce non-zero `ID()`.                                                                                                                                                                                                |
| 5   | **Lint debt cleared**                  | `sse_event.go` (gochecknoglobals + wrapcheck), usermgmt (exhaustruct = the cmdID bug). **All 3 modules: 0 issues.**                                                                                                                                                     |
| 6   | **v3.3.0 tagged + pushed**             | Formal CHANGELOG, ROADMAP, AGENTS.md, TODO_LIST alignment.                                                                                                                                                                                                              |
| 7   | **Server-Timing middleware**           | W3C Server-Timing header support (from another agent's work, committed + pushed).                                                                                                                                                                                       |
| 8   | **Checkpoint-based projection replay** | `StartProjections` gains optional `CheckpointStore`. When set + journal seekable, uses `ReadFrom(checkpoint)` instead of `ReadAll()`. Backward-compatible (nil = full replay). Wired through ServiceConfig, EventSourcedConfig, SQLiteSetupConfig, PostgresSetupConfig. |
| 9   | **ADR-0031**                           | Projection lifecycle decision (PROPOSED): recommends CatchUpSubscriber for v3.4.0.                                                                                                                                                                                      |
| 10  | **ADR index**                          | `docs/adr/INDEX.md` — all 31 ADRs with status badges.                                                                                                                                                                                                                   |
| 11  | **scenario/v3 spike (complete)**       | Both happy-path (`Then`) and error-path (`ThenError`) proven. Error-path uses `event.NewConflict(code, "")` as target — `*Error.Is()` matches by code+family.                                                                                                           |
| 12  | **Pareto execution plans**             | Two plans in `docs/planning/` with mermaid graphs.                                                                                                                                                                                                                      |

### Verification Snapshot

| Metric              | Value                                                |
| ------------------- | ---------------------------------------------------- |
| Build               | All 8 modules: OK                                    |
| Tests (race)        | root + usermgmt + adminui + integration_test: all OK |
| Lint                | root: 0, usermgmt: 0, adminui: 0                     |
| errorfamily         | root: pass, usermgmt: pass                           |
| Coverage (root)     | 93.8%                                                |
| Coverage (usermgmt) | 79.3%                                                |
| Test functions      | 739 (root 126, usermgmt 578, adminui 35)             |
| ADRs                | 31                                                   |
| Commits             | 856 total, 11 this session                           |

### Session Commits

```
6f092d7 feat(usermgmt): checkpoint-based projection replay + Server-Timing Config
9c48db0 refactor(server-timing): adopt nil-receiver pattern, remove enabled field
95630d5 refactor(usermgmt): embed command.BasicCommand in all 20 commands
c1ef807 feat: Server-Timing middleware + bump submodule deps to v3.3.0
c385db3 docs(status): v3.3.0 released — comprehensive status report
d509483 docs(adr): ADR-0031 projection lifecycle + scenario spike + ADR index
d595466 feat(v3.3.0): regression test, changelog, doc alignment
41f1293 docs(status): v3.4.0 upgrade session — comprehensive status report
3c177b0 fix: resolve 4 lint warnings in SSE delegation layer
73b46a2 fix(usermgmt): mint command IDs in 7 constructors that returned zero
aace1d1 chore: upgrade go-cqrs-lite to v3.4.0 across all 8 modules
```

---

## b) PARTIALLY DONE

| #   | Item                                        | Status                                                                                   | Gap                                                                                                                            |
| --- | ------------------------------------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **scenario/v3 BDD adoption**                | Spike proven for both paths.                                                             | Not rolled out across all 578 usermgmt tests. Needs systematic conversion.                                                     |
| 2   | **Checkpoint replay adoption**              | Implementation done, wired through all config structs.                                   | No SQL-backed CheckpointStore implementation yet — consumers must bring their own (or use `storage.NewMemoryCheckpointStore`). |
| 3   | **Offline-first Phase 2**                   | Phase 2a shipped.                                                                        | Phase 2b (IndexedDB, ADR-0030) proposed, not implemented.                                                                      |
| 4   | **CatchUpSubscriber adoption**              | ADR-0031 written with recommendation. Checkpoint replay partially solves the perf issue. | Full adoption (replacing StartProjections with CatchUpSubscriber) deferred — the sync-wait problem remains.                    |
| 5   | **Untracked `server_timing_bench_test.go`** | New file appeared from another agent.                                                    | Not committed — leaving for user to decide.                                                                                    |

---

## c) NOT STARTED

| #   | Item                              | Why It Matters                                                                                                                                                            |
| --- | --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **SQL-backed CheckpointStore**    | Persistent checkpoint survives restarts. Currently no implementation in usermgmt — consumers must use upstream `storage.NewMemoryCheckpointStore` or implement their own. |
| 2   | **scheduling/v3 for eviction.go** | Replace hand-rolled TTL sweep goroutine with durable Scheduler.                                                                                                           |
| 3   | **kv.Cache decorator**            | In-process cache for read models.                                                                                                                                         |
| 4   | **Multi-DB split in presets**     | I/O isolation for production.                                                                                                                                             |
| 5   | **prometheus/v3 metrics**         | CQRS dispatch observability.                                                                                                                                              |
| 6   | **graph/v3 read model tier**      | Nodes + edges for traversal-heavy models.                                                                                                                                 |
| 7   | **deriver/v3**                    | Stateless saga pattern.                                                                                                                                                   |
| 8   | **transport/grpc**                | gRPC dispatch.                                                                                                                                                            |
| 9   | **Email branded type**            | Blocked by event serialization. Needs major version + upcaster.                                                                                                           |
| 10  | **Full scenario/v3 rollout**      | Convert all 578 usermgmt tests to BDD DSL.                                                                                                                                |

---

## d) TOTALLY FUCKED UP

| #   | Issue                         | Severity | Status                                                                                                                                                                                                              |
| --- | ----------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **`go mod tidy` broken**      | Medium   | Pre-existing upstream bug (`eventtest@v0.0.0` local replace in go-cqrs-lite). Affects all consumers. Cannot fix from consumer side.                                                                                 |
| 2   | **Concurrent agent activity** | Low      | Another process is creating files (`server_timing.go`, `server_timing_bench_test.go`) during my sessions. Requiring careful investigation before each commit. Not breaking anything but adds coordination overhead. |
| 3   | **usermgmt coverage 79.3%**   | Low      | Down from 80.1% — the BasicCommand embedding removed some test-covered code paths (the old ID() methods). Still above the 75% gate but trending down.                                                               |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **SQL-backed CheckpointStore** — The checkpoint replay feature is wired but has no persistent store implementation. Add a SQL-backed store to usermgmt (like `SQLSessionStore`) so consumers get persistent checkpoints out of the box.

2. **Full CatchUpSubscriber adoption (v3.4.0)** — The checkpoint replay solves the "no full journal replay" problem but StartProjections is still 155+ LOC of hand-rolled code. Full adoption would eliminate this and add DLQ + crash restart.

3. **scenario/v3 systematic rollout** — The BDD DSL is proven. Converting usermgmt's 578 test functions in batches would dramatically improve test readability and reduce maintenance burden.

### Type Safety

4. **BasicCommand embedding is done** — The cmdID bug class is structurally eliminated. No further action needed here. This is a win.

### Testing

5. **usermgmt coverage gap** — The embedding removed some covered code paths. Need to add tests for the `mustCommand` helper and the panic path (zero aggID).

6. **OAuth2 real-provider integration test** — Still only mock-based.

### Developer Experience

7. **Concurrent agent coordination** — Multiple agents creating files in the same repo causes confusion. Need a protocol for agent-created files.

8. **TODO_LIST has 104 items, most done** — Needs triage to reduce noise.

---

## f) Top 25 Things We Should Get Done Next

| #   | Task                                                         | Impact    | Effort  |
| --- | ------------------------------------------------------------ | --------- | ------- |
| 1   | **SQL-backed CheckpointStore in usermgmt**                   | High      | 2 hours |
| 2   | **Investigate `server_timing_bench_test.go`**                | Low       | 5 min   |
| 3   | **Add mustCommand panic test**                               | Medium    | 15 min  |
| 4   | **Triage 104 TODO items**                                    | Medium    | 2 hours |
| 5   | **Roll out scenario/v3 to usermgmt decider tests (batch 1)** | High      | 1 day   |
| 6   | **Full CatchUpSubscriber adoption (v3.4.0)**                 | Very High | 2 days  |
| 7   | **Contribute eventtest fix upstream**                        | High      | 1 hour  |
| 8   | **scheduling/v3 for eviction.go**                            | Medium    | 1 day   |
| 9   | **kv.Cache decorator for read models**                       | Medium    | 2 hours |
| 10  | **Multi-DB split in presets**                                | Medium    | 3 hours |
| 11  | **prometheus/v3 metrics**                                    | Medium    | 2 hours |
| 12  | **Phase 2b IndexedDB persistence**                           | Medium    | 2 days  |
| 13  | **OAuth2 real-provider integration test**                    | Medium    | 3 hours |
| 14  | **Fuzz tests for OAuth2 state + PKCE**                       | Medium    | 2 hours |
| 15  | **v3.3.1 patch release (unreleased work)**                   | High      | 30 min  |
| 16  | **Write v3.3.x consumer migration guide**                    | Medium    | 2 hours |
| 17  | **Update FEATURES.md**                                       | Low       | 1 hour  |
| 18  | **transport/grpc dispatch**                                  | Low       | 1 day   |
| 19  | **graph/v3 for RBAC**                                        | Low       | 2 days  |
| 20  | **deriver/v3**                                               | Low       | 1 day   |
| 21  | **Email branded type + upcaster**                            | Low       | 1 day   |
| 22  | **Snapshot integration**                                     | Low       | 1 day   |
| 23  | **Update ROADMAP planned sections**                          | Low       | 30 min  |
| 24  | **Clean up docs/status/archive**                             | Low       | 30 min  |
| 25  | **Version-alignment CI check**                               | Low       | 1 hour  |

---

## g) Top #1 Question I Cannot Figure Out Myself

> **Should I commit `server_timing_bench_test.go`?**
>
> Another agent created this file (untracked) during my session. It's a benchmark test for the Server-Timing middleware. I've been leaving untracked files from other agents untouched, but this is the third occurrence and it creates friction.
>
> Should I: (a) review + commit agent-created files when they're good, (b) always leave them, or (c) something else?

---

_Generated 2026-06-29 06:39 CEST_
