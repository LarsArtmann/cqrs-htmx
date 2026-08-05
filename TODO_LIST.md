# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision, v5 plans, and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-08-05 | **Version:** v4.6.1 + `[Unreleased]` (WebSocket dropped — SSE-only; see ADR 0046) | **Modules:** 19 in `go.work` | **`*Service` methods:** 72 (leading v5 indicator; see ROADMAP) | **Coverage:** Root ~93% (gate 90%), openapi 99.0%, usermgmt 81.6% (gate 74%), identity-model 74.9% (gate 70%), dashboardui 84.0% (gate 60%), datastar 96.7% (gate 90%) — recompute via `nix run .#coverage-gate` | **Lint:** All lint-checked modules at 0 issues. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P1 — High impact (release follow-through & doc health)

- [~] **Complete MySQL event-store support.** Dialect, read-model constructors, setup template, migration guide, and error classifier are done (`MySQLDialect`, `NewMySQL*ReadModel`, `mysql_setup.go`, `docs/guides/mysql-setup.md`, `classifyMySQLError`). Remaining: (1) integration test against real MySQL via docker/testcontainers; (2) document MySQL support in the root README; (3) `NewMySQLSetup` convenience constructor + MySQL-backed session/snapshot/checkpoint stores. Evidence: `usermgmt/sql_readmodel_mysql.go`, `go-cqrs-lite/storage/sql/dialect.go`.

---

## P2 — Medium impact (tooling & quality)

- [ ] **Add remaining BuildFlow tools to the flake devShell.** biome, shfmt, and nixfmt are now wired. Still missing: cspell (spell-checking), vitest, jest — needed for full `--no-verify`-free commits on JS/Markdown files. Source: `docs/status/2026-08-05_02-35_templ-components-adoption-deepening.md`.
- [~] **Wire remaining `check-*` apps into CI.** `check-docs-links`, `check-service-methods`, `check-domain-counts`, `check-large-files`, and `check-phantom-version` now run in the CI `checks` and `security` jobs. Remaining: `check-codegen` (needs templ version pinning in CI), `check-templates` (needs workspace mode / local replaces), `check-cqrs-lint` (blocked — Nix-only binary; see P3.1).

---

## P3 — Technical debt & future

- [ ] **Add cqrs-lint strict CI gate to GitHub Actions.** Run cqrs-lint `--strict` in CI to catch catalog/validation findings early. Blocked: cqrs-lint is a Nix-only binary; needs a Go-installable distribution or a Nix CI runner. The flake.nix `check-cqrs-lint` app exists for local use.
- [ ] **Audit display-only structs for dead JSON tags.** `recentEvent` in dashboardui was one case (fixed). Other display-only structs rendered via templ/fmt may carry dead JSON tags that cause lint friction. Systematic `grep` for structs with `json:"..."` tags that are never marshaled.
- [ ] **Add golines alignment to `nix fmt` pipeline.** `golines` is available but not integrated into treefmt. Would catch alignment drift automatically. May need `pkgs.golines` from nixpkgs or a wrapper.
- [ ] **Consider a Go-based markdown link checker.** The current `check-docs-links.sh` uses awk regex which handles common cases but may miss edge cases. A Go checker using goldmark would be more robust. The awk checker + test suite is sufficient for now.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision, v5 plans, and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
