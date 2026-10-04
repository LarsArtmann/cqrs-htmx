# Status Report — scripts/ Reorganization Session

> **2026-10-04 13:14 CEST** — point-in-time snapshot of the 2026-10-04 ~12:20–13:10
> session that reorganized `scripts/` (user prompt: "this is too much, do something
> about it!"). Scope: this session's work and what it noticed on the tree. No
> unrelated research was done. Per the status-report skill, HTML is the canonical
> format; the user's explicit `.md` path demand wins for this report.

## Executive Summary

The flat 50-entry `scripts/` directory is now a 6-entry tree (`checks/`, `selftests/`,
`tools/`, plus unchanged `lib/`, `hooks/`, `testdata/`). All 69 files were re-homed
with `git mv`, and every reference surface was rewired: flake.nix (92 refs),
`.github/workflows/ci.yml` (~40 refs), both git hooks + their templates, 6 self-test
scratch-repo layouts, internal `$SCRIPT_DIR`/`REPO_ROOT` plumbing, and living docs.
Verification: 22/22 self-tests green, `check-modules --report` 27/29, every
CI-parity gate that exercises scripts green. The 2 red check-modules stages and the
cqrs-lint gate are **proven external** (a concurrent session's in-flight
`dashboardui.LayoutFunc` feature), not caused by this session.

| Category | Count |
| --- | --- |
| a) Fully done | 11 |
| b) Partially done | 4 |
| c) Not started (deliberate) | 7 |
| d) Fucked up (mine) | 5 |
| e) Improvements | 7 |
| f) Next tasks | 40 |

---

## a) FULLY DONE

1. **Directory reorganization** — 55 files moved via `git mv` into `scripts/checks/`
   (22 `check-*` gates + `errorfamily_scanner.go` + `erraudit-inventory.sh`),
   `scripts/selftests/` (22 `test-*` fixture self-tests), `scripts/tools/` (9 workflow
   utilities). Evidence: `ls scripts/` = 6 entries; git history (daemon commits
   `1f2daac6`, `7314bc0f`, `b595ee24`).
2. **flake.nix rewired** — all 92 `scripts/...` references updated to the new paths
   (check-modules stage list, individual flake apps, errorfamily, bump-dep, docs
   descriptions). Evidence: `nix flake show --all-systems` evaluates; every flake app
   run this session resolved its script.
3. **CI workflow rewired** — `.github/workflows/ci.yml` ~40 step paths updated;
   YAML parse-validated with python-yaml. Note: the `cd scripts && find . -name
   '*.sh'` shfmt/shellcheck sweep is recursive, so it absorbed the new subdirs with
   zero changes.
4. **Git hooks + templates rewired and kept byte-identical** — `.githooks/pre-commit`,
   `.githooks/pre-push`, `scripts/hooks/*.template` all updated by the same sed set.
   Evidence: `test-install-git-hooks.sh` T3 (hook byte-matches template) green.
5. **Internal script plumbing fixed** — `REPO_ROOT`/`PROJECT_ROOT`/`ROOT` depth
   (`/..` → `/../..`) in both idioms (`BASH_SOURCE` and `$0`); `$SCRIPT_DIR` sibling
   refs in self-tests (`../checks/`, `../lib/`, `../hooks/`, `../testdata/`,
   `../tools/`); three dirname-form `lib/` sources
   (`check-module-isolation.sh:18`, `check-workspace-build.sh:33`,
   `verify-tag.sh:170`); two `$REPO_ROOT`-relative sources in
   `check-docs-freshness.sh`. Evidence: full battery below.
6. **Scratch-repo layouts mirrored** — 6 self-tests that copy scripts into fixture
   trees now build the NEW layout (`$fx/scripts/checks/`, `$scratch/scripts/tools/`):
   test-check-release-train, test-check-css-bundle-classes, test-check-docs-links,
   test-check-status-annotations, test-install-git-hooks, plus the F1 remediation-hint
   assertion in test-check-release-train (stays in lockstep with the gate's printed
   `scripts/tools/bump-dep.sh` hint).
7. **Runtime user-facing output corrected** — `check-release-train.sh`'s printed
   remediation recipe now says `scripts/tools/bump-dep.sh '...' v<ver>` (this string
   is what a developer sees during a red release-train gate).
8. **Auxiliary configs updated** — `.buildflow.yml` comments, `.go-structure-linter.yaml`
   comment, `scripts/lib/replace-exemption.sh` consumer comments, usage docstrings in
   the two `.py` tools and `errorfamily_scanner.go`.
