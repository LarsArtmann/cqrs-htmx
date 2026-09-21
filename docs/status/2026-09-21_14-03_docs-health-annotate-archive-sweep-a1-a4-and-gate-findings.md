# Status Report — docs-health ANNOTATE + ARCHIVE Sweep A1–A4 Complete + Completeness-Gate Findings

**Date:** 2026-09-21 14:03 CEST
**Session scope:** Finish the ANNOTATE + ARCHIVE half of the docs-health standing order (the plan `docs/planning/2026-09-20_15-43_docs-health-completion-and-backlog-superb-plan.md`, workstreams A1–A7 + B1). This session executed **A3, A4 and A5** end-to-end; ran the **A6** completeness gates and surfaced a corpus-level finding that needs a decision. A1/A2 were completed in the immediately preceding continuation and are included here for a whole-picture count.
**Verified state at writing:** tree clean (auto-commit daemon absorbed every rename/edit into heuristic commits — no narrative commit survived, the documented race); `docs/status/archived/` = **402 files**; `docs/status/*.md` = **4 reports + README**; this sweep's 35 files all carry inline strikethroughs (**1,862 `~~` markers**, A3 = 584, A4 = 220).

---

## a) FULLY DONE

1. **A3 — 13 reports (2026-09-18 / 2026-09-19) annotated + archived.** Each got a dated `> ANNOTATED 2026-09-20` blockquote with per-section verdicts + routing, plus inline `~~…~~ done (…)` strikes on every confirmed-done item:
   - `2026-09-18_05-42_dashboardui-adoption-run-2-complete.md` (90 markers)
   - `2026-09-18_05-44_setup-gap-bundle-brutal-self-review.md`
   - `2026-09-18_07-44_broadcast-move-followup-docs-gates-and-cqrs-lint-4.8.1-drift-repair.md`
   - `2026-09-18_07-52_pareto-plan-execution-session-status.md`
   - `2026-09-18_10-15_command-audit-pareto-completion-status.md`
   - `2026-09-18_15-50_gate-session-verification-battery-status.md`
   - `2026-09-18_19-22_library-adoption-run-3-gates-repaired-m12-m13-status.md`
   - `2026-09-19_09-31_adminui-errorpage-adoption-and-css-regression-fixes-status.md`
   - `2026-09-19_10-34_run3-execution-status.md`
   - `2026-09-19_11-48_n16-n17-hardening-status.md`
   - `2026-09-19_16-42_v4.11.0-train-prep-session.md`
   - `2026-09-19_18-03_train-session2-recovery-plan-lint-sweep.md`
   - `2026-09-19_23-55_v4.11.0-train-executed-13-tags-pushed.md`
   **584 `~~` markers**; done items cited to the closing sessions (09-19 run-3 N1–N17, the v4.11.0 train, the 2026-09-20 systemadapter tag), open items left unstruck + routed to `TODO_LIST.md`/`ROADMAP.md`.
2. **A4 — 5 older 2026-09-20 reports annotated + archived** (kept the 3 most recent unarchived per the README convention): `00-25` (v4.11.0 shipped / first-honest CI), `08-40` (post-train sweep / first systemadapter tag), `10-57` (round 2), `11-58` (round 3), `13-22` (round 4). **220 markers**; the round 2–4 adminui program (Phases 1–4) was confirmed shipped by round 5 and struck accordingly.
3. **All 3 `NOTE`-convention-consistent blockquotes written**, including a **relative-link fix** for two archived files whose `../planning/…` links would have broken one directory deeper (`2026-09-19_16-42`, `2026-09-19_18-03` → `../../planning/…`).
4. **A5 — `docs/status/README.md` rewritten to reality:** counts corrected (**402 archived**, spanning **2026-05-03 → 2026-09-20**; **4 unarchived reports** + README + 12 HTML artifacts), the duplicate `archive/` split-brain removal documented (`archived/` is the single archive dir), the annotation convention updated to the **literal-strikethrough** form (`~~item~~ done at \`hash\`` / `done (evidence)` / `done (docs-health pass YYYY-MM-DD)` / `**Won't implement — reason.**`) with "unmarked = open", and the two completeness gates documented verbatim (`grep -rLn '~~'`, `check-rows.py`).
5. **A6 partial — this sweep's own completeness verified clean on the presence gate:** all **35** A1–A4 files exist at `docs/status/archived/` and **every one carries ≥1 `~~`** (0 missing, 0 strike-less).
6. **On-sight doc fixes (from the prior continuation, confirmed still in place):** `dashboardui/README.md:290` stale demo note corrected (published since `dashboardui/v4.8.2`); `.agents/skills/cqrs-htmx/SKILL.md` module table now carries `dashboardui`, `loginpage`, `setup` rows.
7. **Archive-tree consolidation confirmed:** `docs/status/archive/` (the 92-file duplicate) is gone; `docs/status/archived/` is the only archive directory.

