# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-07-26 | **Version:** v4.5.0 (15 modules in `go.work`; see AGENTS.md for per-sub-module versions) | **Coverage:** Root 93.5% (gate 90%), usermgmt 81.0% (gate 74%), identity-model ~41% (no gate) — recompute via `nix run .#coverage-gate` | **Lint:** 0 issues across all linted submodules. Root carries ~80 pre-existing low-severity style nits (varnamelen ×50, staticcheck SA1019 ×18, testpackage ×9, …) — not gate-blocking.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P1 — Release Hygiene

- [ ] **Fix `examples/dashboard-demo/go.mod` zero pseudo-version.** It still requires `github.com/larsartmann/cqrs-htmx/dashboardui/v4 v4.0.0-00010101000000-000000000000` (broken placeholder) instead of `v4.0.0`. This is the **only** remaining inter-module version-ref defect — every other submodule (`usermgmt`, `adminui`, `loginpage`, `identity-model`) was resolved to clean tags (`v4.5.0` / `identity-model/v4.1.0`) in commit `e274540`. Workspace builds are unaffected (go.work local replaces), but `GOWORK=off` resolution for the demo breaks. One-line fix.

---

## P2 — Quality Gates

- [~] **identity-model test coverage.** ~41% coverage (2 test files for 25 source files). No coverage-gate threshold defined for this module in `flake.nix`. Needs tests for the Authz engine, command constructors, event payload round-trips, crypto helpers, and the remaining fold functions (`foldMembership`, `foldBot`). Source: `docs/status/2026-07-23_21-27_identity-model-consolidation-brutal-review.md`.
- [ ] **dashboardui dead code cleanup.** `notImplemented()` at `handlers.go:21` and `renderStatCardsTempl()` at `templ_render.go:25` have **zero callers** (verified). Delete them. FEATURE is honestly marked 🔴 `BROKEN` until cleaned.
- [~] **dashboardui test coverage.** 1 test file (`dashboard_test.go`, 12 tests) for 12 source files. Needs handler-level tests, an SSE bridge test, and payload-rendering tests.
- [ ] **dashboardui `handlers.go` split.** Single file at **1167 lines** — split per domain (overview, events, aggregates, projections, DLQ, audit, snapshots, time-travel).
- [ ] **dashboardui SSE reconnect replay.** The SSE bridge now emits domain event IDs and has heartbeat config (`SSEHeartbeatInterval`), but it does **not** construct a journal-backed store or call `ReplayEvents` for `Last-Event-ID`. A reconnecting client gets live-only data, missing events that fired during disconnect. Also missing: a `Dashboard.Close()` lifecycle contract and a heartbeat-emission test. Source: `docs/status/2026-07-25_02-03_sse-integration-status.html`.

---

## P3 — Technical Debt & Future

- [ ] **MySQL event-store support.** Currently Postgres + SQLite only (dropped in v3.0.0 when `SQLEventStore` was delegated to `go-cqrs-lite/storage`, which has no MySQL dialect). `SQLSessionStore` already supports MySQL.
- [ ] **Offline sync E2E browser testing.** The SharedWorker IndexedDB queue (`sync/sync-worker.js`) is unit-verified but never exercised in a real browser. Needs a Playwright E2E test verifying the queue → retry → ACK cycle and the cross-session `rebuildAndRetry` path. The #1 deferred item across every sync status report since ADR-0029.
- [ ] **Migrate `id.NewAggregateID` → `id.NewStreamID`** across usermgmt. go-cqrs-lite v4 renamed the API; backward-compatible aliases exist, but the migration aligns with upstream naming. Production footprint is small — **2 call sites** in non-test usermgmt code (`import_export.go:155`, `service_oauth2.go:139`; ~8 repo-wide) — the larger ~121-figure count is test fixtures, which can follow once the production sites move.
- [ ] **Delete `usermgmt/sqlite_setup_test.go`.** Carries a `//go:build ignore` tag (excluded from builds) but is stale cruft with compilation errors against the current API. Note: the sibling `//go:build ignore` files (`sqlite_setup.go`, `postgres_setup.go`, `es_setup_core.go`) are the intended integration-path entry points and should stay.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
