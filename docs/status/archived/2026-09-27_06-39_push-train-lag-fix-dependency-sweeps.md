# Session Report: Push Train-Lag Fix — templ-components v1.19.4 + go-health v0.4.1 Sweeps

> ANNOTATED 2026-10-01 (docs-health round 13): the open tail closed — CI on the pushed ranges went green (cqrs-htmx 7/7 witnessed 2026-09-28, run 36382941958), the CHANGELOG convention question resolved by precedent (the 09-27/09-30 sweeps got [Unreleased] entries in the round-13 pass), the pre-commit env-class failures were largely FIXED at source 2026-10-01 (BuildFlow 14-step recovery: treefmt templ wrapper, go-licenses GOROOT shim, allow-serial-runners; residual documented in gotcha 8), the datastar-demo SA1019 pair migrated 2026-10-01, the go-directive policy question is now ROADMAP OQ26, and this HARVEST happened in this pass (per-module example race tests, `go mod verify`, bump-dep `--commit` mode, fsprobe daemon-glob, vulnix policy → TODO_LIST P3 micro-batch). §f rows not struck are adjudicated brainstorm. Archived per the tail convention.

> **Session:** 2026-09-27 ~04:20–06:40 CEST · **Task:** `git push; do fix`
> **Scope:** This session only — what was done, what was missed, what was noticed in tool output along the way.
> **Final state:** `master` pushed through `d1c1bf01`; pre-push CI-parity gates green.

---

## Context

`git push` (27 commits ahead) was blocked by the pre-push release-train gate: **63 train-lag findings** — `templ-components` family at v1.19.2 while v1.19.4 was published (10 modules), `go-health` at v0.4.0 while v0.4.1 was published (3 modules). The fix was the gate's own recipe: dependency sweeps via `scripts/bump-dep.sh`, then re-push.

Commits this session (mine, or daemon racing me):

| Commit     | Content                                                                                                                   |
| ---------- | ------------------------------------------------------------------------------------------------------------------------- |
| `d2c9ad1f` | daemon: pre-existing `.config/metadata.yaml` `importance` removal (foreign change, committed to unblock clean-tree tools) |
| `13252d77` | daemon: dashboardui test formatting (foreign diff from another session/hook pass)                                         |
| `b4369e64` | daemon: **go-health v0.4.1 sweep** (3 modules) + stray `buildflow-fsprobe-*` binary                                       |
| `8ea48c5a` | mine: fsprobe binary deletion                                                                                             |
| `47321d7c` | daemon content + my amended message: **templ-components v1.19.4 sweep** (12 modules)                                      |
| `d1c1bf01` | mine: AGENTS.md gotcha-8 addition (fsprobe strays)                                                                        |

Pushed ranges: `c35cc7a1..47321d7c`, then `47321d7c..d1c1bf01`.

---

## a) FULLY DONE

