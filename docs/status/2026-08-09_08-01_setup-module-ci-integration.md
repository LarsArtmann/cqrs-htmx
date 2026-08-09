# Status Report: setup/v4 Module CI Integration

> **Date:** 2026-08-09 08:01
> **Session scope:** Wire `setup/v4` into all workspace gates (coverage-gate, CI, lint, module-isolation, dep-budgets) + fix 13 lint issues.
> **Task source:** TODO_LIST.md P1 items #1 and #2 (both now removed → CHANGELOG).
> **Commits this session:** `9b39ff1e` (refactor), `522f8e68` (CI+flake), `a450d655` (CHANGELOG). Doc updates (AGENTS/FEATURES/ROADMAP/TODO) uncommitted at report time.

---

## a) FULLY DONE

### Lint fixes (13 issues → 0)

| File | Issue | Fix |
| --- | --- | --- |
| `config.go:29-31` | SA1019 ×3 (deprecated usermgmt re-export aliases) | Migrated to direct `identitymodel.TOTPProvider`/`WebAuthnProvider`/`OAuth2Provider` imports |
| `setup.go:79` | mnd (magic number `25`) | Extracted `const defaultPageSize = 25` |
| `setup.go:49` | exhaustruct (Bundle missing Admin/Dashboard/Login) | Added `//nolint:exhaustruct` (conditional assignment below) |
| `bundle.go:96` | wrapcheck (unwrapped `Service.Close()` error) | Wrapped via `errorfamily.WrapInfrastructure(err, "setup.bundle_close", ...)` |
| `setup_test.go` ×7 | errcheck (unchecked `defer bundle.Close()`) | Converted to `defer func() { _ = bundle.Close() }()` |

### Gate integration

| Gate | Status | Evidence |
| --- | --- | --- |
| `nix run .#coverage-gate` | **PASS** — setup 85.3% (gate 80%) | Added `check_cov setup 80` in `flake.nix:622` |
| `nix run .#lint` | **PASS** — 0 issues across 12 modules | Auto-discovered from `go.work`; setup passes with 0 issues |
| `nix run .#build` | **PASS** — 21 modules including setup | Auto-discovered from `go.work` |
| `nix run .#check-cqrs-lint` | **PASS** — all modules pass strict | Verified post-report |
| `nix flake check --no-build` | **PASS** — all checks passed | Verified post-report |
| CI: build job | **ADDED** — `cd setup && go build ./...` | `ci.yml:149-155` |
| CI: test job | **ADDED** — `cd setup && go test ... -race -coverprofile=coverage.out` | `ci.yml:242-248` |
| CI: coverage check | **ADDED** — 80% threshold | `ci.yml:290-294` |
| CI: lint job | **ADDED** — `golangci-lint (setup)` | `ci.yml:381-389` |
| CI: mod-tidy job | **ADDED** — `setup` in module list | `ci.yml:403` |
| CI: errorfamily scanner | **ADDED** — `setup` in module list | `ci.yml:494` |
| `check-module-isolation.sh` | **ADDED** — `setup` in MODULES array | `scripts/check-module-isolation.sh:26` |
| `check-dep-budgets.sh` | **ADDED** — setup budget 14 (actual 11) | `scripts/check-dep-budgets.sh:28` |
| `go mod tidy` (setup) | **CLEAN** — identity-model promoted to direct | Verified via `go mod tidy -diff` |
| `go vet` (setup) | **PASS** — 0 issues | Verified post-report |
| `check-docs-links.sh` | **PASS** — 187 links | Verified |
| `check-domain-counts.sh` | **PASS** — no drift | Verified |

### Documentation updates

