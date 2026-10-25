# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-10-25 · **Owner:** Lars · **Module count:** 29 (`go.work`)

## The active thread

**Execute the CI-rescue + Lane-D completion plan (Wave R done, Wave D next).**
Plan of record:
[`docs/planning/2026-10-25_08-05_SUPERB-ci-rescue-s32-proof-and-full-micro-breakdown-pareto-plan.md`](docs/planning/2026-10-25_08-05_SUPERB-ci-rescue-s32-proof-and-full-micro-breakdown-pareto-plan.md)
(the 06-20 file keeps the per-milestone scope text for D01–D15).

**Done when:** CI green on a pushed HEAD; the four ratchets (tail, corpus,
branching-flow, domain counts) green in the same run; master-plan Appendix D
reconciled; every TODO_LIST row executed or carrying a dated verdict.

**Progress (2026-10-25):** Wave R executed — the four CI-red causes fixed at the
source (root lint 44→0 via errorfamily conversion + openapi static sentinels;
root coverage 88.1→91.8; branching-flow re-pinned 168→179 with a dated ledger
row; domain counts already green), the S32 proof tests written (and they caught
a REAL gap: `RecordCredentialUse` was never registered in systemadapter's
DomainConfig — fixed, usermgmt v4.16.0 carries the export, strict train 0/0/0),
S32 receipts landed. Remaining: R04 full battery → R05 push + CI watch → R06
receipt, then Wave D (D01 receipt, D04 golangci verdict, D02 docs/doctrine,
D03 tooling, D05–D16).

## The next thread (start immediately after)

**Wave D, in the plan's order: D01 (delivery receipt: Appendix D + CHANGELOG
receipts S35/S41/S44 + S27 TODO harvest) → D04 (golangci alignment verdict) →
D02 (docs & doctrine) → D03 (tooling) → D05–D09 (code-fidelity wave) →
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
