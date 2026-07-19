# TODO List — cqrs-htmx

**Updated:** 2026-07-19 | **Coverage:** 93.6% root, 79.9% usermgmt (CI gates: root 90%, usermgmt 78%) | **Lint:** 0 issues (usermgmt, totp, webauthn, oauth2, adminui, loginpage, integration_test); root has ~187 issues, mostly config noise (depguard allow-list excludes sibling larsartmann/* modules; canonicalheader flags intentional HX-* HTMX-header casing; err113 in tests) — see `docs/reviews/2026-07-19_05-03_code-quality-scan.html` | **Version:** v4.2.1+unreleased (go-cqrs-lite v4.0.1)

## Status Legend

- [ ] OPEN — actionable, not yet started
- [~] PARTIALLY DONE — started but incomplete

> Completed items live in [CHANGELOG.md](CHANGELOG.md). Deferred and rejected ideas live in [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## Open Items

### P3 — Technical Debt & Future

- [x] **Phase 2b — Persistent offline queue** (IndexedDB persistence per ADR-0040) — DONE: SharedWorker queue now persists command envelopes to IndexedDB; drains on spawn (cross-tab/cross-session retry via htmx.ajax); deletes on ACK; degrades gracefully to in-memory when IDB unavailable. ADR-0030 reversed (SUPERSEDED by ADR-0040).
- [x] **Snapshot integration** for high-event-volume aggregates (>10K events/aggregate) — DONE: `SnapshotConfig` (Store/Codec/Strategy) wired through every repository path (NewService, NewEventSourcedSetup, SQLite/Postgres setups). Includes `MemorySnapshotStore` + 4 tests. Zero behavior change when unconfigured.
- [x] **TypedRepository adoption** to eliminate command type assertions across all deciders — PREMISE INVALID: (1) zero command type assertions exist — `command.RegisterTyped[Cmd]` already gives fully-typed handlers (see `es_dispatch.go`); (2) `TypedDecider` binds ONE command type per repository, incompatible with usermgmt's multi-command aggregates (User has Register/ChangeEmail/AddRole/Suspend/...); (3) the current `repo.Execute(ctx, aggID, aggType, decideFn)` + per-command closure pattern is the correct, already-type-safe design for multi-command aggregates.
- [ ] **MySQL support** for event store (currently Postgres + SQLite only)
- [ ] **Property-based tests** for event fold functions (rapid/Hypothesis-style)
- [ ] **Load testing benchmarks** for SSE broadcaster under high fan-out
- [ ] **OpenAPI spec generation** for HTTP endpoints
- [ ] **Admin UI: OAuth2 link/unlink views**
- [ ] **Benchmark dedup.Ring vs old map** for typical journal sizes (100, 1K, 10K, 100K events)

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
