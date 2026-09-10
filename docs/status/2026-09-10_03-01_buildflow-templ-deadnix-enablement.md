# Status Report — BuildFlow templ-generate + deadnix Enablement (Root-Cause Fix)

- **Date:** 2026-09-10 03:01 CEST
- **Session scope:** Config audit + enablement of previously skipped BuildFlow steps (`templ-generate`, `deadnix`), plus `auto_fix` semantics investigation in BuildFlow source.
- **Trigger:** User challenged the config: "our config is pretty fucked up .. like templ disabled!??!"
- **Verdict:** The user was right. The old skip justification was **factually wrong** (documented a root cause that does not exist), and both steps are now enabled, root-caused, and verified green.

---

## Executive Summary

| Item                       | State                                                                                                               |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `templ-generate` skip      | **REMOVED** — root-caused to CWD-dependent `FileName:` emission, NOT a version/bundled-binary conflict              |
| `deadnix` skip             | **REMOVED** — installed in devShell, repo verified dead-code clean                                                  |
| Canonical templ generation | **FLIPPED to repo-root** (matches BuildFlow's always-from-root design); `nix run .#gen` + `.#check-codegen` aligned |
| `_templ.go` files          | 8 regenerated (FileName-only diff, verified)                                                                        |
| AGENTS.md                  | Stale gotcha rewritten with the true root cause                                                                     |
| Daemon commit              | `136ae3c6` (heuristic message, content = verified staged state)                                                     |
| AGENTS.md edit             | **STAGED, UNCOMMITTED** (vulnerable to daemon shredding)                                                            |

---

## a) FULLY DONE

1. **`templ-generate` enabled** — removed from `skip_steps` in `.buildflow.yml`, with an honest comment block documenting the root-canonical decision and the history of the old skip.
2. **`deadnix` enabled** — `pkgs.deadnix` added to the devShell's "BuildFlow pre-commit steps" block in `flake.nix`; removed from `skip_steps`; repo passes `deadnix --fail --quiet .` (exit 0).
3. **Canonicality flip executed** — all 8 `_templ.go` files (7 adminui + 1 loginpage) regenerated from the repo root; diff verified to touch ONLY `FileName:` lines (runtime `templ.Error` values; no test asserts on these strings).
4. **Nix gates aligned** — `nix run .#gen` and `nix run .#check-codegen` rewritten from per-module `cd` loops to root-level generation, so all three paths to generation (nix apps, BuildFlow step, manual) are byte-identical.
5. **Root cause documented** — BuildFlow resolves `templ` via `exec.LookPath` (no bundled binary); nixpkgs templ = `~/go/bin` templ = v0.3.1020 (zero version skew). The real issue: `FileName:` fields are relative to the invocation directory.
6. **AGENTS.md gotcha rewritten** — the false "buildflow bundles its own templ binary" claim replaced with the verified CWD-dependence explanation + rule "NEVER regenerate by cd-ing into module dirs".
7. **`auto_fix` question answered with source evidence** — BuildFlow force-enables AutoFix in `pre-commit` and `dev` modes (`internal/cli/workflow_config.go:474`), so the YAML key only affects full/CI runs. BuildFlow source read, not guessed.
8. **Verification battery (all green)**:
   - `nix run .#check-codegen` → PASSED (post-staging)
   - `nix develop -c deadnix --fail --quiet .` → exit 0
   - `buildflow -s templ-generate` → success (no-op; generation deterministic)
   - `buildflow -s deadnix` → 1/1 passed
   - `go test ./adminui/... ./loginpage/... -count=1` → ok (3.6s / 0.7s)
   - `nix fmt` → 0 changed

## b) PARTIALLY DONE

1. **Pre-commit hook end-to-end verification** — both steps verified individually via `buildflow -s` (full mode), but the actual hook path (`buildflow --build-mode pre-commit --staged-only` inside the 60s budget) was NOT exercised with the new steps active. Phantom-skip behavior for unstaged `.templ` files in pre-commit mode is assumed from BuildFlow source, not observed.
2. **Daemon commit audit** — `136ae3c6` stat shows exactly the 10 expected files, but a formal `git diff HEAD~1..HEAD` content audit against my verified staged state was not performed.
3. **Docs consistency** — READMEs/guides may still describe per-module `templ generate` (the old canonical mode). Not grepped. `check-codegen`'s FAIL message now points to root-canonical instructions, but other docs were not swept.
4. **This session's own docs hygiene** — CHANGELOG entry not written; section (f) below not yet harvested into TODO_LIST/ROADMAP (per docs-health convention).

## c) NOT STARTED

1. **markdown-format normalization** — 257 `.md` files; the skip is still in place (deliberately, but the "one-time pass" has never been scheduled).
2. **BuildFlow upstream improvement** — per-module templ generation provider (would make bare `FileName:` canonical again for all multi-module consumers). Flagged as the upstream-grade alternative; not filed.
3. **a-h/templ upstream question** — whether templ can pin the `FileName:` base dir (would eliminate the whole canonicality question at the source). Not investigated per user's "don't research unrelated" constraint.
4. **HARVEST of this report's next-steps into TODO_LIST/ROADMAP.**

## d) TOTALLY FUCKED UP

1. **The old config documentation lied as fact.** The `.buildflow.yml` comment AND the AGENTS.md gotcha asserted "buildflow bundles its own templ binary which regenerates `_templ.go` files with directory-prefixed FileName: fields, conflicting with the nix-pinned templ version." BuildFlow has no bundled templ binary (LookPath); the versions were IDENTICAL. A wrong root cause was committed as institutional knowledge and survived until challenged. Cost: a full session to debunk a two-line claim.
2. **The daemon race struck again (known pattern, recurring).** Deliberate, verified work (config flip + codegen canonicality) landed as `chore: auto-commit 10 changed file(s) (heuristic)` — the 3rd+ occurrence of this class per AGENTS.md history. The commit message carries zero of the narrative (why canonicality flipped, what the root cause was).
3. **I repeated a documented mistake class.** `nix run .#check-codegen 2>&1 | tail -3; echo "EXIT: $?"` printed `CODEGEN-EXIT: 0` on a FAILED gate — the exact PIPESTATUS-class trap AGENTS.md bans ("never through a pipe"). I caught the failure via the FAIL text, not via the exit code. The rule exists because of me; I then violated it in the same session.
4. **Sloppy opening move.** First tool call of the session was two `read_mcp_resource` calls to nonexistent MCP servers — a wasted round trip before the actual work started.

## e) WHAT WE SHOULD IMPROVE

1. **Root-cause-first config hygiene.** Every skip/exclude comment should carry a falsifiable claim + a verification command. "X conflicts with Y" without the command that proves it is how the templ lie survived months.
2. **Commit at phase boundaries (the standing rule I cannot follow alone).** Critical rules forbid me from committing without an explicit user "commit"; the daemon polls faster than verification tails. This guarantees heuristic-shredding of deliberate work unless the user grants a standing commit authorization or disables the daemon during working sessions.
3. **Exit-code discipline.** Every gate invocation in scripts AND in my shell work: capture `$?` directly (`cmd > /tmp/f 2>&1; echo $?`), never through a pipe. Consider a `scripts/lib/` helper so it's easier to do right than wrong.
4. **Version-skew guard for ambient binaries.** `~/go/bin/templ` shadowing the nix templ is benign ONLY while versions match; a nixpkgs bump silently reintroduces the original failure class. A 1-line version-consistency check in the pre-commit hook would make it impossible to miss.
5. **Mechanical canonicality guard.** `check-codegen` verifies drift, but nothing prevents a human from cd-ing into `adminui/` and regenerating (the trap is now documented, not prevented). Options: a `scripts/gen.sh` wrapper as the only documented entry, or a path check inside check-codegen that detects bare-FileName output and says "you generated from the wrong directory".
6. **Verify the no-op semantics.** `buildflow -s templ-generate` reported "1 no-op" — confirm this is deterministic-output detection (good) vs result-cache skip (potentially stale). Simulate a staged `.templ` change once to watch the step actually regenerate + restage.
7. **Upstream the fix.** Two candidate upstream items came out of this session (BuildFlow per-module templ provider; a-h/templ FileName base-dir flag). Filing them (with `verify-before-filing` discipline) would fix the root cause for every multi-module consumer, not just this repo.

## f) Up to 50 Things To Get Done Next

_Brainstorm per status-report skill — most items below the top tier are ROADMAP fuel for docs-health HARVEST routing. Sorted by impact._

**P1 — close this session's loose ends (do now)**

1. Commit the staged AGENTS.md edit with a narrative message (it documents the true root cause; leaving it staged invites daemon shredding).
2. Write the CHANGELOG.md entry for the templ-generate/deadnix enablement + canonicality flip (repo convention: completed work lives in CHANGELOG, and right now AGENTS.md records a change CHANGELOG doesn't).
3. Audit `136ae3c6` content: `git diff HEAD~1..HEAD` against the intended 10-file set (formal close of the daemon-commit audit).
4. Simulate the real hook once: stage a trivial `.templ` whitespace change, run `buildflow --build-mode pre-commit --staged-only`, confirm templ-generate regenerates + restages within the 60s budget, then revert.
5. Confirm the "1 no-op" semantics of the templ-generate step (deterministic-output detection vs result-cache skip) with the simulation from #4.
6. Grep repo docs (README.md, adminui/, loginpage/, docs/guides/) for stale per-module `templ generate` instructions; update to root-canonical.
7. Verify `.github/workflows/ci.yml` invokes `nix run .#check-codegen` (and that the rewritten app is what CI executes).
8. Run the full gate battery after the flake edit: `nix run .#build`, `nix run .#test`, `nix flake check --no-build`, `nix run .#check-modules`.

**P2 — hardening (this week)**
9. Add a templ version-consistency check to the pre-commit hook (ambient `~/go/bin/templ` vs nix templ; warn/fail on skew).
10. Add bare-FileName detection to `check-codegen` (turn the "wrong directory" trap from a docs rule into a loud error message naming the fix).
11. Create `scripts/gen.sh` as the single documented regeneration entry point (wraps root-level `templ generate` + gofmt), reference it from AGENTS.md and check-codegen's FAIL message.
12. Add an exit-code-safe gate-runner helper in `scripts/lib/` and sweep the most-used gate invocations onto it (kills the PIPESTATUS class for humans too).
13. Run `integration_test` module (fullstack UI suite) — it imports adminui/loginpage panels and should see the regenerated files.
14. Verify examples that embed the UI panels still build hermetically (`setup-demo`, `admin-demo`, `dashboard-demo`) — covered by `.#build` in #8, but confirm explicitly.
15. Check `templ-fmt` (still enabled) for interaction with treefmt/templ-version ownership — does it rewrite `.templ` sources in a way treefmt agrees with?
16. Assess semver impact: next adminui/loginpage tag ships the FileName-prefixed `templ.Error` values (runtime error strings change) — decide patch vs minor and note in the release runbook.
17. Check whether any consumer-facing golden tests elsewhere (setup, e2e) snapshot error strings containing `layout.templ` etc.
18. Evaluate statix (nix linter) — it appeared in BuildFlow's nix_tools.go next to deadnix; check if the step exists, is it passing, should it be enabled like deadnix?
19. Evaluate vulnix step status (binary is in devShell; is the BuildFlow step running green or silently phantom-skipped?).
20. Document the enabled-steps philosophy in the `.buildflow.yml` header comment (what belongs in skip_steps: only phantom steps for this repo type, with a falsifiable reason each).
21. Add `auto_fix` clarification comment in `.buildflow.yml`: the key does NOT disable fixing in pre-commit/dev modes (BuildFlow force-enables there) — the current silence on this misled even this session.

**P3 — upstream / structural (file, don't fix locally yet)**
22. File the BuildFlow issue/PR: per-module templ-generate provider (iterate dirs containing `.templ`, generate in each) — would let bare `FileName:` be canonical again for all multi-module repos (verify-before-filing: prototype against BuildFlow source first).
23. File the a-h/templ question: flag/env to pin `FileName:` base directory (verify in templ source whether the walk-root behavior is configurable before filing).
24. Consider BuildFlow config for language-phantom auto-skip (jest/vitest/pnpm-audit skips exist because BuildFlow can't infer "Go library, no JS") — upstream suggestion with the cqrs-htmx skip list as the case study.
25. Decide buildflow binary pinning strategy: hook runs system-path buildflow (`a3168a2`) while config schema evolves — pin the binary in the flake and have the hook resolve it, eliminating version drift between hook and config.

**P4 — deferred / roadmap fuel**
26. markdown-format: schedule the 257-file dprint normalization as its own session (bulk pass → re-enable step → never think about it again), or explicitly decline and delete the aspiration from comments.
27. eslint skip: confirm biome's scope (4 JS files per biome.json) fully covers what eslint would check; if yes, keep skip with the current comment; if no, close the gap.
28. Consider treefmt-owned deadnix (treefmt has a deadnix plugin) to unify formatter ownership — but keep the BuildFlow step working (same class of double-ownership question as the shfmt fight documented 2026-08-30).
29. Add deadnix/statix to CI checks job alongside existing gates.
30. Sweep for other CWD-dependent generators in the repo's toolchain (sqlc, stringer, govalid, mockgen if any) for the same module-dir vs root mismatch class that templ had.
31. Extend `scripts/test-install-git-hooks.sh` self-test with a templ/deadnix step-firing case so hook regressions are caught by the installer test.
32. Write the canonical-generation decision into a short ADR (ADR-0047+): "repo-root generation is canonical; FileName prefix is runtime-correct" — AGENTS.md gotchas rot; ADRs don't.
33. Silence the `templ not found in go.mod` warning: add `a-h/templ` to root go.mod (tool directive or require) or confirm the warning is cosmetic and document that.
34. `nix run .#bench-spike` after the next natural bench-relevant change (not needed for this one — FileName strings are not on hot paths; noted to prevent cargo-cult re-benching).
35. docs-health ANNOTATE pass over older status reports that mention the templ-generate skip as current truth (at least one prior report references the skip rationale).
36. HARVEST this file's P1-P3 items into TODO_LIST.md / ROADMAP.md (docs-health skill).
37. Consider a `make`-free `just`-free single entry doc: "how to regenerate UI code" one-pager in docs/guides/ (small, but the question has now cost two sessions).
38. Evaluate `--result-cache` interaction with generator steps repo-wide (steps declaring Outputs with deterministic content) — templ-generate benefited; check ln-generate-class steps if ever added.
39. Re-verify `check-templates` (SQL setup files) unaffected — orthogonal, but it's the other "generated-code-shaped" gate; one green run post-flake-edit closes the doubt.
40. Ask BuildFlow whether `skip_steps` entries emit a periodic "this skip may be stale" reminder (the stale-suppression detector exists in cqrs-lint; BuildFlow could mirror it) — upstream suggestion.

**P5 — hygiene backlog (no urgency, list for completeness)**
41. Delete the `htmx.min.js` duplicate exclude line (lines 58-59 both list a `.min.js` pattern; verify whether both globs are needed).
42. Confirm `max_file_size: 350` and `dupl_threshold: 30` are still the intended values (pre-date this session; untouched).
43. Confirm `todo_severity: debug` is still right now that documentation notes proliferate in status docs.
44. Re-read the `exclude` block for entries that became dead after module deletions (e.g. `dist`, `.next` — do these paths exist anywhere?).
45. Verify the devShell `ci` shell (line 187) — it has templ but NOT deadnix; if CI ever runs buildflow steps, deadnix would fail there; align or document.
46. Sweep `scripts/hooks/pre-commit.template` (template of truth) for anything that assumes disabled steps (budget comments, step lists).
47. Check whether `nix run .#gen` is referenced in `check-modules`/gates docs with its OLD description ("adminui templ components") — the description string changed.
48. Consider signing the next deliberate commit with the GOCACHE env prefix per the documented pre-commit inheritance gotcha (daemon commits bypass hooks entirely — another quiet divergence worth a decision).
49. After upstream BuildFlow templ fix lands (if #22 accepted): flip canonicality back to bare FileName in ONE session (regen + flake apps + AGENTS.md + changelog together), never piecemeal.
50. Archive this session'sBuildFlow-source findings (LookPath resolution, pre-commit AutoFix force-enable, step DAG gating) into a `docs/reviews/` or AGENTS.md micro-section so the next config question doesn't start from zero.

## g) Questions I Cannot Figure Out Myself

1. **Standing commit authorization:** The daemon keeps shredding deliberate, verified work into `chore: auto-commit` messages (this session: `136ae3c6`). The critical rules forbid me from committing without your explicit "commit". Do you want to grant a standing authorization to commit at phase boundaries with narrative messages (AGENTS.md's own documented rule), or would you rather tame/disable the auto-daemon during working sessions? I cannot resolve this policy tension myself.

2. **Release intent for the FileName flip:** The regenerated `_templ.go` files change runtime `templ.Error.FileName` values (e.g. `layout.templ` → `adminui/layout.templ`) and will ship with the next adminui/loginpage tags. Should the next family train intentionally include this (I'd classify it patch-level: behavior-identical except error strings), or do you want to hold the flip until the BuildFlow upstream per-module provider question (#22) is decided? This determines whether I sequence a release or leave the change riding master.

3. **markdown-format endgame:** The 257-file dprint normalization has been "pending a one-time pass" since it was skipped. Do you want that bulk pass executed as its own session (huge diff, then the step is re-enabled forever), or is markdown formatting formally not-wanted (in which case the skip comment should say "not planned" instead of "re-enable after", which is a lie by aspiration)?

---

## Verification Evidence (raw)

```
templ versions:  nixpkgs 0.3.1020 == ~/go/bin v0.3.1020
root regen:      8 files modified, diff = FileName: lines only
check-codegen:   FAILED (uncommitted) → staged → PASSED
devshell deadnix: exit 0
buildflow -s templ-generate: success, 1 no-op, 22ms
buildflow -s deadnix: 1/1 passed, 11ms
tests:           adminui ok 3.577s, loginpage ok 0.697s
nix fmt:         formatted 9 files (0 changed)
daemon commit:   136ae3c6 (10 files: 8 _templ.go + flake.nix + .buildflow.yml)
uncommitted:     AGENTS.md (staged)
```

## Session File Footprint

| File                      | Change                                                                     | State                   |
| ------------------------- | -------------------------------------------------------------------------- | ----------------------- |
| `.buildflow.yml`          | Enabled templ-generate + deadnix, honest comments, kept 3 legitimate skips | Committed (`136ae3c6`)  |
| `flake.nix`               | `gen` + `check-codegen` root-canonical rewrite; `pkgs.deadnix` in devShell | Committed (`136ae3c6`)  |
| `adminui/*_templ.go` (7)  | FileName prefix (root-canonical)                                           | Committed (`136ae3c6`)  |
| `loginpage/page_templ.go` | FileName prefix                                                            | Committed (`136ae3c6`)  |
| `AGENTS.md`               | Gotcha rewritten with verified root cause                                  | **Staged, uncommitted** |

_Format note: written as Markdown per explicit user instruction; the status-report skill's canonical format is HTML. One-off override, not propagated as a default._
