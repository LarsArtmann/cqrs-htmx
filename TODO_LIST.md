# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision, v5 plans, and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-08-05 | **Version:** v4.6.1 + `[Unreleased]` (WebSocket dropped — SSE-only; see ADR 0046) | **Modules:** 19 in `go.work` | **`*Service` methods:** 72 (leading v5 indicator; see ROADMAP) | **Coverage:** Root ~93% (gate 90%), openapi 99.0%, usermgmt 81.6% (gate 74%), identity-model 74.9% (gate 70%), dashboardui 84.0% (gate 60%), datastar 96.7% (gate 90%) — recompute via `nix run .#coverage-gate` | **Lint:** All lint-checked modules at 0 issues. Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P0 — Blocking (hermetic build & release)

- [ ] **Publish httputil v0.9.0, bump cqrs-htmx `go.mod`, remove the `go.work` replace.** The hermetic `nix run .#build`/`.#test` (GOWORK=off) is currently broken: `security.go` references `httputil.RecommendedHSTS`/`RecommendedCSP`/`SecurityHeaderSkip`/`ContentTypeOptions` added in the pending v0.9.0 (root `go.mod` is still at v0.8.0). The `go.work` TEMPORARY replace points at `/home/lars/projects/httputil`. Steps: tag httputil v0.9.0, bump root + all sub-module `go.mod`, `go mod tidy`, remove replace, verify `nix run .#test` passes hermetically. Evidence: `go.work:133-138`, AGENTS.md "httputil leverage" gotcha.

---

## P1 — High impact (release follow-through & doc health)

- [~] **Complete MySQL event-store support.** Dialect, read-model constructors, setup template, migration guide, and error classifier are done (`MySQLDialect`, `NewMySQL*ReadModel`, `mysql_setup.go`, `docs/guides/mysql-setup.md`, `classifyMySQLError`). Remaining: (1) integration test against real MySQL via docker/testcontainers; (2) document MySQL support in the root README; (3) `NewMySQLSetup` convenience constructor + MySQL-backed session/snapshot/checkpoint stores. Evidence: `usermgmt/sql_readmodel_mysql.go`, `go-cqrs-lite/storage/sql/dialect.go`.

---

## P2 — Medium impact (tooling & quality)

- [ ] **Publish the `datastar/v4` git tag.** The module is release-ready (71 tests, 96.7% coverage, 0 lint, no local replaces, GOWORK=off clean) but has no published tag — consumers outside the workspace cannot `go get` it. Requires stripping local `replace` directives from demo/integration_test `go.mod` first. Source: ROADMAP "Datastar Future Scope".
- [~] **Wire remaining `check-*` apps into CI.** `check-docs-links`, `check-service-methods`, `check-domain-counts`, `check-large-files`, and `check-phantom-version` now run in the CI `checks` and `security` jobs. Remaining: `check-codegen` (needs templ version pinning in CI), `check-templates` (needs workspace mode / local replaces), `check-cqrs-lint` (blocked — Nix-only binary; see P3.1).
- [ ] **Add missing BuildFlow tools to the flake devShell.** biome, shfmt, nixfmt, cspell, vitest, jest are "not found" in the devShell, forcing `--no-verify` on commits that touch JS/Markdown. Source: `docs/status/2026-08-05_02-35_templ-components-adoption-deepening.md`.
- [ ] **Close `dashboardui/core/` coverage gaps + add to the coverage gate.** `core/` is at 67.2% and ungated. Untested: `ListStreamsPaged` (0%), the `FetchOverview` ProjectionHost health-classification path (48.9%), `ProjectionStats` (25%), `DLQProjectionLinks` with a real host (16.7%), CBOR payload rendering. Source: `docs/status/2026-08-05_06-32_dashboardui-core-extraction-phase2-complete.md`.
- [ ] **Auto-discover modules from `go.work` in `flake.nix`.** Replace the hardcoded module lists in 6+ flake.nix apps (`test`, `lint`, `coverage`, `build`, `test-flake`, `test-fuzz`, `check-cqrs-lint`) with dynamic discovery via `go work edit -json | jq -r '.Mapping[].DiskPath'`. Requires an architecture decision (shell-in-Nix pattern). Documented in AGENTS.md gotchas.
- [ ] **Add httputil `SecurityHeaders` field tests.** `PermissionsPolicy`, `Custom`, `ContentTypeOptions` precedence, and the `SecurityHeaderSkip` (`"-"`) sentinel are additive new branches untested in httputil (pending v0.9.0). Source: `docs/status/2026-08-05_11-09_httputil-adoption-100-session-complete.md`.

---

## P3 — Technical debt & future

- [ ] **Add cqrs-lint strict CI gate to GitHub Actions.** Run cqrs-lint `--strict` in CI to catch catalog/validation findings early. Blocked: cqrs-lint is a Nix-only binary; needs a Go-installable distribution or a Nix CI runner. The flake.nix `check-cqrs-lint` app exists for local use.
- [ ] **Audit display-only structs for dead JSON tags.** `recentEvent` in dashboardui was one case (fixed). Other display-only structs rendered via templ/fmt may carry dead JSON tags that cause lint friction. Systematic `grep` for structs with `json:"..."` tags that are never marshaled.
- [ ] **Add golines alignment to `nix fmt` pipeline.** `golines` is available but not integrated into treefmt. Would catch alignment drift automatically. May need `pkgs.golines` from nixpkgs or a wrapper.
- [ ] **Consider a Go-based markdown link checker.** The current `check-docs-links.sh` uses awk regex which handles common cases but may miss edge cases. A Go checker using goldmark would be more robust. The awk checker + test suite is sufficient for now.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision, v5 plans, and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