9. **Living docs updated** — AGENTS.md (all refs + new "scripts/" Quick Reference row
   documenting the taxonomy), README.md, FEATURES.md, TODO_LIST.md, ROADMAP.md,
   docs/status/README.md, 3 runbooks, release-playbook, v5-removal-inventory.
   Zero stale refs per repo-wide sweep. Archives, ADRs, dated plans, CHANGELOGs,
   `docs/agents-notes.md`, and `attic/` deliberately untouched (point-in-time
   convention).
10. **Verification battery green** — 22/22 self-tests; check-modules --report 27/29
    (the 2 reds external, see d-note); CI-parity gates all green from new paths:
    check-require-tags (831 requires OK), check-service-methods, check-domain-counts,
    check-docs-tail-budget, errorfamily (flake app), check-templates,
    check-command-bijection (20/20), check-large-files --all, check-go-toolchain,
    check-workspace-build (28 modules), check-vcs-cache, both status gates, both css
    gates, erraudit-inventory --gate (TOTAL=0), docs-links, docs-freshness; YAML
    valid; `nix flake show` evaluates; treefmt `0 changed`.
11. **Hygiene** — `scripts/__pycache__/` trashed (pre-existing) plus two `__pycache__`
    dirs my own `py_compile` created.

## b) PARTIALLY DONE

1. **Full-workspace Go test suite NOT run** (`nix run .#test`, and `.#test-all` with
   e2e/examples). What works: zero Go files in this session's diff, so script-scoped
   verification is complete. What remains: the 28-module race suite was never
   executed this session. Blocker: the tree carries a concurrent session's in-flight
   feature (below) — a red suite now would be unattributable noise. Effort: M once
   the tree is quiet.
2. **check-modules is 27/29** — isolation + workspace-build stages red. What works:
   every scripts-exercising stage (25 of 29) green, including all 14 self-test
   stages. What remains: 2 stages red with failure signature
   `undefined: dashboardui.LayoutFunc` / `missing go.sum entry for x/tools@v0.51.0`
   under `GOWORK=off` — the concurrent feature is unpublished, so standalone module
   builds resolve stale published deps. Blocker: external session. Effort: S to
   re-verify once they land.
