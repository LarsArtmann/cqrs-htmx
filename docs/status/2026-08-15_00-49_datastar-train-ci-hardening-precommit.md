# Status Report — Datastar Train, CI Hardening, Pre-Commit Repair

**Date:** 2026-08-15 00:49
**Session:** Continuation after 2026-08-14 blob-purge/push follow-through (`56068b16`)
**Scope:** Execute the post-push follow-up backlog: datastar replace-strip train, check-templates CI wiring, integration_test identity-model migration, pre-commit hook fix, full gate suite.

---

## a) FULLY DONE

### 1. State verification (opening step)

- **Master is synced** — user pushed all 3 pending commits (`d261a80c`, `a2d75cee`, `56068b16`); local == remote at `56068b16`.
- **go-datastar sibling is clean, synced, and the user ALREADY cut + pushed `static/v0.2.0`** (at `5b70bb1`) and `v0.2.0` main tag — the entire datastar train unblocked itself. Q1 of the previous report is answered for the datastar half: no tags needed from me.
- **go-cqrs-lite sibling is mid-refactor and DIRTY** (uncommitted ADR-0128 codec-extraction: `codec/`, `flightrecorder/`, `retry/` dirs deleted, ~70 modified files, 2 unpushed commits). Per standing rules I did not touch it. §4 item 7 (metaengine tag train) stays blocked on that session finishing.

### 2. Datastar replace-strip train (runbook §4 item 8) — commit `f128072d`

Stripped go-datastar-family dev replaces (`go-datastar`, `go-sse`, `go-datastar/static`) from all 4 consumers, each verified hermetically (`GOWORK=off go mod tidy && go build ./... && go vet ./...`):

| Module                   | Replaces dropped                                          | Extra verification   |
| ------------------------ | --------------------------------------------------------- | -------------------- |
| `datastar`               | all 3 (zero replaces remain)                              | full test suite pass |
| `examples/datastar-demo` | 3 upstream (kept `datastar/v4 => local` until tag pushed) | build + vet          |
| `integration_test`       | 3 upstream (kept `datastar/v4 => local`)                  | build + vet          |
| `health`                 | static replace + its TEMPORARY comment block              | build + vet + tests  |

`check-modules` gate green (isolation, budgets, strict drift, replaces, docs-freshness, links).

### 3. Local `datastar/v4.8.0` tag cut for review

