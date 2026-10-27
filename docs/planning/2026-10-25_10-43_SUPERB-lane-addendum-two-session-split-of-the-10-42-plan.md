# Lane addendum to the 10-42 plan: the two-session execution split

**Created:** 2026-10-25 10:43 CET (finalized 13:04) · **Status:** ADDENDUM, not a competing plan.
**Plan of record:** [`2026-10-25_10-42_SUPERB-wave-d-completion-code-fidelity-to-closure-pareto-plan.md`](2026-10-25_10-42_SUPERB-wave-d-completion-code-fidelity-to-closure-pareto-plan.md) — first in time (10:42 vs 10:43), committed (`8c51ec59`), carries the 08-05 supersession banner, and owns the full §7 coverage table. This file supersedes NOTHING.
**What this file amends:** (1) a deterministic lane assignment for the two live sessions over 10-42's W01–W17, so no milestone is executed twice; (2) two TODO row-groups 10-42's §7 table does not route (docs-truth quartet, sync harvested tails) — its "0 orphans" claim does not hold against the 10:36-harvest rows; (3) dated state receipts for the red-CI class that predates 10-42.
**What this file does NOT do:** duplicate 10-42's micro-breakdown. Their §4 table (w01.1–w17.6) is authoritative for every W-milestone; my earlier 224-line v3 draft is withdrawn into this file — only row-groups with NO 10-42 breakdown carry micro-tasks here (§2).

---

## 1. Lane assignment over W01–W17 (binding for both sessions)

File-set boundary drives the split, not preference: the sibling lane is the module trees (code fidelity); this session's lane is tooling, decisions, and docs — zero overlap in write sets.

| 10-42 ID | Content | Lane | Write-set boundary |
| --- | --- | --- | --- |
| W01–W07 | D05–D09 + S36/S39 code fidelity (S37, S38, S42, S46+S49, S58) | **sibling** | `usermgmt/`, `identity-model/`, `systemadapter/`, `adminui/`, `loginpage/`, `transport/`, `health/`, `auditlog/`, `datastar/`, `oauth2/`, `totp/` |
| W08 | D02 receipt verification (this session's D02 work) | **sibling** (verify-only) | read-only + gate runs |
| **W09** | **D10 (S50+S51): `check-upstream-treadmill.sh` + `wave-tag.sh` + fixture self-tests + flake/check-modules/CI wiring** | **me** | `scripts/checks/`, `scripts/selftests/`, `scripts/testdata/`, `flake.nix`, `.github/workflows/ci.yml`, `AGENTS.md` row |
| W10, W11 | D11 loginpage E2E + D12 file splits | **sibling** (W02→W10 chain; splits touch their module trees) | `loginpage/`, `e2e/`, `setup/`, `usermgmt/`, `systemadapter/`, `dashboardui/` |
| W12 | Checkpoint: battery + push + CI watch | **shared** — first-ready pushes; `git log --stat` reconciliation before every push (10-42 guardrail 2) | — |
| **W13** | **D13 owner packet (S55) + S56 watch table** | **me** | `docs/drafts/`, `TODO_LIST.md`, `ROADMAP.md` |
| W14, W15 | D14a/b backlog (dashboardui tails, root-declutter, skill gates, canaries, upstream drafts) | **sibling** (10-42's w14.x/w15.x are the breakdown; module + scripts overlap with their W01–W07 trees) | per w-rows |
| **W16** | **D15 BuildFlow nix-fmt filing + bench window** | **me** | upstream issue + `docs/` receipts |
| W17 | D16 parked-row verdicts + final battery/receipt + CURRENT_FOCUS rewrite | **verdicts: me** (D16 was this session's lane) · **final receipt + focus: the last-standing session** (W17 gates on W13–W16, so normally me) | `TODO_LIST.md`, `docs/status/`, `CURRENT_FOCUS.md` |

Shared-file discipline (both lanes write these): `TODO_LIST.md`, `AGENTS.md`, `CHANGELOG.md`, `docs/status/` — edit → `git add <explicit paths>` → commit immediately; review `git diff --cached` for the other session's hunks before every commit; never `git add -A` on this tree.

## 2. Unrouted row-groups → this session's lane (no 10-42 breakdown exists)

**Docs-truth quartet** (TODO_LIST P2 row, master-plan S22/S24/S25/S26, harvested 2026-10-25):

| ID | Task | Min | V |
| --- | --- | ---: | --- |
| q.1 | S22: `VERSIONING.md` claim-by-claim vs code | 12 | verdict list |
| q.2 | S22: `docs/DOMAIN_LANGUAGE.md` same | 12 | verdict list |
| q.3 | S24: correct co-change measurement command + one run | 12 | number + cmd |
| q.4 | S25: modularization PROPOSAL cite-or-supersede | 10 | link |
| q.5 | S26: two-repo memo (or explicit no-verdict memo) + ROADMAP link | 12 | memo |

**Sync harvested tails** (TODO_LIST P1 pointer row, 2026-10-24 tails):

| ID | Task | Min | V |
| --- | --- | ---: | --- |
| s.1 | e2e `DB_VERSION` 1→2 IndexedDB migration spec | 12 | spec green |
| s.2 | node-level worker-boot/DB-upgrade smoke | 12 | smoke green |
| s.3 | cursor monotonicity guard + test | 12 | test green |
| s.4 | `Family.IsZero()` adoption + guide batch-cost/timeout + CSP/nonce notes | 12 | suites + links |

## 3. State receipts (dated, 2026-10-25)

1. **The red-master class 10-42 could not see:** CI `40181238507` (pushed `65665920`) and `40181909642` (pushed `8c51ec59`) both failed `checks` on the install-git-hooks fixture: `new_scratch()` did not copy the new `check-templ-codegen-staged.sh` into the scratch repo, so T6/T12's installed gates hit "No such file or directory". Root cause confirmed by local reproduction of the fixture shape; **healed by the sibling's `d19249bb`** ("fix: ship templ-codegen gate into hook-installer fixture") — unpushed at the time of this addendum. This addendum's commit push carries it; verify terminal green before starting W09.
2. **10-42's "State at planning" line was stale at birth** (wrote "tree green on pushed a5183fff" while `40181238507` was already red) — harmless to its milestone content; corrected here for the record.
3. **Owner-verdict defaults applied unilaterally in 10-42 §6** (parallel-continue, fixed-expiry lockout, S36/S39 in scope) — consistent with the standing continue-rule and the 10-36 §g defaults; surfaced for the owner in the W13 packet regardless.

## 4. This session's execution order

`P01` (this commit's push + CI watch to green) → **W09** (treadmill + wave-tag, their w09.1–w09.5) → **W13** (owner packet, w13.1–w13.5) → **W16** (BuildFlow filing, w16.1–w16.3) → **W17 verdicts** (w17.1–w17.2) → §2 quartet → §2 sync tails → final battery/receipt (W17 tail, last-standing). Battery + push checkpoint after every ~2 milestones; verdicts from unpiped rc only.

**Done when:** every W-milestone above is closed by its owning lane, the two §2 row-groups are executed or verdicted, exactly one plan of record (10-42) and one addendum (this file) remain, and the final receipt supersedes 10-37 per W17.