3. **cqrs-lint gate red** — `--strict` root walk fails with 3 WARNINGs in
   `setup/setup.go` (event-bus recovery, `Close()` ×2). Proven external: a detached
   worktree at pre-reorg commit `1f2daac6` shows **0** such warnings; `setup.go` was
   last modified in `7314bc0f` (concurrent feature work). Not mine to fix (safety
   rule: never touch diffs I didn't author). Effort: S once they land.
4. **Pre-push hook never live-fired** — the strict release-train + version-drift
   path in `.githooks/pre-push` is verified by self-tests and byte-identical
   templates, but no `git push` happened this session (none requested). CI parity on
   real runners is therefore unproven for the new paths. Effort: S (next push).

## c) NOT STARTED (deliberate deferrals, in priority order)

1. **HARVEST of section (f) into TODO_LIST.md/ROADMAP.md** — the skill's post-report
   rule; awaiting instructions per the session's "then wait" directive.
2. **Repo-root-resolution guard** — a mechanical check that no script in
   `scripts/{checks,tools}` computes repo root with a single `/..` (today's
   incident class: move one level deeper, 6+ scripts silently break). Not written;
   needs a decision whether it lives as a self-test inside an existing gate.
3. **CI/flake stage-list consolidation** — ci.yml's ~40 hardcoded paths and flake.nix's
   check-modules stage list are two sources of truth for the same fact (pre-existing
   split brain, made no worse today). Not started; needs an architecture decision.
4. **Aggregator for the 22 self-tests** — one `nix run .#selftests` app instead of 22
   enumeration points (flake apps + CI steps). Not started.
5. **`reports/jscpd-report.json` stale paths** — tracked + gitignored + generated;
   regenerating it was out of scope. Not started.
6. **`docs/agents-notes.md` war-story entry** for the reorg (the `$0` vs
   `BASH_SOURCE` + scratch-layout coupling lessons). Convention says dated histories
   go there; not written.
7. **Stale-path sweep of archived docs** — deliberately never done (historical
   integrity), but there is no guard preventing a FUTURE doc from linking a dead
   path; `check-docs-links.sh` only validates actual `[x](y)` links, and none point
   at scripts today. Not started (low).

## d) TOTALLY FUCKED UP (this session's own errors, radical-honesty section)

1. **Three sed rounds died on delimiter collisions** — patterns containing `||`
   (`cd "$SCRIPT_DIR/.." || exit 1`) inside `|`-delimited sed expressions →
   `unknown option to 's'`. Worse, one round **left a mangled line**
   (`""$SCRIPT_DIR/...` double-quote corruption) in `test-install-git-hooks.sh`
   that survived until a targeted view caught it. Nothing shipped broken (battery
   green at the end), but the fix loop cost 4 tool round-trips. Root cause: I chose
   `|` as delimiter while patching shell control flow; should have used `@`
   everywhere from the start. Severity: low (caught by syntax sweep + tests).
2. **Multi-file `cp` mis-edit corrupted a scratch layout** — my first scratch-layout
   edit turned a 3-file `cp` into a 4-source/single-destination form, silently
   copying `check-large-files.sh`, `check-release-train.sh`, `check-version-drift.sh`
   into `scripts/tools/` instead of `scripts/checks/` in the fixture repo. T6/T12
   failed with "No such file or directory" and T7 passed for the WRONG reason
   (missing script also blocks the commit). Fixed by splitting into two `cp`
   commands. Severity: medium (test false-positive pattern — T7's green was
   meaningless at that moment).
3. **Idiom-blind first audit** — my path-fix sweep matched only `BASH_SOURCE` forms;
   the `$0` forms (`check-workspace-build.sh`, `check-command-bijection.sh`,
   `check-templates.sh`, `erraudit-inventory.sh`) were missed and surfaced LATE:
   check-modules failed mid-run ("no go.work at .../scripts"), and three gates
   (bijection, templates, erraudit) failed their direct post-fix runs. Also the
   `})/lib/` sed pattern missed the `"$(dirname ...)"/lib/` form, killing the first
   check-modules run. Root cause: audited for one idiom instead of grepping for ALL
   `dirname` variants before writing the sed set. Severity: medium (wasted a full
   check-modules cycle; all caught before commit).
4. **`--no-verify` commit without naming the failed step** — the docs-phase commit
   used the documented BuildFlow-fallback after a pre-commit step failure, but I
   classified the failure by-class (known deterministic non-devShell noise) without
   capturing WHICH step failed — the justification rests on AGENTS.md gotcha 8's
   pattern match, not on a recorded step name. Also the daemon committed the docs
   phase (`759da4f6`) before my explicit commit could land, so the fallback commit
   ultimately never happened — the rationale trail lives only in this report.
5. **Self-inflicted pycache noise** — my `python3 -m py_compile` verification created
   two new `__pycache__` dirs inside the freshly reorganized tree; caught and
   trashed, but it momentarily re-dirtied exactly the directory I was cleaning up.
   Severity: trivial.

**Not mine, but the tree is not fully green (attribution record):**
concurrent session's `dashboardui.LayoutFunc` feature — `dashboardui/layoutfunc.go`
+ `layout_embed_test.go` + `setup/config.go:414` + `setup.go:377` references, landed
partially via daemon commits amid this reorg; currently causes the 2 red
check-modules stages and the 3 cqrs-lint WARNINGs. Evidence trail: worktree probe at
`1f2daac6` (0 warnings) vs current (3).

## e) WHAT WE SHOULD IMPROVE

1. **Audit idioms BEFORE writing the fix** — for any path-movement refactor, grep for
   ALL path-computation idioms (`BASH_SOURCE`, `$0`, `%~dp0`-style, `__file__`,
   `filepath.Dir`, hard-coded `scripts/`) and enumerate the forms first. Would have
   saved 2 failed rounds + 1 mid-gate failure today.
2. **Delimiter discipline for sed in shell code** — default to `@` (or a char proven
   absent from the pattern) whenever a pattern can contain `|` (shell `||`, pipes).
   Better: for structural shell edits, prefer the `edit`/`multiedit` tools over sed
   entirely — exact-match beats regex for one-off structural changes.
3. **Run the mega-gate before declaring internal plumbing done** — I ran targeted
   gate probes first and check-modules late-ish; the mid-run failure
   (`check-workspace-build` root resolution) was findable earlier by running
   check-modules right after the move. Order: move → seds → bash -n → check-modules
   → self-tests → docs.
4. **`wait-tree-quiet` as a pre-ritual for big moves** — I ran `preflight-tree-check`
   at start (clean), but the concurrent session started DURING my work. A
   wait-tree-quiet window before beginning wouldn't have prevented that, but a
   re-check before the risky seds would have flagged the foreign dirty file
   (`dashboardui/layoutfunc.go`) earlier and avoided one false-attribution detour.
5. **Test false-positives need assertion audits** — T7 passing while its precondition
   was broken (missing script blocks commit for the wrong reason) shows
   negative-outcome tests should assert on the ERROR MESSAGE, not just the exit
   code. Candidate improvement for test-install-git-hooks (and worth a fleet lesson).
6. **Edit-tool friction in shared trees** — 3 "modified since read" rejections cost
   round-trips; in a repo with a live auto-commit daemon and concurrent sessions,
   re-view immediately before every `edit` on files touched by seds in the same
   session.
7. **Report .md-vs-.html override handling** — the status-report skill now has a
   recorded standing exception for dispatch prompts; this session's explicit `.md`
   demand is a third instance of the same user preference. If it recurs, promote it
   to the skill's default-path note (crush-config repo change, not this repo).

## f) NEXT TASKS (40, ranked by impact; HARVEST input)

| # | Task | Impact | Effort | Category |
| --- | --- | --- | --- | --- |
| 1 | Re-run check-modules + cqrs-lint once the concurrent LayoutFunc feature lands/publishes; annotate this report with results | High | S | Quality |
| 2 | If the concurrent session is abandoned: take over setup.go (3 cqrs-lint WARNINGs: bus recovery, Close ×2) + finish/publish LayoutFunc or revert it | High | M | Bug |
| 3 | Live-fire the pre-push hook: next `git push` runs strict release-train + version-drift on the new paths (watch the fresh-tag TTL gotcha) | High | S | Quality |
| 4 | Verify CI green on next push — the shfmt/shellcheck step (`cd scripts && find`) now recurses 3 new dirs; confirm stage count + runner parity | High | S | Quality |
| 5 | Add repo-root-resolution guard: fail if any `scripts/{checks,tools}` script computes root with single `/..` (mechanizes today's incident class) | High | M | Quality |
| 6 | Single `nix run .#selftests` aggregator app running all 22 test-* scripts with a printed candidate count (kills enumeration drift) | High | M | Quality |
| 7 | Decide ci.yml/flake stage-list consolidation: generate one stage manifest consumed by both (removes a pre-existing split brain) | High | L | Quality |
| 8 | Assert error MESSAGE (not just exit code) in test-install-git-hooks negative cases (T7-class false positives) | Medium | S | Quality |
| 9 | Run `nix run .#test` (28-module race suite) on a quiet tree; then `.#test-all` (incl. e2e/examples) | High | M | Quality |
| 10 | Regenerate or untrack `reports/jscpd-report.json` (tracked+ignored+stale paths) | Low | S | Cleanup |
| 11 | Document `.go.work.check-workspace-build.sum` policy (tracked generated artifact; when is it committed vs regenerated) in AGENTS gotchas | Low | S | Documentation |
| 12 | Taxonomy naming pass: `erraudit-inventory.sh` lives in checks/ but reads like a tool — rename to `check-erraudit-inventory.sh` or document the exception in AGENTS scripts/ row | Low | S | Cleanup |
| 13 | treefmt coverage gap: `scripts/hooks/*.template` files are not `*.sh`, so shfmt/shellcheck never see the hook templates — add a glob or symlink-check | Medium | S | Quality |
| 14 | Fold the reorg war story ($0 vs BASH_SOURCE idioms, scratch-layout coupling) into docs/agents-notes.md per the dated-histories convention | Low | S | Documentation |
| 15 | Add a one-line pointer in AGENTS gotcha 3 to the sed `|`-delimiter trap (patterns with `\|\|`) | Low | S | Documentation |
| 16 | Confirm `install-git-hooks.sh --verify` on THIS repo (self-test proves scratch; live repo unverified this session) | Low | S | Quality |
| 17 | Final post-daemon stale-path sweep (`scripts/check-`, `scripts/test-` etc.) once the concurrent session's commits settle | Low | S | Cleanup |
| 18 | Consider `scripts/**/__pycache__/` explicit .gitignore entry (cosmetic; currently covered by a global rule) | Low | S | Cleanup |
| 19 | `nix flake check` full evaluation (heavier than the `show` probe used this session) on a quiet tree | Medium | M | Quality |
| 20 | Run `buildflow` full mode once the tree is green (fleet-standard quality verdict) | Medium | M | Quality |
| 21 | Explicit .gitignore/docs note that fresh reports in docs/status/ root are Gate-1-ungated until archived (prevents future surprise at archive time) | Low | S | Documentation |
| 22 | HARVEST this report's (f) section into TODO_LIST.md/ROADMAP.md (docs-health HARVEST mode) | High | S | Documentation |
| 23 | Decide whether the reorg warrants a CHANGELOG entry at the next train (repo tooling vs consumer-facing; current convention says no) | Low | S | Documentation |
| 24 | Annotate this report when items 1–4 resolve (Gate-1 compliance happens at archive time) | Low | S | Documentation |
| 25 | Verify flake app descriptions mentioning script paths stayed accurate after sed (spot-check 3 apps' meta.description) | Low | S | Documentation |
| 26 | Consider pinning `shellcheck` directive headers (`source=` hints) in self-tests updated to `../lib/` paths (test-check-version-drift.sh:63 hints at old form) | Low | S | Cleanup |
| 27 | Add the root-resolution idiom list (BASH_SOURCE, $0, __file__) as a comment header in the new AGENTS scripts/ row for future movers | Low | S | Documentation |
| 28 | Next train: release-playbook's updated script paths get their first real exercise — walk §6 verify-tag steps verbatim as a dry run | Medium | M | Quality |
| 29 | Check whether any tooling outside this repo (fleet scripts, BuildFlow providers) invokes cqrs-htmx's scripts by absolute path — reorg is invisible to consumers, but fleet tooling isn't a consumer | Medium | M | Quality |
| 30 | Consider `git mv`-ing this report to docs/status/archived/ + annotate at next status cycle per convention | Low | S | Documentation |
| 31 | Evaluate whether `attic/batch-release.sh` (stale path comments, archived) should state its archived-status header explicitly | Low | S | Documentation |
| 32 | Spot-check 5 random self-tests' scratch trees for OTHER hidden copies of scripts (only 6 known couplings were found; a grep for `cp .*(check-\|lib/)` confirmed the set — re-confirm post-churn) | Medium | S | Quality |
| 33 | Keep bench discipline: no `--save-baseline` re-pin needed (scripts content-identical, ns/op unaffected) — record as explicit no-op | Low | S | Documentation |
| 34 | When cqrs-lint's Go-installable distribution exists (existing ROADMAP item), CI gains the gate — note the new script path in that future wiring | Low | S | Documentation |
| 35 | gitleaks/codespell on-demand pass over the moved tree (paths changed; secrets profile unchanged — cheap reassurance) | Low | S | Quality |
| 36 | Add `.githooks/pre-commit`'s manual-edit block comments to the check-modules docs-freshness scope if hook-comment drift ever matters (currently unchecked) | Low | S | Documentation |
| 37 | Confirm no JUSTFILE/Makefile reappeared (fleet rule: flake.nix owns automation) | Low | S | Cleanup |
| 38 | Promote the `.md` status-report preference to the crush-config skill if it recurs once more (cross-project lesson, crush-config repo commit) | Low | S | Documentation |
| 39 | After next train: verify the release-train gate's printed remediation path (`scripts/tools/bump-dep.sh`) against a REAL lag scenario (currently only fixture-tested) | Medium | S | Quality |
| 40 | Keep the scripts/ tree at 6 entries: any NEW gate must land in checks/ + selftests/ + flake/CI/docs wiring atomically (gotcha 19) — add this sentence to the AGENTS scripts/ row | Low | S | Documentation |

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **The concurrent `LayoutFunc` session** — is it still active and about to land
   (then I hold gates 1–2 and re-verify after), or abandoned mid-flight (then should
   I take over: finish/publish or revert the dashboardui/setup changes)? I tried:
   git log/status forensics only — intent is unknowable from the tree.
2. **What did "this is too much" mean?** — I interpreted it as a NAVIGATION problem
   and reorganized (50 flat entries → 6 dirs, zero scripts deleted, per gotcha 19's
   atomic-gate convention). If you meant the NUMBER of gates/scripts itself
   (consolidation/dedup pass), that is a different, larger task — which is it?
3. **Should the ~40 hardcoded ci.yml script paths collapse into one generated stage
   manifest shared with flake.nix's check-modules list?** It removes a real split
   brain but restructures CI YAML and adds generation machinery. Keep explicit
   duplication (simple, greppable) or generate (single source of truth)? I could not
   find a stated fleet preference.

---

*Prepared by Crush (glm-5.3-flash). Evidence inline; raw gate outputs in session
transcript. Report is point-in-time; annotate, never rewrite.*
