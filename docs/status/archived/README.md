# Status Archive — Bulk-Pass Manifest

One line per file for every MULTI-file archive pass (single-file archives carry their manifest in the commit message). A bulk archive without a manifest is invisible history.

| Pass | File | Classification | Deciding reason |
| --- | --- | --- | --- |
| round-21 (2026-10-11) | `2026-10-07_02-11_art-dupl-dedup-8-groups-status.md` | ARCHIVE — superseded | Intra-round status; the same round's completion report (`2026-10-07_03-13_…dispositioned.md`) covers every item with verdicts quoted (and its item 12 requested this supersession annotation). Follow-ups harvested + completed via TODO_LIST row 52. |
| round-21 (2026-10-11) | `2026-10-07_04-03_art-dupl-t1-round2-status.md` | ARCHIVE — superseded | Intra-round status; `2026-10-07_04-08_…comprehensive-status.md` re-covers the round WITH the self-review this file lacks. Follow-ups harvested + completed via TODO_LIST row 52 (which cites this file's §c). |

Round-21 scope note (2026-10-11): this pass time-boxed to the clearly-settled supersession pair; the remaining ~25 live reports (tail budget 3) owe a full ANNOTATE sweep in round-22 — completion reports with open f)-lists (e.g. the 10-07 art-dupl completions, the 10-09 frontend-sync series) stay LIVE until every numbered item is resolved inline.

Earlier passes: pre-2026-10 archives were moved one-at-a-time or by passes whose manifests lived in the pass's own report (this README was created 2026-10-11 to satisfy the `docs/status/README.md` claim that had drifted file-less since round-19).

## Sweep log (moved here from `docs/status/README.md` on 2026-10-24)

Counts are deliberately absent: they rot. The pass history is what matters.

The archive tail has been swept repeatedly: 2026-09-09 (full backlog), 2026-09-20 (the 38-report tail), 2026-09-21 (the 34-report tail), 2026-09-22 (the 8-report superb-plan tail + the plan itself), 2026-10-01 round-13 (the 09-22→09-30 tail — every report annotated with a dated blockquote + evidence-backed inline strikes, open items routed to TODO_LIST/ROADMAP, then archived), 2026-10-01 round-14 (the 10-01 pair — the 02:59 BuildFlow-recovery and 06:36 round-13 reports got W0–W3-receipt top-up blockquotes + inline strikes for everything the train/gate/erraudit sessions then resolved, then archived; the same pass triaged the six 2026-08-30 planning files — upstream-asks draft archived, gated-work index + cqrs-lint distribution draft annotated, three live owner-gated decisions confirmed leave-in-place — and confirmed the HTML corpus keep-in-place decision), 2026-10-04 round-17 (the 10-01→10-04 tail — every report annotated with a dated blockquote + evidence strikes verified against the tree (CI run 37178906141: test green on published pins; master red on exactly one job — the loginpage 7/5 dep budget) and routed to TODO_LIST/ROADMAP, then archived; the two `docs/drafts/2026-10-04-signing-cloneevent-encoding*` files got RESOLVED-SHIPPED blockquotes and stay in place), and 2026-10-06 round-18 (the 10-04→10-06 tail — every report annotated with a dated blockquote + evidence strikes verified against the tree (the identity-auth-hardening wave tags, the 11-module sweep train, push `e2f6be27`, CI green 37392674249, the M01–M15 hardening execution across three sibling sessions) and routed to TODO_LIST/ROADMAP, then archived; the six 2026-08-30/09-23 planning files re-confirmed: 3 annotated, 3 owner-gated LEAVE-IN-PLACE; the HTML corpus keep-in-place decision re-honored).

Reports archived before 2026-09-09 predate the annotation convention and are legacy-exempt (see `docs/status/README.md` § Older dialects). Gate 1 enforces the current-convention era; Gate 2 keeps the row shapes honest.

## Round-22 archive manifest (2026-10-24)

Scope: the clearly-superseded series-internal reports — each has a later report in the SAME series
that re-covers the work with the outcome and a self-review. Classification: **ARCHIVE —
superseded**. Every file got a dated `> ANNOTATED 2026-10-24` blockquote naming its successor, plus
the `~~this report~~ SUPERSEDED` marker; relative links were re-based for the new depth; 7 pre-existing
PARTIAL table rows were normalized by `scripts/tools/normalize-status-rows.py` before the gate
passed. The live tail went 58 → 44 and `docs/status/TAIL_BASELINE` was lowered in the same commit.