**Net:** A3 + A4 + A5 complete; A1–A4 total **35 reports annotated**, `docs/status/` reduced to the intended 4-report tail.

---

## b) PARTIALLY DONE

1. **A6 completeness — `check-rows.py` is NOT clean on this sweep's own 35 files.** 5 files have mixed tables (some rows struck, some left open) — **113 flagged lines**:
   - `2026-09-14_13-56_otel-64-to-90-sprint-execution.md` (table @32: 3/4)
   - `2026-09-17_13-11_templ-components-deep-dive-session-status.md` (table @41: 3/6)
   - `2026-09-17_13-23_library-deep-dive-audit-status.md` (tables @35: 2/6, @48: 5/10)
   - `2026-09-17_18-31_stability-and-extraction-analysis.md` (tables @36: 1/7, @48: 1/8)
   - `2026-09-17_21-04_adminui-templ-components-migration-and-lan-demo.md` (tables @50: 6/7, @64: 5/10)
   **Judgment:** the CLEAN rows are **genuinely-open items** deliberately left unstruck (e.g. "Commit hygiene", "Score 68/100 rubric recount", "Verification protocol never exercised", upstream filings, the 8097 listener) — this is the tool's documented "judge and report, don't hide" case, not a missed-done class. Still formally a gate failure.
