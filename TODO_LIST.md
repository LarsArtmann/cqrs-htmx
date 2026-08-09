# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision, v5 plans, and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-08-09 | **Version:** v4.7.0 (released 2026-08-07) + `[Unreleased]` (security middleware consolidation, httputil v0.11.0) | **Modules:** 21 in `go.work` | **`*Service` methods:** 72 (leading v5 indicator; see ROADMAP) | **Coverage:** Root ~93% (gate 90%), openapi 99.0%, usermgmt 81.6% (gate 74%), identity-model 74.9% (gate 70%), dashboardui 83.3% (gate 60%), datastar 97.4% (gate 90%) — recompute via `nix run .#coverage-gate` | **Lint:** All 11 lint-checked modules at 0 issues (2026-08-09). Recompute uncapped: `GOEXPERIMENT=jsonv2 golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 ./...` per module.

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P1 — High impact (release follow-through & doc health)

- [ ] **Create `examples/fullstack-demo/` — the kitchen-sink composition proof.** Mount root App + usermgmt Service + totp + adminui + dashboardui + loginpage + SSE broadcaster in one runnable example. Proves all modules compose, surfaces integration friction, serves as the copy-paste template for real apps. No example currently wires more than 2 UI modules together. Source: `docs/architecture-understanding/2026-08-09_05-36_module-integration-composability.html` (Finding 5).

- [ ] **Create `cqrs-htmx/setup/v4` module — internal wiring kit.** New optional Go module (zero external deps) providing: `setup.DashboardConfig(svc *usermgmt.Service) dashboardui.Config` (eliminates 12-field manual wiring), `setup.MountAll(mux, opts) *MountedPanels` (mounts admin + dashboard + login + auth routes in one call), `setup.FullstackConfig` struct. Depends only on internal siblings. Added to `go.work`; flake.nix `forEachGoModule` apps pick it up automatically. Source: architecture review Finding 1 + 2.

- [~] **Complete MySQL event-store support.** Dialect, read-model constructors, setup template, migration guide, integration test (testcontainers), and error classifier are done (`MySQLDialect`, `NewMySQL*ReadModel`, `mysql_setup.go`, `docs/guides/mysql-setup.md`, `classifyMySQLError`, `mysql_integration_test.go`). Remaining: (1) document MySQL support in the root README; (2) `NewMySQLSetup` convenience constructor + MySQL-backed session/snapshot/checkpoint stores. Evidence: `usermgmt/sql_readmodel_mysql.go`, `go-cqrs-lite/storage/sql/dialect.go`.

---

## P2 — Medium impact (tooling & quality)

- [ ] **Document Broadcaster duality and expose underlying `sse.Broadcaster`.** Root's `cqrshtmx.Broadcaster` and datastar's `datastar.Broadcaster` both wrap `sse.Broadcaster[sse.Event]` but are separate types with no shared abstraction. Add a `Raw()` accessor (or interface) so consumers can share one fan-out hub for cross-transport scenarios. Write `docs/guides/sse-and-datastar.md`. Source: architecture review Finding 3.

- [ ] **Add fullstack integration test to `integration_test/`.** Mount adminui + dashboardui + loginpage against a real `*usermgmt.Service`. Verify: admin panel renders with seeded user, dashboard shows projection health, login page renders correct auth buttons based on Service config. No integration test currently mounts any UI module. Source: architecture review Finding 6.

- [ ] **Create `cqrs-htmx/health/v4` module — go-health + go-health-dashboard integration.** New optional Go module bridging usermgmt projection health → go-health checks. `health.NewProbe(svc, opts)` auto-registers `ProjectionStatusProvider` as a health check. `health.NewDashboard(probe, opts)` returns a pre-configured `*dashboard.Dashboard`. Separate module so consumers who don't need it pay zero dep cost. Source: architecture review §3.

