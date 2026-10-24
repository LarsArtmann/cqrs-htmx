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
