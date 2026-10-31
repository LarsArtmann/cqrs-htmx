# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-10-31 07:50 · **Owner:** Lars · **Module count:** 29 (`go.work`)

## The active thread

**Round 45 landed and pushed (`82ddd1b8`): the samber/do × health thread is CLOSED** —
round-2 verdict 9/10, all five owner decisions executed on defaults (T12 wire rename
shipped; `RecorderChain` docs migrated; T13 closed; `InvokeAs` filing authorized+drafted;
erraudit strict = advisory month started), the 3-family train cut locally and the
push blocker resolved (train lag 0), five new gates landed (readme-badges, pr-secrets,
dependabot watch, coverage self-test, check-modules parity lock), and the backlog
burned: 5 TODO rows closed, 15 parked rows re-audited with dated verdicts.
Receipt: [`docs/status/2026-10-31_07-45_round-45-decision-batch-family-train-and-gate-hardening_status.md`](docs/status/archived/2026-10-31_07-45_round-45-decision-batch-family-train-and-gate-hardening_status.md).

**Done when** (met): the final battery 5× rc=0 on a PUSHED commit — check-modules 78/78,
`.#test` rc=0, coverage-gate 15/15, `.#lint` rc=0, erraudit --gate TOTAL=0, pre-push
gates green.

## The next thread (pick ONE)

1. **Advisory month for erraudit strict** (passive): let `.#check-erraudit-strict`
   accumulate a green month at TOTAL ≤ 303, then flip `--strict --gate` (TODO watch row).
2. **File the samber/do `InvokeAs` issue** (owner click): draft ready at
   `docs/proposals/upstream/samber-do-invokeas-map-order-scan.md`, repro verified twice
   on v2.1.0; M01 verdict 4 authorizes.
3. **10-42 Wave-D remainder** (W01 S37 tails, W07 schema stamping) on its own trigger —
   the two dashboardui tail sub-items (page-size off-menu pins, browser e2e smoke) ride
   the next dashboardui change (verified: pins exist; browser legs are CI-gated).

## Ground rules that keep this honest

1. **One thread at a time.** Anything not listed above is in [TODO_LIST.md](TODO_LIST.md) and waits.
2. **Never add a row to `TODO_LIST.md` without a verify command** or a named evidence path.
3. **Never leave a finished row in `TODO_LIST.md`** — it moves to [CHANGELOG.md](CHANGELOG.md).
4. **Closed history is not documentation.** `*/archived/` is out of the reading path by design;
   open it only for provenance questions.
5. **Master stays green.** Never push a stack whose battery is red on content (the 15-57 hold rule).
