# Status: Green tree — Wave R closed, D04 executed, CI green on pushed HEAD

**Written:** 2026-10-25 10:15 CET · **Supersedes:**
[`2026-10-25_09-45_wave-r-rescue-executed_status.md`](archived/2026-10-25_09-45_wave-r-rescue-executed_status.md)
(its §g.1 golangci question is ANSWERED and executed; §g.2/§g.3 carried here).
**Plan of record:** [`docs/planning/2026-10-25_08-05_SUPERB-ci-rescue-s32-proof-and-full-micro-breakdown-pareto-plan.md`](../planning/2026-10-25_08-05_SUPERB-ci-rescue-s32-proof-and-full-micro-breakdown-pareto-plan.md)
**Tree:** clean, master == origin/master at `a5183fff` · **CI: run `40179492715` ALL JOBS SUCCESS** (lint (.) under golangci 2.15.0, test incl. root coverage, checks, module-architecture; `architecture-gate` is PR-only by design).

## §a FULLY DONE this session (committed and pushed)

| Work | Receipt |
| --- | --- |
| **D04 golangci alignment executed (the 09-45 §g.1 decision)** — direction: UP. The 2.15-only wrapcheck findings were measured at **189 across 12 modules** (not 13 in one: my first piped `tail -30` hid 176 of them, see §d.1) and every one is the repo-sanctioned `go-error-family` wrapping layer (`New*`/`Wrap*` constructors + `.WithCause`/`.WithContext`/`.WithContextAny` builders — mandatory since stdlib `errors.New`/`fmt.Errorf` are banned). All 15 wrapcheck-enabled `.golangci.yml`s now carry `wrapcheck.ignore-package-globs` for `go-error-family` (+ `go-cqrs-lite/event`) plus the three builder sigs; CI action pin `v2.14.0 → v2.15.0`; AGENTS lint row, root CHANGELOG Fixed entry, TODO row closed; 3 `//nolint:wrapcheck` directives that became unused were removed | `bcc64baa`; `nix run .#lint` rc=0 on 15 modules; CI `lint (.)` + all 11 matrix lints green at 2.15.0 |
| **branching-flow second re-pin (179 → 179)** — post-v4.16.0 drift was a pure transpose: the `DecideRecordCredentialUse` export added 9 lines above the three frozen `system_exports.go` swap-hazards (:56/:62/:91 → :65/:71/:100). Dated ledger row + re-pinned SARIF in ONE commit; corpus baseline bumped in the same commit after the ratchet caught the row's lines | `a5183fff`; gate +0/−0 |
| **R04 full battery at push HEAD** — `.#test` race all modules rc=0 · `.#lint` rc=0 (2.15.0) · `.#coverage-gate` PASSED · `.#check-modules --report` **55/55** rc=0 | outputs captured 10:0x CET |
| **R05 delivery** — 14-commit range pushed through UNskipped strict pre-push gates (867 requires 0/0/0, drift clean, 866 tags resolve); CI watched to terminal green | `cde4e417..a5183fff`; run `40179492715` |
| **R05.3 tag-hygiene evidence table** (see §g.2) — assembled; recommended verdict leave-and-document | below |

## §b PARTIALLY DONE

- **Wave D**: D04 ✅ (§a). D01–D03, D05–D16 not started (§c).

## §c NOT STARTED

D01 (delivery receipt: Appendix D, CHANGELOG S35/S41/S44, S27 harvest), D02 (docs/doctrine S45/S52/S53/S54), D03 (tooling S47/S48), D05–D09 (code fidelity S37/S38/S42/S46+S49/S58), D10 (S50/S51 scripts), D11 (S57 loginpage E2E), D12 (file splits), D13 (owner packet), D14a/b (harvested backlog), D15 (BuildFlow filing + bench window), D16 (parked-row audit).

## §d TOTALLY FUCKED UP (honest)

