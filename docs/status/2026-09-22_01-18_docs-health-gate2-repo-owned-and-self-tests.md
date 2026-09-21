# Status Report — Docs-Health Gate 2 Made Repo-Owned + Self-Tests for Both Gates (session 2026-09-22 01:18)

**Date:** 2026-09-22 01:18 CEST
**Session scope (this run):** resume the docs-health standing order from `docs/planning/2026-09-20_15-43_docs-health-completion-and-backlog-superb-plan.md`. Concrete goal: close A6 for real — give both status gates fixture self-tests, make the row check (Gate 2) repo-owned and green-by-policy instead of a manual command pointing at a read-only skill asset, and wire what can be wired.
**Governing rules honored:** verify before strike; atomic edits; never revert another session's WIP; no repo-wide `--fix`; no destructive git ops; no `rm` (temp dirs via `mktemp` + trap).
**Repo state at writing:** branch `master`, tree **clean**; HEAD `4de015bf` ("docs(status): v4.12.0 train report — gates root-caused, 14 tags, CI green"). A sibling session cut the v4.12.0 family train and pushed it while this run was in flight. Every file this run wrote was absorbed by the auto-commit daemon as heuristic commits (`f34322c0`, `520ca386`, `64e97a07`).

---

## a) FULLY DONE

### 1. Deconfliction with the sibling session (the open Q3, resolved by reading)

Read `docs/status/2026-09-21_18-20_round6-decisions-executed-tc-v1.19-theme-sse-v007-gates.md` in full as the first action. It already completed, verified, and pushed:

| Overlapping task | Sibling status |
| --- | --- |
| C7 templ-components v1.19 train prep | **Done** — v1.19.0 authored, tagged, pushed; all 12 consuming `go.mod`s bumped; both CSS bundles rebuilt |
| C8 adminui theme toggle (M089) | **Done** — library `ThemeScript`/`ThemeToggle`, `@custom-variant dark`, e2e gate test 10/10 |
| C6 `/sse` posture one-pager | **Resolved** — `setup.Config.SSEFilter` (Option B), test green, decision doc marked RESOLVED |
| C5 ProjectionLayer v5-removal finalization | **Done** — `// Deprecated:` markers, ADR-0051, v5-removal-inventory §5b |
| C3/C4 V007 clusters 2+3 | **Done** — spike merged (`99be42f7`); cluster 1 gated by ADR-0051 criterion |
| C9 cqrs-lint Go-installable distribution | **Explicitly NOT authorized** |
| D7 datastar-demo rebrand | **Explicitly NOT authorized** |

Net effect: seven C-tier items were removed from this run's scope before any work was done. Without this read I would have re-done a pushed release train.

### 2. Reconnaissance facts established (each verified by command, not assumed)

- `scripts/check-dep-budgets.sh:57` **already** excludes standalone comment lines (`/\//` guard) — the sibling's one open gate blocker was fixed before this run; the "OVER BUDGET" class is gone.
- `git tag -l '*v4.12*'` returned nothing at recon time — the v4.12.0 train had not been cut when I started.
- `scripts/hooks/pre-commit.template` (76 lines, read in full) exports `GOCACHE`/`GOLANGCI_LINT_CACHE` via `scripts/lib/go-cache-env.sh` but **never sets `GOTOOLCHAIN`** — that is the root cause of every commit from a bare shell failing the hook.
- `flake.nix:39` pins `goPkg = pkgs.go_1_27` and the devShell exports `GOTOOLCHAIN=local`; ambient `go version` is **go1.26.7** with `GOTOOLCHAIN=local`; `go.work:1` and root `go.mod:3` both demand **1.27.1**.
- `GOTOOLCHAIN=go1.27.1 go version` → `go1.27.1` **works** (toolchain already cached at `$GOPATH/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-amd64`). So the hook fix is a one-line export, not a network dependency.

### 3. `scripts/test-check-status-annotations.sh` — fixture self-test for Gate 1 (NEW)

