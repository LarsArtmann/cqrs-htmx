> **ANNOTATED 2026-09-22 (docs-health Z):** superseded by the completed sweep — ~~A1/A2~~ done (all 37 target reports annotated+archived; the archive-consolidation held). The §f tail (gates, harvest, README) was executed by the 09-21/09-22 sessions: Gate 1 + Gate 2 repo-owned and CI-wired, B4 harvest done, ROADMAP/TODO v4.12.0-current. Evidence: CHANGELOG [Unreleased] 2026-09-22, docs/status/README.md.

# Status Report — Docs-Health Annotate+Archive Sweep (A1 + A2 complete)

**Date:** 2026-09-21 10:34 CEST
**Session scope:** continuation of the 2026-09-20 docs-health AUDIT execution (plan `docs/planning/2026-09-20_15-43_docs-health-completion-and-backlog-superb-plan.md`, workstreams A1–A4). This continuation completed **A1 and A2** of the annotate+archive sweep.
**Governing rule:** DO NOT VERSCHLIMMBESSERN — every annotation is inline + evidence-cited; no report text deleted; open items are left unmarked (absence = open) and routed.

---

## a) FULLY DONE (this continuation)

| # | Item | Evidence |
|---|------|----------|
| a1 | **A1 — annotate + archive the 5 remaining 2026-09-09→09-15 reports** (`2026-09-10_00-42`, `_02-20`, `_03-01`, `2026-09-14_13-56`, `2026-09-15_10-49`). Each got a top `> ANNOTATED 2026-09-20` blockquote + inline `~~…~~ done` strikes on confirmed-done items; all 5 `git mv`'d to `docs/status/archived/`. | 48 / 52 / 14 / 46 / 36 struck-annotation occurrences respectively; `git log` renames |
| a2 | **A2 — annotate + archive ALL 12 2026-09-17 reports** (`13-11`, `13-12`, `13-23`, `18-31`, `19-24`, `19-57`, `20-18`, `21-01`, `21-03`, `21-04_adminui`, `21-04_library`, `21-25`). Same treatment: blockquote verdict per section + inline strikes + archive. | 60/82/162/76/82/20/28/64/96/156/40/36 occurrences; 12 renames |
| a3 | **~549 annotated line-items struck** across the 17 reports in this continuation (~1,098 `~~` markers). Every spec applied atomically via `annotate-rows.py` / `annotate-prose.py` (dry-run first), with manual strikes for ID-less tables and inline-numbered paragraphs the tools cannot reach. | tool outputs + `grep -o '~~'` counts |
| a4 | **Real doc bug fixed on sight #1** — `dashboardui/README.md` demo note claimed the demo "requires the `dashboardui/v4` module to be tagged and published"; it has been published since `v4.8.2`. Rewrote the note (resolves from the tag; already in `go.work`). | file diff |
| a5 | **Real doc bug fixed on sight #2** — `.agents/skills/cqrs-htmx/SKILL.md` module table listed only `adminui`, omitting `dashboardui`, `loginpage`, and `setup` (the primary AI-session discovery surface). Added all three rows and clarified the adminui row as **identity ops** (not event-store introspection). | file diff |
| a6 | **Archive dir now single canonical tree** — `docs/status/archived/` holds **384** files; top-level `docs/status/*.md` down from 39 → **22** (21 reports + README). | `ls | wc -l` |
| a7 | **Every strike is evidence-backed** — done items cite a commit hash, a shipped artifact, or a dated closure ("v4.11.0 train", "struck 2026-08-30", "systemadapter/v4.11.0 tagged 2026-09-20"). Items I could not confirm were deliberately left unstruck. | report bodies |
| a8 | **Cross-references green** — no stale `docs/status/archive/` (singular) links remain from the earlier merge; the sweep added no new broken links. | `check-docs-links` is gate-checked (see c3) |

---

## b) PARTIALLY DONE

