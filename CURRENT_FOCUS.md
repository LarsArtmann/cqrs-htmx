# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-10-25 · **Owner:** Lars · **Module count:** 29 (`go.work`)

## The active thread

**Execute Wave D (Wave R is DONE — tree green on pushed HEAD `a5183fff`, CI `40179492715` all-success).**
Plan of record:
[`docs/planning/2026-10-25_10-42_SUPERB-wave-d-completion-code-fidelity-to-closure-pareto-plan.md`](docs/planning/2026-10-25_10-42_SUPERB-wave-d-completion-code-fidelity-to-closure-pareto-plan.md)
(supersedes 08-05 for execution order; the 06-20 file keeps the per-milestone
scope text for D01–D15). Two-session lane split (binding):
[`docs/planning/2026-10-25_10-43_SUPERB-lane-addendum-two-session-split-of-the-10-42-plan.md`](docs/planning/2026-10-25_10-43_SUPERB-lane-addendum-two-session-split-of-the-10-42-plan.md)
— sibling executes W01–W08/W10/W11/W14/W15 (module code), this docs session
executes W09/W13/W16/W17-verdicts + the quartet + sync tails.

**Done when:** master-plan Appendix D reconciled; every TODO_LIST row
executed or carrying a dated verdict; the four ratchets (tail, corpus,
branching-flow, domain counts) green in the same run; CI green on the
final pushed HEAD.

**Progress (2026-10-25 10:15):** Wave R + D04 CLOSED — R01–R03 receipts
landed earlier (S32 proof tests caught and fixed the unwired
systemadapter registration; usermgmt v4.16.0), D04 executed (golangci
**2.15.0 everywhere**: the 189 wrapcheck findings on the sanctioned
`go-error-family` layer are `ignore-package-globs`'d in all 15
wrapcheck-enabled configs; CI pin moved up to match the flake), the full
battery green (test rc=0 · lint rc=0 · coverage-gate PASS ·
check-modules 55/55), pushed through strict gates, CI terminal green.
Receipt: [`docs/status/2026-10-25_10-15_green-tree-receipt_status.md`](docs/status/archived/2026-10-25_10-15_green-tree-receipt_status.md).

## The next thread (start immediately after)

**Wave D remainder, in the plan's order: D01 (delivery receipt: Appendix D +
CHANGELOG receipts S35/S41/S44 + S27 TODO harvest) → D02 (docs & doctrine:
S45/S52/S53/S54) → D03 (tooling: S47/S48) → D05–D09 (code-fidelity wave) →
D10–D16.** Everything is micro-brokened in the plan's §4.2 tables.

**Done when:** each milestone's verify command is green and its rows/receipts
are committed (guardrails in the plan's §2).

## Resolved since the last update

**S32 (WebAuthn sign-counter persistence) is DONE and proven.** Login persists
assertion state (`RecordCredentialUse` → `CredentialUsed`), regressions are
conflicts, UP/UV round-trip honestly, and the proof tests exposed + fixed the
unwired systemadapter registration (usermgmt v4.16.0). Tags: identity-model
v4.14.0, usermgmt v4.15.0/v4.16.0, usermgmt/webauthn v4.13.0.

**The docs-corpus thread is closed.** Tail 38 → 1, 617+77 files evicted with a
restore manifest, keep-set re-derived (49), planning triaged, TODO_LIST
reworked. The four hygiene gates (tail, corpus, todo, archive) run in
check-modules + CI.

## Ground rules that keep this honest

1. **One thread at a time.** Anything not listed above is in [TODO_LIST.md](TODO_LIST.md) and waits.
2. **Never add a row to `TODO_LIST.md` without a verify command** or a named evidence path.
3. **Never leave a finished row in `TODO_LIST.md`** — it moves to [CHANGELOG.md](CHANGELOG.md).
4. **Closed history is not documentation.** `*/archived/` is out of the reading path by design;
   open it only for provenance questions.
