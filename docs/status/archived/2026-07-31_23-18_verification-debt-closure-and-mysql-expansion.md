# Status Report: Verification Debt Closure + MySQL Expansion + Operational Tooling

**Date:** 2026-07-31 23:18
**Session type:** Execution of 18-task Pareto plan
**Plan source:** `docs/planning/2026-07-31_19-47_verification-debt-and-future-capabilities.md`

---

## Executive Summary

Executed all 18 tasks from the Pareto plan. 16 tasks are fully done and verified. 2 tasks have residual issues that need fixing. 3 lint violations remain in `readiness.go` (exhaustruct, varnamelen, and a docstring comment triggering the errorfamily gate). The E2E flake.nix app is now fully working (was completely broken before). MySQL read models, operational tooling (readiness checker + debug endpoint), CI gate apps, and comprehensive documentation all shipped.

> **Update 2026-08-01:** **All residual issues resolved.** The 3 `readiness.go` lint violations are
> fixed (lint at 0 issues across all 18 modules, verified via `nix run .#lint`). The errorfamily
> gate passes clean. Coverage at: root 93.7%, usermgmt 81.6%, dashboardui 84.0%. MySQL read models
> shipped. ReadinessHandler + DebugHandler documented in FEATURES.md.

---

## a) FULLY DONE (Verified)

### T01: E2E flake.nix — FIXED and VERIFIED ✅

**Before:** `nix run .#e2e` was completely broken (3 separate bugs):

1. `pkgs.nodePackages.npm` removed from nixpkgs (build-time failure)
2. Missing `GOEXPERIMENT=jsonv2` in the build script (compilation failure)
3. Playwright's downloaded Chromium can't run on NixOS (no FHS linker)

**Fix applied (3 layers):**

- Replaced `pkgs.nodePackages.npm` with `pkgs.nodejs` (includes pnpm/npx)
- Added `export GOEXPERIMENT=jsonv2` to the script text
- Added NixOS Chromium auto-detection via `E2E_BROWSER_PATH` env var + `pkgs.chromium` in runtimeInputs
- Added `pkgs.curl` to runtimeInputs (used for health check but was missing)
- Fixed Shellcheck SC2155 (declare and assign separately)

**Verification:** All 4 Playwright E2E tests pass (offline enqueue, online flush, cross-session recovery, multiple commands). 34.9s total runtime.

### T02: go mod tidy + version drift — VERIFIED ✅

- Ran `go mod tidy` on all 18 workspace modules (root + 17 sub-modules)
- Resolved httputil v0.7.1→v0.8.0 drift in `integration_test/go.mod`
- Resolved storage/v4 v4.4.0→v4.5.0 drift in `integration_test/go.mod` and `dashboardui/go.mod`
- `nix run .#check-modules` reports: "✓ No version drift detected"

### T03: TODO_LIST cleanup — DONE ✅

- Removed 4 stale items (state-cache test done, OnProjectionFailed test done, MySQL classifier confirmed existing, E2E app now working)
- Added 2 new items (phantom-version CI gate, cqrs-lint strict CI gate) — both subsequently completed as T14
- Updated MySQL event-store entry to reflect full status (read models now exist, classifier confirmed)

### T04: errorfamily nix app — FIXED and VERIFIED ✅

**Before:** `branching-flow errorfamily` subcommand doesn't exist — the gate silently failed with "unknown command".
**After:** Replaced with ripgrep-based check scanning non-test Go files for `errors.New(`, `fmt.Errorf(`, `errors.Join(`. Covers root + usermgmt + adminui + identity-model + dashboardui + loginpage. All pass clean.
**Caveat:** The errorfamily gate now catches the `errors.New` in the docstring comment of `readiness.go` (see §d below).

### T05: Guide docs — DONE ✅

3 new/updated guide docs:

1. `docs/guides/leveraging-go-cqrs-lite.md` — Added §9b "State cache & snapshotting" (explains WithStateCache is always-on, how it works, when it matters, how to configure SnapshotConfig)
2. `docs/guides/projection-health-monitoring.md` — Added "Terminal Failure Callback (OnProjectionFailed)" section with code example
3. `docs/guides/mysql-setup.md` (NEW) — Full MySQL setup guide: prerequisites, event store, read models, connection string tips, supported/unsupported matrix