2. **A6 completeness — the corpus-wide presence gate fails for 340 archived files**, and this is the session's biggest finding (see d1/d2):
   - **294 files have NO annotation marker of any kind** (all 2026-05 → 2026-08 legacy archive).
   - **~42 files carry an `ANNOTATED` blockquote but no inline markers** (blockquote-only convention — the skill's "#1 failure mode" shape, predating the strikethrough convention).
3. **A7 (health report) not printed** — deliberately deferred to after the report you are reading, and now entangled with the corpus finding above.
4. **B1 (CHANGELOG entries) not written**, **B4 (harvest) not run** — both pending.
5. **A8–A10, B2–B3 and the C/D tiers untouched** (planning-plan archive, DOMAIN_LANGUAGE verify, AGENTS.md size, docs-lint gate, FEATURES re-audit, bench, V007, etc.).

---

## c) NOT STARTED

1. **A7** — the AUDIT health report inline (Accuracy + Fitness, per-doc table).
2. **B1** — CHANGELOG `[Unreleased]` entry for this sweep + the two on-sight doc fixes.
3. **B4** — consolidate the routed open items from the 35 annotated reports into `TODO_LIST.md`/`ROADMAP.md` with citations.
4. **Decision + execution on the legacy corpus** (294 unannotated + ~42 blockquote-only): annotate, or formally exempt and document.
5. **A8** — annotate/archive the unarchived `docs/planning/*` plans.
6. **A9** — verify `docs/DOMAIN_LANGUAGE.md` against code.
7. **A10** — AGENTS.md size decision (**still ~120 KB**, rubric calls >100 KB "Broken").
8. **B2 / B3** — docs-lint gate for `/v4`-less imports; FEATURES `FULLY_FUNCTIONAL` re-audit.
9. **C tier** (bench-spike idle re-run, V007 clusters 1+2, ProjectionLayer v5 finalization, SSE remainder, templ-components v1.19 prep, cqrs-lint Go-installable, ADR-001 verdict) and **D tier** (upstream asks, gate hardening, bump playbook, blob-purge prep, example smokes, ROADMAP triage, datastar-demo rebrand, systemadapter Volume test) — all untouched this session.

---

## d) TOTALLY FUCKED UP

1. **I inherited a false premise about the corpus and did not test it before working.** The plan estimated "38 unarchived status reports, 37 without any annotation" and A1–A4 were scoped to the *unarchived tail*. Only at the A6 gate did I discover that **294 files already sitting in `archived/` have never been annotated at all** (2026-05 → 2026-08). I should have run `grep -rLn '~~'` / the marker-density check at the START of the sweep, not at the gate. Net cost: the "finish the standing order" claim is weaker than the counts suggested.
2. **`check-rows.py` was never run per-file as I annotated** — I ran it once, at the end, over the whole corpus. That is exactly the "run it over every annotated file before declaring a pass done" rule, inverted. Result: 5 of my own files carry mixed tables I would have judged deliberately as I went.
3. **The daemon shredded 100% of this session's narrative commits** — every `git mv` and every annotation landed inside `chore: auto-commit` heuristic commits (the documented unwinnable race, 5th+ recurrence). No narrative commit exists for A3/A4/A5; `git log` shows only heuristic messages for ~35 renames + edits. The mitigation (content verified intact) held, but attribution is archaeology again.
4. **Annotate-tool shape gaps cost hand-edits and a retry:** `annotate-prose.py` cannot reach inline-numbered paragraphs, and `annotate-rows.py` cannot handle ID-less or descriptive-first-cell tables — so section-`a` evidence tables were left unstruck and several bullet sections were annotated only via the blockquote. One command (`b)`/`c)` on `2026-09-19_10-34`) failed atomically ("expected 1 match, found 0") because those sections are bullets, not numbered — no damage, one wasted call.
5. **I twice used kind `v` on items that were genuinely-open** in an earlier turn and produced a `done (open …)` marker, then reverted — a self-inflicted false-positive class that the "leave open items untouched" rule exists to prevent.
6. **No CHANGELOG entry was written while the work was fresh.** By the time this report was requested, the changes were already daemon-committed with no `[Unreleased]` narration, so B1 now has to reconstruct them post-hoc.

---

## e) WHAT WE SHOULD IMPROVE

1. **Gate-first, not gate-last.** Any annotate sweep should open with the completeness probes (`grep -rLn '~~'` + `check-rows.py`) to establish the true starting state, then re-run them per-file as each file lands.
2. **Scope the gate to the convention.** `grep -rLn '~~' archived/` presumes the whole archive used the strikethrough convention; this repo's archive is **mixed-vintage** (legacy 2026-05–08 unannotated; 09-09→09-14 blockquote-only; 09-15+ strikethrough). The README now documents the gates, but the corpus reality needs either annotation or a documented exemption.
3. **Decide the legacy corpus once, explicitly.** 294 May–Aug reports: annotating them individually is ~200 hours of low-value work (almost every "next task" is obsolete); a single dated `> LEGACY-EXEMPT` header per file would be ~294 cheap edits; a README-level exemption is zero edits. The skill's "So what?" test favors an exemption, but it must be a **recorded decision**, not a silent omission.
4. **Normalize the blockquote-only files** (~42) that carry an `ANNOTATED` block but zero inline markers — either add inline markers or add a one-line "no numbered items to resolve" note to each block.
5. **A machine-checkable corpus gate.** Extend `check-docs-freshness` (or add a small script) to assert: every `archived/*.md` file has either an `ANNOTATED`/`LEGACY-EXEMPT` block AND a marker, so the mixed-vintage drift can never regrow silently.
6. **Commit at the phase boundary — really.** The daemon won every race; explicit per-phase commits (with the sanctioned `--no-verify` + justification on the known hook reasons) would have preserved the A3/A4/A5 story. The standing order authorizes commits here.
7. **Report gate failures as findings, not silence.** The 340-file presence gap and 5 mixed tables should have been surfaced in-conversation the moment A6 ran, not deferred into this report.

---

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

**Finish this sweep (do first):**
1. Decide the legacy corpus: annotate / `LEGACY-EXEMPT` header / README exemption (Q1).
2. Run `check-rows.py` per annotated file and adjudicate the 5 mixed tables (strike, un-strike, or document as deliberately-open inline).
3. Normalize the ~42 blockquote-only files (inline markers or an explicit "nothing to resolve" note).
4. Write B1: CHANGELOG `[Unreleased]` for A3/A4/A5 + the two on-sight doc fixes.
5. Write A7: the AUDIT health report (Accuracy + Fitness, per-doc table, visible math).
6. Add the corpus-completeness assertion to `check-docs-freshness` (e5).
7. B4: harvest the routed open items from the 35 reports into `TODO_LIST.md`/`ROADMAP.md` with citations.
8. Audit each of the 35 files' blockquotes for router accuracy (no item routed to a doc that doesn't carry it).

**Docs items still open from the plan:**
9. A8 — annotate/archive the unarchived `docs/planning/*` execution plans.
10. A9 — verify `docs/DOMAIN_LANGUAGE.md` against code; fix or file findings.
11. A10 — AGENTS.md size decision + execution (prune / split / declared exception).
12. B2 — docs-lint gate rejecting `/v4`-less `larsartmann/cqrs-htmx/*` imports in living docs.
13. B3 — re-audit FEATURES `FULLY_FUNCTIONAL` rows in batches against code.
14. Reconcile `docs/status/README.md`'s "4 unarchived" target against the actual tail policy (keep-3 vs keep-4 incl. this report).