16 assertions, all green; shellcheck clean (after replacing two `A && B || C` helpers with `if/then/else` — SC2015). Cases: gated+annotated passes; missing `~~` fails; `~~` without `ANNOTATED` fails; **epoch boundary (exactly 2026-09-09) is gated, not exempt**; legacy report exempt; undated file fails; missing directory fails; explicit directory argument honored.

Wired in three places in the same change:
- flake app `nix run .#test-status-annotations`;
- `check-modules` stages array **and** the sequential path (`flake.nix:877`, `flake.nix:908`);
- CI `checks` job, plus the Gate 1 step itself which was **not previously in CI at all** (`.github/workflows/ci.yml:676-682`).

### 4. `scripts/check-status-rows.py` — Gate 2 is now repo-owned and green-by-policy (NEW)

The row check previously existed only as `~/.config/crush/skills/docs-health/assets/check-rows.py` — a read-only skill install, unrunnable on a clean checkout and impossible to run in CI. The new repo-owned checker encodes the **adjudicated policy** from `docs/status/README.md`:

- **FAIL** a `PARTIAL` row (cells within one row disagree — struck some cells, unstruck others). That is the real format error, and it is what the 2026-09-16 F12.3 miss shipped through.
- **REPORT, never fail**, a *deliberately mixed* table (struck rows = done, unstruck rows = open) — first-class by convention.
- Tildes inside inline code spans never count as strikethrough.

Corpus run: `✓ row gate: 405 file(s) free of PARTIAL rows (19 deliberately-mixed table(s) reported, first-class by convention)` — the 19 count **exactly reproduces** the prior session's manual `check-rows.py` finding scope (19 INCOMPLETE tables across 8 files), while being green because every one of those 19 is an adjudicated mixed table, not a format error. No split brain: the skill asset is the annotator's authoring aid (reports mixed as INCOMPLETE for a human to judge); this is the adjudicated gate.

### 5. `scripts/test-check-status-rows.sh` — fixture self-test (NEW)

12 assertions, all green. Cases: all-struck passes; PARTIAL row fails and is named; mixed table passes **and is counted**; code-span tildes are not strikethrough; header-only table passes; directory argument expands to `*.md`; missing path fails loudly and separately.

### 6. Flake apps added

`check-status-rows` and `test-status-rows` added to `flake.nix` (with `pkgs.python3` pinned in `runtimeInputs`, so the app is hermetic rather than depending on ambient python). `nix flake check --no-build` → **all checks passed** (both new apps evaluate).

---

## b) PARTIALLY DONE

1. **The row gate is half-wired.** The flake **apps** exist and evaluate, but `check-status-rows` / `test-status-rows` are **not** in the `check-modules` stages array or the sequential path, and **not** in CI. This is the exact "test script committed but wired into nothing" state that an archived 2026-08-03 report already diagnosed as dead code — I reproduced it in miniature and must finish it.
2. **`docs/status/README.md` Gate 2 section is stale.** Line 65 still says "Gate 2 (manual, per audit)" and documents `python3 ~/.config/crush/skills/docs-health/assets/check-rows.py …`. The repo-owned command is not referenced anywhere yet.
3. **The new flake apps were never actually run** (`nix run .#check-status-rows`, `.#test-status-rows`): verified by flake evaluation only; the check app pins `pkgs.python3` while the CI step and README would use ambient python3 — that asymmetry is untested.
4. **No narrative commit survived.** All four new files landed as daemon heuristic commits. The pre-commit hook is still unrunnable from a bare shell (Q1) and I did not fix it despite having proved the one-line fix.
5. **Gate-2 fidelity was spot-checked, not diffed.** My checker's 19-mixed-table count matches the skill's 19-INCOMPLETE count in aggregate; I did not diff the per-file offender lists to prove they are the same 19.

---

## c) NOT STARTED

