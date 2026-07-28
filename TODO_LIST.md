# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-07-29 | **Version:** v4.6.1 (15 modules in `go.work`; see AGENTS.md for per-sub-module versions) | **Coverage:** Root 93.7% (gate 90%), openapi 99.0%, usermgmt 80.9% (gate 74%), identity-model 74.9% (gate 70%), dashboardui 72.5% (gate 60%) | **Lint:** All 15 modules at 0 issues. Zero SA1019 deprecation warnings. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P1 — High Impact (recurring across 3+ sessions)

_(None open. Previous P1 items completed: identity-model coverage gate added to flake.nix at 70% threshold, `.golangci.yml` exclusion audit confirmed zero masked bugs, dashboardui write-operation handler tests added.)_

---

## P2 — Medium Impact (code quality & test gaps)

- [ ] **dashboardui index handler tests.** The write-operation handlers (DLQ replay/delete/purge, projection reset, time-travel detail, snapshot delete) are now fully tested (`handlers_write_test.go`, 16 tests). The remaining untested handlers are the index/list views: `timeTravelIndexHandler`/`renderTimeTravelIndex` and `snapshotsIndexHandler`/`renderSnapshotsIndex`. These are read-only render handlers (lower correctness risk than write handlers). Evidence: `dashboardui/handlers_timetravel.go:16`, `dashboardui/handlers_snapshots.go:17` — no corresponding test coverage for these index paths.

---

## P3 — Technical Debt & Future

- [ ] **MySQL event-store support.** Currently Postgres + SQLite only (dropped in v3.0.0 when `SQLEventStore` was delegated to `go-cqrs-lite/storage`, which has no MySQL dialect). `SQLSessionStore` already supports MySQL. Requires adding a MySQL dialect to `go-cqrs-lite/storage` (external repo).
- [~] **Offline sync E2E browser testing.** The SharedWorker IndexedDB queue (`sync/sync-worker.js`) is unit-verified and E2E infrastructure is built (`e2e/` directory: Go test server, Playwright config, 4 test scenarios in `sync.spec.ts`). A **production bug** in `sync-client.js` (FormData serialization — `cfg.parameters` is a `FormData` object in HTMX 2.x that `postMessage` cannot clone) was fixed in this session (syncVersion bumped to `1.2.0`). The E2E tests need to be re-run in a browser environment to verify the fix resolves the enqueue failure. Remaining: verify tests pass with Chromium, add `e2e/README.md`, integrate into flake.nix/CI.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