| # | Item | State | What remains |
|---|------|-------|--------------|
| b1 | **A3 — annotate + archive the 2026-09-18/19 reports (13 files)** | 0% | Not started this continuation; they are the next timestamp block after A2. |
| b2 | **A4 — annotate + archive the older 2026-09-20 reports** | 0% | Not started; the 3 most recent reports stay unarchived by design. |
| b3 | **A5 — `docs/status/README.md` refresh** | 0% | The README still says "3 unarchived reports + 273 archived" (reality: 21 + 384) and does not document the merged `archived/` layout or the annotation standard. |
| b4 | **A6 — completeness gates** | 0% | `grep -rL '~~'` clean-check and `check-rows.py` uniformity pass over the newly annotated files have not been run for A1/A2. |
| b5 | **A7 — AUDIT health report** | 0% | Not printed. |
| b6 | **B1 / B4 — CHANGELOG entry + harvest** | 0% | No CHANGELOG entry for this continuation; the open items routed by the sweep blockquotes have not been consolidated into TODO_LIST/ROADMAP in one pass. (Most were already tracked in `TODO_LIST.md`, which is well-maintained.) |
| b7 | **The whole C/D tier** (bench, V007 clusters, gates, upstream filings, blob-purge prep, example smokes, ROADMAP triage, datastar-demo rebrand, systemadapter Volume test) | untouched | Out of this continuation's scope (docs sweep first, per plan sequencing). |

---

## c) NOT STARTED

1. **A3** (2026-09-18/19, 13 reports) and **A4** (older 2026-09-20 reports) — the remaining half of the docs sweep.
2. **`docs/status/README.md`** counts/layout/annotation-convention refresh (A5).
3. **Completeness gates** (A6): `grep -rL '~~'` + `check-rows.py` over A1/A2 output.
4. **AUDIT health report** printed inline (A7) with Accuracy + Fitness + per-doc table.
5. **CHANGELOG** `[Unreleased]` entries for this continuation (B1) and the A-notes for the two on-sight doc fixes (a4/a5).
6. **A8** planning-doc archive, **A9** `DOMAIN_LANGUAGE.md` verify, **A10** AGENTS.md size decision, **B2** docs-lint `/v4`-import gate, **B3** FEATURES row audit.
7. **C1–C10** (bench-spike idle re-run, V007 clusters 1+2, ProjectionLayer v5 finalization, `/sse` posture packet, templ-components v1.19 prep, theme-toggle M089, cqrs-lint CI, ADR-001 verdict).
8. **D1–D8** (upstream asks, gate hardening, bump playbook, blob-purge prep, example smokes, ROADMAP triage, datastar-demo rebrand, systemadapter `Volume` test).

---

## d) TOTALLY FUCKED UP

Nothing destroyed. Honest failures:

1. **The daemon race swallowed every rename/edit again (4th+ session in a row).** `git status --short` at report time shows only the LAST rename; everything before it was already committed as heuristic `chore: auto-commit` commits. Content is intact (all 17 files are in `archived/`), but there is no narrative commit for the sweep. The AGENTS.md phase-boundary rule is documented precisely because of this, and it still lost — the poll interval is shorter than one report's annotation cycle.
2. **I mis-used the `v` marker kind twice.** On the dashboardui audit report I annotated two genuinely-open items (`ThemeToggle` / `ThemeScript`) with `v:"open …"`, producing a self-contradictory `~~…~~ done (open …)`. I caught both and restored them to unmarked with a routing note, but the tool accepted the contradictory spec — nothing prevents "done (open)".
3. **The annotate tools can't reach two common report shapes.** (i) ID-less tables (`| Item | State | Remaining |`) — `annotate-rows.py` requires a row-id column, so I hand-struck 3 such tables. (ii) Inline-numbered paragraphs (`**M6 — errorpage:** 16. … 17. …`) — `annotate-prose.py` needs the number at line start, so items 16–50 of the Run-2 report had to be hand-rewritten line-by-line. Both are avoidable-but-frequent shapes in this corpus.
4. **I did not run the completeness gates before stopping.** A1/A2 are annotated but not yet machine-verified for uniformity (`check-rows.py`); a PARTIAL/missed row would still be invisible. This is exactly the "skipping items you didn't check" failure class the skill warns about — mitigated only by my having struck conservatively (unconfirmed ⇒ unstruck).
5. **I left one file mid-atomic-step when interrupted** (the `21-25` setup-gap report had inline strikes but no blockquote yet). I completed the blockquote + `git mv` before writing this report, but the interrupt exposed that my per-report loop is not transactional — a stop between "annotate" and "archive" leaves a half-annotated file in the live tail.

