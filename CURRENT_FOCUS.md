# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-10-25 16:00 · **Owner:** Lars · **Module count:** 29 (`go.work`)

## The active thread

**Finish Wave D on the two-session tree: the docs lane is CLOSED, the push is held on
the module lane's lint red, and the module lane (W04–W15) continues.**
Plan of record:
[`docs/planning/2026-10-24_05-36_SUPERB-master-plan-docs-eviction-delivery-and-backlog-burn-down.md`](docs/planning/2026-10-24_05-36_SUPERB-master-plan-docs-eviction-delivery-and-backlog-burn-down.md)
+ the 10-42/10-43 lane split (docs lane: W09/W13/W16/W17-verdicts/quartet/sync-tails — ALL DONE).

**State:** sync tails shipped as assets v1.6.0 (`5e0fb368`); battery run green
except lint — **7 findings, all in the sibling's W03 code** (root/transport ×5,
auditlog ×1, usermgmt ×2; inventory in the receipt §b1). 14 commits sit unpushed;
pushing them before that lint is green would turn master's CI red.

**Done when:** sibling lint findings fixed → `nix run .#lint` rc=0 → push the
stack through the strict pre-push gates → CI terminal green → `.#verify-remote-tags`
→ push receipt appended.

## The next thread (start immediately after)

**The module lane's plan of record continues: W04 (identity-model tails) → W05–W08
→ W10/W11 → W12 checkpoint → W14/W15 → W17-final receipt** (their sequencing lives
in `docs/status/2026-10-25_15-22_*.md` §f). First action there: the missing
usermgmt/CHANGELOG S39 receipt, then the lint fixes above (their §b2 gap).

## Resolved since the last update

**The docs lane of Wave D is closed** (receipt:
[`docs/status/2026-10-25_15-57_wave-d-lane-closed-sync-tails-v1-6-0-battery-run-push-held-on-sibling-lint_status.md`](docs/status/archived/2026-10-25_15-57_wave-d-lane-closed-sync-tails-v1-6-0-battery-run-push-held-on-sibling-lint_status.md)):
W09 master-healing, W13 owner packet (S55+S56, on origin), W16 fixed-upstream
verdict, W17 verdicts ×8, docs-truth quartet, sync tails s.1–s.4 (cursor
monotonicity guard + v1→v2 upgrade pins + guide spec/budget/CSP sections).
Master green at `3ff9e0de`; battery: test rc=0 · coverage rc=0 ·
check-modules 58/59 (branching-flow +22, owner-gated) · lint red = sibling's.

## Ground rules that keep this honest

1. **One thread at a time.** Anything not listed above is in [TODO_LIST.md](TODO_LIST.md) and waits.
2. **Never add a row to `TODO_LIST.md` without a verify command** or a named evidence path.
3. **Never leave a finished row in `TODO_LIST.md`** — it moves to [CHANGELOG.md](CHANGELOG.md).
4. **Closed history is not documentation.** `*/archived/` is out of the reading path by design;
   open it only for provenance questions.
5. **Master stays green.** Never push a stack whose battery is red on content (the 15-57 hold rule).