**C tier (technical backlog, untouched):**
15. C1 — bench-spike idle re-run; re-pin per policy if the 10% gate trips.
16. C3/C4 — V007 clusters 1+2 (`stack.Materialize` → metaengine; `stack.Bundle` → `system.New`).
17. C5 — ProjectionLayer v5-removal finalization (docs + inventory link).
18. C2 — SSE hardening optional remainder (reconnect e2e, fuzz edges, journal bench already landed).
19. C6 — `/sse` endpoint posture decision packet (user call).
20. C7 — templ-components v1.19 train prep (sharp cards → CSS rebuild + screenshot pass).
21. C8 — adminui/dashboardui theme-toggle decision (M089 Option B vs keep `prefers-color-scheme`).
22. C9 — make cqrs-lint Go-installable, then wire `check-cqrs-lint` into CI.
23. C10 — ADR-001 appkit default-flip verdict (fold into `RunHandler`).

**D tier (upstream / tooling / hygiene, untouched):**
24. D1 — file the remaining upstream asks (templ-components Button children slot, `wire.Action` swap/push-url, exhaustruct_v5 promoted-key panic; go-cqrs-lite #35/#36 already filed).
25. D2 — gate-hardening bundle (verify-tag guard, flake check with builds, MD024, LICENSE check, flock; `forEachGoModule` silent-empty-fail in CI; coverage-gate fail-loud).
26. D3 — bump-playbook runbook + `bump-dep.sh` + `go work sync` policy.
27. D4 — blob-purge prep (v4 branch, setup-demo blob, backup refs) — prepare only, await approval.
28. D5 — example smoke tests (basic, datastar-demo, catalog-demo, samber-do SSE).
29. D6 — ROADMAP candidate triage (metrics recorder, SQL checkpoint/DLQ, hydrator B/C/D, setup surface).
30. D7 — `examples/datastar-demo` rebrand.
31. D8 — systemadapter `Volume` hint provenance + regression test.
32. Check in `e2e/tests/admin-screenshots.spec.ts` (the adminui harness is still /tmp-only).
33. Sort the tenants read model deterministically (map-order diff noise in every visual gate).
34. Add an adminui CSP/inline-handler sweep test (dashboardui has one; adminui doesn't).
35. Ratchet dashboardui coverage gate 60 → 80 (actual 91.6%).
36. Add `--allow-unpublished-next` to the release-train pre-commit gate (removes train-phase `--no-verify`).
37. Expand the CI mod-tidy job's module list (6 modules missing).
38. Wire `check-cqrs-lint` + fmt-marker/CSP/a11y tests into CI.
39. Sourcegraph/naming/architecture review passes deferred from the plan's tail (optional).
40. Run `nix run .#check-cqrs-lint` and triage the ~11 pre-existing 4.8.1-era findings (installed now 4.11.2).
41. Refresh AGENTS.md paragraphs still describing the pre-v4.11.0 replace world.
42. Add a `nix run .#render-bench` app (hand-typed bench invocations today).
43. Decide `docs/screenshots/` track-vs-trash.
44. Triage the 2+2 e2e/server + datastar-demo golangci findings.
45. dependabot `open-pull-requests-limit` review.
46. CI ubuntu-latest pin/migration decision (Oct 19 runner change).
47. Add `pull.rebase true` policy + pre-amend `git fetch` guard (daemon-race ergonomics).
48. Ground-truth-first rule: start every browser/assertion script with a DOM dump (captured lesson).
49. Add screenshot-hash-uniqueness assertion into the checked-in harness, not just the shell loop.
50. protect the green state: record "zero replaces / zero train lag / CI green" as the tripwire for the next train plan.

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **The 294-file legacy corpus (2026-05 → 2026-08): annotate it, stamp a `> LEGACY-EXEMPT` header on each file, or declare a README-level exemption?** It is all pre-v4 history whose "next tasks" are overwhelmingly obsolete. Annotating individually is ~200 hours of low-value work; a header is ~294 cheap edits; a README exemption is zero edits but leaves the presence gate knowingly red. My recommendation: **README-level exemption + a `check-docs-freshness` rule that requires the exemption to be documented** — but this is the one call that determines whether "the archive is fully annotated" can ever be claimed.
2. **Are the 5 mixed-vintage files' blockquote-only annotations (and the 5 mixed tables this sweep produced) acceptable as "deliberately open", or do you want every table uniform (all-struck or all-untouched) at the cost of un-striking confirmed-done rows?** The skill's gate wants uniformity; honesty wants open rows left unstruck. I need your precedence call before normalizing.
3. **Release-notes packaging (carried 3 sessions now):** tags + CHANGELOG only, or 14× GitHub Releases bodies generated from the module CHANGELOG sections? It is the last open question blocking the v4.11.0 post-train hygiene list to zero.

---

*Report only — no research beyond this session's own trail. Per the harness contract, no manual commit: the auto-commit daemon has already absorbed the sweep; this file will be picked up the same way. Awaiting instructions.*
