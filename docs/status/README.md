# docs/status/ — Historical Session Snapshots

**These files are point-in-time snapshots, not living documents.**

Each report in this tree captures what someone knew at the end of a work session. Reports are **append-only history** — never rewritten in place. A file from 2026-06-16 reflects what was true on 2026-06-16, not what is true today.

## How to read these

- **A reader who opens one of these** should treat it as a historical artifact, like a git commit message or a lab notebook page.
- **For current state**, see the living docs at the repository root:
  - `FEATURES.md` — what works today, by status
  - `TODO_LIST.md` — actionable work, current
  - `AGENTS.md` — non-obvious project context for AI sessions
  - `README.md` — what this project is, today

## Layout (reorganized 2026-09-09; archive consolidated 2026-09-20)

| Path                               | Contents                                                                     |
| ---------------------------------- | ---------------------------------------------------------------------------- |
| `docs/status/*.md`                 | The most recent session reports only (unarchived tail; currently 0 after the round-17 pass) |
| `docs/status/archived/`            | 467 archived session reports (2026-05-03 → 2026-10-06)                       |
| `docs/status/*.html`               | 12 generated HTML report artifacts (see "HTML corpus" below)                 |
| `docs/planning/`                   | Active plans; superseded ones move to `docs/planning/archived/`              |
| `docs/reviews/archived/`           | Archived review documents                                                    |
| `docs/modularization/archived/`    | Archived module-assessment documents                                         |
| `docs/plans/archived/`             | Archived implementation plans                                                |
| `docs/feedback/processed/`         | Processed feedback documents                                                 |
| `docs/architecture-understanding/` | Generated self-contained HTML architecture reports (see "HTML corpus" below) |

`docs/status/archived/` is the ONLY archive directory — the older duplicate `docs/status/archive/` (a 92-file split-brain) was merged into it and removed.

## Annotation convention

### Current convention (reports dated 2026-09-09 onward)

When a historical report is verified against the current tree, it receives:

1. A dated **`> ANNOTATED YYYY-MM-DD`** blockquote at the top of the file, summarizing the verification verdict per section (DONE-SHIPPED / OPEN-TRACKED / OBSOLETE / HARVEST-CATCH, with evidence), and routing the remaining open items to `TODO_LIST.md`/`ROADMAP.md`.
2. **Inline strikethrough on every resolved item**: `~~original line~~ done at \`hash\``,` ~~original line~~ done (evidence)`,` ~~original line~~ done (docs-health pass YYYY-MM-DD)`, or` ~~original line~~ **Won't implement — reason.**`. **Unmarked items are the "open" signal** — never strike an item you did not verify.

Annotations are **additive only** — the original report text is never deleted. Cross-references in living docs point at the archived paths.

### Older dialects (reports dated before 2026-09-09) — LEGACY-EXEMPT

The inline-strikethrough convention was established on **2026-09-09**. Everything archived before that date predates it and is **exempt from the presence gate**:

| Dialect                        | Dates                   | Files | Marker                                         |
| ------------------------------ | ----------------------- | ----- | ---------------------------------------------- |
| Inline strikethrough (current) | 2026-09-09 onward       | 37    | `~~…~~ done (evidence)` + dated blockquote     |
| Prose blockquote               | 2026-06-17 → 2026-09-07 | 72    | `> ANNOTATED …` verdict blockquote, no strikes |
| Unannotated                    | 2026-05-03 → 2026-08-09 | 268   | none                                           |

