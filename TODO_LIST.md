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

- [~] **MySQL event-store support via go-cqrs-lite Dialect.** Event-store-only MySQL dialect added (`MySQLDialect` with `?` placeholders, MySQL-specific DDL, `IsDuplicateKeyError` extended, `classifyMySQLError` error classifier implemented in go-cqrs-lite `storage/sql/classify_init.go`). `dialectToUpstream` updated. `storage/v4` at v4.5.0. Read model constructors shipped (`NewMySQLUserReadModel`, etc.). Remaining: (1) add integration test against real MySQL (docker/testcontainers), (2) document MySQL support in README, (3) consider `NewMySQLSetup` convenience constructor and MySQL-backed session/snapshot/checkpoint stores.

- [ ] **Auto-discover modules from go.work in flake.nix.** Replace hardcoded module lists in flake.nix apps (`test`, `lint`, `coverage`, `build`, etc.) with dynamic discovery via `go work edit -json | jq -r '.Mapping[].DiskPath'`. Currently each new module requires manual updates to 6+ flake.nix apps AND `.github/workflows/ci.yml`. Requires an architecture decision: the shell-in-Nix pattern makes dynamic generation non-trivial. Documented in AGENTS.md gotchas.

- [ ] **Single-source domain model counts.** Generate event/command counts (currently "21 events, 20 commands") from identity-model code instead of hardcoding in 5+ docs (AGENTS.md, ROADMAP.md, FEATURES.md, TODO_LIST.md, guides). A small Go script counting struct types in `events.go`/`commands.go` could produce a canonical number.

- [ ] **Add golines alignment to nix fmt pipeline.** `golines` is available (`~/go/bin/golines`) but not integrated into `nix fmt` (treefmt). Adding it would catch alignment drift automatically. May need `pkgs.golines` from nixpkgs or a wrapper.

- [ ] **Audit display-only structs for dead JSON tags.** `recentEvent` in dashboardui was one case (fixed). Other display-only structs rendered via templ/fmt may carry dead JSON tags that cause lint friction. Systematic `grep` for structs with `json:"..."` tags that are never marshaled.

- [ ] **Add `nix run .#check-docs-links` to pre-commit hook.** Catch broken markdown links before commit. Risk: scans ALL .md files (not just staged), so pre-existing broken links could block unrelated commits. Consider running only on staged `.md` files or adding a `--staged-only` flag.

---

## P1 — High Impact (structural debt from 2026-08-05 architecture review)

- [ ] **Extract OAuth2 sub-service from `*Service` as v5 decomposition prototype.** `*Service` has 74 methods (+22 since ADR-0038 on 2026-07-19, trending to ~100 by October). ADR-0038 defers full decomposition to v5, but extracting one focused sub-service validates the composition pattern within v4. OAuth2 (8 methods in `service_oauth2.go`) is the best candidate: self-contained, few cross-domain dependencies. Create an `OAuth2Service` struct that holds the OAuth2-specific fields (providers, stateStore, successURL, errorURL), extract the 8 methods, and have `*Service` embed or hold it. This is a same-package struct decomposition (not a sub-package move — that's blocked by ADR-0019). Validates ADR-0038's proposed pattern before committing to v5. Evidence: `docs/architecture-understanding/2026-08-05_01-40_deep-architecture-review.html`, `docs/modularization/2026-08-05_PROPOSAL.html`.

- [ ] **Remove identity-model re-export layer in v5.** 26 usermgmt files re-export identity-model types (160 deprecated markers added 2026-08-05). Removal is a breaking change (consumers must switch `usermgmt.UserID` → `identitymodel.UserID`), bundled with the v5 major bump alongside ADR-0019/0038. Track: no consumer has requested this; v5 is undated. The deprecation markers are the retirement path — they signal consumers to migrate now. Decision: remove in v5 (confirmed by maintainer 2026-08-05).

- [ ] **Remove httputil re-export layer in v5.** 3 root-module files (`csrf_reexport.go`, `ratelimit_reexport.go`, `server_timing_reexport.go`) re-export 39 symbols from `github.com/larsartmann/httputil`. All now carry `// Deprecated:` markers (added 2026-08-05). Internal callers, examples, tests, and docs migrated to direct `httputil.*` imports. Removal bundled with the v5 major bump. SecurityHeaders split-brain resolved: httputil enriched with richer config fields (pending v0.9.0 publish); `security.go` now deprecated alias + delegating wrapper. **Remaining:** publish httputil v0.9.0, bump cqrs-htmx `go.mod`, remove `go.work` replace. See `docs/guides/leveraging-httputil.md` for the migration table.

---

## P3 — Technical Debt & Future

- [ ] **Add cqrs-lint strict CI gate to GitHub Actions.** Run cqrs-lint `--strict` in CI to catch catalog/validation findings early. Blocked: cqrs-lint is a Nix-only binary; needs either a Go-installable distribution or a Nix CI runner. The flake.nix `check-cqrs-lint` app exists for local use.

- [ ] **Consider a Go-based markdown link checker.** The current `check-docs-links.sh` uses awk regex extraction which handles common cases but may miss edge cases (nested brackets, multi-line links). A Go-based checker using a proper Markdown parser (goldmark) would be more robust. The awk checker + test suite (`scripts/test-check-docs-links.sh`) is sufficient for now.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
