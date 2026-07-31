# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-07-31 | **Version:** v4.6.1 (15 modules in `go.work`; see AGENTS.md for per-sub-module versions) | **Coverage:** Root 93.7% (gate 90%), openapi 99.0%, usermgmt 80.9% (gate 74%), identity-model 74.9% (gate 70%), dashboardui 78.7% (gate 60%) | **Lint:** All 15 modules at 0 issues. Zero SA1019 deprecation warnings. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P1 — High Impact (blocking verification or correctness)

- [ ] **Publish httputil v0.8.0 and remove go.work local replace.** The httputil consolidation moved Server-Timing, CSRF core, and keyed rate limiting into `httputil`; cqrs-htmx re-exports them via type/var aliases (`server_timing_reexport.go`, `csrf_reexport.go`, `ratelimit_reexport.go`). `go.mod` still references `v0.7.1` (which lacks the consolidated symbols). The `go.work` local replace (`=> /home/lars/projects/httputil`) keeps the workspace build green, but hermetic builds (`nix run .#test` under GOWORK=off) fail on missing symbols. Until httputil v0.8.0 is tagged and the replace is removed, canonical nix gates cannot verify the build. Source: `docs/planning/2026-07-30_02-15_httputil-consolidation-superb-plan.md`.

- [ ] **Run canonical nix verification gates after httputil publication.** Multiple sessions since 2026-07-22 (httputil consolidation, cqrs-lint remediation rounds 1–2, identity-model enhancements, dashboardui sprint) used raw `go build`/`go test` instead of `nix run .#test`, `nix run .#lint`, `nix run .#coverage-gate`, `nix flake check`. Blocked by the httputil replace above; once resolved, run all four gates and verify all 15 modules pass. This is the #1 recurring verification gap flagged across 10+ status reports.

---

## P2 — Medium Impact (code quality & tooling)

- [ ] **Upgrade cqrs-lint from Nix v0.2.2 to latest build.** The installed binary (`/run/current-system/sw/bin/cqrs-lint` v0.2.2) lacks comma-separated rule ID support (`//cqrs-lint:ignore(E004,E006)`). This causes 4 stale-suppression warnings in `examples/dashboard-demo/` where dual-rule findings need two separate comment lines. The go-cqrs-lite source (`5ee3832e`) already implements comma-separated parsing. Upgrading eliminates all 4 stale warnings cleanly. Source: `docs/status/2026-07-31_03-41_cqrs-lint-suppression-remediation-round2.md`.

- [ ] **dashboardui coverage improvements.** Overall 78.7% (gate 60%), but several handlers have thin coverage: `overviewStats` (48.9% — 7 data-source branches), `renderDLQ` (42.9% — populated-entry path untested), `dlqDetailHandler` (54.5%), `snapshotDetailHandler` (50.0%), `dlqIndexHandler` (58.3%), `eventDetailHandler` (28.6%), `loadRecentEvents` (46.2%). Needs integration-style tests with real memory stores. Source: `docs/status/2026-07-30_22-21_p2-dashboardui-coverage-and-todo-reconciliation.md` items f.7–f.21.

- [ ] **Document cqrs-lint suppression syntax in AGENTS.md Gotchas.** Two sessions (cqrs-lint 79-finding remediation + round 2) struggled with suppression placement. Key learnings: (a) suppression checks line + line-above only; (b) v0.2.2 doesn't support comma-separated rules despite source implementing it; (c) go.mod comments work for module-level findings (E003); (d) stale detector flags standalone-line-above suppressions when the rule fires on line+1; (e) the "standalone line above + inline" pattern is the only working approach for dual-rule findings under v0.2.2. Source: `docs/status/2026-07-30_23-21_cqrs-lint-79-finding-remediation.md` item e.6 + round 2 item e.5.

- [ ] **Integrate E2E tests into flake.nix/CI.** The offline sync E2E Playwright tests (`e2e/`, 4 scenarios, all passing) have a `README.md` and test infrastructure but are NOT wired into the nix build or CI pipeline. Needs a `nix run .#e2e` app or CI step. Source: `docs/status/2026-07-29_00-17_offline-sync-e2e-browser-testing.md`.

- [ ] **Fix `decoder.go:22` unparam finding.** `readBodyForDecode[T]` always returns zero-value `T` (the function reads body bytes but never populates `T`; callers unmarshal into `out` afterward). The `T` return is structurally unnecessary. Pre-existing, flagged by `unparam` linter in dedup round 3/4. Source: `docs/status/2026-07-29_23-38_dedup-round4-t2-zero-clones-brutal-self-review.md` item f.6.

- [ ] **Fix `dashboardui/sse_replay_test.go:182` data race.** `httptest.ResponseRecorder` is accessed from both the test goroutine (`buf.String()`) and the SSE handler goroutine (`buf.Write` via heartbeat), breaking `-race` for the entire dashboardui module. Pre-existing, noted in dedup rounds 3 and 4. Source: `docs/status/2026-07-29_23-07_dedup-round3-zero-clones.md` section d + `docs/status/2026-07-29_23-38_dedup-round4-t2-zero-clones-brutal-self-review.md` item f.7.

---

## P3 — Technical Debt & Future

- [ ] **MySQL event-store support via go-cqrs-lite Dialect.** Currently Postgres + SQLite only (MySQL was dropped in v3.0.0 when `SQLEventStore` was delegated to `go-cqrs-lite/storage`, which has no MySQL dialect; `SQLSessionStore` already supports MySQL). `storage/sql/dialect.go` has 3 dialects (Postgres, SQLite, DuckDB) behind an 11-method `Dialect` interface. Event-store-only MySQL support is LOW effort (~half a day): clone `PostgresDialect` → `MySQLDialect` (`?` placeholders, `LONGBLOB`/`JSON`/`DATETIME(6)` types), add MySQL error-1062 detection to `IsDuplicateKeyError`. Full SQL backend (snapshots/checkpoints/KV/views/projections) is MEDIUM (~2-3 days) — the blocker is UPSERT syntax divergence (`ON CONFLICT` → `ON DUPLICATE KEY UPDATE`) across ~8 call sites. Recommendation: ship event-store-only first; add UPSERT abstraction when non-event-store MySQL is requested. Requires adding a MySQL dialect to `go-cqrs-lite/storage` (external repo). Source: `docs/guides/leveraging-go-cqrs-lite.md` evaluation + ROADMAP.md.

- [~] **Offline sync E2E browser testing.** The SharedWorker IndexedDB queue (`sync/sync-worker.js`) is unit-verified and E2E infrastructure is built (`e2e/` directory: Go test server, Playwright config, 4 test scenarios in `sync.spec.ts`, `README.md`). All 4 E2E tests pass (offline enqueue, online flush, cross-session recovery, multiple commands). syncVersion at `1.3.0`. Remaining: integrate into flake.nix/CI (see P2 above).

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