| Pass | File | Classification | Deciding reason |
| --- | --- | --- | --- |
| round-22 (2026-10-24) | `2026-10-07_03-13_art-dupl-dedup-round-complete-8-groups-dispositioned.md` | ARCHIVE — superseded | [`2026-10-07_04-08_art-dupl-t1-round2-comprehensive-status.md`](2026-10-07_04-08_art-dupl-t1-round2-comprehensive-status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-09_04-04_frontend-sync-protocol-adr0056.md` | ARCHIVE — superseded | [`2026-10-09_05-17_frontend-sync-protocol-completion-and-self-review.md`](2026-10-09_05-17_frontend-sync-protocol-completion-and-self-review.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-09_04-15_SUPERB-round-execution-20-tasks-done-preflight-refuses-foreign-wave_status.md` | ARCHIVE — superseded | [`2026-10-09_15-38_brutal-self-review-superb-round-20-tasks_status.md`](2026-10-09_15-38_brutal-self-review-superb-round-20-tasks_status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-09_17-19_SUPERB-execution-phase1-status.md` | ARCHIVE — superseded | [`2026-10-09_19-44_SUPERB-samber-do-health-session-brutal-self-review_status.md`](2026-10-09_19-44_SUPERB-samber-do-health-session-brutal-self-review_status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-09_18-10_SUPERB-samber-do-health-execution-phases-2-5_status.md` | ARCHIVE — superseded | [`2026-10-09_19-44_SUPERB-samber-do-health-session-brutal-self-review_status.md`](2026-10-09_19-44_SUPERB-samber-do-health-session-brutal-self-review_status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-11_10-58_tag-wave-staged-sync-e2e-green-sync-asset-fixes_status.md` | ARCHIVE — superseded | [`2026-10-12_05-23_train-lag-unblock-atomic-wave-ci-repair.md`](2026-10-12_05-23_train-lag-unblock-atomic-wave-ci-repair.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-11_18-28_pareto-continuation-p10-p12-delivered-wave-blocked-self-review_status.md` | ARCHIVE — superseded | [`2026-10-11_18-40_wave-blocker-disproven-sibling-held-p15-repro_status.md`](2026-10-11_18-40_wave-blocker-disproven-sibling-held-p15-repro_status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-13_19-30_wave-completion-adminui-deferral-battery-green_status.md` | ARCHIVE — superseded | [`2026-10-13_22-33_wave-closure-adminui-unblock-go-appkit-lane-masked-red-revealed_status.md`](2026-10-13_22-33_wave-closure-adminui-unblock-go-appkit-lane-masked-red-revealed_status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-13_21-10_wave-tail-tooling-go-appkit-lane-adminui-unblock-sighted_status.md` | ARCHIVE — superseded | [`2026-10-13_22-33_wave-closure-adminui-unblock-go-appkit-lane-masked-red-revealed_status.md`](2026-10-13_22-33_wave-closure-adminui-unblock-go-appkit-lane-masked-red-revealed_status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-14_20-50_family-alignment-x3-gate-bugs-systemadapter-root-cause_status.md` | ARCHIVE — superseded | [`2026-10-14_20-54_wave-tail-execution-gate-bugs-systemadapter-root-cause-self-review_status.md`](2026-10-14_20-54_wave-tail-execution-gate-bugs-systemadapter-root-cause-self-review_status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-17_09-34_pareto-plan-push-saga-inherited-systemadapter-red.md` | ARCHIVE — superseded | [`2026-10-17_10-41_docs-truth-wave-execution-halfway_status.md`](2026-10-17_10-41_docs-truth-wave-execution-halfway_status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-19_02-42_github-issue-sweep-fix-session_status.md` | ARCHIVE — superseded | [`2026-10-19_08-47_issue-sweep-execution-complete_status.md`](2026-10-19_08-47_issue-sweep-execution-complete_status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-19_04-38_issue-sweep-execution-train-complete_status.md` | ARCHIVE — superseded | [`2026-10-19_08-47_issue-sweep-execution-complete_status.md`](2026-10-19_08-47_issue-sweep-execution-complete_status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |
| round-22 (2026-10-24) | `2026-10-19_08-47_issue-sweep-execution-complete_status.md` | ARCHIVE — superseded | [`2026-10-19_12-18_issue-sweep-self-review-and-next-50_status.md`](2026-10-19_12-18_issue-sweep-self-review-and-next-50_status.md) is the same series' later report (outcome + self-review); nothing unique stays open here. |

Not in this pass: the remaining 44 live reports. They are NOT bulk-archived — a report lands here
only when its items are resolved or its series has a successor. The ratchet gate now forbids the
tail from growing, so the rest can follow in verified batches.