- Signed (`gpg.format=ssh`), message stripped of any embedded SSH signature block (verified: exactly 1 `BEGIN SSH SIGNATURE` occurrence = the tag's own).
- Points at `f128072d` (the strip commit). **NOT pushed** — pushing is the user's call.

### 4. check-templates wired into CI — commit `e7aeee97`

- **Root-caused why it wasn't CI-ready:** the old script mutated `go.work` with an absolute-path `stack/mysql` replace (sibling-only) and required `stack/mysql v4.2.0` **which does not exist upstream** (max: `v4.1.0`). Also discovered published `stack/postgres v4.2.0` is broken in isolation (references unreleased `storage/v4` API: `NotificationListener`, `NewPostgresBus`) — it only compiled via local workspace replaces. `v4.3.0` fixed it.
- **Rewrote `scripts/check-templates.sh`:** now fully hermetic (`GOWORK=off`, runs in `usermgmt/` dir, resolves `sqlite v4.3.0` / `postgres v4.3.0` / `mysql v4.1.0` from the proxy, adds `go mod tidy` step the old workspace-mode masked, version notes documented in header). Identical behavior local + CI.
- **Added `SQL setup template compile check` step** to the `checks` job in `.github/workflows/ci.yml` (after codegen drift).
- Verified locally: script passes AND restores tree cleanly (trap-based restore).

### 5. integration_test → direct identity-model imports — commit `f6599348`

- Enumerated raw SA1019s (was reported as 22; actual: **31 findings, 24 symbols, 10 test files**).
- Pre-verified safety: all flagged `usermgmt.*` symbols are transparent aliases (`ErrForbidden = identitymodel.ErrForbidden`) or delegating wrappers (`func ParseUserID(...) { return identitymodel.ParseUserID(...) }`) — zero behavior change.
- Migrated all 31 sites to `identitymodel.*` (matching adminui's import style), added the import to all 10 files, dropped the now-dead staticcheck text exclusion from `integration_test/.golangci.yml`.
- Verified: build, vet, staticcheck (0 SA1019), full configured lint (0 issues), tests pass.

### 6. Pre-commit hook repaired — commit `af090852`

- Root-caused: 4 buildflow steps failed because their binaries were absent from the flake devShell — `tsc` (kills type-check + tsconfig-check), `go-licenses` (license-check), `vulnix`.
- Added `pkgs.typescript`, `pkgs.go-licenses`, `pkgs.vulnix` to devShell; verified all three resolve in `nix develop`; **verified `buildflow --build-mode pre-commit --staged-only` now exits 0**.
- `git commit` works without `--no-verify` again.

### 7. check-cqrs-lint gate fixed — flake.nix (uncommitted, see b)

- Root-caused: the app was the only gate without `GOWORK=off`; workspace-mode package loading broke on the 3 modules whose deps touch go-cqrs-lite symbols (`systemadapter`: `undefined: projectionadapter`; `health`/`auditlog`: `undefined: do`) because the dirty sibling tree poisons go.work replaces. All 3 pass with `GOWORK=off`.
- Added `export GOWORK=off` + `GOEXPERIMENT=jsonv2` to the app (matching every other gate). Gate now green: "All modules pass cqrs-lint strict."

---

## b) PARTIALLY DONE

1. **Full gate suite — ALL 13 GREEN**: `check-modules`, `build` (26 modules), `lint` (0 issues/15 modules), `test` (17 suites, race), `coverage-gate` (15/15 thresholds, e.g. root 93.4%, datastar 97.5%, health/auditlog 100%), `check-codegen`, `check-templates`, `check-cqrs-lint` (after fix), `nix flake check --no-build`, `test-fuzz` (PASS), `test-flake` (3/3 runs PASS).
2. **flake.nix cqrs-lint fix uncommitted** — the edit is verified but not yet committed; it rides with this report commit.

## c) NOT STARTED

1. **systemadapter train (§4 item 7)** — blocked on go-cqrs-lite finishing its codec-extraction refactor and cutting `metaengine/projectionadapter v4.5.0` + `metaengine/sqliteengine v4.0.2` tags. Sibling is dirty; not my call to touch.
2. **Pushing `datastar/v4.8.0` tag + master** (5 unpushed commits after this report: `f128072d`, `e7aeee97`, `f6599348`, `af090852`, + this report + the local tag). User owns the push, per standing rule.
3. Post-push: strip the remaining `datastar/v4 => local` replaces in datastar-demo + integration_test.
4. **Runbook §4 item 8 / living docs pass** for this session's datastar strip + tag (AGENTS.md replace-pile bullet, runbook §4, CHANGELOG) — deferred to a single docs pass, see f/7.
5. Previous report's §f backlog items (32) — untouched this session.

## d) TOTALLY FUCKED UP (honest ledger)

1. **Buildflow's `--fix` runs wrote UNVERIFIED dependency bumps mid-session.** My hook-repair verification (`buildflow pre-commit` with a staged file) let golangci-lint/go mod tooling "fix" files outside my intent: it bumped `totp v4.7.0→v4.8.0` in examples/admin-demo **for a tag that does not exist** (unresolvable), bumped `go-webauthn/x`, and churned `adminui/styles.css` (no trailing newline). I caught it only because I `git status` after every operation — otherwise the next hermetic build would have failed mysteriously. Reverted the two bad files, kept the two correct ones (webauthn v4.8.0 exists; identity-model direct require is right). Lesson: treat `buildflow pre-commit` as a MUTATING tool, always diff-inspect its output before committing, and never trust staged-only scoping.
2. **Wasted a round on the SA1019 count**: relied on the previous report's "22" instead of enumerating first (actual 31). Minor, but the enumeration should always come first.
3. **Committed the strip pass BEFORE running the module test suites for datastar-demo/integration_test** (only build+vet for those two; datastar itself got tests). Risk was low (go.mod-only changes) and CI covers it, but the discipline I claimed ("hermetic verify each") was only fully applied to 2 of 4 modules pre-commit. Post-commit gates covered the gap (test gate green).
4. **Stale-workspace gopls noise**: project diagnostics showed 7,287 phantom errors all session (broken go-sse import etc.) caused by the dirty sibling poisoning the LSP's workspace view. Correctly ignored (builds are truth) but it degrades every Edit-tool call's signal — worth an LSP restart next session once the sibling lands.

## e) WHAT WE SHOULD IMPROVE

1. **Add a repo guard against nonexistent-version bumps**: a tiny script asserting every `github.com/larsartmann/*` require in all go.mod files resolves against `git ls-remote --tags` would have made buildflow's phantom totp bump impossible to commit. Could fold into `check-version-drift.sh --strict`.
2. **Pin buildflow's mutating behavior in pre-commit**: the hook runs `golangci-lint --fix` + go tooling over the repo; with the hook now WORKING again (devShell fixed), the fatcontext/dupword nolint discipline in AGENTS matters again — and so does post-hook `git diff` review. Consider `--staged-only` behavior verification as a standing habit.
3. **Make `check-templates.sh` CI step also assert the restore contract**: the script's trap-restore is load-bearing; a CI follow-up step `git diff --exit-code -- usermgmt/` after it would catch a restore regression (currently only checked manually).
4. **go-cqrs-lite release hygiene**: `stack/postgres v4.2.0` is a published tag that cannot compile against its own declared deps — that's an upstream broken-tag incident worth an ADR/retraction check over there (their side, not mine, but it cost this session a debugging cycle and will cost any external consumer).
5. **Consistent GOWORK=off in ALL flake gate apps**: check-cqrs-lint was the odd one out and silently depended on workspace mode. A one-line audit of every app for the env pair would close the class. (Done for cqrs-lint; errorfamily/others appear GOWORK-safe via goApp wrapper but not audited this session.)
6. **LSP health check at session start**: the 7k stale diagnostics indicate the workspace view broke when the sibling went dirty; a quick `crush` LSP restart habit saves noise on every edit.

## f) NEXT UP TO 50 (Pareto-ordered)

1. Commit the flake.nix cqrs-lint fix (rides with this report).
2. **User pushes**: master (`f128072d..HEAD+1`) + `datastar/v4.8.0` tag.
3. After push: strip `datastar/v4 => local` replaces from `examples/datastar-demo` + `integration_test`; hermetic verify; commit.
4. After push: verify proxy serves `datastar/v4.8.0` (pkg.go.dev / `go get` dry-run).
5. Wait for go-cqrs-lite refactor to land; then cut `metaengine/projectionadapter v4.5.0` + `metaengine/sqliteengine v4.0.2` (strip workspace replaces in tag commits) — §4 item 7.
6. Strip systemadapter + examples/system-demo metaengine replaces; bump requires; cut `systemadapter/v4.8.0`.
7. Single living-docs pass for this session: AGENTS.md (replace-pile bullet post-strip, datastar tag state, buildflow-mutation gotcha, cqrs-lint GOWORK fix), runbook §4 item 8 DONE, CHANGELOG entries (strip, CI wiring, migration, devShell, cqrs-lint gate).
8. Add nonexistent-larsartmann-version guard to drift check (see e1).
9. Add post-check-templates `git diff --exit-code` CI assertion (see e3).
10. Audit all flake gate apps for GOWORK=off consistency (see e5).
11. Fix `adminui/styles.css` missing trailing newline properly (formatter config) — the churn source.
12. Fix phantom `totp`/`oauth2` v4.8.0 alignment: either cut those two tags (their trees haven't changed — but family versioning says one coordinated version) or leave at v4.7.0 deliberately and note it.
13. Blob purge from pushed master history (previous report Q2 — force-push + re-cut 10 tags; still awaiting user decision).
14. Delete `backup/pre-blob-purge` branch (previous Q3 — awaiting user).
15. v4 branch blob rewrite (TODO_LIST P3).
16. integration_test: move off `datastar/v4 => local` (dup of 3).
17. Migrate remaining usermgmt re-export consumers where deprecated (root module's own `_reexport.go` markers stay until v5).
18. auditlog viewer integration into fullstack UI suite (backlog).
19. golines integration into `nix fmt` via treefmt (backlog).
20. goldmark-based link checker for anchors (backlog; current checker is path-only).
21. cqrs-lint distribution via CI artifact (backlog).
22. Consider a `docs/guides/datastar-integration.md` version note: datastar v4.8.0 resolves upstream-published deps now.
23. Re-run `nix flake check` full (with builds) once devShell inputs settle — this session ran `--no-build`.
24. Upstream: propose go-cqrs-lite retraction/ADR for broken `stack/postgres v4.2.0` (see e4).
25. Add `systemadapter` to CI `test` job (currently covered by nix gates; CI lacks a systemadapter test step? verify — build step exists).
26. Coverage-gate note in AGENTS: refresh actual numbers from this run (usermgmt 81.5%, identity-model 75.5%, dashboardui 83.4%, systemadapter 89.6%).
27. Sweep remaining `[~]` TODO_LIST items for staleness against this session's changes.
28. LSP restart + workspace resync once go-cqrs-lite lands.
29. Add e2e/offline-sync smoke to CI (currently manual/Playwright local).
30. Consider `skip_steps` addition for buildflow vulnix if it proves noisy in practice (first working run had 0 findings; watch it).

(30 items — the remaining backlog from the previous report's §f carries the rest.)

## g) QUESTIONS FOR YOU (cannot self-answer)

1. **Push scope**: push master + the local `datastar/v4.8.0` tag now yourself, or should I stage anything else first (e.g. also cut `totp/v4.8.0` + `oauth2/v4.8.0` tags — their module trees are at the family-aligned state but no v4.8.0 tags exist for them, which is why buildflow's phantom bump had to be reverted)? Note: those two tags would be no-op code-wise (their last commits are from the original family cut) but would complete the "one coordinated family version" convention.
2. **Broken upstream tag policy**: published `stack/postgres v4.2.0` cannot compile against published `storage/v4` (release-train skew — verified this session). Do you want me to draft an upstream issue/retraction proposal for go-cqrs-lite (using the verify-before-filing discipline), or leave it to that repo's own session?
3. **Blob in pushed history (carried from last report, still unanswered)**: accept the 27MB blob in pushed master history, or schedule the force-push purge (rewrite `5604e810..`, re-cut all 10 pushed tags, coordinate with anyone who pulled)?

---

## Gate Results (this session, in run order)

| Gate                          | Result                          |
| ----------------------------- | ------------------------------- |
| check-modules                 | ✅ green (post-strip)           |
| build (26 modules, hermetic)  | ✅ green                        |
| lint (15 modules)             | ✅ 0 issues                     |
| test (17 suites, -race)       | ✅ green                        |
| coverage-gate (15 thresholds) | ✅ green                        |
| check-cqrs-lint               | ✅ green (after GOWORK=off fix) |
| check-codegen                 | ✅ green                        |
| check-templates               | ✅ green (hermetic rework)      |
| nix flake check --no-build    | ✅ green                        |
| test-fuzz                     | ✅ PASS                         |
| test-flake                    | ✅ green (3/3 runs)             |
| buildflow pre-commit (hook)   | ✅ exit 0 (after devShell fix)  |
