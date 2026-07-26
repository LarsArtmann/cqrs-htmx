# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-07-26 | **Version:** v4.6.0 (15 modules in `go.work`; see AGENTS.md for per-sub-module versions) | **Coverage:** Root 93.5% (gate 90%), usermgmt 80.9% (gate 74%), identity-model ~41% (no gate) — recompute via `nix run .#coverage-gate` | **Lint:** `nix run .#lint` fails on root (~610, varnamelen-dominated), usermgmt (~330, SA1019-dominated), and dashboardui (~150) on pre-existing style nits + `id.*AggregateID` SA1019 deprecation; the other 7 modules pass clean. Non-release-blocking. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P2 — Quality Gates

- [~] **identity-model test coverage.** ~41% coverage (2 test files for 25 source files). No coverage-gate threshold defined for this module in `flake.nix`. Needs tests for the Authz engine, command constructors, event payload round-trips, crypto helpers, and the remaining fold functions (`foldMembership`, `foldBot`). Source: `docs/status/2026-07-23_21-27_identity-model-consolidation-brutal-review.md`.
- [~] **dashboardui test coverage.** SSE reconnect replay, initial backfill, heartbeat emission, and Close lifecycle now tested (`sse_replay_test.go`, 4 tests; 16 tests total across 2 files). Remaining: handler-level tests and payload-rendering tests.
- [ ] **dashboardui `handlers.go` split.** Single file at **1158 lines** — split per domain (overview, events, aggregates, projections, DLQ, audit, snapshots, time-travel).
- [ ] **Triage lint nits.** Root ~610 (varnamelen 405, exhaustruct 61, staticcheck 37, errcheck 27, canonicalheader 24), usermgmt ~330 (staticcheck SA1019 271 — `id.NewAggregateID` deprecation), dashboardui ~150. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module. Non-release-blocking.

---

## P3 — Technical Debt & Future

- [ ] **MySQL event-store support.** Currently Postgres + SQLite only (dropped in v3.0.0 when `SQLEventStore` was delegated to `go-cqrs-lite/storage`, which has no MySQL dialect). `SQLSessionStore` already supports MySQL.
- [ ] **Offline sync E2E browser testing.** The SharedWorker IndexedDB queue (`sync/sync-worker.js`) is unit-verified but never exercised in a real browser. Needs a Playwright E2E test verifying the queue → retry → ACK cycle and the cross-session `rebuildAndRetry` path. The #1 deferred item across every sync status report since ADR-0029.
- [ ] **Migrate `id.NewAggregateID` → `id.NewStreamID`** across usermgmt. go-cqrs-lite v4 renamed the API; backward-compatible aliases exist, but the migration aligns with upstream naming. Production footprint is small — **2 call sites** in non-test usermgmt code (`import_export.go:155`, `service_oauth2.go:139`; ~8 repo-wide) — the larger ~121-figure count is test fixtures, which can follow once the production sites move.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
