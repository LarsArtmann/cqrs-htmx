# Status Report — Push Unblock Attempt & Broken templ-components Release

**Date:** 2026-10-05 15:42 CEST
**Session:** Continuation of the branching-flow analysis pass — plan authoring, dependency-train alignment, and push attempt.
**Scope:** (1) author and commit the Pareto plan for the branching-flow findings; (2) `git commit` + `git push` as instructed; (3) root-cause the push block.
**Format note:** The `status-report` skill's canonical output is HTML; the operator explicitly requested `.md` at `docs/status/<YYYY-MM-DD_HH-MM_WELL-NAMED>.md`. Override honored and recorded (recurring divergence, see §e-8).

---

## Session outcome in one line

The plan is written and committed; 12 dependency-family alignment commits landed green; **`git push` is blocked by an upstream broken release (`templ-components@v1.20.0`)** that this repo cannot fix.

---

## a) FULLY DONE

Evidence = command output in this session; SHAs are local.

- **Pareto execution plan authored** → `docs/planning/2026-10-05_15-04_SUPERB-branching-flow-ratchet-pareto-plan.md` (299 lines). Contains: Pareto 1%/4%/20%/other breakdown, a 25-row comprehensive table (30–100 min), an 84-row fine-grained table (≤12 min), a Mermaid execution graph, a decisions ledger freezing every rejected analyzer class with its reason, and do-not-verschlimmbessern guards. Committed `6e5b7404`.
- **Key discovery recorded:** `branching-flow all` ships `--baseline` (SARIF diff, fail-on-NEW), `--format sarif/json/markdown/…`, and `--exclude-generated`. This makes the ratchet a config change, not a build. Evidence: `branching-flow all --help`.
- **12 exact-anchor family alignment commits, all green** (each module passed hermetic `tidy + go mod verify + build + vet` via `scripts/tools/bump-dep.sh`):
  - `larsartmann/go-output` + `/d2 /delimited /graph /markdown /markup /plantuml /serialization /table /tree` → `v0.38.4` (10 commits)
  - `larsartmann/go-health` → `v0.5.0`
  - `larsartmann/samber-do-auditlog` → `v0.11.0`
- **Push blocker root-caused with hard evidence.** `templ-components@v1.20.0`'s published root `go.mod` requires submodules at workspace placeholder pseudo-versions:
  ```
  templ-components/charts/echarts   v1.20.0-00010101000000-000000000000
  templ-components/datastar         v1.20.0-00010101000000-000000000000
  templ-components/errorpage        v1.20.0-00010101000000-000000000000
  templ-components/htmx             v1.20.0-00010101000000-000000000000
  ```
  `git ls-remote` grep for `00010101000000` = empty → those tags do not exist → the release is unconsumable. Root v1.20.0 downloads, but any consumer bump explodes with `errorpage@v1.20.0-0001…: unknown revision`.
- **Failed sweep reverted cleanly, zero damage.** `git status` clean; the auto-revert (`git diff --name-only | xargs git restore`) worked exactly as designed.
- **Both prior artifacts committed:** status report `8dbf7146` (daemon) + plan `6e5b7404`.

---

## b) PARTIALLY DONE

- **The push.** Everything is staged/committed locally (16 commits ahead of `origin/master`), but the pre-push `check-release-train --strict-lag 0` gate fails on 64 templ-components lag items. **What works:** docs + valid dep alignments committed. **Blocked by:** upstream broken release. **Effort to finish:** S once upstream re-releases.
- **Dependency alignment.** go-output/go-health/samber-do-auditlog are aligned (lag for those = 0). **Still open:** templ-components (64 items) — not mechanically alignable. **Effort:** M (blocked).
- **The 25/84-task plan.** Authored and committed, **not executed** (this was a planning+push session). Effort: the plan itself is the artifact.
- **Upstream issue for the broken release.** Not filed. **Blocker:** none — just not started; I offered it and awaited the call. Effort: S.

---

## c) NOT STARTED

- **`git push`** — blocked (not a failure of intent; external limit).
- **File the templ-components upstream issue** (`verify-before-filing` gate first). Still wanted: YES, high value.
- **HARVEST** of plan items into `TODO_LIST.md`/`ROADMAP.md` (mandated by both `pareto-planning` and `status-report`; deliberately deferred).
- **All 25 comprehensive plan tasks** (T1–T25) — including the ratchet foundation (T1), the three safe fixes (T3–T5), and the credential decision memo (T7).
- **templ-components CSS-bundle rebuilds** (`.#build-adminui-css`/`.#build-dashboardui-css`) — only relevant if/when the family becomes consumable.
- **`nix fmt` on the new planning `.md`** — not run.
- **The `--no-verify` push decision** — deferred to owner.

---

## d) TOTALLY FUCKED UP

Ranked by real severity, honestly:

