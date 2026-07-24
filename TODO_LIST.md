# TODO List — cqrs-htmx

**Updated:** 2026-07-24 | **Version:** v4.5.0 (15 modules in go.work; see AGENTS.md for per-sub-module versions) | **Coverage:** Root 93.5% (gate 90%), usermgmt 81.0% (gate 74%) — recompute via `nix run .#coverage-gate` | **Lint:** 0 issues across all submodules. Root has ~75 pre-existing low-severity style nits (varnamelen, testpackage, etc.).

## Status Legend

- [ ] OPEN — actionable, not yet started
- [~] PARTIALLY DONE — started but incomplete

> Completed items live in [CHANGELOG.md](CHANGELOG.md). Deferred and rejected ideas live in [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P1 — Release Hygiene

- [ ] **Fix inter-module version refs.** usermgmt/adminui/loginpage reference root as `v4.4.0` (should be `v4.5.0`); usermgmt references identity-model via pseudo-version `v4.0.0-20260723162555-beae91131538` (should be `v4.1.0`); `examples/dashboard-demo/go.mod` uses broken zero pseudo-version `v4.0.0-00010101000000-000000000000` for dashboardui. These don't affect workspace builds (go.work replaces) but block `GOWORK=off` resolution for external consumers. Root cause: `batch-release.sh` strips replaces without re-resolving requires.

---

## P2 — Quality Gates

- [~] **identity-model test coverage.** ~41% coverage (2 test files for 25 source files). No coverage gate threshold defined for this module in flake.nix. Needs tests for Authz engine, command constructors, event payload round-trips, crypto helpers. See `docs/status/2026-07-23_21-27_identity-model-consolidation-brutal-review.md`.
- [ ] **dashboardui dead code cleanup.** `notImplemented()` at `handlers.go:21` and `renderStatCardsTempl()` at `templ_render.go:25` are never called. Delete or wire in.
- [~] **dashboardui test coverage.** 1 test file (`dashboard_test.go`, 12 tests) for 12 source files. Needs handler-level tests, SSE bridge tests, payload rendering tests.
- [ ] **dashboardui `handlers.go` split.** Single file at 1136 lines — split per domain (overview, events, aggregates, projections, DLQ, audit, snapshots, time-travel).

---

## P3 — Technical Debt & Future

- [ ] **MySQL support** for event store (currently Postgres + SQLite only; dropped in v3.0.0 when SQLEventStore was delegated to go-cqrs-lite/storage which has no MySQL dialect)
- [ ] **Offline sync E2E browser testing.** SharedWorker IndexedDB persistence (`sync/sync-worker.js`) is functional but not browser-tested. Needs Playwright E2E test verifying queue→retry→ACK cycle.
- [ ] **Migrate `id.NewAggregateID` → `id.NewStreamID`** across usermgmt (121 call sites). go-cqrs-lite v4 renamed the API; backward-compatible aliases exist but the migration aligns with upstream.
- [ ] **Delete `usermgmt/sqlite_setup_test.go`.** Has `//go:build ignore` tag (excluded from builds) but is stale cruft with compilation errors.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
