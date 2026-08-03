# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-08-03 | **Version:** v4.6.1 (19 modules in `go.work`; see AGENTS.md for per-sub-module versions) | **Coverage:** Root 93.3% (gate 90%), openapi 99.0%, usermgmt 81.6% (gate 74%), identity-model 74.9% (gate 70%), dashboardui 84.0% (gate 60%), datastar 96.7% (gate 90%) — recompute via `nix run .#coverage-gate` | **Lint:** All 19 modules at 0 issues. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P2 — Medium Impact (code quality & tooling)

- [ ] **Upgrade cqrs-lint from Nix v0.2.2 to latest build.** The installed binary (`/run/current-system/sw/bin/cqrs-lint` v0.2.2) lacks comma-separated rule ID support (`//cqrs-lint:ignore(E004,E006)`). This causes 4 stale-suppression warnings in `examples/dashboard-demo/` where dual-rule findings need two separate comment lines. The go-cqrs-lite source (`5ee3832e`) already implements comma-separated parsing. Upgrading eliminates all 4 stale warnings cleanly. Source: `docs/status/2026-07-31_03-41_cqrs-lint-suppression-remediation-round2.md`.

- [~] **MySQL event-store support via go-cqrs-lite Dialect.** Event-store-only MySQL dialect added (`MySQLDialect` with `?` placeholders, MySQL-specific DDL, `IsDuplicateKeyError` extended, `classifyMySQLError` error classifier implemented in go-cqrs-lite `storage/sql/classify_init.go`). `dialectToUpstream` updated. `storage/v4` at v4.5.0. Read model constructors shipped (`NewMySQLUserReadModel`, etc.). Remaining: (1) add integration test against real MySQL (docker/testcontainers), (2) document MySQL support in README, (3) consider `NewMySQLSetup` convenience constructor and MySQL-backed session/snapshot/checkpoint stores.

- [ ] **Add datastar module to GitHub Actions CI workflow.** `.github/workflows/ci.yml` was expanded to include identity-model, dashboardui, and loginpage but does NOT include the datastar module (which didn't exist when CI was last updated). Add build+test+lint+coverage (90% threshold) for datastar. Source: `docs/status/2026-08-03_19-34_datastar-cleanup-session-3-status.md`.

- [ ] **Fix templ version mismatch between buildflow pre-commit hook and nix.** The pre-commit hook's `go-generate` step regenerates `_templ.go` files with directory-prefixed `FileName:` fields, conflicting with the nix templ version's bare filenames. Forces `--no-verify` on templ-touching commits. Fix by either (a) pinning the same templ version in buildflow as in nix, or (b) excluding `*_templ.go` from buildflow's go-generate step in `.buildflow.yml`. Recurring since 2026-08-01 (documented in AGENTS.md).

---

## P3 — Technical Debt & Future

- [ ] **Add cqrs-lint strict CI gate to GitHub Actions.** Run cqrs-lint `--strict` in CI to catch catalog/validation findings early. Blocked: cqrs-lint is a Nix-only binary; needs either a Go-installable distribution or a Nix CI runner. The flake.nix `check-cqrs-lint` app exists for local use.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
