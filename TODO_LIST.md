# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision, v5 plans, and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-08-12 | **Version:** v4.7.0 (released 2026-08-07) + `[Unreleased]` (async projection startup, ActorID consolidation/ADR-0111, ADR-0114 tombstone migration, security middleware consolidation, httputil v0.11.0, setup module CI integration, Broadcaster Raw() accessor, fullstack UI integration test, systemadapter module + system/metaengine integration) | **Modules:** 24 in `go.work` (13 production + 10 examples + e2e/server) | **`*Service` methods:** 73 (leading v5 indicator; see ROADMAP) | **⚠️ BUILD BROKEN:** go-cqrs-lite master (`af4b60841`) reverted ADR-0111 API — root/usermgmt/identity-model/setup fail to compile. Coverage/lint numbers below are from session 21-19 (pre-drift), marked `[unverified]`. Fix requires adapting to reverted API or restoring go-cqrs-lite types. | **Coverage (last verified 2026-08-12 session 21-19):** Root ~93% (gate 90%) `[unverified]`, openapi 99.0%, usermgmt ~81% (gate 74%) `[unverified]`, identity-model ~75% (gate 70%) `[unverified]`, dashboardui ~83% (gate 60%), datastar 97.4% (gate 90%), setup ~88% (gate 80%) `[unverified]` | **Lint:** 0 issues across 12 modules (2026-08-09, pre-drift). systemadapter excluded (work-in-progress).

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P0 — Blocking (build broken)

- [ ] **Fix go-cqrs-lite upstream drift (ADR-0111 API reverted).** go-cqrs-lite master (`af4b60841`, committed 2026-08-12 21:42) reverted the ADR-0111 branded ActorID types: `event.WithActor` is gone (only `event.WithUserID` exists), `id.ActorID`/`id.ActorKind`/`id.ActorUser` are gone, `record.CommonMetadata.ActorID` reverted from branded struct to plain `string`. All 4 core modules (root, usermgmt, identity-model, setup) fail to compile. Fix requires EITHER adapting cqrs-htmx to the reverted API (downgrade ADR-0111) OR restoring go-cqrs-lite to have the ADR-0111 types. Evidence: `context.go:246` (`event.WithActor`), `audit_log.go:24` (`id.ActorID`), `identity-model/id.go:97` (`id.ActorKind`). Source: `docs/status/2026-08-12_21-54_docs-health-audit-self-review.md`.

---

## P1 — High impact (release follow-through & doc health)

- [~] **Complete MySQL event-store support.** Dialect, read-model constructors, setup template, migration guide, integration test (testcontainers), and error classifier are done (`MySQLDialect`, `NewMySQL*ReadModel`, `mysql_setup.go`, `docs/guides/mysql-setup.md`, `classifyMySQLError`, `mysql_integration_test.go`). Remaining: (1) document MySQL support in the root README; (2) `NewMySQLSetup` convenience constructor + MySQL-backed session/snapshot/checkpoint stores. Evidence: `usermgmt/sql_readmodel_mysql.go`, `go-cqrs-lite/storage/sql/dialect.go`.

- [ ] **Run full nix verification gates after recent changes.** The ActorID consolidation (ADR-0111), ADR-0114 tombstone migration, async projection startup, and go-cqrs-lite snapshot breakage repair were each verified with per-module `go test`/`golangci-lint`, but the full workspace gates (`nix run .#coverage-gate`, `nix run .#lint`, `nix run .#check-cqrs-lint`, `nix run .#test-fuzz`, `nix run .#test-flake`, `nix flake check --no-build`) have not been run as a unified suite since 2026-08-09. Source: `docs/status/archived/2026-08-12_21-19_async-projection-startup-verified.md`.

---

## P2 — Medium impact (tooling & quality)