1. **I repeated the predecessor's §d.1 pipe mistake within an hour of reading it.** The first `nix run .#lint | tail -30` showed "13 issues: wrapcheck: 13" — webauthn's own module summary — and I briefly concluded only webauthn was affected. The full output held **189 findings across 12 modules**. The reconciliation against the 09-45 report's "201" is what exposed it. Verdicts must come from unpiped rc, never a tail window.
2. **Daemon race, again**: 17 of D04's 21 files were auto-committed (`c5fa1bee`) while the verification lint ran; my descriptive message landed on the 4-file remainder. Recovered by soft-reset squash (`bcc64baa`) — content was never at risk, but the first commit message lied about its own diff.
3. **Corpus-baseline miss on the ledger commit**: the branching-flow ledger row added docs lines and `check-modules` went red on `corpus-budget-ratchet` one run later. Amended the baseline into the same commit. The ratchet worked as designed; my sequencing didn't.

## §e IMPROVE

- Run gates unpiped (or capture `PIPESTATUS[0]`); a truncated tail is not a verdict — twice proven now.
- Any commit that adds lines under `docs/` (ledger rows included) must bump `docs/CORPUS_BASELINE` in the same commit — run the corpus check BEFORE committing, not one battery later.
- Lint-stability note for 2.15.0: `exhaustruct` is deprecated in favor of `exhaustruct_v5` (warning-only today) and golangci warns about `//nolint:erraudit` directives it doesn't know (they belong to the standalone erraudit tool; inert). Both are warning-class; a rename/ignore pass is a small future hygiene item, NOT a gate change.

## §f NEXT (in execution order)

1. **D01** delivery receipt: master-plan Appendix D reconciliation, CHANGELOG receipts S35/S41/S44, S27 TODO harvest.
2. **D02** docs & doctrine: S45 EventHandler docs, S52 train-preflight mandate, S53 credential memo, S54 v5 migration skeleton.
3. **D03** tooling: `bump-dep --lint` leg, templ fmt→generate chain.
4. **D05–D09** code fidelity waves, then D10–D16 per the plan's §4.2 micro-tasks.

## §g Owner gates (recommendations recorded, decisions not executed)

1. **Tag hygiene (carried, evidence complete).** CI triggers only on `master`/`v4` branch pushes — tags never get their own runs, so "tag CI" = the master run of the tagged commit:

   | Tag | Commit | Location | CI story | Importer risk | Recommended verdict |
   | --- | --- | --- | --- | --- | --- |
   | `identity-model/v4.14.0` | `9a556294` | on origin/master | era of red runs (`40167243876`/`40171760161`) — module code compiled; its own test-suite gap was the red | none (tests not compiled by consumers) | **leave-and-document** |
   | `usermgmt/v4.15.0` | `819be075` | on origin/master | same era | none | **leave-and-document** |
   | `usermgmt/webauthn/v4.13.0` | `f165854f` | on origin/master | same era (the webauthn test call-sites were the non-compiling suite) | none | **leave-and-document** |
   | `usermgmt/v4.16.0` | `651fa40b` | on master since this push | verify-tag.sh guards at tag time; superset tree green in `40179492715` | none | healthy — no action |

   All three red-era tags are superseded by green content on the same major line (v4.16.0 carries the fix); retraction would burn resolvable versions to no importer benefit. Owner may override.
2. **Minor-version ratification (carried from 09-45 §g.3):** the `FinishLogin` → `([]byte, error)` signature change shipped as minor (one official implementor; CHANGELOG notes technically-breaking) and the auth-audit journal riding usermgmt v4.15.0. Recommend: ratify as-documented (pre-v5 changelog prominence), flag both in the v5 migration plan (D02/S54).
3. **Future hygiene (small, not gates):** `exhaustruct` → `exhaustruct_v5` rename when adopting it deliberately; consider teaching the flake lint app to fail on `level=warning` unknown-linter drift if it ever becomes load-bearing.
