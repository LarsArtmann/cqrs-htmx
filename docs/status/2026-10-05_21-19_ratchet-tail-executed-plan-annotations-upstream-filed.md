# Status Report — Ratchet Tail: Plan Execution, Annotations, Upstream Filing

**Date:** 2026-10-05 21:19 CEST
**Session:** Continuation of the branching-flow ratchet session — executing the carry-forward todo list from the round-2 plan (`docs/planning/2026-10-05_15-48_SUPERB-push-unblock-ratchet-and-release-guards-plan.md`), operator order: "GET SHIT DONE! The WHOLE TODO LIST!"
**Format note:** The `status-report` skill is HTML-canonical; the operator explicitly demanded `.md` at this path — standing repo override, flagged here per skill policy.
**Report basis:** this session's run only (started from the WAITING state after the 20-31 report). Supersedes the 20-31 report's "remaining work" list.

---

## a) FULLY DONE

All with evidence; session commits `05213817`-era (daemon trio), `47a88a88`, `471c0933`, `ee59ac0b` (+ daemon `c57693dd`/`9f2586ed` carrying 3 annotation files), `3af81157`.

- **T5 exhaustruct finding FIXED — better than planned.** Instead of re-placing the fragile trailing nolint (golines had already stranded it once), `plainBodyOptions` joined `exhaustruct_v5`'s ignore-patterns in `.golangci.yml` (same precedent class as `readinessDetail` — zero/partial literals ARE the intent). Both nolints deleted; formatter-proof forever. Verified: root module `golangci-lint run` → **0 issues**; root module race tests green (4.06s).
- **Baseline ratcheted 663 → 659 and gate GREEN.** Regenerated SARIF, class-level diff proved a PURE ratchet-down (-2 FLAG_PARAM at the `plainBodyWriter` def — T5's target; -2 PHANTOM_TYPE at its call sites; zero new findings anywhere), minified to 807 KB, installed, ledger/README/AGENTS counts updated. `nix run .#check-branching-flow` → **rc 0, +0 added** (the gate was RED at session start). The uncommitted-baseline guard fired correctly mid-flow (by design) and was satisfied by committing code + baseline together.
- **R7: 12 dep commits KEPT + rationale recorded.** Recorded in the round-2 plan annotation (row R7) and open question Q3: all 12 on origin, individually verified green last session; revert = force-push (forbidden) or pointless churn; re-evaluate only if upstream re-releases differently. (Honesty note: see §d — the 7.1 re-enumeration was NOT independently redone.)
- **R14/T12 HARVEST executed** (`471c0933`): 4 actionable P2 items (consumability pre-flight gate R2, bump-dep pre-flight R18, credential decision memo R11, runbook updates R20) + 2 P3 items (analyzer-subset tuning R15, per-module count proof R12) into TODO_LIST; ROADMAP gains **OQ28** (credential 4× intent — owner question ① from the 20-31 report, now a standing open question) + a dated residual block (R10, R19, R21–R23, R25); T17 closed won't-implement per the ledger verdict. Deduped against existing TODO content; evidence cited per item.
- **T13/R16 ANNOTATE executed**: the 14-59 analysis report (§b/§c inline verdicts, §f items 1–40 per-item done/routed/won't-implement markers, §g questions answered or routed, Status line superseded), plus BOTH plans (all 25 task-table rows each with verdicts; Q1/Q3 answered, Q2 → OQ28) — `> ANNOTATED 2026-10-05` blockquotes on all three.
- **R17 UPSTREAM ISSUE FILED: [templ-components#27](https://github.com/LarsArtmann/templ-components/issues/27).** All 5 verify-before-filing gates passed BEFORE drafting: proxy go.mod for v1.20.0 shows 4 submodule requires at zero-commit pseudo-versions; ls-remote proves all 4 real submodule tags EXIST (so the diagnosis sharpened from "no tags" to "replaces present at tag time kept pseudo-versions" — icons/utils resolved correctly, the 4 with replaces did not); proxy v1.19.4 proves the regression (all six clean); no fix tag exists (v1.20.0 latest); no duplicate issue (the concurrent session's templ#1449 work is a different repo — ownership question ③ resolved: nobody had filed). Voice check 0 FAIL/0 WARN; draft receipt kept in `docs/drafts/` (never /tmp); fleet-watch TODO row updated with the receipt.
- **CHANGELOG entries written** (`3af81157`): ratchet gate, drift guard, plainBodyOptions + T4 doc contract, codegen regen fix, SyntheticUserID lint fix — 4 entries across Added/Fixed/Changed.
- **Final verification ran honestly**: `nix run .#check-modules -- --report` — 28/33 green including BOTH session-owned stages (`branching-flow` ✅ — red at session start — and its self-test ✅), status-annotations/rows/docs-links all ✅.

---

## b) PARTIALLY DONE

- **Final verification (attribution complete, closure blocked).** 5 of 33 stages red: `workspace-build`, `version-drift`, `release-train`, `css-bundle-classes`, `docs-freshness`. ALL five trace to the templ-components v1.20.0 situation (the exact break filed as #27) hitting the concurrent session's in-flight family sweep — a daemon commit (`97a6aafb`, 20:37, templ-components v1.20.0 requires across 12+ modules) landed mid-session and a FURTHER uncommitted sweep (28 go.mod/go.sum + setup-demo CSS) was live in the working tree at verification time. Not fixable locally: the only local unblock is a go.work replace-shield (owner-gated by the plan's decision ledger) or upstream v1.20.1. Left untouched per the never-revert-foreign-changes rule.
- **R7 (decision recorded, enumeration skipped).** The verdict and rationale are recorded, but sub-task 7.1 ("enumerate the 12 commits + their content") was not independently re-executed this session — I relied on the prior session's recorded verification. The verdict is robust regardless (revert path is forbidden), but the evidence chain has one inherited link.

---

## c) NOT STARTED

Deliberately routed this session, not started (living in TODO_LIST/ROADMAP per the harvest):

- **TODO_LIST P2:** R2 family-release consumability pre-flight gate · R18 bump-dep pre-flight · R11 credential decision memo · R20 runbook updates.
- **TODO_LIST P3:** R15 analyzer-subset tuning · R12 per-module candidate-count proof.
- **ROADMAP:** R10 strong-id high rows · R19 expiring upstream-broken allowlist proposal · R21 credential data-model review · R22 phantom/mixins re-eval · R23 JSON digest · R25 small-findings bundle.
- **Owner-gated:** OQ28 answer (credential intent), push timing once upstream lands, replace-shield interim decision.

---

## d) TOTALLY FUCKED UP!

Nothing shipped broken (lint 0, tests green, gate rc 0, tree state honest) — but this session had TWO real editing failures, both caught and repaired in-flow, both worth naming:

- **I deleted a function's arguments with a careless multiedit.** Repairing the errors.go nolint, my `new_string` was just `handleErrorCore(` — silently dropping `w, r, err, loginRedirect, plainBodyWriter(...)`. Caught by immediately viewing the result; repaired; verified by lint (compiles) + race tests. Root cause: I wrote the replacement pair as a prefix instead of the full block.
- **I broke the round-2 plan's table schema mid-annotation.** Done rows got 7 cells, open rows 6 (I redesigned the table shape mid-edit without recounting). Detected only AFTER applying; fixed with a 13-edit repair pass; verified mechanically (awk cell counts: exactly 27×7 per comprehensive table). Root cause: cell-count verification came after the edit, not as part of it.
- **First §f annotation batch failed on a false contiguity assumption.** My old_string joined report items 1–15 as one block, but subsection headings (`### Analysis hardening`, `### Gate / CI integration`) sit between them. And on the partial-failure report I initially misread WHICH edits had failed (assumed §g; it was the two §f blocks) — one wasted verification cycle before re-viewing the actual file.

---

## e) WHAT WE SHOULD IMPROVE!

Process (this session's lessons):

1. **Multi-line Go edits: never hand-shrink `new_string`.** For call-site edits, replace the exact full block or use `lsp_replace_symbol`/single surgical edits. The errors.go incident is the canonical near-miss.
2. **Markdown table rewrites: verify cell counts as PART of the edit loop**, not after — one awk pass per edited table would have caught the 6-vs-7 split before the repair pass.
3. **On partial multiedit application, diff-read every target region immediately** — do not infer which edits failed from the ordering of the result message (I got it backwards once).
4. **Use `nix run .#preflight-tree-check` before batch doc commits during daemon-active periods.** The daemon raced me once this session (took 3 of 5 staged files into heuristic commits — content verified complete, but attribution got messy).
5. **Do not restructure historical docs' schemas.** Annotating rows = correct; I also renamed both plans' table headers to "ID & task (annotated 2026-10-05)" to fit the merged verdict column — a borderline rewrite of a historical artifact's structure (flagged as question g3).
6. **R7-class decisions: re-run the enumeration step even when a prior session recorded it** — a decision receipt should be rebuildable from this session's evidence, not inherited.

---

## f) Top next tasks (ranked; later items are ROADMAP fuel)

### Unblock / release path
1. **Watch templ-components#27** → upstream v1.20.1 re-cut (fix: drop the 4 submodule replaces + tidy + tag, per the issue's Fix section).
2. **Complete (or revert) the in-flight templ-components v1.20.0 sweep** — 28 modified go.mod/go.sum sit uncommitted in the tree (concurrent session owns; do not touch without coordination).
3. **After the sweep settles: rebuild both CSS bundles** (`.#build-adminui-css` / `.#build-dashboardui-css`) — `css-bundle-classes` red, the family-bump same-change rule.
4. **After the sweep settles: fix the `docs/agents-notes.md:89` "uniform at v1.19.4" claim** — `docs-freshness` red.
5. **Re-run `nix run .#check-modules -- --report` on a quiet tree** — this session's run measured a foreign in-flight sweep (version-drift mid-sweep).
6. **Push decision (owner)** once upstream lands + local gates green — pre-push hook enforces strict release-train.

### TODO_LIST (routed this session)
7. **R2: family-release consumability pre-flight gate** — checker + self-test + flake app + check-modules + CI + README, atomic (gotcha 19); kills the #27 class at the source.
8. **R18: bump-dep pre-flight** — detect placeholder pseudo-versions, error "upstream release is unconsumable" instead of the misleading `unknown revision`.
9. **R11: credential 4× decision memo** (read-only; feeds OQ28).
10. **R20: runbook updates** — validate-before-align, rc-capture discipline, moving-target stop-rule into `dependency-train-bump.md`.
11. **R15: analyzer-subset tuning** for the ratchet gate.
12. **R12: per-module candidate-count proof** (gotcha-2 class guard).

### Quality / follow-through
13. **Re-verify the T3 drift-guard test under workspace mode** once templ-components unblocks — GOWORK=off usermgmt tests pin the PUBLISHED root module, so the behavioral test currently asserts published-root behavior.
14. **Annotate the 20:31 status report** as superseded by this one (its §g owner questions ①/③ are now resolved: OQ28 filed, #27 filed; ② resolved by the baseline refresh + amendment).
15. **Existing TODO watch items:** loginpage coverage re-pin (quiet window), bench-spike quiet-window retry, treefmt-nix#545 / BuildFlow#29 / templ-components#27 responses.

### ROADMAP fuel (routed, not worker tasks)
16. R10 strong-id 4 high rows · 17. R19 expiring allowlist proposal · 18. R21 credential data-model review · 19. R22 phantom/mixins re-eval · 20. R23 JSON nightly digest · 21. R25 small-findings bundle (navItem comments, Capabilities.Has*, cqrs-lint overlap doc, suppression convention, verdict template, FP-rate tracking).

### Owner decisions
22. OQ28 credential intent · 23. Replace-shield interim yes/no · 24. Push timing.

---

## g) Questions I cannot answer myself

1. **OQ28 (credential 4× duplication):** intentional anti-corruption/Published-Language boundary set (keep), or sprawl to consolidate? I can see the four field sets and boundary roles (`CredentialCore` / `CredentialView` / `credentialData` / `webAuthnUserCred`); the domain intent is yours. The R11 memo is queued as your input, but the call itself gates any code.
2. **Interim replace-shield:** until templ-components ships a consumable v1.20.1, should a go.work `replace templ-components => /home/lars/projects/templ-components` be added (recorded, load-bearing) to unblock workspace builds — or do we hold the line and keep workspace-mode red? The plan's decision ledger says no shield without your approval; the concurrent session's sweep makes this live NOW.
3. **Historical-doc annotation form:** I renamed both plans' table headers to "ID & task (annotated 2026-10-05)" to carry verdicts inline. Acceptable annotation, or should historical docs keep their original header schema (verdicts in a trailing column instead)? Deciding this wrong either erodes "never rewrite history" or makes every future plan annotation awkward.

**Status:** WAITING FOR INSTRUCTIONS.