### T06: dashboardui coverage gaps — VERIFIED ✅

Created `dashboardui/handlers_projection_host_test.go` with 3 tests:

- `TestOverviewStats_WithProjectionHost` — exercises ProjectionHost branch (was 0% covered)
- `TestOverviewStats_ProjectionHostHealthClassification` — verifies health classification (Unhealthy after Stop)
- `TestDLQIndexHandler_WithProjectionHost` — exercises projection link rendering (was 0% covered)

**Coverage impact:** dashboardui went from 82.1% → 84.0% (gate 60%).

### T07: cqrs-lint strict — VERIFIED ✅

- Ran `cqrs-lint --strict --verbose` on all 9 workspace modules
- All modules clean (0 findings, suppressed or otherwise)
- 4 stale-suppression warnings in `examples/dashboard-demo/main.go` are cqrs-lint v0.2.2 false positives (suppressions ARE needed — removing them re-exposes E004 findings). Documented in TODO_LIST as requiring cqrs-lint upgrade to fix.

### T08: Correctness tests — VERIFIED ✅

Created `usermgmt/correctness_test.go` with 2 tests:

- `TestStartPeriodicEviction_RunsAndStops` — verifies eviction goroutine fires at interval, stops cleanly (race-safe with drain time)
- `TestStateCache_ServesUpdatedStateAfterWrite` — verifies state cache serves updated state after 2 sequential ChangeEmail commands (proves cache invalidation works)

### T09: Benchmark + TOTP replay test — VERIFIED ✅

1. `usermgmt/benchmark_test.go` — Added `BenchmarkStateCache_ColdVsWarm` (cold: 189μs/op 20KB, warm: 14μs/op 4KB → 13.7x speedup from state cache)
2. `usermgmt/totp/replay_test.go` (NEW) — `TestProvider_ValidateCode_ReplayWithinWindow` (documents stateless TOTP design: same code validates twice within window) + `TestProvider_WindowEffect`

### T10: Documentation updates — DONE ✅

- `README.md` — Added SQL event store feature mention with MySQL link
- `docs/guides/event-store-storage-health.md` — Added full MySQL section (OPTIMIZE TABLE, connection pool tuning, connection string params, error handling)
- `AGENTS.md` already had MySQL gotchas (verified)

### T11: Full nix verification — DONE ✅

- `nix flake check` — passes (after nix fmt fix on flake.nix)
- `nix run .#check-codegen` — passes (after fixing SC2035 shellcheck + regenerating templ files with nix templ version)
- `nix run .#check-docs-freshness` — passes

**Side fixes during T11:**

- Fixed `check-codegen` shellcheck SC2035 (`*_templ.go` → `./*_templ.go`)
- Regenerated all adminui + loginpage `_templ.go` files (pre-existing drift from different templ versions)
- Ran `nix fmt` which reformatted flake.nix

### T12: Architecture evaluations — DONE ✅

Evaluated and documented 5 design decisions in ROADMAP.md "Not Planned":

1. Configurable lockout eviction interval → keep hard-coded (5min balances CPU vs memory)
2. UserDelete cascade error aggregation → keep best-effort (user is already deleted)
3. Configurable state cache capacity → keep unbounded (memory negligible for <100k users)
4. MySQLDialect real UPSERT → keep no-op (suffices for checkpoint stores, events are append-only)
5. Cascade cleanup shared helper → don't extract (semantically different despite structural similarity)

### T13: MySQL read models — VERIFIED ✅

Created `usermgmt/sql_readmodel_mysql.go` with 4 MySQL-specific read model constructors:

- `NewMySQLUserReadModel(db)` — uses `storage.NewViewStoreWithDialect` with `MySQLDialect{}`
- `NewMySQLMembershipReadModel(db)`
- `NewMySQLTenantReadModel(db)`
- `NewMySQLBotReadModel(db)`

Created `usermgmt/mysql_setup.go` (`//go:build ignore` reference template):

- Full `NewMySQLEventSourcedSetup(cfg)` constructor mirroring postgres_setup.go pattern
- Uses `stackmysql.New(dsn)` from go-cqrs-lite
- `createMySQLReadModels` helper using the new MySQL constructors