- **CHANGELOG.md**: Added entry under `[Unreleased] → Changed` (gate integration) + `[Unreleased] → Fixed` (13 lint issues).
- **TODO_LIST.md**: Removed both P1 items (#1 "Wire setup into CI" and #2 "setup module needs flake/CI integration"). Updated header stats (12 lint modules, setup coverage 85.3%/gate 80%).
- **AGENTS.md**: Added setup to Architecture module list, updated lint count 11→12, coverage gate count 10→11, added setup to "Root module local replaces" gotcha, noted setup as the model for SA1019 migration.
- **FEATURES.md**: Updated header stats (12 lint modules, setup coverage).
- **ROADMAP.md**: Updated header stats (12 lint modules, 11 coverage gates, setup 85.3%/gate 80%).

---

## b) PARTIALLY DONE

### Nothing in this session was left partially complete.

Both P1 TODO items are fully resolved. The setup module is wired into every gate that uses a hardcoded module list.

---

## c) NOT STARTED (noticed but out of scope)

- **`nix run .#test` (full race suite)** — I ran `coverage-gate` (per-module `go test ./... -count=1 -coverprofile=...` without `-race`) and setup's own `go test -race`, but did NOT run the full `nix run .#test` across all 14 suites simultaneously. Coverage-gate passed, which means all modules' tests pass, but without `-race` in the gate. The setup-specific race test passed independently.
- **CI YAML schema validation** — I added 5 edit blocks to `ci.yml` but did not run a YAML linter. The structure mirrors existing entries exactly, so it should be fine, but it was not formally validated.
- **SKILL.md check** — Quick grep confirmed no stale "11 lint" / "10 gate" references, but a full review was not done.

---

## d) TOTALLY FUCKED UP

### Nothing was irreversibly broken.

However, there are things I should have done differently:

1. **I almost missed 3 lint categories.** My initial mental model was "add setup to coverage-gate and CI." It wasn't until I ran `golangci-lint run` that I discovered 13 real issues (SA1019 ×3, errcheck ×7, mnd ×1, exhaustruct ×1, wrapcheck ×1). The TODO said "lint config" but I initially glossed over it. **Lesson:** always run the actual lint command before estimating scope.

2. **The gopls "identity-model should be direct" warning persisted** after my `go mod tidy` promoted it. This was a stale LSP cache, not a real issue — but I didn't conclusively verify this until the very end. I should have run `go mod tidy` immediately after the first edit, then confirmed the warning cleared.

3. **I didn't notice the untracked `systemadapter/` module until the final `git status`.** Another agent created it and added it to `go.work`. This is unrelated to my work, but it appeared in my diff output and I initially didn't flag it clearly enough. I correctly left it untouched, but the discovery was late — I should have run `git status` at the START of the session to understand the working tree state.

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements (this session)

1. **Run `git status` at session start.** I dove straight into research without checking what was already uncommitted. The `systemadapter/` module and `usermgmt/system_exports.go` were already in the working tree. A start-of-session `git status` would have given me full context.

2. **Run the FULL verification suite, not just targeted checks.** I ran coverage-gate, lint, build, module-isolation, dep-budgets, cqrs-lint, flake-check, vet, docs-links, domain-counts — but I did NOT run `nix run .#test` (the full race-enabled test suite). Coverage-gate runs tests without `-race`. For a change that touches test files, the full race suite matters.

3. **Estimate scope by running the actual commands first.** "Lint config" in the TODO was vague. Running `golangci-lint run` immediately would have revealed the 13 issues upfront and let me plan the full scope.

### Codebase improvements (noticed during this session)

4. **The `defer func() { _ = x.Close() }()` pattern is verbose.** Seven identical conversions in `setup_test.go`. An alternative: add `(*setup.Bundle).Close` to the errcheck `exclude-functions` list in `.golangci.yml`, matching how `(*os.File).Close`, `(*sql.DB).Close`, etc. are handled. This would be one config line instead of seven code changes. However, the explicit pattern is more visible and self-documenting — a tradeoff.

5. **Coverage thresholds are inconsistent.** The project has no single formula: root 90%, usermgmt 74%, identity-model 70%, dashboardui 60%, datastar 90%, adminui 66%, loginpage 79%, auth 80%, setup 80%. Some are tight (loginpage: 79.9% actual, 79% gate = 0.9% headroom), others are generous (dashboardui: 83.3% actual, 60% gate = 23.3% headroom). A documented policy ("gate = floor(actual - 5%)") would make future additions deterministic.

6. **CI workflow is repetitive.** Each module has 4-5 copy-pasted step blocks (build, test, coverage-check, lint) with identical `env:` blocks. A matrix strategy or a reusable composite action would eliminate ~200 lines of YAML duplication. This is a known tradeoff (GitHub Actions matrix limitations with `working-directory`), but worth revisiting.

7. **The `systemadapter/` module appeared during this session** (another agent's work). It's untracked and in `go.work`. This is a coordination risk — two agents working on module structure simultaneously. The auto-commit daemon already committed my setup work; if the other agent's work conflicts, rebasing could be messy.

8. **SA1019 suppression is a ticking bomb.** Adminui has 133 suppressed deprecation warnings, integration_test has 22. Setup now demonstrates the clean path (direct identity-model imports). The v5 re-export retirement (ADR-0047) will force this migration, but the longer it's deferred, the more call sites accumulate. Setup proves the migration is mechanical — it could be scripted.

9. **`check-dep-budgets.sh` has manual comments that drift.** The budget comments say "11 current" but dep counts change as `go mod tidy` resolves things. The script could auto-compute and display the current count (it already does in the output), making the comments advisory-only rather than authoritative.

10. **The setup module's `Bundle.Close()` now wraps the error** via `errorfamily.WrapInfrastructure`. This is correct for lint compliance, but it changes the error identity — callers doing `errors.Is(err, someSentinel)` on the inner error would need `errors.Unwrap()`. The `errorfamily.Error` type implements `Unwrap()`, so this works, but it's worth documenting that setup wraps close errors.

---

## f) Up to 50 things to get done next

### Immediate follow-up (this session's work)

1. **Run `nix run .#test`** — full race-enabled test suite across all 14 suites to confirm nothing broke.
2. **Commit the remaining doc updates** (AGENTS.md, FEATURES.md, ROADMAP.md, TODO_LIST.md) — currently unstaged.
3. **Validate CI YAML** — run `yamllint` or `actionlint` on `.github/workflows/ci.yml`.
4. **Investigate `systemadapter/` module** — understand what the other agent is building and whether it conflicts with setup or changes module counts.

### P1 — High impact (from TODO_LIST)

5. **Complete MySQL event-store support** — document in README, add `NewMySQLSetup` constructor. (Only remaining P1 item.)
6. **Document `RecommendedSecurityMiddleware()` recipe** in `docs/guides/leveraging-httputil.md` (P3 item, but quick).

### P2 — Medium impact (from TODO_LIST)

7. **Document Broadcaster duality** — expose underlying `sse.Broadcaster`, write `docs/guides/sse-and-datastar.md`.
8. **Add fullstack integration test** to `integration_test/` — mount adminui + dashboardui + loginpage against real Service.
9. **Create `cqrs-htmx/health/v4` module** — go-health + go-health-dashboard integration.
10. **Create `cqrs-htmx/auditlog/v4` module** — samber-do-auditlog integration.
11. **Write `docs/guides/fullstack-wiring.md`** — integration guide for setup + health + auditlog.
12. **Add remaining BuildFlow tools to devShell** — cspell, vitest, jest.
13. **Wire `check-codegen` and `check-templates` into CI** — currently blocked on templ version pinning and workspace mode.
14. **Migrate adminui to direct identity-model imports** — eliminates 133 SA1019 suppressions (v5 prerequisite).
15. **Migrate integration_test to direct identity-model imports** — eliminates 22 SA1019 suppressions.

### Quality and consistency

16. **Standardize coverage threshold policy** — document the formula for setting new module gates.
17. **Refactor CI workflow to use matrix/reusable workflows** — eliminate ~200 lines of copy-pasted YAML.
18. **Add `(*setup.Bundle).Close` to errcheck exclude** — simplify test code (alternative to the verbose `defer func() { _ = ... }()` pattern).
19. **Auto-compute dep budget comments** — make `check-dep-budgets.sh` self-documenting.
20. **Add `setup` to the `examples/samber-do-demo`** as an alternative composition pattern.
21. **Write a setup module guide** — `docs/guides/setup-module.md` showing the one-call composition root.
22. **Add setup module to README** — it's not mentioned in the root README.md.
23. **Review whether setup should have its own `.golangci.yml`** — currently inherits root's config. Adminui, loginpage, dashboardui, datastar, integration_test, and identity-model all have their own. Setup is the only lint-checked module without one.
24. **Add integration test for setup module** — test actual HTTP requests through the full Bundle.Mount pipeline with auth providers.
25. **Benchmark setup.New() startup time** — composition root creates many sub-systems; ensure it's fast.

### Technical debt

26. **Add cqrs-lint strict CI gate** — blocked on Nix-only binary distribution (P3).
27. **Cross-module dep version drift** — bump all cross-module refs before next release tag.
28. **Re-investigate datastar/go-sse architecture** — ADR or migration (P3).
29. **Add golines alignment to `nix fmt`** — catch alignment drift automatically.
30. **Consider Go-based markdown link checker** — goldmark-based, more robust than awk.

### v5 preparation

31. **Plan re-export layer retirement** — setup demonstrates the pattern; script the adminui migration.
32. **Evaluate whether setup should absorb the `examples/samber-do-demo` patterns** — DI composition vs. setup's direct composition.
33. **Consider a `setup/v4` sub-package for SQL-backed stores** — `setup/sql` with SQLite/Postgres/MySQL defaults.
34. **Add OpenAPI spec generation to setup** — auto-generate spec from mounted routes.
35. **Add graceful shutdown to setup.Bundle** — `Shutdown(ctx)` with timeout.

### Documentation

36. **Update SKILL.md with setup module** — coverage guide, quick-start recipe.
37. **Add setup to FEATURES.md module inventory** — it's in AGENTS but not FEATURES.
38. **Write ADR for setup module** — document the composition root decision.
39. **Add setup to the fullstack-wiring guide decision tree** — "do you need one-call setup? → use setup/v4".
40. **Document the middleware ordering** that setup.Bundle.Middleware() applies.

### Testing

41. **Add race test for setup.Bundle.Close()** — concurrent close calls.
42. **Add test for setup with real SQL read model** — not just in-memory.
43. **Add test for setup with WebAuthn/TOTP/OAuth2 providers** — currently all nil.
44. **Add test for setup.Bundle.Mount route conflicts** — what happens if consumer routes overlap.
45. **Fuzz test setup.Config.withDefaults()** — ensure no panic on weird inputs.

### Observability

46. **Add structured logging to setup.New()** — log which components were created.
47. **Add startup health check to setup.Bundle** — verify all sub-systems are healthy after New().
48. **Expose setup.Bundle component count** — for debugging "what got wired".
49. **Add setup.Bundle.Stats()** — return a snapshot of wired components (for dashboards).
50. **Add pprof endpoint option to setup.Bundle.Mount()** — development profiling.

---

## g) Questions I CANNOT figure out myself

1. **Should the setup coverage gate threshold be 80% or higher?** I chose 80% (actual 85.3%, 5.3% headroom). Other modules range from 60% (dashboardui, 23% headroom) to 90% (root/datastar, ~3% headroom). There's no documented formula. Should I tighten to 83% (matching the ~2-3% headroom pattern of root/datastar), or is 80% the right amount of breathing room for a new module?

2. **Should setup get its own `.golangci.yml`?** Every other lint-checked module (adminui, loginpage, dashboardui, datastar, integration_test, identity-model) has its own config file. Setup currently inherits the root config. It works (0 issues), but it's the only module without one. Is this intentional (setup is "close to root") or an oversight?

3. **The untracked `systemadapter/` module (another agent's work) appeared during this session.** Should I investigate whether it affects the setup module or the gate counts I just updated, or leave it entirely to the other agent? It's in `go.work` now, which means `forEachGoModule` apps already include it — but the hardcoded-list gates (coverage-gate, dep-budgets, module-isolation) do NOT include it yet.

---

_Reporting basis: this session's work only. No external research conducted beyond verifying gate commands._
