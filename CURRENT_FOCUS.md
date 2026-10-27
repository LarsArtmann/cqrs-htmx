# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-10-27 01:30 · **Owner:** Lars · **Module count:** 29 (`go.work`)

## The active thread

**Wave D module lane is EXECUTED to its final receipt; three owner questions gate the
epilogue.** Plan of record:
[`docs/planning/2026-10-25_10-42_SUPERB-wave-d-completion-code-fidelity-to-closure-pareto-plan.md`](docs/planning/2026-10-25_10-42_SUPERB-wave-d-completion-code-fidelity-to-closure-pareto-plan.md)

**State (round-35 receipt, 2026-10-26/27):** W12 checkpoint green; W14.1–W14.6 and
W15.2–W15.5 landed (the W14.3 export-walker hang was the session's P1 — root-caused,
stall-guarded, CI-healed); the final battery is CI-authoritative (box load 150+
made the local legs an honest refusal). Master green. Remaining on THIS lane:
nothing executable without the owner.

**Done when:** the owner answers the round-34 §g packet — (1) branching-flow
baseline refresh (+46/−40 drift, held), (2) upstream filings + W04 defaults
ratification (held), (3) release timing for loginpage `NoWebAuthn` + the dashboardui
fixes (default taken: batch into the next train, not mid-wave) — plus the two
dashboardui tail sub-items (page-size off-menu, browser e2e smoke) that ride the next
dashboardui change.

## The next thread (start immediately after)

Whatever the owner's §g answers unlock first: the S55 packet chains (cqrs-lint fleet
swap → CI wiring, ReadLastN upstream), the branching-flow re-identification +
baseline refresh as one act, or the release train. The sibling lane (W09/W13/W16)
closes independently; the 10-42 plan's terminal DONE line waits for both lanes.

## Resolved since the last update

**The module lane's W14+W15+W17 verdicts** (receipt:
[`docs/status/2026-10-27_01-30_round-35-w14-w15-landed-ci-battery-of-record-8-verdicts-parked-rows-held-gates-named_status.md`](docs/status/archived/2026-10-27_01-30_round-35-w14-w15-landed-ci-battery-of-record-8-verdicts-parked-rows-held-gates-named_status.md)):
master healed from the w14.3 CI-killer (cursor-ignoring fake → OOM class), journal
walkers stall-guarded everywhere, snapshot 404/500 split, boost-inheritance guard,
root-files allowlist gate, coverage thresholds single-sourced + fail-loud, tooling
canaries a–g, skill-maintenance gates ×3 (caught real skill drift), 8 parked-row
verdicts confirmed, TODO swept (58 rows, hygiene green).

## Ground rules that keep this honest

1. **One thread at a time.** Anything not listed above is in [TODO_LIST.md](TODO_LIST.md) and waits.
2. **Never add a row to `TODO_LIST.md` without a verify command** or a named evidence path.
3. **Never leave a finished row in `TODO_LIST.md`** — it moves to [CHANGELOG.md](CHANGELOG.md).
4. **Closed history is not documentation.** `*/archived/` is out of the reading path by design;
   open it only for provenance questions.
5. **Master stays green.** Never push a stack whose battery is red on content (the 15-57 hold rule).
