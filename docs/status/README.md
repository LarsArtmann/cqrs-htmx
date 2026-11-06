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

| Path                               | Contents                                                                                    |
| ---------------------------------- | ------------------------------------------------------------------------------------------- |
| `docs/status/*.md`                 | The unarchived tail — newest session reports only. **Counts are derived, never written here; run the command in "File counts" below.** |
| `docs/status/archived/`            | Closed history — every archived session report. Counts derived (see below); bulk-pass manifest in `archived/README.md` |
| `docs/status/*.html`               | Historical: generated HTML artifacts were evicted 2026-10-24 (see "HTML corpus policy")                                      |
| `docs/planning/`                   | Active plans; superseded ones move to `docs/planning/archived/`                             |
| `docs/reviews/archived/`           | Archived review documents                                                                   |
| `docs/modularization/archived/`    | Archived module-assessment documents                                                       |
| `docs/plans/archived/`             | Archived implementation plans                                                              |
| `docs/feedback/processed/`         | Processed feedback documents                                                               |
| `docs/architecture-understanding/` | Generated architecture snapshots (d2/svg/HTML); uncited ones evicted 2026-10-24, the two load-bearing composability reviews remain |

> **Archive eviction (prepared 2026-10-24, owner-gated).** 578 of the 591 files under the archive
> directories are unreferenced closed history. The measured keep-set, the dry-run verdict, and the
> one-command removal/restore flow live in [`../ARCHIVE_EVICTED.md`](../ARCHIVE_EVICTED.md).

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