1. **Push blocker diagnosed.** Pre-push hook output read end-to-end; 63 lag findings mapped to two family bumps; the gate's fix recipe followed exactly (exact-anchor `$` patterns, same-version family merged into one prefix sweep — sanctioned by the recipe note).
2. **go-health v0.4.0 → v0.4.1 swept** across `health`, `integration_test`, `examples/samber-do-demo`. Evidence: `b4369e64`; `bump-dep.sh` per-module hermetic tidy+build+vet all PASS; absence assertion green (no `v0.4.0` pins remain); `go` directives untouched at 1.27.1.
3. **templ-components family v1.19.2 → v1.19.4 swept** (root + errorpage/htmx/icons/utils/datastar) across 12 modules: adminui, dashboardui, e2e/server, 6 examples, health, integration_test, setup. Evidence: `47321d7c`; 12/12 PASS in `bump-dep.sh`; stale-pin absence assertion RC=1 (clean).
4. **Breaking-change check done BEFORE sweeping.** Both target changelogs read (local clones): go-health v0.4.1 = floor fix only; templ-components v1.19.3/.4 = fixes + formatting canonicalization, no removals. Verified non-breaking rather than assumed.
5. **CSS bundles rebuilt in the same change as the family bump** (the AGENTS.md requirement): `build-adminui-css` + `build-dashboardui-css` RC=0; rebuild byte-identical (no diff). Both gates green: `check-css-bundles` (canonical, canaries present) and `check-css-bundle-classes` (class sets exact: 1021/1005 tokens — mechanized proof that v1.19.4's templ-fmt canonicalization is formatting-only for consumers).
6. **Full test suite green:** `nix run .#test` → 18 packages `ok`, RC=0 (race detector on), including all 5 swept modules that this gate covers.
7. **Strict release-train gate green with fresh cache:** `--refresh-cache --strict-lag 0` → 851 requires, 0 unpublished, 0 replace-exempted, 0 lag.
8. **Pushed twice; every pre-push gate green at push time** (release-train strict, version-drift strict, published-tag existence: all 851 requires resolve).
9. **AGENTS.md updated** with the fsprobe-stray trap (failed pre-commit BuildFlow runs strand `buildflow-fsprobe-*` binaries; daemon commits them). `d1c1bf01`.

## b) PARTIALLY DONE

1. **Behavioral test verification of the 13 swept modules — 5/13 covered.** `nix run .#test` explicitly excludes `^(e2e/|examples/)` (flake.nix:221), so `e2e/server` + 6 swept examples got build+vet only. At least `examples/catalog-demo` and `examples/dashboard-demo` have `*_test.go` files that never ran this session. Remaining: one per-module `go test -count=1 -race` pass over the 8 excluded modules. Effort S. No blocker.
2. ~~**CHANGELOG.md not updated for either sweep.** Repo CHANGELOG is append-only and maintained via docs commits, but no entry was written for the two dependency bumps (unknown whether dep sweeps conventionally get entries — needs a `git log -S` convention check, then possible backfill). Effort S.~~ done 2026-10-01 — both sweeps got CHANGELOG [Unreleased] entries in the round-13 pass (precedent set; the standing policy for FUTURE sweeps stays an owner call, see ROADMAP OQ26's sibling question in the 09-30 report g1).
3. ~~**CI on GitHub unobserved.** Push succeeded; the pre-push hook mirrors CI's strict gates locally, so risk is low — but the actual CI run result for `c35cc7a1..d1c1bf01` was not checked. Effort S (`gh run list`).~~ done — CI green witnessed 2026-09-28 (run 36382941958, 7/7).
4. **Commit-message hygiene inconsistent.** templ-components sweep got a proper message via guarded amend (`47321d7c`); the go-health sweep permanently carries the daemon heuristic message (`b4369e64` "auto-commit 7 changed file(s)") because my hook-failing commit attempt lost the race. Content is correct in both; readability of one is degraded.

## c) NOT STARTED

1. **`go mod verify`** across the 13 touched modules — item 4 of the go-ecosystem-upgrade skill's verification gate; never executed (see d-2).
2. **docs-health HARVEST** of section (f) below into `TODO_LIST.md`/`ROADMAP.md` — deliberately deferred: the user instructed report-then-wait. This report's (f) is the harvest input.
3. ~~**BuildFlow doctor warning triage** observed in pre-commit output (19 modules need tidy, stale vendorHash FOD, go-line flipflop) — noticed, logged in (f), not acted on (out of session scope).~~ resolved — the vendorHash alarm was DISPROVEN 2026-10-01 (it belongs to the external `.#benchstat` package, byte-identical fleet pin); the go-line flipflop is ROADMAP OQ26; the gomod-freshness class is standing noise documented in gotcha 8.

## d) TOTALLY FUCKED UP

Nothing shipped broken — every gate was green before each push. But the session had four real mistakes:

1. **Repeated a known-failing action and paid for it.** The FIRST commit attempt (metadata.yaml) failed on the pre-commit BuildFlow environment class (golangci-lint in examples, bare-shell go 1.26.7 vs workspace 1.27.1 — the documented gotcha-8 failure). Diagnosis was correct: environment class, not content. I then ran the FULL hook AGAIN for the go-health sweep commit instead of going `--no-verify` immediately. Same deterministic failure, ~90 s wasted, and the daemon won the race — the go-health sweep is permanently labeled `chore: auto-commit 7 changed file(s) (heuristic)`. One env-class failure is diagnosis; the second was anti-pattern.
2. **Silently skipped a loaded skill's checklist item.** `go mod verify` is on the go-ecosystem-upgrade verification gate I had loaded and was following; I never ran it and never consciously decided to skip it. (tidy's success makes go.sum inconsistency unlikely, but "likely fine" is not the gate.)
3. **No pre-change baseline.** The skill's Phase 1 mandates recording build+test state BEFORE touching anything. I never ran the test suite before the sweeps — the 27 unpushed commits' test state was assumed, not verified. Had a post-sweep test failed, I could not have distinguished regression from pre-existing breakage. Zero-harm outcome by luck, not by process.
4. **Violated the rc-capture rule on the very first command.** `git push 2>&1 | tee /tmp/...; echo "EXIT: $?"` — printed `EXIT: 0` while the push had FAILED (the echo captured tee's exit code, the exact PIPESTATUS-class trap documented in AGENTS.md gotcha 3). I read the output text and wasn't misled, but the mechanical check lied. Later commands used the correct `cmd > log 2>&1; rc=$?` form.

## e) WHAT WE SHOULD IMPROVE

1. **Commit-first, assert-later under daemon contention.** Between the templ-components sweep PASS and my commit, I ran absence assertions + go-directive greps — the daemon committed first. Under a fast-polling daemon, verification that doesn't gate the push should run AFTER committing (on committed state), not before. Impact: lost commit messages + confusing attribution; happens every multi-step change.
2. **bump-dep.sh stops at build+vet.** The repo's canonical sweep tool runs tidy+build+vet but no `go mod verify` and no tests — so a mechanical sweep CANNOT satisfy the go-ecosystem-upgrade verification gate as written. Either extend the script (`go mod verify` + optional `--test` flag) or document the intended follow-up pass.
3. **Pre-commit hook is devShell-only in practice.** From a bare shell it deterministically fails (env class), strands fsprobe binaries, and thereby trains sessions to `--no-verify`. Fix: make `scripts/hooks/pre-commit.template` source `scripts/lib/go-cache-env.sh` (or export GOTOOLCHAIN alignment) so BuildFlow's Go steps see the 1.27.1 toolchain.
4. **Auto-commit daemon commits junk binaries.** `buildflow-fsprobe-*` strays get swept into history (happened this session). A daemon-side ignore glob is one line.
5. **No canonical gate tests e2e/examples.** Every `forEachGoModule` test app excludes `^(e2e/|examples/)` by design; nothing else covers them. Sweeps that touch 8 such modules verify them only to build+vet. Either a dedicated `#test-all` app or an explicit docs statement that examples are build-only artifacts.
6. **rc-capture discipline decays.** The documented trap (pipes eat exit codes in this shell) still bit once this session. Consider a `scripts/lib/rc-guard.sh` helper or just: never pipe on commands whose rc matters.

## f) Next tasks (session-adjacent, ranked)

| #  | Task                                                                                                                                  | Impact | Effort | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Run `go test -count=1 -race ./...` in e2e/server + the 6 swept examples; fix anything red                                             | High   | S      | Quality       |
| 2  | Check CI result for pushed range `c35cc7a1..d1c1bf01` (`gh run list`)                                                                 | High   | S      | Verification  |
| 3  | `go mod verify` across the 13 swept modules                                                                                           | Medium | S      | Quality       |
| 4  | `git log -S` CHANGELOG convention for dep sweeps; backfill entries for v1.19.4 + v0.4.1 if conventional                               | Medium | S      | Documentation |
| 5  | Make `.githooks/pre-commit` (via template) devShell-independent: source go-cache-env.sh / GOTOOLCHAIN alignment for BuildFlow steps   | High   | M      | Tooling       |
| 6  | Teach auto-commit daemon to ignore `buildflow-fsprobe-*`                                                                              | High   | S      | Tooling       |
| 7  | Extend bump-dep.sh: `go mod verify` per module + optional `--test` flag                                                               | Medium | S      | Tooling       |
| 8  | Decide fleet go-directive policy: `go 1.27.1` patch form here vs the patch-floor-poisoning class go-health v0.4.1 just fixed          | High   | M      | Decision      |
| 9  | BuildFlow doctor: 19 modules flagged needing `go mod tidy` (gomod-freshness) — run `nix run .#deps` in a quiet window                 | Medium | S      | Quality       |
| 10 | Stale `vendorHash` FOD (flake.nix:107) — `buildflow -s nix-hash-fix --fix`                                                            | Medium | S      | Quality       |
| 11 | go-line flipflop warning: `go` directive changed 20× in 20 commits — align go-version-auto-configure vs go-mod-update dispositions    | High   | M      | Quality       |
| 12 | go-work-paths warning: 13 `use` paths lack the `/v4` suffix their go.mod module names carry — confirm deliberate, then silence or fix | Medium | S      | Cleanup       |
| ~~13~~ | ~~datastar-demo SA1019s: migrate to `go-datastar/broadcast` constructors (deprecated aliases die in v5)~~ done 2026-10-01 (BuildFlow recovery session; direct `broadcast.NewBroadcaster` require)                                                                                                                           | ~~Low~~    | ~~S~~      | ~~Cleanup~~       |
| 14 | samber-linter fails 80% of runs — exclude from BuildFlow or fix root cause                                                            | Low    | S      | Tooling       |
| 15 | vulnix: 65 nix-infra CVE findings in pre-commit output — triage real vs noise                                                         | Medium | L      | Security      |
| 16 | flake.nix missing `mainProgram` (flake-meta-checker)                                                                                  | Low    | S      | Cleanup       |
| ~~17~~ | ~~HARVEST this report's (f) into TODO_LIST/ROADMAP via docs-health~~ done 2026-10-01 (round-13: survivors → TODO_LIST P3 micro-batch; go-directive → OQ26)                                                                  | ~~Medium~~ | ~~S~~      | ~~Documentation~~ |
| 18 | Orphan `adminui/styles.css` + `dashboardui/styles.css` buildflow outputs (gotcha 9): ignore or delete                                 | Low    | S      | Cleanup       |
| 19 | Consumer-level golden for PolledRegion interval rendering (library golden re-baselined in v1.19.4; dashboardui polls)                 | Low    | M      | Quality       |
| 20 | Root-cause the `.config/metadata.yaml` `importance` field removal (foreign change, provenance unknown)                                | Low    | S      | Cleanup       |
| 21 | Consider a sanctioned daemon-pause (or session marker) for multi-step sweeps                                                          | Medium | M      | Tooling       |
| 22 | Document the --no-verify justification template for env-class failures in the runbook (it lives only in gotcha 8 prose today)         | Low    | S      | Documentation |
| 23 | BuildFlow binary stale vs HEAD (`e881e96` vs `c9f7094`) — rebuild/reinstall                                                           | Low    | S      | Tooling       |
| 24 | Examples with tests but no gate (catalog-demo, dashboard-demo, …) — fold into task 5's coverage decision                              | Medium | S      | Quality       |
| 25 | Skipped-then-committed `d2c9ad1f`/`13252d77` foreign diffs: confirm with the owning session that daemon-committing them was intended  | Low    | S      | Process       |

## g) Questions I cannot answer myself

1. **Go-directive policy (task 8):** this repo pins `go 1.27.1` (patch form) in every module — the exact anti-pattern go-health v0.4.1 was released to fix, yet AGENTS.md declares 1.27.1 the fleet-wide floor. Tried: AGENTS.md (says "fleet-wide floor"), go-health changelog (says patch floors poison consumers) — the two authoritative sources disagree. Keep 1.27.1 or normalize to `go 1.27`?
2. **Daemon etiquette:** is amending the daemon's unpushed commits to restore proper messages acceptable (I did it once, hash-guarded), or should heuristic messages stand? Is there a sanctioned way to pause the daemon during multi-step sweeps?
3. **Test-coverage policy for examples/e2e:** the exclusion from every `forEachGoModule` test app is explicit (flake.nix:221) — is that a deliberate speed/cost choice, or should a `#test-all` app close the gap so dependency sweeps get behavioral verification on all consumers?

---

_Point-in-time snapshot; append-only. Sections (f)/(g) are the handoff surface — HARVEST before this file goes stale._