- [ ] **`systemadapter/v4` lint remediation + gate integration.** The work-in-progress systemadapter module has 104 lint issues (contextcheck ×10, SA1019 ×43, exhaustruct ×23, err113 ×4, errcheck ×4, mnd ×4, wsl_v5 ×7, wrapcheck ×2, goimports ×3, gci ×1, nlreturn ×3). The module is currently excluded from `nix run .#lint` via flake.nix regex. Remediation requires: (1) migrate `usermgmt.*State` to direct `identity-model` imports (43 SA1019), (2) add `.golangci.yml` with appropriate exclusions or add nolint directives, (3) use `errorfamily` constructors instead of `fmt.Errorf` (err113), (4) add coverage-gate threshold, (5) add to CI workflow, `check-module-isolation.sh`, `check-dep-budgets.sh`. Evidence: `systemadapter/domain_config.go`, `systemadapter/projections.go`. Source: `docs/status/2026-08-09_08-36_system-metaengine-integration.md`.

- [ ] **Expand fullstack UI integration test.** The 3-test suite in `integration_test/fullstack_ui_test.go` covers only rendering and unauthenticated blocking. Missing assertions (from original TODO spec): (1) admin panel renders WITH a seeded user (register → session → GET /admin/ → assert user data in HTML), (2) dashboard shows projection health data (assert projection names/health appear in rendered HTML), (3) login page renders correct auth buttons based on Service config (configure with TOTP → verify button appears; without → verify absence). Evidence: `integration_test/fullstack_ui_test.go`. Source: `docs/status/2026-08-09_08-36_broadcaster-raw-api-and-fullstack-ui-test.md`.

- [ ] **Create `cqrs-htmx/health/v4` module — go-health + go-health-dashboard integration.** New optional Go module bridging usermgmt projection health → go-health checks. `health.NewProbe(svc, opts)` auto-registers `ProjectionStatusProvider` as a health check. `health.NewDashboard(probe, opts)` returns a pre-configured `*dashboard.Dashboard`. Separate module so consumers who don't need it pay zero dep cost. Source: architecture review §3.

- [ ] **Create `cqrs-htmx/auditlog/v4` module — samber-do-auditlog integration.** New optional Go module: `auditlog.WithAuditLog(opts) []do.HookProvider` for one-line DI audit logging. `auditlog.MountReport(mux, report)` for the HTML visualization viewer. Closes the gap explicitly noted in the samber-do-demo status doc. Source: architecture review §3.

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
- [ ] **Rewrite `origin/v4` branch history to strip 3 remaining binary blobs (~27.7 MB).** The master branch was cleaned via `git filter-repo` (731.8 MB → 0 blobs), but the `v4` branch has independent binary contamination (`examples/basic/basic` 9.8MB, `examples/datastar-demo/datastar-demo` 8.9MB ×2) that does not share ancestry with master. Requires `git filter-repo` on the v4 branch + force-push. Source: `docs/status/2026-08-09_06-15_git-binary-cleanup-history-rewrite.md`.
- [ ] **Write ADR-0048: Liveness/Readiness Decoupling.** Document the architectural decision to decouple HTTP server liveness from projection readiness via `AsyncStartup` + `ProjectionReadinessCheck`. The repo has 47+ ADRs; this decoupling is architecturally significant. Source: `docs/status/archived/2026-08-12_21-19_async-projection-startup-verified.md`.
- [ ] **Cross-reference async startup from `docs/guides/projection-health-monitoring.md`.** The existing projection health guide doesn't link to the new `docs/guides/async-projection-startup.md` guide. Quick doc cross-reference.
- [ ] **Extract `ActorID.AsUserID() (UserID, bool)` helper.** Three call sites (`audit_log.go`, `authz_roles.go`, `session.go`) have the same `if actorID.Kind() == ActorUser { NewUserID(actorID.String()) }` kind-guard pattern. A helper would eliminate boilerplate and prevent future bugs (forgetting the guard). Source: `docs/status/archived/2026-08-11_17-31_actorid-cleanup-self-review.md`.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision, v5 plans, and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