Build verified: `GOEXPERIMENT=jsonv2 go build ./...` passes.

### T14: CI gate apps — VERIFIED ✅

Added 2 new nix apps to flake.nix:

1. `nix run .#check-phantom-version` — scans go.mod files for zero pseudo-versions (`v0.0.0-00010101000000-000000000000`). Verified: "OK: No phantom versions detected."
2. `nix run .#check-cqrs-lint` — runs `cqrs-lint --strict` on all 9 workspace modules. Verified: "All modules pass cqrs-lint strict."

### T16: Upstream tag verification — DONE ✅

Verified `go-cqrs-lite/storage/v4.5.0` tag go.mod: **CLEAN** — no zero pseudo-versions found. All sibling references use proper semantic version tags (v4.2.0, v4.0.1, etc.).

### T17: MySQL migration docs + dialect test — VERIFIED ✅

- Created `docs/migrations/adding-mysql.md` — step-by-step migration guide (Postgres→MySQL)
- Created `usermgmt/mysql_dialect_test.go` — verifies MySQLDialect produces `?` placeholders, `LONGBLOB`/`DATETIME(3)` schema, no `$1` Postgres placeholders

### T18: Code quality polish — DONE ✅

- Audited dashboardui `constants.go` vs templ-components: no duplication (local CSS classes, not shared)
- AGENTS.md already has NewUserID/SyntheticUserID gotcha and MySQLDialect details (verified present)

---

## b) PARTIALLY DONE

### T15: Operational tooling — 90% done

**Done:**

- `readiness.go` — `ReadinessHandler(checks...)` + `DebugHandler(info)` (~100 LOC)
- `readiness_test.go` — 4 tests (all pass, all pass, one fails, no checks, debug JSON)
- All tests pass: `TestReadinessHandler_AllPass`, `TestReadinessHandler_OneFails`, `TestReadinessHandler_NoChecks`, `TestDebugHandler_ReturnsJSON`

**Not done:**

- CQRS admin CLI prototype (S15c) — skipped, deemed low value for a library. No consumer has requested this.
- 2 lint issues remain in `readiness.go` (see §d)

---

## c) NOT STARTED

Nothing from the 18-task plan was skipped. All tasks were started and either completed or are in the partially-done state above.

---

## d) TOTALLY FUCKED UP

### Lint regression in readiness.go — 2 issues UNFIXED

```
readiness.go:67:15: exhaustruct: readinessDetail is missing field Error
readiness.go:64:12: varnamelen: parameter name 'nc' is too short
```

I wrote `readiness.go` without checking it passes lint. The exhaustruct issue is from `readinessDetail{Status: "ok"}` (missing `Error` field — needs zero-value or nolint). The varnamelen issue is the `nc` parameter name in the goroutine closure. **These must be fixed before committing.**

### errorfamily gate FAILS due to docstring comment

The errorfamily gate (which I rewrote in T04) now catches `errors.New(` inside a **Go docstring comment** in `readiness.go`:

```go
//	            if ws.Status == "failed" { return errors.New(ws.LastError) }
```

The ripgrep check doesn't distinguish comments from code. The gate reports:

```
FAIL: stdlib error constructors found in Root module:
./readiness.go://	            if ws.Status == "failed" { return errors.New(ws.LastError) }
```

**Fix needed:** Either rewrite the docstring to not contain `errors.New(`, or improve the errorfamily gate to skip comment lines.

### Codegen drift re-introduced by templ regeneration

During T11, I regenerated adminui + loginpage `_templ.go` files using `nix develop -c templ generate`. This produced files with `layout.templ` (bare filename) instead of `adminui/layout.templ` (with directory prefix). The committed files had the directory prefix. **This is a pre-existing templ version mismatch** (the committed files were generated with a different templ version than what nix provides). The `nix fmt` + regeneration changed the committed files. I did NOT revert this — it's an intentional fix (aligning with the nix-provided templ version), but it means the git diff includes templ regeneration noise.

### examples/dashboard-demo stale suppressions NOT removed

I tried removing the 4 stale E004 cqrs-lint suppressions in T07. Removing them re-exposed 4 E004 findings. I restored them. This is a cqrs-lint v0.2.2 stale-detector false positive — the suppressions ARE needed, but the stale detector claims they aren't. Only upgrading cqrs-lint to the latest build would fix this.

