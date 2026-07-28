# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-07-28 | **Version:** v4.6.1 (15 modules in `go.work`; see AGENTS.md for per-sub-module versions) | **Coverage:** Root 93.4% (gate 90%), openapi 99.0%, usermgmt 80.9% (gate 74%), identity-model 74.9% (**no gate yet — see P1 below**), dashboardui at 55% gate | **Lint:** All 15 modules at 0 issues. Zero SA1019 deprecation warnings. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P1 — High Impact (recurring across 3+ sessions)

- [ ] **Add identity-model coverage-gate threshold to `flake.nix`.** identity-model is the ONLY module without a coverage gate (`nix run .#coverage-gate` checks 8 modules; identity-model absent). Coverage is now 74.9% (verified). Add `check_cov identity-model 70` (or similar) to the `coverage-gate` app. One-line flake.nix edit. Flagged as open in 5+ status reports since 2026-07-23. Evidence: `flake.nix` coverage-gate app (lines ~330-350) lists 8 modules, identity-model not among them.

---

## P2 — Medium Impact (code quality & test gaps)

- [ ] **Audit `.golangci.yml` exclusions for masked bugs.** The lint triage to 0 issues (2026-07-28) achieved clean modules partly via `.golangci.yml` exclusions. The 23-02 self-critique admits: "I solved lint by excluding, not fixing — the 'zero the linter' anti-pattern." Review each exclusion (`exhaustruct` patterns for `readModelCore`/`InMemoryStore`/`IndexSpec`/`identitymodel.User`, `wrapcheck` re-export wrappers, `goconst` catalog strings, `funlen` event-catalog registration, disabled linters `canonicalheader`/`testpackage`/`makezero`) and confirm none masks a real bug. Convert suppressible nolints to named constants or explicit cases where feasible. Evidence: `.golangci.yml` exclusions section; `docs/status/2026-07-28_23-02_*` section (e).
- [ ] **dashboardui handler tests — DLQ, projection reset, time-travel, snapshot delete.** dashboardui has 4 test files (29 tests total) but no tests for: DLQ replay/delete/purge handlers, projection reset handler, time-travel detail handler (with events), snapshot delete handler. These are write-operation handlers (correctness-critical). Recurring gap flagged in 4+ status reports. Evidence: `dashboardui/handlers_dlq.go`, `dashboardui/handlers_projections.go`, `dashboardui/handlers_timetravel.go`, `dashboardui/handlers_snapshots.go` — no corresponding `*_test.go` coverage for these handler paths.

---

## P3 — Technical Debt & Future

- [ ] **MySQL event-store support.** Currently Postgres + SQLite only (dropped in v3.0.0 when `SQLEventStore` was delegated to `go-cqrs-lite/storage`, which has no MySQL dialect). `SQLSessionStore` already supports MySQL.
- [ ] **Offline sync E2E browser testing.** The SharedWorker IndexedDB queue (`sync/sync-worker.js`) is unit-verified but never exercised in a real browser. Needs a Playwright E2E test verifying the queue → retry → ACK cycle and the cross-session `rebuildAndRetry` path. The #1 deferred item across every sync status report since ADR-0029.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