1. **I attempted a prefix sweep (`larsartmann/go-output`) instead of reading the recipe's `$` anchors first.** It pulled in `daghtml`/`escape` (own trains, no v0.38.4) and failed with `unknown revision`. The tool reverted, but this was a **wasted attempt caused by my shortcut**, and it is the exact trap the tool's own header documents. Severity: low (no damage). Root cause: I optimized for fewer commands over reading the tool contract.
2. **I repeated a known shell trap.** The first gate re-run reported `GATE_RC=0` because I piped through `tail` and captured tail's rc — the `PIPESTATUS`/`$?`-after-pipe trap is AGENTS gotcha 3, which I had literally just cited. Severity: low (corrected immediately). Root cause: carelessness.
3. **12 unpushable dependency commits.** They are individually valid and green, but they cannot be pushed (upstream block) and may need rebasing if another session re-releases templ-components differently. Net effect: local churn that may or may not be wasted. Severity: medium. Root cause: I aligned "everything the gate named" before checking whether the target release was even consumable.
4. **The requested deliverable (push) did not happen.** Cause is external (broken upstream), and I root-caused it rather than forcing it — but the operator's explicit instruction is unmet.

Nothing in the repo is broken; no working code was damaged; tree is clean.

---

## e) WHAT WE SHOULD IMPROVE!

**Process (this session):**
1. **Read the tool's own contract before invoking it.** The `$`-anchor rule was in `bump-dep.sh`'s header; skipping it cost a failed attempt. Rule: if a script documents a flag/trap, read it before the first call.
2. **Never capture rc through a pipe.** Capture `cmd > file 2>&1; rc=$?`, always — even on "quick" re-checks.
3. **Detect the moving target and stop.** Lag went 36 → 64 between two `--refresh-cache` runs. When the number *grows* despite fixes, the correct move is to stop and find out *why the upstream published*, not to sweep again.
4. **Check consumability before aligning to a version.** A gate says "newer version published"; it does not say "newer version is consumable". Validate a candidate's `go.mod` for placeholder pseudo-versions (`-00010101000000`) *before* a family sweep.

**Tooling/design (repo + upstream worth filing):**
5. **`bump-dep.sh` should pre-flight the target release.** Detect `-00010101000000-000000000000` placeholder pseudo-versions in a candidate family's `go.mod` and refuse with "upstream release is broken" instead of the misleading downstream `unknown revision …/<sub>/go.mod` error.
6. **`check-release-train` needs an "upstream-broken" escape.** Today the only consumer-side options are "infinite red" or `--no-verify`. A recorded, expiring allowlist ("this published tag is known-unconsumable; suppress lag until TTL") would be honest and safe.
7. **A release-time guard in `templ-components` itself** — a publish step that refuses to tag a root module whose `go.mod` contains workspace placeholder versions. This is the root cause of everything above; the fix belongs upstream.
8. **Recurring skill/format divergence.** Both `status-report` and `pareto-planning` are HTML-canonical, yet every operator dispatch in this repo demands `.md`. Consider recording a repo-local standing exception so agents stop flagging it every time.

**Self-critique (asked explicitly):**
9. **What did I forget?** — (i) read `bump-dep.sh`'s contract first; (ii) capture rc directly; (iii) validate release consumability before sweeping; (iv) stop when lag grew; (v) `nix fmt` the new plan; (vi) record the pre-existing unpushable state (origin had not moved → the repo has been unpushable for a while, a signal I read late).
10. **What could I have done better?** — Ordered the work as *diagnose-then-act*: before touching pins, run one cheap probe (`git ls-remote` + fetch the candidate `go.mod`) to prove the target is consumable. That single check would have prevented items d1–d3.
11. **What could I still improve?** — Turn d1–d4 into a **pre-flight script** (`scripts/checks/check-family-release-consumable.sh`) so the next agent gets a one-line verdict instead of rediscovering the placeholder trap.

---

## f) Top next tasks (up to 50; ranked; later items are ROADMAP fuel)

### Push unblock (highest priority)
1. File the templ-components upstream issue: root `v1.20.0` go.mod contains workspace placeholder pseudo-versions → unconsumable — **High / S / Bug-report**
2. Decide push path (wait for re-release vs `--no-verify`) — **High / S / Decision**
3. When upstream re-releases, re-run the templ sweep + CSS bundles + gate + push — **High / M / Release**
4. Re-verify go-output/go-health/samber-do-auditlog alignments survive any rebase — **Medium / S / Release**

### Guard against the whole class
5. Add consumability pre-flight to `bump-dep.sh` (placeholder-version detection) — **High / M / Tooling**
6. Add `scripts/checks/check-family-release-consumable.sh` — **High / M / Tooling**
7. Propose an "upstream-broken" expiring allowlist to `check-release-train` — **Medium / M / Tooling**
8. Record the broken-release episode in `docs/agents-notes.md` (gotcha-24 family) — **Medium / S / Documentation**