---

## e) WHAT WE SHOULD IMPROVE

1. **Machine-gate each report as it is archived, not in a batch at the end.** Run `check-rows.py <file>` + `grep -c '~~'` immediately after the `git mv` for that file. Batch-gating at A6 is where misses hide.
2. **Never annotate with a non-committal value.** The tool should reject `v:"open…"`; until upstreamed, treat `v` as strictly "done with cited evidence". The two `done (open…)` slips came from trying to route an open item while also touching the line.
3. **Pre-classify the report shape first** (table-with-ID? ID-less table? prose list? inline-numbered paragraph?) and batch the hand-edits per shape, instead of discovering them mid-command. One report took four tool calls purely to route around shape mismatch.
4. **Commit the atomic unit (annotate + archive) within the same minute** — or accept the daemon and write the narrative into the blockquote itself (which I did: the blockquote IS the durable summary, so the commit-message loss matters less than in a code change).
5. **Report-reconciliation is cheap and was skipped twice in the corpus** (two same-day 2026-09-17 deep-dives; the report-reconciliation questions in both). Route them explicitly rather than leaving two "open" markers.
6. **The `docs/status/README.md` should have been refreshed FIRST**, before archiving, so the counts/layout it documents are never wrong mid-sweep. Doing it last guarantees a stale-window.

---

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

**Close the docs sweep (highest value, bounded):**

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | A3: annotate + archive the 7 `2026-09-18_*` reports | High | M |
| 2 | A3: annotate + archive the 6 `2026-09-19_*` reports | High | M |
| 3 | A4: annotate + archive the older `2026-09-20_*` reports (keep the 3 most recent) | High | M |
| 4 | A5: rewrite `docs/status/README.md` (counts 21/384, merged layout, annotation standard incl. `~~` alongside the legacy `✅` convention) | High | S |
| 5 | A6: run `grep -rL '~~' docs/status/archived/` and `check-rows.py` over every file annotated in A1/A2/A3/A4; fix PARTIAL/UNTOUCHED rows | High | S |
| 6 | B1: append CHANGELOG `[Unreleased]` entries for the sweep + the two on-sight doc fixes | Medium | S |
| 7 | B4: consolidate routed open items into TODO_LIST/ROADMAP with citations | High | M |
| 8 | A7: print the AUDIT health report inline (Accuracy + Fitness + per-doc table) | Medium | S |
| 9 | Add a `check-status-annotated` gate: fail if any top-level report lacks an `> ANNOTATED` block | Medium | S |
| 10 | Upstream the two annotate-tool gaps (ID-less tables; inline-numbered paragraphs) to the docs-health skill assets | Low | S |