The prose-blockquote and unannotated eras are **historically complete as written** — their still-open items were harvested by later docs-health sweeps (2026-09-09, 2026-09-20, 2026-09-21). Retro-annotating 340 legacy files would be a [Verschlimmbesserung](https://en.wikipedia.org/wiki/Verschlimmbessern) with near-zero information gain, so the gate is deliberately scoped to the current convention era. New reports always use the current convention.

### Mixed tables are first-class (not a defect)

A table may mix **struck rows (done)** with **unstruck rows (open)**. That is exactly what "absence of a marker IS the open signal" means at row granularity, and it is how a report records partial completion honestly. The repo gate `scripts/checks/check-status-rows.py` counts such tables and reports them as `deliberately-mixed` — informational, **never** a reason to un-strike a verified-done row. The only row-level failure that matters is a `PARTIAL` row — a single row whose cells disagree (some struck, some not).

### Completeness gates

Both gates are repo-owned, CI-wired (the `checks` job), and run inside `nix run .#check-modules`:

```bash
# Gate 1: presence + dated blockquote for every gated-era report.
bash scripts/checks/check-status-annotations.sh

# Gate 2: no PARTIAL rows in any struck table; deliberately-mixed tables
#   are counted and reported (first-class by convention), never failed.
python3 scripts/checks/check-status-rows.py

# Fixture self-tests for both gates (also CI-wired).
bash scripts/selftests/test-check-status-annotations.sh
bash scripts/selftests/test-check-status-rows.sh

# Fixer for the PARTIAL-row class Gate 2 catches: completes a row by striking
#   every remaining cell (the whole-row-strike policy above). Idempotent; the
#   checker and fixer are proven to agree by the self-test. `--dry-run` first.
python3 scripts/tools/normalize-status-rows.py --dry-run
bash scripts/selftests/test-normalize-status-rows.sh
```

A third, **advisory** gate keeps the live tail honest: `scripts/checks/check-docs-tail-budget.sh`
warns (exit 0) when `docs/status/*.md` (excluding this README) exceeds 3 reports,
so a growing tail surfaces at sweep boundary instead of being re-discovered.
It is deliberately **not** a blocking `check-modules` stage — a mid-session tail
of two or three reports is legitimate. `--strict` exits 1 for a manual/train-time
check. Fixture self-test: `bash scripts/selftests/test-check-docs-tail-budget.sh`.

Ambient `python3` is the contract for Gate 2 (CI runners ship it); the flake apps
(`nix run .#check-status-rows` / `.#test-status-rows` / `.#normalize-status-rows`
/ `.#test-normalize-status-rows` / `.#check-docs-tail-budget` /
`.#test-check-docs-tail-budget`) additionally pin `pkgs.python3` for hermetic use.
The docs-health skill's authoring-time `check-rows.py` asset remains the
annotator's aid — it reports mixed tables as `INCOMPLETE` for a human to
adjudicate; scope-identity with the repo gate was verified per-file and per-line
(same 19 mixed tables) on 2026-09-22.

**Adjudication log:** the 8 files above were individually inspected on 2026-09-21; every mixed table pairs verified-done rows with genuinely-open ones, and the two `PARTIAL` rows in `2026-08-05_11-46` (hand-annotated before the tooling existed) were normalized to full-row strikethrough.

## HTML corpus policy

The `*.html` files in `docs/status/` and `docs/architecture-understanding/` (~60 files) are **generated, self-contained report artifacts** (styled snapshots produced for a specific session or review) and are **exempt from annotation sweeps and freshness gates**. They are deliverables, not claims to maintain: their content is superseded by the living docs, and no markdown-source counterpart is expected. Do not edit them in place; archive or regenerate instead. **Standing decision (docs-health round 14, 2026-10-01): the 12-file `docs/status/*.html` corpus is KEEP-IN-PLACE** — archiving would be churn without information gain (they are already exempt from every gate and indexed by this README), and the TODO_LIST item that asked for the deliberate decision is closed with this line as the record.

## File counts

- `docs/status/`: **0 unarchived reports** (the 2026-10-06 round-18 docs-health pass annotated + archived the entire 17-report tail in one sweep) + README + 12 HTML artifacts.
- `docs/status/archived/`: **467 archived reports** (2026-05-03 → 2026-10-06).
- The archive tail has been swept repeatedly: 2026-09-09 (full backlog), 2026-09-20 (the 38-report tail), 2026-09-21 (the 34-report tail), 2026-09-22 (the 8-report superb-plan tail + the plan itself), 2026-10-01 round-13 (the 22-report 09-22→09-30 tail — every report annotated with a dated blockquote + evidence-backed inline strikes, open items routed to TODO_LIST/ROADMAP, then archived), 2026-10-01 round-14 (the 10-01 pair — the 02:59 BuildFlow-recovery and 06:36 round-13 reports got W0–W3-receipt top-up blockquotes + inline strikes for everything the train/gate/erraudit sessions then resolved, then archived; the same pass triaged the six 2026-08-30 planning files — upstream-asks draft archived, gated-work index + cqrs-lint distribution draft annotated, three live owner-gated decisions confirmed leave-in-place — and confirmed the HTML corpus keep-in-place decision), and **2026-10-04 round-17 (the 13-report 10-01→10-04 tail — every report annotated with a dated blockquote + evidence strikes, open items verified against the tree (CI run 37178906141: test green on published pins; master red on exactly one job — the loginpage 7/5 dep budget) and routed to TODO_LIST/ROADMAP, then archived; the two `docs/drafts/2026-10-04-signing-cloneevent-encoding*` files got RESOLVED-SHIPPED blockquotes and stay in place)**, and **2026-10-06 round-18 (the 17-report 10-04→10-06 tail — every report annotated with a dated blockquote + evidence strikes verified against the tree (the identity-auth-hardening wave tags, the 11-module sweep train, push `e2f6be27`, CI green 37392674249, the M01–M15 hardening execution across three sibling sessions) and routed to TODO_LIST/ROADMAP, then archived; the six 2026-08-30/09-23 planning files re-confirmed: 3 annotated, 3 owner-gated LEAVE-IN-PLACE; the HTML corpus keep-in-place decision re-honored)**. Of the 467 archived reports, **102 carry the current inline-strikethrough convention** (gated by `check-status-annotations.sh`), 72 carry the older prose blockquote, and the rest predate annotation entirely (legacy-exempt). Gate 1 enforces the current-convention era; Gate 2 keeps the row shapes honest.

### Round-17 archive manifest (2026-10-04)

All 13 files: classification **ARCHIVE** (dated blockquote + evidence strikes + open items routed), `git mv` → `docs/status/archived/`.

| File | Deciding evidence |
| --- | --- |
| `2026-10-01_10-30_pareto-w0-w3-train-gates-erraudit-status.md` | T06 erraudit closed same evening; T13–T25 + W10 shipped 10-01/02; T09 answered (3 filed / 2 retired); go.work → OQ27; rawIDToken executed (D3) |
| `2026-10-01_12-30_docs-health-round14-full-pass-status.md` | Its own §b/§c debt closed (composite 15:50, D-index exists, LEAVE verdict recorded); §g1 moot; archive bar applied by rounds 15–17 |
| `2026-10-01_15-55_pareto-t06-close-battery-tail-status.md` | T08 local half + T09 done same evening; open residue (bench, next train, D5) routed |
| `2026-10-01_17-01_t06-tail-session-self-critique-status.md` | §b breadth closed overnight (test-all 28/28); 25 strikes; owners routed |
| `2026-10-01_17-36_tooling-round-2-shared-tree-session-status.md` | Heavy gates landed; the report's own ARCHIVE-pass ask executed by this pass |
| `2026-10-02_07-09_round15-consolidated-status.md` | Open items all routed (bench/train/D-list/E005); T08-half + stale-suppressions + archive-pass strikes |
| `2026-10-02_08-55_round15-tail-self-critique-status.md` | f16 done by this pass; OQ21 moot; §f50 superseded; the rest routed as written |
| `2026-10-03_03-30_v007-verification-cqrs-lint-determinism-session.md` | V007 disposition settled (ADR-0051 + inventory §5c); archive ask executed; remainder owner-gated and routed |
| `2026-10-03_04-01_pareto-plan-v5-autoupgrade-push-blocked.md` | Push blocker resolved next morning (alignment executed, push landed); plan M1–M28 live in TODO |
| `2026-10-04_05-49_loginpage-critical-review-and-hardening-session.md` | Both filed follow-ups (NoOAuth2, CSP-nonce) shipped; README/AGENTS/CHANGELOG truth-passes verified; review HTML annotated |
| `2026-10-04_05-51_push-unblock-and-signing-regression.md` | Arc closed: upstream fix landed + re-pinned; patch-tag thread MOOT (only integration_test requires signing); dep-budget = last red |
| `2026-10-04_06-52_signing-v433-fix-landed-and-verified.md` | CI sharpened to one red job; lifecycle debts closed by this pass; §g questions resolved/MOOT |
| `2026-10-04_06-54_pareto-plan-episode4-tail-execution.md` | Test failures sharpened (CI green on pins; sibling-tree state; dep-budget is the train blocker); T5–T9 verified still open + routed |

### Round-18 archive manifest (2026-10-06)

All 17 files: classification **ARCHIVE** (dated `> ANNOTATED 2026-10-06` blockquote + evidence strikes + open items routed), `git mv` → `docs/status/archived/`. Verification anchors: the identity-auth-hardening wave (usermgmt v4.14.0 + setup v4.14.0 + dashboardui v4.13.0; ADR-0055), the 11-module sweep train (root v4.13.1 + usermgmt v4.14.1 + identity-model v4.12.1 + 8 patch tags; templ-components v1.20.1 heal), the alignment push `e2f6be27` (CI all-green 37392674249), and the M01–M15 cross-repo hardening execution (22:34 + 23:20 + 21:35 reports, GCL receipts).

| File | Deciding evidence |
| --- | --- |
| `2026-10-04_12-09_crm-identity-feedback-verification.md` | Every §f seed executed by the M1–M25 hardening plan (ADR-0055, gate + self-wrap + DisplayName + wrapper-check gate); CRM acceptance ran; 41 strikes; browser-level ceremony + CSRF note routed |
| `2026-10-04_13-14_dashboardui-integration-modes_status.md` | Autodetect + `Config.Layout` shipped as dashboardui/v4.13.0, consumed by setup v4.14.x; B1/B2 + c3/c6/c7 + §f1–5/15/49 struck; embed demo/guide + panels layer routed |
| `2026-10-04_13-14_scripts-reorg-status.md` | Reorg held through 5 pushes + green CI; external reds shipped (LayoutFunc); §b1–b4 + §f1–4/9/22–24/30 struck; root-guard/aggregator/manifest-consolidation routed |
| `2026-10-04_15-50_identity-auth-hardening-execution-self-review.md` | Wave tagged + pushed; CI verified; the 3 "pre-existing" failures root-caused (gotcha 25) + fixed 10-05; goldens regenerated; f4/6/9–11/13/18/20/21 + g1 struck |
| `2026-10-05_12-48_maximization-pass-self-review.md` | P1 fix + gotcha 25 canon; push done (`e2f6be27`); StreamMarker train landed; statusToBadgeMap closed MOOT+ADOPTED; b3–b5 + f1–4/6/11/12/17/19/33 + g1 struck |
| `2026-10-05_14-59_branching-flow-full-analysis.md` | Already annotated 2026-10-05 by its own session (ratchet `94a92e6f`, safe fixes, harvest `471c0933`); carried into the archive unchanged |
| `2026-10-05_15-42_push-unblock-broken-templ-components-release.md` | UNBLOCKED: v1.20.1 healed, #27 filed, family swept, bundles rebuilt, push landed; b1–b4 + c1–c4/c7 + f1–4/9–11/13–16/21 + g1–g3 struck |
| `2026-10-05_20-31_ratchet-shipped-safe-fixes-upstream-poison.md` | All loose ends closed by the 21:19 session (exhaustruct ignore-pattern, 663→659, R7, harvest, #27, CHANGELOG) + Q1 end-to-end (v1.20.1 → push); b/c + f1–12/18 + g2/g3 struck |
| `2026-10-05_20-32_round17-plan-execution-and-divergence-recovery.md` | Blockers dissolved: train ran (11 patch tags), push `e2f6be27`, CI all-green; b2–b5 + f1–5/8/41/47–49 struck; A7/A9/A11–A14 + owner rows routed |
| `2026-10-05_21-19_ratchet-tail-executed-plan-annotations-upstream-filed.md` | Its §b blocker resolved end-to-end (v1.20.1 + sweep + `7c74307a` + green check-modules + push); final-verification + f1–6/14 + g2/g3 struck |
| `2026-10-06_03-04_alignment-push-train-and-ci-green-repair.md` | End state held (CI sustained green); golden TODO struck this pass; stale agents-notes claim fixed `7c74307a`; b7/c7/f1/f2/f18 struck; battery/tooling/collection items routed |
| `2026-10-06_19-31_alignment-stall-env-triage-phase1-prep.md` | Alignment completed + M01–M15 executed same evening (three sibling sessions); b1/b3 + c + f1–5/9–29/46 struck; flightrecorder push blocker + M16–M27 routed |
| `2026-10-06_19-31_SUPERB-hardening-m11-m12-done-m13-m15-pending.md` | Fully superseded by 21:35 (M13–M15 done, questions answered by execution); ANNOTATED + §b2/§c struck; 5 PARTIAL rows normalized to whole-row strikes |
| `2026-10-06_21-35_SUPERB-hardening-m11-m15-complete.md` | M01–M10 closed same night by the sibling lane (22:34 + 23:20); §2 + §5 items 1–10 + §6-Q1 struck; flightrecorder + M16–M27 + forensics routed |
| `2026-10-06_22-34_SUPERB-phase1-complete-phase2-m06-midedit.md` | Phase 2 closed by 23:20 (M06 `7584b7a84` completed; M07/M08/M10 landed + receipted); b1 + c-row + f1–6 + g2/g3 struck; push blocker + full-gate battery routed |
| `2026-10-06_23-20_SUPERB-phase2-complete-m05-m10.md` | Lane M05–M10 verified + receipted; Q2/Q3 confirmed; Q1 (flightrecorder/push) + M16–M27 + GCL TODO items stay open; 1 PARTIAL row normalized |
| `2026-10-06_23-23_SUPERB-m05-m10-brutal-status-review.md` | The honest gaps (F22, GCL gate compliance, metaengine `-race`, doc comment) harvested into TODO_LIST; f30/f31 struck; push authorization + history-readability owner calls routed |

Also archived in the same sweep (planning corpus): `docs/planning/archived/2026-10-01_06-47_pareto-round13-superb-execution-plan.md` and `..._15-04_pareto-round14-gates-green-first.md` — both carry dated OUTCOME blockquotes, every executable item shipped 2026-10-01/02, residue (bench, T08 fleet swap, D-list) routed in TODO_LIST; the round-16 and episode-4 plans stay ACTIVE in `docs/planning/`.

## Why not "update them all" ad hoc?

Bulk-editing historical files without verification would be a [Verschlimmbesserung](https://en.wikipedia.org/wiki/Verschlimmbessern) — a well-intentioned change that makes things worse. The 2026-09-09 sweep did the verified version of this work once (every report read and checked against CHANGELOG/git/code before annotation). New reports: write them, and route anything still-open into the living docs so the next sweep stays small.