### Harvest + loop-closing
9. HARVEST plan T1–T14 into `TODO_LIST.md` — **High / S / Documentation**
10. HARVEST plan T15–T25 into `ROADMAP.md` — **Medium / S / Documentation**
11. Annotate the branching-flow report + plan with done-markers — **Low / S / Documentation**
12. `nix fmt` the planning `.md` — **Low / S / Quality**

### The plan's own 1% / 4% (executable now, independent of push)
13. T1 — SARIF baseline + `.#check-branching-flow` gate — **High / L / Tooling**
14. T3 — `commandOptionApplier` drift guard — **High / S / Quality**
15. T4 — `pendingTOTPStore` typed key (+ gotcha-25 display-form check) — **High / M / Quality**
16. T5 — `plainBodyWriter` options struct — **High / S / Quality**
17. T2 — triage-decisions ledger — **Medium / M / Documentation**
18. T6 — isolate the 4 high `strong-id` rows — **Medium / S / Analysis**
19. T7 — credential 4× decision memo — **High / L / Quality**
20. T8 — per-module coverage proof — **Medium / S / Analysis**
21. T9 — wire the gate into `check-modules` + CI — **High / L / Tooling**
22. T11 — AGENTS entry for `branching-flow` — **Medium / S / Documentation**
23. T14 — analyzer-subset tuning — **Medium / M / Tooling**
24. T23 — memory note (tool + commands) — **Low / S / Documentation**

### Process hardening (from §e)
25. Add a "validate consumability before aligning" note to the dependency runbook — **Medium / S / Documentation**
26. Add rc-capture guidance to the shell-reference runbook — **Low / S / Documentation**
27. Add a "moving-target lag" stop-rule to the release-train runbook — **Low / S / Documentation**
28. Draft the `--no-verify` justification convention (gotcha 6) as a copy-paste block — **Low / S / Convention**

### Speculative / ROADMAP
29. T10 — JSON/SARIF nightly digest — **Low / M / Tooling**
30. T15 — navItem dup comment — **Low / S / Cleanup**
31. T16 — `Capabilities` `Has*()` helpers — **Low / M / Quality**
32. T17 — `flagparam` low item — **Low / S / Quality**
33. T18 — credential data-model review — **Medium / L / Quality**
34. T19 — `phantom` re-eval — **Low / M / Analysis**
35. T20 — `mixins` non-wire re-eval — **Low / M / Quality**
36. T21 — branching-flow ↔ cqrs-lint overlap — **Low / M / Analysis**
37. T22 — suppression-reason convention — **Low / M / Convention**
38. T24 — verdict-table template — **Low / M / Process**
39. T25 — FP-rate tracking — **Low / M / Process**
40. Add `crypto/mldsa`/go-licenses devShell shim adoption (carried from AGENTS gotcha 22) — **Low / M / Tooling**
41. Explore `branching-flow` v0.2.0 vs upstream releases — **Low / S / Analysis**
42. Add `branching-flow` to the devShell for discoverability — **Low / S / Tooling**
43. Decide whether `contextguard` becomes a permanent cheap gate — **Low / S / Tooling**
44. Add a scheduler/reminder for family-lag sweeps (avoid surprise blocks) — **Low / M / Tooling**
45. Document the placeholder-pseudo-version failure mode in the dependency runbook — **Medium / S / Documentation**
46. Add a release-time guard to `templ-components` (refuse to tag broken root go.mod) — **High / M / Upstream**
47. Audit other Lars multi-module repos for the same broken-root-release class — **Medium / L / Analysis**
48. Add a CI check that any pinned family root's go.mod is consumable — **Medium / M / Tooling**
49. Track the "unpushable for N commits" metric as a health signal — **Low / M / Process**
50. Reconcile the two local status/plan artifacts' staleness after the next train — **Low / S / Documentation**

---

## g) Questions I cannot answer myself (Top 3)

1. **Push path:** Wait for a fixed `templ-components` re-release, or push now with `--no-verify` (accepting red CI on an upstream-caused gate)? I can't decide policy — the repo's own TODO marks pushes as "owner — your call". *(unblocks items 2/3/4)*
2. **Upstream filing:** Should I file the templ-components issue (and does it go to that repo's tracker under my usual voice)? I tried to answer it by inspecting the tag/go.mod evidence — the release is definitely broken — but whether/when to file, and where, is yours. *(unblocks item 1)*
3. **The 12 dep commits:** Keep them as-is (they will need rebasing if upstream re-releases differently), or revert them so the tree only carries the docs commits until the family wave is handled centrally? I can't judge whether your release process wants consumer-side alignment scattered across commits. *(unblocks items 4/9)*

**Status:** WAITING FOR INSTRUCTIONS.
