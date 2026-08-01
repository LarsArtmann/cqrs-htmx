# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-08-01 | **Version:** v4.6.1 (18 modules in `go.work`; see AGENTS.md for per-sub-module versions) | **Coverage:** Root 93.7% (gate 90%), openapi 99.0%, usermgmt 81.6% (gate 74%), identity-model 74.9% (gate 70%), dashboardui 84.0% (gate 60%) — recompute via `nix run .#coverage-gate` | **Lint:** All 18 modules at 0 issues. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P2 — Medium Impact (code quality & tooling)

- [ ] **Upgrade cqrs-lint from Nix v0.2.2 to latest build.** The installed binary (`/run/current-system/sw/bin/cqrs-lint` v0.2.2) lacks comma-separated rule ID support (`//cqrs-lint:ignore(E004,E006)`). This causes 4 stale-suppression warnings in `examples/dashboard-demo/` where dual-rule findings need two separate comment lines. The go-cqrs-lite source (`5ee3832e`) already implements comma-separated parsing. Upgrading eliminates all 4 stale warnings cleanly. Source: `docs/status/2026-07-31_03-41_cqrs-lint-suppression-remediation-round2.md`.

- [~] **MySQL event-store support via go-cqrs-lite Dialect.** Event-store-only MySQL dialect added (`MySQLDialect` with `?` placeholders, MySQL-specific DDL, `IsDuplicateKeyError` extended, `classifyMySQLError` error classifier implemented in go-cqrs-lite `storage/sql/classify_init.go`). `dialectToUpstream` updated. `storage/v4` at v4.5.0. Read model constructors shipped (`NewMySQLUserReadModel`, etc.). Remaining: (1) add integration test against real MySQL (docker/testcontainers), (2) document MySQL support in README, (3) consider `NewMySQLSetup` convenience constructor and MySQL-backed session/snapshot/checkpoint stores.

- [~] **State cache invalidation-after-write test.** WithStateCache is wired in all 4 usermgmt repositories. Two tests exist (`TestStateCache_InterceptsWritePathLoad`, `TestSnapshot_WritePathConsultsSnapshot_OnCacheMiss`). Remaining: verify cache is busted on command dispatch (not just on service restart), and write a benchmark quantifying first (cold) vs second (cache hit) Execute.

- [ ] **Add catalog-demo smoke test.** `examples/catalog-demo/` has zero test files — the only example module without any test coverage. A basic `main_test.go` that imports and initializes the demo would catch build-breakage. Source: docs/status/2026-07-28_09-58.

- [ ] **Make errorfamily CI gate comment-aware.** The current ripgrep-based `nix run .#errorfamily` scan matches `errors.New(`/`fmt.Errorf(` inside Go comments and docstrings (caused a false positive on `readiness.go`). Either filter comments properly or use a Go AST-based scanner.

---

## P3 — Technical Debt & Future

- [ ] **Add phantom-version CI gate to .buildflow.yml.** Detect go-cqrs-lite zero pseudo-version issues at build time. Source: planning doc.

- [ ] **Add cqrs-lint strict CI gate to .buildflow.yml.** Run cqrs-lint `--strict --verbose` in CI to catch catalog/validation findings early. Source: planning doc.

- [ ] **Create `loginpage/CHANGELOG.md`.** All other sub-modules (totp, webauthn, oauth2, identity-model) have CHANGELOGs. loginpage is the only sub-module without one. Either create it with entries aligned to root releases, or document the decision to leave it absent.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