- **A6 residual:** finish the wiring in (b)1–(b)3, then run the two apps.
- **Q1 hook toolchain fix** — `GOTOOLCHAIN` export (template + active hook + reinstall), despite the fix being proven available.
- **A7** — AUDIT health report printed inline (Accuracy + Fitness, per-doc table, visible math).
- **B1** — CHANGELOG `[Unreleased]` for the annotation sweep, archive merge, README truth, `/v4` import-path fix, and now the two new gates.
- **B4** — harvest surviving open items from the 35+ annotated reports into `TODO_LIST.md`/`ROADMAP.md` with citations; dedupe; header count.
- **A8** — annotate/archive the unarchived `docs/planning/*` execution plans.
- **A9** — `docs/DOMAIN_LANGUAGE.md` verified against code.
- **A10** — AGENTS.md size decision executed (decided SPLIT in annotation, still not executed; and now competing with the sibling's pending AGENTS cqrs-lint-row refresh).
- **B2** — docs-lint gate for `/v4`-less imports; **B3** — FEATURES `FULLY_FUNCTIONAL` re-audit.
- **Remaining C/D tier after deconfliction:** C1 bench-spike idle re-run; D1 upstream asks; D2 gate-hardening bundle; D3 bump playbook + `bump-dep.sh`; D4 blob-purge prep; D5 example smoke tests; D6 ROADMAP triage; D8 systemadapter `Volume > 0` test. (C2/C3/C4/C5/C6/C7/C8 done by sibling; C9/C10/D7 out of scope or unauthorized.)
- **Z** — final gate battery + verification.

---

## d) TOTALLY FUCKED UP (honest)

1. **I planned from a stale snapshot.** The todo list and report I resumed from described a tree at `a87424ca` with 405 archived files and v4.12.0 uncut. The actual tree was already several commits ahead and the sibling had cut and pushed a **14-tag release train**. I only noticed because `git log` after the fact showed `4de015bf … 14 tags, CI green`. My first tool call should have been `git log`/`git status`, and my second a check for status reports newer than my summary. I got lucky: reading the sibling report for Q3 happened to be the same action.
2. **I built Gate 2's checker correctly and then left it unwired.** Adding `check-status-rows.py` + its self-test + two flake apps without touching `check-modules` or CI is the precise anti-pattern an archived report in this very corpus calls out ("a test script that isn't run automatically is dead code"). I did it for Gate 1's self-test and then did not repeat the discipline for Gate 2. Half-wiring is worse than not wiring, because the next session must diff the flake to discover it.
3. **My own self-test missed a real count bug.** The first `check_file` returned `len(problems)` — which counts the "table at line N" header line *plus* each PARTIAL row — so one PARTIAL row printed "2 PARTIAL row(s)". My tests asserted the exit code and the substring `PARTIAL`, never the count, so they passed. I found the bug by eye in the printed output. A self-test that asserts only substrings is a weak test; I wrote a weak test.
4. **I did not fix the hook even after proving the fix.** I established that `GOTOOLCHAIN=go1.27.1` resolves to a cached 1.27.1 toolchain and that `go-cache-env.sh` (already sourced by the hook) is the natural home for a dynamic "align with the go.work floor" export. I then wrote nothing, so all four files landed as heuristic daemon commits and the Q1 mystery survives another session.
5. **Verification asymmetry left in place.** The flake apps pin `pkgs.python3`; my CI step and the eventual README command will use ambient python3. I did not decide which is canonical, so the gate now has two possible runtimes.
6. **`test-check-status-annotations.sh` was shellcheck-broken on first write** (SC2015 ×2) — caught only because I ran shellcheck. Good habit, weak first draft.

---

## e) WHAT WE SHOULD IMPROVE

1. **Atomic per-gate changes.** A gate ships as ONE change: checker + fixture self-test + flake app + `check-modules` stage + CI step + README command. Anything less is half-wired by definition. Plan the six edits before writing line one.
2. **Reconcile at session start, always.** `git log -n 10` + `ls -t docs/status/*.md | head` before planning. In a repo with a fast auto-commit daemon and concurrent sessions, any inherited todo list is a hypothesis, not a fact.
3. **Assert counts, not just substrings.** Self-tests should compare exact numbers ("1 PARTIAL row") and exact offender paths wherever the gate prints them.
4. **Fix the hook's toolchain now.** The pre-commit hook silently bypassing for every shell except `nix develop` means "commit at phase boundaries" — a rule this repo has lost narrative commits over repeatedly — is currently unenforceable.
5. **Single-source the annotation policy.** The convention now lives in three descriptions (README prose, `check-status-annotations.sh` header, `check-status-rows.py` docstring). They agree today; nothing keeps them agreeing.
6. **Decide python ownership.** Either python3 goes in the devShell (then CI, README, and flake apps are uniform) or it does not (then the flake app must be the only entry point and the README must say so).
7. **Prefer one checker over two where policy allows.** The skill asset and the repo gate classify rows the same way but judge mixed tables differently. That divergence should be stated once, in the README, with a pointer from each tool.
8. **Stop-loss on scope creep.** A6 was nominally "adjudicate the gate"; it has now produced two scripts, two self-tests, four flake apps, a CI step, and still has two wiring edits outstanding. Bound it: finish the wiring, then move to A7/B1.

---

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

**Tier 0 — finish this run's half-wired state (minutes)**

1. Add `check-status-rows` to the `check-modules` stages array and the sequential path in `flake.nix`.
2. Add `test-status-rows` to the same two lists.
3. Add both to the CI `checks` job beside the annotation gate steps.
4. Run `nix run .#check-status-rows` and `nix run .#test-status-rows` for real.
5. Update `docs/status/README.md` lines 57–70: Gate 2 = `bash scripts/check-status-rows.py` (repo-owned, CI-runnable); keep the skill command as the annotator's authoring aid and say so explicitly.
6. Decide python3 ownership (devShell vs flake-app-only) and make CI/README consistent with it.
7. Diff my 19 mixed tables against the skill's 19 INCOMPLETE lines per file to prove scope equality.
8. Strengthen the row self-test to assert the exact PARTIAL count and the exact file name.

**Tier 1 — the hook (Q1)**

9. Add dynamic toolchain alignment to `scripts/lib/go-cache-env.sh` (read `go.work`'s `go` directive; export `GOTOOLCHAIN=go<floor>` only when ambient is older — never downgrade a newer ambient toolchain).
10. Reinstall via `scripts/install-git-hooks.sh --force`; verify with `--verify`.
11. Prove it with a real commit from a bare shell (non-devShell) that reaches buildflow.
12. Extend `scripts/test-install-git-hooks.sh` with a toolchain-alignment case.
13. If buildflow ignores the exported `GOTOOLCHAIN`, fall back to documenting the devShell commit path and stop trying to patch it.

**Tier 2 — close the docs loop**

14. A7 — print the AUDIT health report inline (Accuracy + Fitness, per-doc table, visible math, no file written).
15. B1 — CHANGELOG `[Unreleased]`: annotation sweep, archive merge, README truth pass, `/v4` import-path fix, Gate 1 gate + self-test, Gate 2 repo-owned gate + self-test.
16. B4 — harvest surviving open items from the 35+ annotated reports into `TODO_LIST.md`/`ROADMAP.md` with `file:line` citations.
17. B4 — dedupe against existing entries (the DOMAIN_LANGUAGE duplicate; the V007 cluster).
18. B4 — update the TODO_LIST header count.
19. A8 — classify the ~19 `docs/planning/*` plans (live vs superseded) and archive the dead ones, fixing inbound links.
20. A9 — `docs/DOMAIN_LANGUAGE.md` terms vs code; fix drift or file it.
21. A10 — inventory AGENTS.md temporal-pollution lines; extract lean core; verify retained paths resolve.
22. B2 — add the `/v4`-less-import rule to `check-docs-freshness.sh` + failing/passing fixtures + `check-modules` wiring.
23. B3 — batch-audit root/usermgmt/UI `FULLY_FUNCTIONAL` FEATURES rows; downgrade unverifiable ones.
24. Refresh `docs/status/README.md` archived-count line (405) if the sibling archived more.

**Tier 3 — the deconflicted technical tail**

25. C1 — check load, attempt `nix run .#bench-spike`, record the verdict (or the refusal).
26. C1 — re-pin the raw baseline only if the 10% gate trips, in the same change.
27. D8 — systemadapter `Volume > 0` regression test + system-demo display verification.
28. D5 — `examples/basic` + `examples/datastar-demo` smoke tests.
29. D5 — catalog-demo + samber-do SSE smoke tests.
30. D6 — triage ROADMAP candidates (metrics recorder, SQL checkpoint/DLQ, hydrator B/C/D, setup surface).
31. D2 — verify-tag tag-message guard, `nix flake check` with builds, MD024 exclusion, LICENSE check, coverage flock.
32. D3 — `docs/runbooks/dependency-train-bump.md` + `scripts/bump-dep.sh` + `go work sync` policy.
33. D4 — blob list + sizes for the v4 branch; filter-repo and setup-demo purge recipes (no push).
34. D1 — re-verify and file the templ-components asks (a–e).
35. D1 — file the go-cqrs-lite decouple/injection asks.
36. C10 — draft the ADR-001 appkit verdict section with comparison-report evidence.

**Tier 4 — cross-session hygiene**

37. Sibling handoff: confirm their open items (dep-budget fix — already landed; cqrs-lint posture; AGENTS cqrs-lint row; full e2e rerun on the final tree).
38. Adopt the sibling's pending AGENTS.md rows instead of racing them.
39. Remove the stale worktrees (`../cqrs-htmx-v007`, `/tmp/cqrs-head`) and the round-5 demo server if still running.
40. Re-run the full gate battery (`check-modules`, `check-docs-freshness`, `check-docs-links`, both status gates) before declaring any docs sweep done.
41. Add a "new gate checklist" to AGENTS.md (checker + self-test + app + check-modules + CI + README).
42. Sweep `docs/status` for links broken by archive moves (sibling's item).
43. Verify the two new scripts are inside the large-file guard's scope and shellcheck-clean in CI's `find scripts -name '*.sh'`.
44. Consider a `docs-check` composite app (freshness + links + both status gates + both self-tests).
45. Measure `check-modules --report` wall time after adding two more stages; keep the composite fast.
46. Decide whether `check-status-rows.py` should also scan `docs/status/*.md` (unarchived tail), not just `archived/`.
47. Add a mixed-table count regression line to the row gate's output target doc so the 19 is tracked.
48. Re-verify annotation gate counts after the sibling's v4.12.0 report lands in `docs/status/`.
49. Annotate + archive this session's report once its items resolve (the standing convention).
50. Z — commit per phase once the hook is fixed, then final verification.

---

## g) ASK ME UP TO 3 QUESTIONS

**Q1 — the pre-commit hook's toolchain (unchanged from last session, now with proof).**
The hook runs `go` with the ambient 1.26.7 under `GOTOOLCHAIN=local` while `go.work` requires 1.27.1, so every commit from a bare shell bypasses the hook and lands as a heuristic daemon commit. I have now proven `GOTOOLCHAIN=go1.27.1` resolves instantly to a **cached** 1.27.1 toolchain, and that `scripts/lib/go-cache-env.sh` (already sourced by the hook at line 35) is the natural home for a dynamic "align with the go.work floor, never downgrade" export. Do you want me to (a) make that change + reinstall + prove it with a real commit, or (b) leave the hook alone and keep the documented `--no-verify`/devShell fallback?

**Q2 — who owns python3?**
The two new flake apps pin `pkgs.python3` (hermetic), while CI and the future README command would use the runner's/ambient python3. Which is canonical: (a) add `python3` to the devShell and treat ambient python as the contract everywhere, (b) make the flake apps the only sanctioned entry point (CI then calls `nix run .#check-status-rows`), or (c) rewrite the row gate without python (awk/go) to avoid the question entirely?

**Q3 — scope after this report.**
You said to report and wait. When you release me, what is the priority order: (a) finish the half-wired Gate 2 state (items 1–8, minutes), then re-run the deconflicted A7/B1/B4 tail; (b) go straight to A7/B1/B4 and leave the wiring to a follow-up; or (c) adopt the sibling's remaining release-tail items (cqrs-lint posture, AGENTS rows, full e2e) as mine, since they now touch the same files I would edit (AGENTS.md, CI, flake)?

---

*Session paused pending Q1–Q3 and further instructions. Nothing is blocked except the decisions above; the half-wired Gate 2 state is the only known-unfinished work this run created.*