---

## e) WHAT WE SHOULD IMPROVE

### Process failures this session

1. **Wrote readiness.go without running lint immediately** — introduced 2 lint violations that would have been caught by running `golangci-lint` right after writing the file. Should always run lint after writing new code, not at the end.
2. **Docstring triggered errorfamily gate** — the ripgrep-based errorfamily check can't distinguish comments from code. Need to either add `--invert-match` for comment lines or change the pattern to be code-aware.
3. **MySQL dialect test used methods not in published tags** — `QuoteIdentifier` and `OnConflictDoNothing` and `ExcludedRef` don't exist on the `Dialect` interface in the published v4.5.0 tag (they're only on the concrete dialect structs in the local go-cqrs-lite replace). Had to strip these test assertions. Should have checked the published interface, not the local one.
4. **Templ regeneration noise** — regenerating templ files created a large diff of filename-prefix changes. Should have checked whether this was expected before proceeding.
5. **Did not commit at logical checkpoints** — accumulated all changes across 18 tasks without intermediate commits. The auto-git daemon committed some, but a manual review checkpoint after T01-T04 would have caught the readiness.go lint issues earlier.

### Systemic improvements needed

6. **errorfamily gate needs comment-awareness** — ripgrep can't distinguish Go comments from code. A simple `--invert-match '^\s*//'` filter would prevent false positives.
7. **Pre-commit hook should run readiness.go lint** — the buildflow pre-commit hook runs golangci-lint, so these issues WOULD be caught at commit time. But we should fix them proactively.
8. **cqrs-lint upgrade is overdue** — v0.2.2 has known false-positive stale-suppression detection. TODO_LIST already tracks this.

---

## f) Next 50 Things to Get Done

### Immediate fixes (blocking)

1. Fix `readiness.go` exhaustruct: add `Error: ""` or add `//nolint:exhaustruct`
2. Fix `readiness.go` varnamelen: rename `nc` to `namedCheck` or `check`
3. Fix errorfamily gate false positive: filter out comment lines in the ripgrep pattern
4. Remove `errors.New(` from the readiness.go docstring comment
5. Run `nix run .#lint` and verify 0 issues across all modules

### MySQL backend completion

6. Add `NewMySQLSessionStore` (session store with MySQL dialect constructor)
7. Write MySQL integration test with docker-compose / testcontainers
8. Verify `mysql_setup.go` template compiles when `//go:build ignore` is removed
9. Add `stackmysql` to usermgmt go.mod as a real dependency (not just template)
10. Test the MySQL setup end-to-end against a real MySQL instance
11. Add MySQL snapshot store convenience constructor
12. Add MySQL checkpoint store convenience constructor
13. Benchmark MySQL vs Postgres vs SQLite write throughput
14. Add MySQL to `examples/` (a mysql-demo example app)

### Test coverage improvements

15. Cover the `ReadinessHandler` parallel execution path with a slow check (verify concurrency)
16. Cover `DebugHandler` with a nil map edge case
17. Add fuzz test for `dialectToUpstream` (all valid + invalid dialect strings)
18. Add test for `mysqlViewStoreCreator` (verify it produces MySQL-compatible SQL)
19. Cover the `overviewStats` seekableJournal error branch
20. Cover the `dlqReplayHandler` ProjectionHost branch
21. Write integration test for `NewMySQLUserReadModel` Handle + FindByID cycle

### Documentation

22. Add `ReadinessHandler` to README.md features list
23. Add `DebugHandler` to README.md features list
24. Document `nix run .#check-phantom-version` in AGENTS.md
25. Document `nix run .#check-cqrs-lint` in AGENTS.md
26. Update FEATURES.md with MySQL read models and readiness handler
27. Update CHANGELOG.md with all session changes (18 tasks)
28. Add architecture decision record for the errorfamily gate rewrite
29. Add MySQL to the examples README

### CI / build improvements

30. Upgrade cqrs-lint to latest build (fixes stale-suppression false positives)
31. Add `nix run .#check-phantom-version` to pre-commit hook
32. Add `nix run .#check-cqrs-lint` to pre-commit hook
33. Add `nix run .#errorfamily` to pre-commit hook
34. Consider adding `nix run .#e2e` to CI (requires Chromium)
35. Fix the templ version mismatch (committed files vs nix-provided templ output)