- [ ] **Create `cqrs-htmx/auditlog/v4` module — samber-do-auditlog integration.** New optional Go module: `auditlog.WithAuditLog(opts) []do.HookProvider` for one-line DI audit logging. `auditlog.MountReport(mux, report)` for the HTML visualization viewer. Closes the gap explicitly noted in the samber-do-demo status doc. Source: architecture review §3.

- [ ] **Write `docs/guides/fullstack-wiring.md` — the integration guide.** Single guide showing the 4 integration paths (setup-only, +health, +auditlog, full-stack). Decision tree: "do you need health checks? → add health/v4". Cross-link from README and SKILL.md. Source: architecture review Step 7.

- [ ] **Add remaining BuildFlow tools to the flake devShell.** biome, shfmt, and nixfmt are now wired. Still missing: cspell (spell-checking), vitest, jest — needed for full `--no-verify`-free commits on JS/Markdown files. Source: `docs/status/archived/2026-08-05_02-35_templ-components-adoption-deepening.md`.
- [~] **Wire remaining `check-*` apps into CI.** `check-docs-links`, `check-service-methods`, `check-domain-counts`, `check-large-files`, and `check-phantom-version` now run in the CI `checks` and `security` jobs. Remaining: `check-codegen` (needs templ version pinning in CI), `check-templates` (needs workspace mode / local replaces), `check-cqrs-lint` (blocked — Nix-only binary; see P3.1).

- [ ] **Migrate adminui to direct identity-model imports.** Eliminates 133 SA1019 suppression warnings and the scoped text-based exclusion in `adminui/.golangci.yml`. v5 prerequisite — see ROADMAP "Re-export Layer Retirement". High effort (~26 files, ~133 call sites) but zero API change (type aliases are transparent).
- [ ] **Migrate integration_test to direct identity-model imports.** Eliminates 22 SA1019 suppression warnings. Same pattern as the adminui migration above.

---

## P3 — Technical debt & future

- [ ] **Add cqrs-lint strict CI gate to GitHub Actions.** Run cqrs-lint `--strict` in CI to catch catalog/validation findings early. Blocked: cqrs-lint is a Nix-only binary; needs a Go-installable distribution or a Nix CI runner. The flake.nix `check-cqrs-lint` app exists for local use.
- [ ] **Update `docs/guides/leveraging-httputil.md` with `RecommendedSecurityMiddleware()` recipe.** Document the zero-arg factory that bundles SecurityHeaders + Nonce + Recovery, and the `RecommendedPermissionsPolicy` constant. Both adminui and dashboardui delegate to it. Source: `docs/status/archived/2026-08-09_04-47_security-middleware-consolidation.md`.
- [ ] **Cross-module dep version drift after v4.7.0 tagging.** Published submodule tags reference stale sibling versions (e.g., `adminui/v4.7.0` still depends on `usermgmt/v4.6.1`). Next release should bump all cross-module refs before tagging. Source: `docs/status/archived/2026-08-07_22-23_release-v4.7.0-and-retry-investigation.md`.
- [ ] **Re-investigate datastar/go-sse architecture decision.** The prior analysis (`docs/status/archived/2026-08-07_06-25_datastar-go-sse-analysis-self-review.md`) claimed go-sse cannot produce Datastar wire format — but go-sse has `KeyedLines`/`SendKeyed`/`SendLines` designed for Datastar. The exclusion is a design choice (Patch coupling to SDK), not a technical incompatibility. Needs either an ADR documenting the decision or a migration to go-sse.
- [ ] **Add golines alignment to `nix fmt` pipeline.** `golines` is available but not integrated into treefmt. Would catch alignment drift automatically. May need `pkgs.golines` from nixpkgs or a wrapper.
- [ ] **Consider a Go-based markdown link checker.** The current `check-docs-links.sh` uses awk regex which handles common cases but may miss edge cases. A Go checker using goldmark would be more robust. The awk checker + test suite is sufficient for now.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision, v5 plans, and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
