# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-07-31 | **Version:** v4.6.1 (15 modules in `go.work`; see AGENTS.md for per-sub-module versions) | **Coverage:** Root 93.6% (gate 90%), openapi 99.0%, usermgmt 81.7% (gate 74%), identity-model 74.9% (gate 70%), dashboardui 82.1% (gate 60%) | **Lint:** All 15 modules at 0 issues. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P1 — High Impact (blocking verification or correctness)

- [ ] **Write proper state-cache test for WithStateCache.** The sprint wired `decider.WithStateCache` in all 4 usermgmt repositories, but the existing `TestSnapshot_WritePathConsultsSnapshot` assertion was weakened to pass (asserting "neither snapshot nor LoadFromVersion was called" instead of verifying the cache-hit path). Need: (1) restore the strict snapshot assertion by disabling cache in that test, (2) write `TestStateCache_AcceleratesSecondLoad` verifying the cache-hit path. Source: self-review §d.3.

- [ ] **Fix E2E flake.nix app.** The `nix run .#e2e` app was added but never tested. It assumes `bun`/`npx` is available but doesn't add them to nix `runtimeInputs`. Would fail in a pure nix environment. Fix: add `pkgs.nodejs` (or `pkgs.bun`) to `runtimeInputs`, or document as "requires system node". Source: self-review §d.6.

---

## P2 — Medium Impact (code quality & tooling)

- [ ] **Upgrade cqrs-lint from Nix v0.2.2 to latest build.** The installed binary (`/run/current-system/sw/bin/cqrs-lint` v0.2.2) lacks comma-separated rule ID support (`//cqrs-lint:ignore(E004,E006)`). This causes 4 stale-suppression warnings in `examples/dashboard-demo/` where dual-rule findings need two separate comment lines. The go-cqrs-lite source (`5ee3832e`) already implements comma-separated parsing. Upgrading eliminates all 4 stale warnings cleanly. Source: `docs/status/2026-07-31_03-41_cqrs-lint-suppression-remediation-round2.md`.

- [ ] **Write OnProjectionFailed runtime test.** `projectionhost.WithOnFailed` is wired via `EventSourcedConfig.OnProjectionFailed` / `ServiceConfig.OnProjectionFailed` but the callback is unverified at runtime. Need a test that registers a projection that always fails, then verifies the callback fires after terminal failure (5 crash-restarts). Source: self-review §e.7.

- [ ] **Add MySQL error classifier.** `MySQLDialect` was added to go-cqrs-lite `storage/sql/dialect.go` but `classify_init.go` has no `classifyMySQLError` function. MySQL errors won't be classified into error families (Transient/Conflict/etc.). Source: self-review §e.6.

- [~] **dashboardui coverage improvements.** Overall improved from 78.7% → 82.1% (gate 60%), but two handlers still have thin coverage: `overviewStats` (51.1% — ProjectionHost branch uncovered), `dlqIndexHandler` (58.3% — ProjectionHost link rendering uncovered). Need tests with a projectionhost.Host mock.

- [~] **MySQL event-store support via go-cqrs-lite Dialect.** Event-store-only MySQL dialect added (`MySQLDialect` with `?` placeholders, MySQL-specific DDL, `IsDuplicateKeyError` extended). `dialectToUpstream` updated. `storage/v4` at v4.5.0. Remaining: (1) add `classifyMySQLError` error classifier, (2) add integration test against real MySQL, (3) document MySQL support in guides/README/FEATURES, (4) consider `NewMySQLSetup` convenience constructor. Source: self-review §e.6, §f.44.

- [~] **Integrate E2E tests into flake.nix/CI.** `nix run .#e2e` app added to flake.nix but is broken (missing `nodejs`/`bun` in runtimeInputs, never actually tested). Fix runtimeInputs then verify all 4 Playwright scenarios pass. Source: self-review §d.6.

---

## P3 — Technical Debt & Future

- [~] **Offline sync E2E browser testing.** The SharedWorker IndexedDB queue (`sync/sync-worker.js`) is unit-verified and E2E infrastructure is built (`e2e/` directory: Go test server, Playwright config, 4 test scenarios in `sync.spec.ts`, `README.md`). All 4 E2E tests pass (offline enqueue, online flush, cross-session recovery, multiple commands). syncVersion at `1.3.0`. Remaining: fix and verify the flake.nix e2e app (see P1 above).

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
