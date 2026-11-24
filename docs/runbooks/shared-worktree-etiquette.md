# Runbook: Shared-Worktree Etiquette (two sessions, one tree)

Operating procedure for running two (or more) agent sessions concurrently in
this repo. The 2026-11-24 dual-wave (adminui next-wave + identity-merge wave,
both finishing the same evening) hit every class below at least once; this
page turns the scar tissue into rules. Recovery procedures for the commit
races live in `daemon-commit-races.md`; this page is the PREVENTION half.

## Rule 1: claim disjoint file sets, in writing

Every dispatched subagent (and every lane of your own work) carries an
explicit OWNERS list and an explicit DO-NOT-TOUCH list. The merge-wave
session's `handler_merge.go` route rows landed in the WRONG file for the
adminui surface gate's scanner (gotcha 70) precisely because the file
layout was designed without reading the sibling's gate. Read the other
lane's gates BEFORE designing your file layout.

## Rule 2: never interleave a bump-dep sweep with anything

`scripts/tools/bump-dep.sh` refuses on a dirty tree and warns on a live
sibling sweep (live-PID + fresh-log probes, round-38 interleave). Sweep
ONLY on a quiet tree; the gate-printed FIX RECIPE names the exact
pattern+version per family (the `$` anchor means end-of-MODULE-PATH, not
end-of-line — if the sweep reports "no go.mod requires", the quoting was
wrong, not the anchor).

Hermetic verification inside a sweep fails on the red-by-construction
modules (they reference unpublished own-family symbols). Sweep with
`--no-lint`, then verify workspace-mode build+test yourself, and commit the
failed-verify modules' version edits with a message naming the class —
that is what the 2026-11-24 go-codec sweep did, and the train went 0-lag.

## Rule 3: batteries pollute each other — sequence them

The e2e battery ports (18923-18941) are shared on this worktree. A battery
run started while a sibling edits adminui source mid-flight produces
transient reds that waste a triage cycle (both 2026-11-24 sessions
misattributed each other's in-flight edits before proving the real bugs).
The kill ritual (`scripts/tools/e2e-kill-strays.sh`) now REFUSES while a
`playwright test` process is live (rc=3, `--force` overrides) — coordinate
instead of killing.

During a sibling's active lane: run TARGETED specs only, and leave the
full battery to whoever closes the tree.

## Rule 4: templ regen is a single-writer resource

`nix run .#gen` collides with a sibling's regen. On collision: wait 60s,
retry, max 3. `updates=0` in the gen output means the tree was already
canonical — verify the generated file carries your change instead of
re-running blindly (gotcha 67).

## Rule 5: verify modes — workspace vs hermetic

Workspace mode (go.work replaces) sees ALL in-tree code and is green on a
healthy tree. Hermetic mode (GOWORK=off) builds against PUBLISHED tags and
is red-by-construction for any module referencing unpublished own-family
symbols — planned-red, not broken. Before triaging a hermetic red, check
whether the failing symbol exists in the tree (then it is the tag-gap
class, gotcha 30) or not (then it is real).

## Rule 6: cross-session failure attribution needs a named mechanism

"Misattributed to the concurrent session's fixtures" was wrong twice in
the 2026-11-24 adminui wave — the real bugs were ours (per-role tenant
counting, an ambiguous revoke selector), proven by the sibling's subagent.
Rule: before blaming the sibling, name the exact mechanism by which their
change causes your failure, and bisect to isolate. A failure you cannot
mechanism-explain is probably yours.

## Rule 7: DONE-claims cite artifacts, not inferences

A subagent (or session) claiming a task done must cite route rows, test
names, or file:line evidence. The 2026-11-24 wave's first final report
claimed T08 done when it had never been dispatched; verification proved
the claim false and the honest re-report rebuilt trust. Never accept
"landed" without a receipt you can check.

## Quick checklist before dispatching a second session

- [ ] Disjoint file OWNERS/DO-NOT-TOUCH lists written into both prompts
- [ ] Read the sibling lane's gates before designing file layouts
- [ ] Sweeps scheduled on a quiet tree only (bump-dep refuses otherwise)
- [ ] Battery runs sequenced (targeted-only during a sibling's lane)
- [ ] templ regen collision protocol in every prompt (wait 60s × 3)
- [ ] Both lanes know the workspace/hermetic red-by-construction classes
- [ ] DONE-claim receipts required (artifacts, not inference)