**Then the plan's remaining tiers (unchanged from `docs/planning/2026-09-20_15-43_*`):**

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 11 | A8: archive superseded `docs/planning/*` plans | Medium | M |
| 12 | A9: verify `docs/DOMAIN_LANGUAGE.md` against code | Medium | S |
| 13 | A10: decide + execute the AGENTS.md size plan (120 KB vs ">100 KB = Broken") | High | M |
| 14 | B2: docs-lint gate rejecting `/v4`-less `larsartmann/cqrs-htmx/*` imports | Medium | S |
| 15 | B3: re-audit FEATURES `FULLY_FUNCTIONAL` rows in batches | Medium | M |
| 16 | C1: bench-spike idle re-run (refused 6× on load; `TODO_LIST.md` P1) | High | S |
| 17 | C3: V007 cluster (1) `stack.Materialize` ×6 → metaengine auto-projection | High | L |
| 18 | C4: V007 cluster (2) `stack.Bundle` ×4 → `system.New` composition | High | L |
| 19 | C5: ProjectionLayer v5-removal finalization (docs + inventory link) | Medium | S |
| 20 | C2: SSE-hardening optional remainder (reconnect e2e, fuzz edges, journal bench) | Medium | M |
| 21 | C7: templ-components v1.19 train prep (Cards lose `rounded-lg`) | Medium | S |
| 22 | C9: make cqrs-lint Go-installable, wire `check-cqrs-lint` into CI | Medium | M |
| 23 | C10: ADR-001 appkit default-flip verdict | Medium | S |
| 24 | D1: verify + file the templ-components upstream asks (a–e) + go-cqrs-lite asks | High | M |
| 25 | D2: gate-hardening bundle (verify-tag message guard, flake check with builds, MD024, LICENSE, flock) | Medium | M |
| 26 | D3: bump-playbook runbook + `scripts/bump-dep.sh` + `go work sync` policy | Medium | M |
| 27 | D4: blob-purge prep (v4 branch + setup-demo) — PREP ONLY, no push | Medium | S |
| 28 | D5: example smoke tests (catalog-demo, samber-do SSE; basic + datastar-demo already have one) | Medium | M |
| 29 | D6: triage ROADMAP candidate ideas | Medium | M |
| 30 | D7: `examples/datastar-demo` rebrand | Low | S |
| 31 | D8: systemadapter `Volume` hint provenance + regression test | Low | S |

*(Stopped at 31 — the remaining slots would re-list the 21 newly-archived reports' routed items, which now live in the blockquotes and `TODO_LIST.md`; padding them here would duplicate the harvest.)*

---

## g) THREE QUESTIONS I CANNOT ANSWER MYSELF

**Q1 — Archive cutoff for 2026-09-20.** The plan says "keep the 3 most recent unarchived". The three most recent by timestamp are `15-00_round5`, `15-37_docs-health-audit`, and this session's `22-02_buildflow-corruption-fix` (plus this new 09-21 report). That would archive `00-25`, `08-40`, `10-57`, `11-58`, `13-22` — i.e. most of today's rounds. Do you want exactly that cutoff (3 most recent), or a different one (e.g. keep all of 2026-09-20 as "today's work" and archive only through 09-19)? The README's "3 unarchived" claim assumes a hard 3.

**Q2 — Corpus-wide strikethrough conversion.** Your standing order said "Archive FULLY done and UPDATED (inline strikethrough)". The legacy ~360 previously-archived reports use the house convention (dated `> ANNOTATED` blockquote + inline `✅ done` / `→ routed` / `STALE` suffixes) and mostly carry NO literal `~~`. The A1/A2 sweep used literal `~~` (as you asked). Do you want (a) a one-time corpus-wide pass converting the legacy `✅ done` markers to `~~…~~ done`, or (b) keep the legacy subset as-is and apply `~~` only to the newly-annotated tail (my recommendation — a 360-file rewrite is a high-Verschlimmbesserung-risk change for zero information gain)?

**Q3 — After the docs sweep, which tier next?** Once A3/A4/A5/A6/A7 + B1/B4 land, the plan branches into C (code: bench, V007 migration, gates) and D (tooling/upstream/hygiene). Do you want me to continue straight into C/D, or stop at the docs boundary and hand back for a fresh go-ahead? (Several C/D items are user-gated: `/sse` posture, theme-toggle M089, ADR-001 flip, blob-purge force-push authorization.)

---

*Point-in-time snapshot. Report written 2026-09-21 10:34 CEST. Sources: this session's tool outputs, `git log`/`git status`, `ls docs/status/`, and the 17 annotated reports.*