# Equivalence meta-test (T05, 2026-10-08): the checker and fixer import the
#   shared scripts/lib/status_table.py; this pushes 7 adversarial fixtures
#   through BOTH and proves the fixer normalizes exactly the rows the gate
#   flags — the split-brain class is structurally dead, not just currently
#   aligned. Also a check-modules + CI stage.
bash scripts/selftests/test-status-table-equivalence.sh
```

A third gate keeps the live tail honest in two modes. The absolute target is 3 reports
(`docs/status/*.md`, excluding this README); `--strict` exits 1 against it for a manual check.
The **blocking** mode is `--ratchet`: it reads `docs/status/TAIL_BASELINE` and fails when the
tail grows past it, so the corpus can only ratchet down — a new report must replace an archived
one, and archiving must lower the baseline in the same commit. That is what turns "the tail grew
for weeks while the gate printed a warning" into a red build. Both modes run in `check-modules`
and CI. Fixture self-test (10 cases incl. ratchet at/over/below/missing/malformed baseline):
`bash scripts/selftests/test-check-docs-tail-budget.sh`.

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

**Standing decision (superseded 2026-10-24):** docs-health round 14 (2026-10-01) had classified
the generated `*.html` artifacts as KEEP-IN-PLACE. The 2026-10-24 corpus-eviction master plan
(S21/S29) re-adjudicated: they are **generated, self-contained report artifacts** whose content
is superseded by the living docs — 14 status HTML reports + every uncited research/architecture
HTML snapshot were **evicted** by the same measured rule as the archive eviction (living-doc
citation = keep; restore via [`../ARCHIVE_EVICTED.md`](../ARCHIVE_EVICTED.md)). The 12 cited
research reports and the 2 composability reviews survive as load-bearing design rationale.
Future generated HTML artifacts follow the same rule: cite one from a living doc if it is
load-bearing, otherwise it is evictable closed history.

## File counts

Counts are **derived, never hand-maintained** (killed by docs-health round 22, 2026-10-24 — the
previous table claimed a number 11 short of reality and stayed wrong for a week).

```bash
# live markdown reports — exactly what the tail-budget gate counts
ls docs/status/*.md | grep -v 'README.md' | wc -l

# generated HTML artifacts (exempt from annotation and freshness gates)
ls docs/status/*.html | wc -l

# closed history, all passes
find docs/status/archived -name '*.md' | wc -l

# the live tail against its budget (exit 1 under --strict)
bash scripts/checks/check-docs-tail-budget.sh
```

The complete per-pass manifest lives in [`archived/README.md`](archived/README.md); the round
manifests below are the in-place records of the largest passes. Gate 1
(`check-status-annotations.sh`) enforces the current-convention era (reports dated 2026-09-09
onward); Gate 2 (`check-status-rows.py`) keeps table-row shapes honest. Anything older than the
epoch is legacy-exempt by design.

### Round-17 archive manifest (2026-10-04)

All 13 files: classification **ARCHIVE** (dated blockquote + evidence strikes + open items routed), `git mv` → `docs/status/archived/`.

| File                                                                  | Deciding evidence                                                                                                                              |
| --------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `2026-10-01_10-30_pareto-w0-w3-train-gates-erraudit-status.md`        | T06 erraudit closed same evening; T13–T25 + W10 shipped 10-01/02; T09 answered (3 filed / 2 retired); go.work → OQ27; rawIDToken executed (D3) |
| `2026-10-01_12-30_docs-health-round14-full-pass-status.md`            | Its own §b/§c debt closed (composite 15:50, D-index exists, LEAVE verdict recorded); §g1 moot; archive bar applied by rounds 15–17             |
| `2026-10-01_15-55_pareto-t06-close-battery-tail-status.md`            | T08 local half + T09 done same evening; open residue (bench, next train, D5) routed                                                            |
| `2026-10-01_17-01_t06-tail-session-self-critique-status.md`           | §b breadth closed overnight (test-all 28/28); 25 strikes; owners routed                                                                        |
| `2026-10-01_17-36_tooling-round-2-shared-tree-session-status.md`      | Heavy gates landed; the report's own ARCHIVE-pass ask executed by this pass                                                                    |
| `2026-10-02_07-09_round15-consolidated-status.md`                     | Open items all routed (bench/train/D-list/E005); T08-half + stale-suppressions + archive-pass strikes                                          |
| `2026-10-02_08-55_round15-tail-self-critique-status.md`               | f16 done by this pass; OQ21 moot; §f50 superseded; the rest routed as written                                                                  |
| `2026-10-03_03-30_v007-verification-cqrs-lint-determinism-session.md` | V007 disposition settled (ADR-0051 + inventory §5c); archive ask executed; remainder owner-gated and routed                                    |
| `2026-10-03_04-01_pareto-plan-v5-autoupgrade-push-blocked.md`         | Push blocker resolved next morning (alignment executed, push landed); plan M1–M28 live in TODO                                                 |
| `2026-10-04_05-49_loginpage-critical-review-and-hardening-session.md` | Both filed follow-ups (NoOAuth2, CSP-nonce) shipped; README/AGENTS/CHANGELOG truth-passes verified; review HTML annotated                      |
| `2026-10-04_05-51_push-unblock-and-signing-regression.md`             | Arc closed: upstream fix landed + re-pinned; patch-tag thread MOOT (only integration_test requires signing); dep-budget = last red             |
| `2026-10-04_06-52_signing-v433-fix-landed-and-verified.md`            | CI sharpened to one red job; lifecycle debts closed by this pass; §g questions resolved/MOOT                                                   |
| `2026-10-04_06-54_pareto-plan-episode4-tail-execution.md`             | Test failures sharpened (CI green on pins; sibling-tree state; dep-budget is the train blocker); T5–T9 verified still open + routed            |

### Round-18 archive manifest (2026-10-06)

All 17 files: classification **ARCHIVE** (dated `> ANNOTATED 2026-10-06` blockquote + evidence strikes + open items routed), `git mv` → `docs/status/archived/`. Verification anchors: the identity-auth-hardening wave (usermgmt v4.14.0 + setup v4.14.0 + dashboardui v4.13.0; ADR-0055), the 11-module sweep train (root v4.13.1 + usermgmt v4.14.1 + identity-model v4.12.1 + 8 patch tags; templ-components v1.20.1 heal), the alignment push `e2f6be27` (CI all-green 37392674249), and the M01–M15 cross-repo hardening execution (22:34 + 23:20 + 21:35 reports, GCL receipts).

| File                                                                        | Deciding evidence                                                                                                                                                                              |
| --------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `2026-10-04_12-09_crm-identity-feedback-verification.md`                    | Every §f seed executed by the M1–M25 hardening plan (ADR-0055, gate + self-wrap + DisplayName + wrapper-check gate); CRM acceptance ran; 41 strikes; browser-level ceremony + CSRF note routed |
| `2026-10-04_13-14_dashboardui-integration-modes_status.md`                  | Autodetect + `Config.Layout` shipped as dashboardui/v4.13.0, consumed by setup v4.14.x; B1/B2 + c3/c6/c7 + §f1–5/15/49 struck; embed demo/guide + panels layer routed                          |
| `2026-10-04_13-14_scripts-reorg-status.md`                                  | Reorg held through 5 pushes + green CI; external reds shipped (LayoutFunc); §b1–b4 + §f1–4/9/22–24/30 struck; root-guard/aggregator/manifest-consolidation routed                              |
| `2026-10-04_15-50_identity-auth-hardening-execution-self-review.md`         | Wave tagged + pushed; CI verified; the 3 "pre-existing" failures root-caused (gotcha 25) + fixed 10-05; goldens regenerated; f4/6/9–11/13/18/20/21 + g1 struck                                 |
| `2026-10-05_12-48_maximization-pass-self-review.md`                         | P1 fix + gotcha 25 canon; push done (`e2f6be27`); StreamMarker train landed; statusToBadgeMap closed MOOT+ADOPTED; b3–b5 + f1–4/6/11/12/17/19/33 + g1 struck                                   |
| `2026-10-05_14-59_branching-flow-full-analysis.md`                          | Already annotated 2026-10-05 by its own session (ratchet `94a92e6f`, safe fixes, harvest `471c0933`); carried into the archive unchanged                                                       |
| `2026-10-05_15-42_push-unblock-broken-templ-components-release.md`          | UNBLOCKED: v1.20.1 healed, #27 filed, family swept, bundles rebuilt, push landed; b1–b4 + c1–c4/c7 + f1–4/9–11/13–16/21 + g1–g3 struck                                                         |
| `2026-10-05_20-31_ratchet-shipped-safe-fixes-upstream-poison.md`            | All loose ends closed by the 21:19 session (exhaustruct ignore-pattern, 663→659, R7, harvest, #27, CHANGELOG) + Q1 end-to-end (v1.20.1 → push); b/c + f1–12/18 + g2/g3 struck                  |
| `2026-10-05_20-32_round17-plan-execution-and-divergence-recovery.md`        | Blockers dissolved: train ran (11 patch tags), push `e2f6be27`, CI all-green; b2–b5 + f1–5/8/41/47–49 struck; A7/A9/A11–A14 + owner rows routed                                                |
| `2026-10-05_21-19_ratchet-tail-executed-plan-annotations-upstream-filed.md` | Its §b blocker resolved end-to-end (v1.20.1 + sweep + `7c74307a` + green check-modules + push); final-verification + f1–6/14 + g2/g3 struck                                                    |
| `2026-10-06_03-04_alignment-push-train-and-ci-green-repair.md`              | End state held (CI sustained green); golden TODO struck this pass; stale agents-notes claim fixed `7c74307a`; b7/c7/f1/f2/f18 struck; battery/tooling/collection items routed                  |
| `2026-10-06_19-31_alignment-stall-env-triage-phase1-prep.md`                | Alignment completed + M01–M15 executed same evening (three sibling sessions); b1/b3 + c + f1–5/9–29/46 struck; flightrecorder push blocker + M16–M27 routed                                    |
| `2026-10-06_19-31_SUPERB-hardening-m11-m12-done-m13-m15-pending.md`         | Fully superseded by 21:35 (M13–M15 done, questions answered by execution); ANNOTATED + §b2/§c struck; 5 PARTIAL rows normalized to whole-row strikes                                           |
| `2026-10-06_21-35_SUPERB-hardening-m11-m15-complete.md`                     | M01–M10 closed same night by the sibling lane (22:34 + 23:20); §2 + §5 items 1–10 + §6-Q1 struck; flightrecorder + M16–M27 + forensics routed                                                  |
| `2026-10-06_22-34_SUPERB-phase1-complete-phase2-m06-midedit.md`             | Phase 2 closed by 23:20 (M06 `7584b7a84` completed; M07/M08/M10 landed + receipted); b1 + c-row + f1–6 + g2/g3 struck; push blocker + full-gate battery routed                                 |
| `2026-10-06_23-20_SUPERB-phase2-complete-m05-m10.md`                        | Lane M05–M10 verified + receipted; Q2/Q3 confirmed; Q1 (flightrecorder/push) + M16–M27 + GCL TODO items stay open; 1 PARTIAL row normalized                                                    |
| `2026-10-06_23-23_SUPERB-m05-m10-brutal-status-review.md`                   | The honest gaps (F22, GCL gate compliance, metaengine `-race`, doc comment) harvested into TODO_LIST; f30/f31 struck; push authorization + history-readability owner calls routed              |
| `2026-10-07_02-11_art-dupl-dedup-8-groups-status.md`                        | Round 1 mid-state (art-dupl `-t 4`); superseded same day by the 03-13 completion report                                                                                                        |
| `2026-10-07_03-13_art-dupl-dedup-round-complete-8-groups-dispositioned.md`  | Round 1 complete: 8 groups dispositioned (3 extracted, 5 accepted); release train + 3 owner questions routed forward                                                                           |
| `2026-10-07_04-03_art-dupl-t1-round2-status.md`                             | Round 2 (art-dupl `-t 1`, 113 groups): 7 extracted + verified, 106 accepted; post-round re-run 113 → 91 shown; full per-group triage                                                           |
| `2026-10-07_04-08_art-dupl-t1-round2-comprehensive-status.md`               | Round 2 comprehensive status + self-review: honest gaps (skipped gates d1/d2, manifest miss d5, batched-lint d3) + 40 ranked next steps + 3 owner questions                                    |

Also archived in the same sweep (planning corpus): `docs/planning/archived/2026-10-01_06-47_pareto-round13-superb-execution-plan.md` and `..._15-04_pareto-round14-gates-green-first.md` — both carry dated OUTCOME blockquotes, every executable item shipped 2026-10-01/02, residue (bench, T08 fleet swap, D-list) routed in TODO_LIST; the round-16 and episode-4 plans stay ACTIVE in `docs/planning/`.

### 2026-10-09 tail-pass archive manifest

Both files: classification **ARCHIVE** (dated `> ANNOTATED 2026-10-09` blockquote + evidence strikes + open items routed), `git mv` → `docs/status/archived/`. Verification anchors: origin CI green (run 37851669797 on `c5bbc693`), fixture conversion `70b1289e`, R12/T8 `baeffce1`+`c5d223de`, R15/T14 `ab23e247`, family-walk receipt `23b54cc2`, runbook docs `b2a12066`, the landed frontend-sync feature (`278b16c2`, ADR-0056). The manifest lives HERE (not `archived/README.md`) — the annotation gate requires every `archived/*.md` to carry a date prefix.

| File                                                                        | Deciding evidence                                                                                                                                                          |
| --------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `2026-10-08_22-02_train-ci-repair-family-walk-fix-fixture-helper-status.md` | Patch train + httputil v1.4.2 + family-walk fix all shipped and CI-verified; 55 strikes; survivors routed to TODO rows + decision table D14/D15 + the train-trust-plan row |
| `2026-10-09_00-06_fixture-consolidation-upstream-wave-branching-flow-hardening_status.md` | Fixtures landed, upstream wave aligned, branching-flow ratchet re-pinned to 229; battery + owner lanes routed; the sync session landed after this report                 |

### docs-health round 47 archive manifest (2026-11-05)

All 3 files: classification **ARCHIVE** (dated `> ANNOTATED 2026-11-05` blockquote + evidence strikes + routed verdicts via `annotate.go repair`), gates BEFORE the move (`check-rows --strict` + `bare-rows` green per file), `git mv` → `docs/status/archived/`. Tail 5 → 2, `TAIL_BASELINE` 3 → 2 in the same commit. Verification anchor: the v4.16.0 family train (root/setup/adminui/dashboardui/loginpage v4.16.0 + usermgmt v4.19.0, cut+pushed 2026-11-05; release-train 897 requires / 0 unpublished / 0 lag; residual CI red = the upstream go-cqrs-lite catalog class only — the 06-38/06-41 reports).

| File | Deciding evidence |
| ---- | ----------------- |
| `2026-11-04_17-28_push-unblock-complete-sweep5-gate-green-stack-on-origin_status.md` | Its §g1 GO answered by execution: the six-tag family train shipped 2026-11-05; auditlog retract moot via upstream v0.12.2; daemon policy → OQ20; standing backlog → TODO rows |
| `2026-11-04_23-05_root-package-split-master-sync-self-review_status.md` | Split landed as PR #40 (`c6b6547d`); pre-struck by the 2026-11-05 planning pass (21 items), remainder resolved/routed this pass (plan M02 battery, ROADMAP tooling/docs-polish, OQ20); its own §f49 asked for this archive |
| `2026-11-05_02-47_pr40-pr39-land-release-train-remaining_status.md` | The release train it gated was executed 2026-11-05; zero-CI window dissolved (no gap existed, 06-38 §a11); gotchas 49/50 landed; CHANGELOG [v4.16.0]/[v4.15.0] sections cut the same day; tails → TODO rows / plan M02–M05 |

**Corpus ledger (baseline 43687 → 43992 this pass → 44069 after the 07-24 sibling session's own re-true; live truth = `docs/CORPUS_BASELINE`):** Round-47 report pass: 44069 → 44153 (the round-47 status report ~+95 + the 06-38 ARCHIVED blockquote ~+5; its 94 item verdicts append inline, zero new lines). the 06-38 report (+149) and the 06-41 report (+128) landed after the previous baseline; this pass added +28 (three ANNOTATED blockquotes ~16 lines + this manifest/ledger table 12 lines; the 169 item verdicts append inline, zero new lines). Round-47-tail session 2 (17-40 report): 44153 → 44184 (sibling receipts) → 44297 (+113: the 17-40 status report; the §f8 annotation + archived skip marker append inline, zero new lines). `TAIL_BASELINE` 3 → 4 same-change for the fourth live report (the sibling's 2→3 raise precedent; the 06-41 archive stays deliberately with the 07-24 lane per round-47 §f20). Round-46 M10–M13 session (18-15 report): 44297 → 44372 (ledgered in that report). Dropdown-fix session (this pass, 2026-11-05 evening): 44372 → 44373 (+1: the CHANGELOG adminui dropdown-anchor entry, the AGENTS templ-components anchoring bullet, and the branching-flow triage row net +1 after the corpus gate's own count; no new reports — `TAIL_BASELINE` stays 5). Same commit also raises the root-files baseline 31 → 33 (`templ-hash.nix` + `templ-vendorHash.nix`, the templGen derivation's hash files from `8710b96d` — the benchstat `hash.nix`/`vendorHash.nix` precedent). Same evening, dropdown-fix session's exit report: 44373 → 44457 (+84: the 22-31 status report) with `TAIL_BASELINE` 5 → 6 same-change (the fourth report of the day; next docs-health round archives 06-41/07-24/09-53/17-40/18-15). Fleet libs utilization audit (2026-11-05): 44457 → 45575 (+1118: 19 per-library reports under `docs/review/libs/` + `index.html` (uncounted — md-only metric) + the `LIBRARY_INTEGRATIONS.md` superseded banner + this series row). Same thread's self-review report (00-27, 2026-11-06): 45575 → 45646 (+71, the delivery + self-review status report) with `TAIL_BASELINE` 7 → 8 same-change. Pareto plan session (00-37, 2026-11-06): 45646 → 45891 (+245: the follow-through plan + the systemcatalog-unblocked errata fixes in the audit files + the TODO_LIST harvest row). Both-issues session (issues #38 + #25, 2026-11-06): 45891 → 46024 (+133: ADR 0060 + its INDEX row, the actor-and-audit-trail session-lifecycle section and its stale "24 events" true, event-count trues 25 → 27 across AGENTS/root-README/usermgmt-README, and the round-23 archive pass's two ANNOTATED blockquotes + archived/README manifest row) with `TAIL_BASELINE` 8 → 6 same-change (round-23 archived 06-41 + 07-24 to the round-47 successor; the sibling's 00-27 report had raised the tail to 8). Post-report true-up (02:15, 2026-11-06): 46024 → 46136 (+112: the sibling's 01-57 adminui-superbification report ~+111 + this session's branching-flow triage row +1) with `TAIL_BASELINE` 6 → 7 same-change (the sibling's report, sixth live). Plan-execution session (02-44, 2026-11-06): 46136 → 46293 (+157: this 02-44 execution status report + the event-catalog guide's 27-event table true + the health README nonce-CSP section) with `TAIL_BASELINE` 7 → 8 same-change (this report, eighth live). Both-issues exit addendum (03:10, 2026-11-06): 46293 → 46295 (+2: the DONE follow-through blockquote on the 23-57 both-issues report — one ledgered paragraph line); tail unchanged at 8. Both-issues follow-through exit report (03:19, 2026-11-06): 46295 → 46463 (+168: the 03-19 follow-through status report) with `TAIL_BASELINE` 8 → 9 same-change (this report, ninth live; next docs-health round drains the settled 2026-11-05 set). Adminui-superbification wave-7 close-out (04:0x, 2026-11-06): 46463 → 46468 (+5 net: the TODO_LIST next-train P1 row, the adminui/setup README config rows — ExtraNav/DashboardURL/AdminExtraNav/AdminDashboardURL — and this ledger sentence) with the round-47 archive pass (09-53 → 00-27 successor, resolved) taking `TAIL_BASELINE` 9 → 8 same-change. Fleet-plan wave-4 resumption (05:24, 2026-11-06): 46468 → 46533 (+65: the 05-24 wave-4 status report — which also carries the recovered essentials of the vanished 04-29 wave-3 report, a gotcha-47 daemon snapshot-restore loss confirmed by `--diff-filter=A` over its pathspec returning nothing — plus the round-23 archive pass's SUPERSEDED blockquote and manifest row) with the round-23 archive pass (17-40 → 02-44 successor, superseded) holding `TAIL_BASELINE` at 8 same-change. Re-export eviction plan session (07:16, 2026-11-06): 46533 → 46939 (+406: the `2026-11-06_07-16_SUPERB-direct-imports-re-export-eviction-and-root-burndown-pareto-plan.md` plan of record — 53 root production files / 7,594 LOC measured, 718 LOC of aliasing catalogued across 9 root files plus 176 `usermgmt` aliases, W0 executable on v4 and W1–W3 riding the single v5 window); tail unchanged at 8 (no new status report). Command/query permission-middleware design triage (07-24, 2026-11-06): 46939 → 47363 (+424: the 07-24 triage status report — root CQRS/decoder/authz symbol inventory, the dispatch-boundary gap, the exact-match matcher blocker at `identity-model/authz_model.go:24`, six self-attributed failures, 50 next items; this ledger sentence appends inline, zero new lines) with `TAIL_BASELINE` 8 → 9 same-change (this report, ninth live; nothing was implemented — the session was read-only). Dispatch-boundary permission-gate plan (07:30, 2026-11-06): 47363 → 47825 (+462: `docs/planning/2026-11-06_07-30_SUPERB-command-permission-gate-dispatch-boundary-pareto-plan.md` — 15 verified planning facts with symbol:line, the 1 %/4 %/20 %/rest Pareto split, 10 autonomous decisions, 15 milestones ≈1,075 min, 105 micro-tasks ≤12 min, mermaid graph with 3 kill points, §f coverage proof, 4 errata (the sync-push claim corrected, the stale maturity-ratchet TODO row named); tail unchanged at 9 — a plan is not a status report). Direct-imports W0 execution (08:10, 2026-11-06): 47825 → 48269 (+444 total = this session's +258 (ADR-0062 93 lines + its INDEX row 1, `v5-removal-inventory` class 1b 26, the `v4-to-v5` per-layer section 84, the plan's §11 execution log 39, CURRENT_FOCUS 13, AGENTS 2, plus the treefmt reflow +1 — the TODO_LIST row updated in place, 0 net) plus the sibling session's ~185 that landed after this baseline was frozen (ADR-0063 + its INDEX row and the docs it touched)); tail unchanged at 9 (no new status report). Direct-imports W0 status report (08:04, 2026-11-06): 48269 → 48779 (+510: the 08-04 execution status report at 397 lines plus ~113 of sibling growth that landed in the same window) with `TAIL_BASELINE` 9 → 10 same-change (this report, tenth live; the next docs-health round drains the settled 2026-11-05 set). W0 post-review report (08:16, 2026-11-06): 48779 → 49106 (+327: the 08-16 "what the wave forgot" report — the AGENTS.md Architecture clause and the skill's rule 1 were prose edits on existing lines, so the whole delta is the report itself) with `TAIL_BASELINE` 10 → 11 same-change (this report, eleventh live; nothing was archived, so the tail keeps the two 2026-11-06 W0 reports until the next docs-health round). Dispatch-boundary gate execution (08:42, 2026-11-06): 49106 → 49779 (+673: the 08-42 execution status report plus the ADR-0063 gotcha-54 ledger entry, the PERMISSIONS L5 level + inventory row, the guide §5 dispatch-boundary section, the skill/README/reference additions and the plan receipt — several of those deltas landed inside sibling/daemon commits after the baseline had frozen) with `TAIL_BASELINE` 11 → 12 same-change (this report, twelfth live; the next docs-health round drains the settled 2026-11-05/06 set). Dispatch-gate hot-path and bench-experiment close-out (09:05, 2026-11-06): 50049 → 50088 (+39: ADR-0063's lock-free read-path consequence, the plan's "The §d bench experiments, run and resolved" subsection, and the CHANGELOG allocation true-up — the `authz_dispatch.go` copy-on-write change and its concurrency test are code-only and add zero corpus lines); tail unchanged at 12 (no new status report). Dispatch-gate verification report (09:06, 2026-11-06): 50088 → 50585 (+497: the 09-06 verification status report — the fifteen-milestone re-verification, the hot-path mutex removal and its mutation-checked race test, the allocation deltas, six self-attributed failures, and the 50-item next list) with `TAIL_BASELINE` 11 → 12 same-change (the sibling's archive pass had drained two W0 reports to 11 before this one landed; the next docs-health round drains the settled 2026-11-05/06 set). Corpus-baseline repair (10:30, 2026-11-06): the committed `docs/CORPUS_BASELINE` (from `24b653d6`) held an UNRESOLVED merge conflict (`50585` vs a stale stashed `49803`), so `--ratchet` refused the file outright ("must contain a single integer"); resolved to the measured truth 50,588 = upstream 50585 + this session's dependency-train failure-class row (+1) + 2 lines of sibling drift that landed after the freeze; tail unchanged at 12. Docs-health pass (11:55, 2026-11-06): 50,588 → 50,649 (+61 = this session's +5 net — the re-export plan's §1.2 re-measure note +6, the Push bullet's 6 → 5 collapse −1, two zero-new-line inline corrections — plus ~56 of sibling growth that landed after the freeze, the `docs/guides/actor-and-audit-trail.md` session-lifecycle section, whose own ledger row is the sibling's to write). The pass fixed the five rule-F findings that had redded `docs-gates annotate` (AGENTS.md, CHANGELOG.md, TODO_LIST.md + two status reports) and resolved the committed merge conflict in this very baseline file; tail unchanged at 12.

## Why not "update them all" ad hoc?

Bulk-editing historical files without verification would be a [Verschlimmbesserung](https://en.wikipedia.org/wiki/Verschlimmbessern) — a well-intentioned change that makes things worse. The 2026-09-09 sweep did the verified version of this work once (every report read and checked against CHANGELOG/git/code before annotation). New reports: write them, and route anything still-open into the living docs so the next sweep stays small.