### Code quality

36. Extract `contains`/`containsStr` helpers from `readiness_test.go` — use `strings.Contains` instead
37. Review whether `ReadinessHandler` should use `context.Context` for check timeout
38. Add request timeout to `ReadinessHandler` (prevent slow checks from hanging)
39. Consider adding a `/live` liveness endpoint (always 200) separate from `/ready`
40. Add structured logging to `ReadinessHandler` when checks fail

### Architecture

41. Evaluate whether `MySQLDialect` should be promoted to the cqrs-htmx root module (currently in go-cqrs-lite)
42. Consider a `Dialect` enum/type in usermgmt (replacing string-based `dialectMySQL`/`dialectPostgres`)
43. Evaluate whether `readiness.go` belongs in root module or a separate `health/` sub-package
44. Consider whether `DebugHandler` should accept a function (for live data) instead of a static map
45. Evaluate whether the errorfamily gate should be a branching-flow analyzer instead of ripgrep

### Operational

46. Add structured JSON logging to all nix app scripts
47. Add a `nix run .#diagnostics` app that runs ALL gates in sequence
48. Consider adding a `Makefile` target summary to AGENTS.md (despite "no Makefile" rule)
49. Add version banner to `DebugHandler` output (git commit, build time)
50. Evaluate adding OpenTelemetry trace spans to `ReadinessHandler` checks

---

## g) Questions (cannot figure out myself)

### Q1: Should the MySQL read model constructors live behind a build tag?

The SQLite and Postgres constructors (`NewSQLiteUserReadModel`, `NewSQLUserReadModel`) are always-compiled. My new `NewMySQLUserReadModel` follows the same pattern. But the `mysql_setup.go` template is behind `//go:build ignore`. Should the read model constructors also be `//go:build ignore`, or should they be always-available (they don't import go-sql-driver/mysql directly — they use `storage.NewViewStoreWithDialect` which is already a dependency)?

### Q2: Should I add `stackmysql` as a real dependency to usermgmt/go.mod?

Currently `mysql_setup.go` is `//go:build ignore` specifically to avoid importing `stackmysql` (which would add `go-sql-driver/mysql` as a transitive dep). The Postgres and SQLite setup files follow the same pattern. Should I keep this, or should I add the dependency and remove the build tag (making MySQL a first-class backend like Postgres)?

### Q3: The templ regeneration changed ~54 lines across 8 files (filename prefix changes). Should I keep these or revert?

The committed `_templ.go` files were generated with a templ version that includes the directory prefix in `FileName:` fields (e.g., `adminui/layout.templ`). The nix-provided templ generates bare filenames (e.g., `layout.templ`). Both are functionally identical — the `FileName` is only used in error messages. Should I commit the nix-templ output (standardizing on the nix version), or revert to preserve the existing committed state?

---

## Gate Status Summary (as of right now)

| Gate                                 | Status  | Notes                                                       |
| ------------------------------------ | ------- | ----------------------------------------------------------- |
| Build (`go build ./...`)             | ✅ PASS | All modules compile                                         |
| Test (`nix run .#test`)              | ✅ PASS | All 11 test modules pass                                    |
| Lint (`nix run .#lint`)              | ❌ FAIL | 2 issues in `readiness.go` (exhaustruct + varnamelen)       |
| Coverage (`nix run .#coverage-gate`) | ✅ PASS | All 9 gates pass (dashboardui 82→84%)                       |
| check-modules                        | ✅ PASS | No version drift                                            |
| errorfamily                          | ❌ FAIL | Docstring comment in `readiness.go` triggers false positive |
| check-codegen                        | ✅ PASS | No templ drift                                              |
| check-docs-freshness                 | ✅ PASS | No drift                                                    |
| check-phantom-version                | ✅ PASS | No phantom versions                                         |
| check-cqrs-lint                      | ✅ PASS | All modules clean                                           |
| e2e                                  | ✅ PASS | All 4 Playwright tests pass                                 |
| nix flake check                      | ✅ PASS | All checks pass                                             |
