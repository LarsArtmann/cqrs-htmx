# Status Report: Identity merge wave (issues #46, #47, #45, #43): M07 code landed without its tests, seven new lint warnings, paused

**Reported at:** 2026-11-24 08:44 CET (session window 08:31 to 08:44 CET).
**Trigger:** owner told me to finish the whole TODO list without stopping, then interrupted for this report. The session is paused here.
**Tree at report time:** `4ebc0943` (master, 45 commits ahead of `origin/master`, working tree clean). Nothing was pushed, tagged, closed or commented on GitHub by this session.
**Plan of record:** `docs/planning/2026-11-24_06-30_SUPERB-identity-merge-open-issue-wave-pareto-plan.md`.
**Predecessor report:** `docs/status/2026-11-24_08-22_identity-merge-wave-m06-tested-first-lint-run-hydrate-gaps-found-paused_status.md`.

## Scope: what this report verified, and what it did not

VERIFIED by command in this session, on the tree above (every Go command with `GOEXPERIMENT=jsonv2`):

| Check                                                                                         | Command                                                                                                                        | Result                                                                                                                                                                                                    |
| --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Formatting of every Go file in `usermgmt/` and `identity-model/`                             | `gofmt -l identity-model/*.go usermgmt/*.go`                                                                                   | empty, 08:42:12                                                                                                                                                                                          |
| usermgmt production build                                                                     | `go build ./usermgmt/`                                                                                                         | ok after every Go edit, last at 08:42:12                                                                                                                                                                 |
| usermgmt vet, test binaries included                                                          | `go vet ./usermgmt/`                                                                                                           | ok after every Go edit, last at 08:42:12                                                                                                                                                                 |
| M04 decider tests, M05 posture tests, M06 read-model tests, after the dispatch refactor      | `go test ./usermgmt/ -run 'TestUserReadModel_Merge\|TestReadModels_HandleEveryDeclaredEventType\|MergeUsers\|MergeUserInto\|TransferBot\|UnmergedLosers\|TestCommandPostures' -count=1 -v` | 53 PASS lines (30 functions and 23 subtests), 0 failures, `ok`, 08:35. Not re-run after the M07 code landed                                                                                         |
| Full usermgmt suite on the M07 code                                                           | `go test ./usermgmt/ -count=1`                                                                                                 | one FAIL, `TestSnapshotShape_SQLStoreDropsStampToday` (foreign, d6), 19.1 s, 08:41. The suite holds no test that drives a merge through the SQL model, so this proves only that nothing else broke |
| Command bijection gate                                                                        | `bash scripts/checks/check-command-bijection.sh`                                                                               | `constants: 27 registrations: 24`, the three tenant commands missing, exit 1 (expected until M11)                                                                                                         |
| golangci-lint for the usermgmt module, through BuildFlow                                      | `buildflow -s "golangci-lint [usermgmt]" --format finding`                                                                     | run 1 (08:32): 4 warnings. Run 2 (08:42, with the M07 code): 10 warnings (section b)                                                                                                                     |

NOT verified, and why:

- **M07 has no test of its own.** Micro-tasks 40, 41, 42, 42a, 42b and 42c exist as production code only. Every claim about tombstones, hydrate and `FindByEmailSQL` below comes from reading the code I wrote, not from a test.
- **No test ran with `-race`.** The repo's own test command uses it.
- **cqrs-lint did not run.** It will judge the new `userIndex` type and the two new exported methods.
- identity-model was not re-run: nothing in that module changed this session.
- The domain-count gate, `check-modules`, coverage and e2e were not run.
- `adminui`, `setup`, `systemadapter`, `dashboardui`, `integration_test` were not built or tested. The sibling session keeps editing `adminui`.
- The three files of M07 reached the tree through a daemon commit at 08:42:18 (`4ebc0943`), before any test for them existed.

## What this session did, in order

1. 08:31. Resumed on a clean tree at `c8f296e8`, 40 ahead. Recreated the todo list. Read four skills (`cqrs-htmx`, `buildflow`, `go-cqrs-lite`, `how-to-golang`), the plan in full and the 08:22 report.
2. 08:32. Fixed the ignored `*errorfamily.Error` at `es_decide_merge_test.go:841`. Lint run 1: 4 warnings left (`maintidx` on `RegisterCommands`, index 19 against a limit of 20; three `unused`).
3. 08:34. Moved the two merge registrations from `RegisterCommands` into `registerMergeCommands` (`es_dispatch.go`, 10 lines more). gofmt, build and vet clean.
4. 08:35. Re-ran the M04, M05 and M06 tests in one command: 53 PASS lines.
5. 08:36. Plan file: decisions D15 to D21, micro-tasks 42a to 42c, a per-milestone lint line in section 9, the expected-reds note in section 8, the M06 row, the M18 correction, the micro-task 107 path. Daemon commit `abaa0bfe`.
6. 08:37. Finished the M07 reading: the view store (`crud.go`, `mapper.go`, `store.go`), the `kv` cache, the callers of the three SQL finders, `StreamID` generation. Answers G6, G7 and G8 and adds G9 (below).
7. 08:39. Wrote the M07 production code in three files. Build and vet clean.
8. 08:41. Full usermgmt suite on that code: only the foreign snapshot red.
9. 08:42. Lint run 2: 10 warnings, seven of them in the M07 code.
10. 08:42:18. The daemon committed the three M07 files as `4ebc0943`, untested.
11. 08:44. Owner interrupt; this report.

## a) FULLY DONE

| Milestone                | What is done                                                                                                                                                                                                                       | Evidence                                                                                     |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| M01 to M04, M06          | Unchanged since the 08:22 report                                                                                                                                                                                                   | see there; re-run green at 08:35                                                             |
| M05 registrations        | `registerMergeCommands` holds the two merge registrations; `RegisterCommands` is back under the maintainability limit                                                                                 | lint run 2 has no `maintidx` finding; bijection gate still shows the three tenant gaps only   |
| Lint debt of the 08:22 report | `errcheck` and `maintidx` are gone                                                                                                                                                                                               | lint run 2                                                                                   |
| Plan bookkeeping         | D15 to D21; micro-tasks 42a to 42c; per-milestone lint line; expected-reds note; M06 row; M18 row; micro-task 107                                                                                                                 | `abaa0bfe`                                                                                   |

## b) PARTIALLY DONE

| Milestone  | What exists                                                                                                                                                                              | What is missing                                                                                                                                                                                                                                                                                                            |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| M05        | Three registrations, three postures                                                                                                                                                     | The bijection gate stays red until M11. Three `unused` warnings stay until M11                                                                                                                                                                                                                                             |
| M07        | Production code for micro-tasks 40, 41, 42, 42a, 42b, 42c: tombstone first, sync skip, retired-stream guard, hydrate replay, `FindByEmailSQL` resolution                                  | **No test.** Seven lint warnings: `nilnil` ×4 (`sql_readmodel.go:219, 331, 339, 351`), `exhaustruct` ×1 (`sql_hydrate.go:206`, `userIndex` built without `merges`), `nolintlint` ×2 (`sql_readmodel.go:211, 261`, two `//nolint:exhaustruct` directives I added that the linter does not need). No `-race`, no cqrs-lint |
| M08        | Micro-task 47 (audit labels) as code                                                                                                                                                    | Untested. Micro-tasks 44 to 46 and 48                                                                                                                                                                                                                                                                                      |
| M18        | Knob and the sibling's red test, both from earlier sessions                                                                                                                             | Micro-tasks 105 to 109                                                                                                                                                                                                                                                                                                     |
| Plan file  | Rows M01 to M06, D1 to D21                                                                                                                                                              | No M07 row. D22 and D23 (below) and the four residual divergences R1 to R4 are not recorded. G9 is not recorded                                                                                                                                                                                                           |

Micro-tasks, counted from the list: the plan now has **145** (142 plus 42a, 42b, 42c). **44 have code**: numbers 1 to 36, 39, 40, 41, 42, 42a, 42b, 42c, 47. Of those, **36 are verified by their own tests** (1 to 30, 34 to 39, unchanged). **Nine are compile-only** (31, 32, 47, 40, 41, 42, 42a, 42b, 42c). One is half verified (33).

## c) NOT STARTED

M09 to M17 and M19 to M25: sixteen milestones, 101 micro-tasks (43 to 46, 48 to 142).

## d) TOTALLY FUCKED UP (or nearly)

**d1. M07 code landed before its test, a third time in the wave.** `4ebc0943` holds the tombstone, hydrate and finder logic with no test that exercises any of it. The earlier two (`41158979`, `3e9ada5e`) did not compile; this one compiles and the suite is green on it, which makes it look more finished than it is. The hydrate replay is the most intricate logic of the wave so far (two passes, a dead-chain rule, a claimant search) and nothing pins it. Nothing is pushed.

**d2. Seven lint warnings in one edit.** The package bans a `nil` error beside a `nil` value (`nilnil`); I used that shape four times for "no live row". I added two `//nolint:exhaustruct` directives the linter did not need, which is itself a warning. I did not look at the rule list before writing.

**d3. The plan now states D19 to D21 as decisions while the behaviour they describe has no test.** They are reversible, but a reader of the plan cannot tell that from the rows.

**d4. A test run that proves less than it looks.** I ran the full suite at 08:41 and it was green except the foreign red. I must not count that as M07 verification; section f lists the tests that are still owed.

**d5. Foreign reds, unchanged and untouched.**

| Module          | Test                                                 | State               | Origin                                                                               |
| --------------- | ---------------------------------------------------- | ------------------- | ------------------------------------------------------------------------------------ |
| `usermgmt`      | `TestSnapshotShape_SQLStoreDropsStampToday`          | red                 | its own message prescribes the fix: flip the test to assert preservation, close TODO row 90 (M23) |
| `setup`         | `TestRunWithAppkit_ServerTimingHeaderDefaultAndOptOut` | red, 2 of 3 (not re-run today) | sibling session, test-first for issue #43                                    |
| `systemadapter` | `TestArchitectureCatalog_EventParityWithUsermgmt`    | red (not re-run)    | the six new events are missing from its derivation; cleared by M16                  |

**d6. The daemon still mixes both sessions' files.** Four of its five commits in this window carry the sibling's `adminui` files; three of those also carry one file of mine each (`cf546d43`, `453bcfdb`, `abaa0bfe`). Only `4ebc0943` is mine alone. Micro-task 140 cannot separate the waves in history.

**d7. 45 commits ahead, nothing pushed.** The three questions of the 08:22 report are still unanswered, so the push scope is still undecided.

## What I forgot, what I could have done better, what I can still improve

- **Write the M07 tests first.** The gap list G1 to G8 of the 08:22 report was precise enough to write failing tests from. I wrote the code instead.
- **Read the package's lint rules before writing:** `nilnil`, `nolintlint` and `exhaustruct` each cost a round trip.
- **Record G9, D22, D23 and R1 to R4 in the plan in the same edit as D19 to D21.** The plan is the only thing that survives a context loss.
- **The M07 test design exists only in this report.** Section f lists it; it should also be a plan row.
- **`-race` and cqrs-lint are still not in my loop.** Third report in a row.
- **The tail budget grows again:** this is the 23rd live report in `docs/status/` against `TAIL_BASELINE` 16. The ratchet is red until M22 re-true it; the three earlier reports of today did the same.

## Decisions made this session

Recorded in the plan (D19 to D21) and not recorded yet (D22, D23):

| #   | Decision                                                                                                                                                                                                                                                                                                                    | Recorded |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| D19 | The SQL tombstone is written before the in-memory removal and carries the pointer inside `Data` (`merged_into`), the retired user's snapshot and the merge time in `updated_at`. A failed write aborts `Handle` before memory changes                                                                                         | yes      |
| D20 | `Hydrate` replays: every live and merged row claims its email and accounts keep-first in key order, then each merge re-points its row's entries to its direct survivor in merge-time order. A merged row's external account moves only while the final live survivor still holds it. With no live survivor, a live row holding the same entry since after the merge takes it over | yes      |
| D21 | `FindByEmailSQL` answers with live rows only: a merged row is replaced by the live view at the end of its chain, once. `FindByIDSQL` answers with the tombstone itself (`Tombstoned` true)                                                                                                                               | yes      |
| D22 | A retired stream owns no row in SQL. `Handle` skips the sync for every event on a retired stream, not only `UserMergedInto` (G9)                                                                                                                                                                                              | **no**   |
| D23 | `FindByEmailSQL` tells rows apart by the user ID inside `Data`, because `UserView` carries no key column. The walk over stored rows is bounded by `sqlMergeChainLimit` (10,000)                                                                                                                                              | **no**   |

Residual divergences of the hydrate replay, accepted for now and not recorded in the plan:

| #   | Case                                                                                                                                                                                                                                         | Before the restart                | After the restart                    |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------- | ------------------------------------ |
| R1  | Key order stands in for registration order. Two users registered in the same millisecond by different processes can tie. Unchanged from before the wave                                                                                       | oldest owns the lookup            | key order decides                    |
| R2  | Survivor deleted; a user **created before** the merge changes its email to the released alias after the deletion. The row carries no email-acquisition time                                                                                  | that user owns the alias          | not found                            |
| R3  | Survivor deleted; a user **created after** the merge adopted the alias while the survivor was still alive                                                                                                                                 | not found (keep-first)            | that user owns the alias             |
| R4  | A consumer dispatches `MergeUserInto` for a loser whose external accounts the survivor never adopted. The gate cannot tell "unlinked after adoption" from "never adopted". The orchestrated `MergeUsers` path always adopts first, so it cannot hit this | subject answers with the survivor | subject not found                    |

## Findings of this session

| #  | Finding                                                                                                                                                                                                                                                                                   | Covered by the plan?                 |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| G6 answered | `Scan` orders rows by key; a user's key is its ULID, so rows arrive in registration order. The keep-first replay stands on that (R1)                                                                                                                                                    | no row; add to the M07 row           |
| G7 answered | `FindByEmailSQL`, `FindByIDSQL` and `CountSQL` have no production caller in this repo; only tests use them. Their consumers are outside the repo, so the new semantics (D21) are a documentation matter for M20                                                                      | docs in M20                          |
| G8 confirmed by reading | `Cache.Set` writes through and stores a copy, so a cached view becomes the tombstone. Still untested                                                                                                                                                                        | test in section f                    |
| G9 (new) | After a hydrate the drain can deliver any pre-merge event on a retired stream. `syncToSQL` calls the pointer-following `FindByID`, so the survivor's row would have been written under the loser's key, as a live row, over the tombstone. The 08:22 list only named `UserMergedInto` (G1) | **no micro-task.** Code is in (D22)  |

## Files touched this session (all auto-committed by the daemon)

| File                                   | Change                                                                                                                                                                                                                                        | Commit      |
| -------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- |
| `usermgmt/es_decide_merge_test.go`     | one line: the ignored error at 841                                                                                                                                                                                                            | `cf546d43`  |
| `usermgmt/es_dispatch.go`              | `registerMergeCommands`, split from `RegisterCommands`                                                                                                                                                                                        | `453bcfdb`  |
| the plan file                          | D15 to D21, micro-tasks 42a to 42c, section 8 and 9 notes, rows M05, M06, M18, task 107                                                                                                                                                        | `abaa0bfe`  |
| `usermgmt/es_readmodel.go`             | `row` (raw row, no pointer), `Retired`, `MergeTarget` (direct pointer, not followed), `isRetired`; `dropOnRetiredStream` through `isRetired`                                                                                                  | `4ebc0943`  |
| `usermgmt/sql_readmodel.go`            | `Handle` (tombstone first, retired guard), `retireToSQL`, `storedUser`, `userViewOf`, `FindByEmailSQL` with resolution, `liveAfterMerge`, `viewUserID`, `sqlMergeChainLimit`                                                                   | `4ebc0943`  |
| `usermgmt/sql_hydrate.go`              | user `Hydrate` through `userIndex`: `add`, `addMerged`, `claimFirst`, `applyMerges`, `settle`, `release`, `emailClaimant`, `accountClaimant`, `liveEnd`, `holdsExternalAccount`                                                                 | `4ebc0943`  |
| this report                            |                                                                                                                                                                                                                                               |             |

`MergeTarget` has no caller yet: M09 needs it to tell a merged identity from a live one. `row` is used by `retireToSQL`.

## e) WHAT WE SHOULD IMPROVE

1. **Test first for every read-model change.** Three times is a pattern, not an accident.
2. **Lint before the first test run of a milestone**, not after the code: the seven warnings would have been visible at 08:40.
3. **Write the test list into the plan before the code.** The hydrate replay has at least eleven distinct cases; a list in the plan survives the session.
4. **Record every finding in the plan the moment it is found.** G9, D22, D23 and R1 to R4 wait since 08:37.
5. **Name "not found" with a typed result, not `nil, nil`.** The `nilnil` rule is right: `storedUser` returning `nil, nil` for "already a tombstone" reads as a bug to the next reader.
6. **A shared file list per wave.** Two sessions, one tree, one daemon: the commits cannot be separated afterwards.
7. **Status reports should not pile up against the tail budget.** Four live reports for one wave; archive the two oldest into one at M22.

## f) Next: 50 work packages (the 101 open micro-tasks plus the gaps found today)

| #  | Package                                                                                                                                                                                                                     | Micro-task       |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| 1  | Fix the seven lint warnings: four `nilnil` (a typed "no live row" answer), one `exhaustruct`, remove two unused `nolint` directives                                                                                          | M07              |
| 2  | Test: a merge through the SQL model: tombstone row by ID, pointer in `Data`, `CountSQL` is 1, survivor's `Data` has `merged_user_ids`                                                                                         | 43               |
| 3  | Test: fresh hydrate restores pointers, aliases and `MergedUserIDs`; every lookup answers as before the restart                                                                                                                | 43, 42, 42a      |
| 4  | Test: unlinking an inherited account on the survivor survives the restart                                                                                                                                                    | 42b              |
| 5  | Test: `FindByEmailSQL` resolves merged rows, never returns a tombstone, returns a survivor once                                                                                                                              | 42c              |
| 6  | Test: view cache enabled, the cached loser view becomes the tombstone                                                                                                                                                        | G8               |
| 7  | Test: pre-merge events delivered again after a hydrate: no row returns in memory, the tombstone is not overwritten                                                                                                           | D18, G9          |
| 8  | Test: deleted survivor releases an email alias and an account to a later claimant across the restart                                                                                                                         | D20              |
| 9  | Test: transitive chain across the restart; pointers name the direct survivor                                                                                                                                                 | 42               |
| 10 | Test: keep-first ownership across the restart in both directions (older loser merged, younger duplicate live, and the reverse)                                                                                                | G6               |
| 11 | Test: a loser without a row leaves a pointer-only tombstone; a survivor that does not exist                                                                                                                                   | G2               |
| 12 | Test: `UserMergedInto` delivered again after a hydrate keeps the first full tombstone (no downgrade to the pointer-only one)                                                                                                 | D19              |
| 13 | Plan file: M07 row, D22, D23, R1 to R4, G9, the count 145                                                                                                                                                                     | bookkeeping      |
| 14 | Lint, cqrs-lint and one full usermgmt run with `-race` for M07                                                                                                                                                               | M07              |
| 15 | Casbin: `UserMergedInto` removes every role of the user                                                                                                                                                                       | 44               |
| 16 | Casbin: `TenantMergedInto` clears the domain                                                                                                                                                                                  | 45               |
| 17 | Bot read model handler for `BotTransferred`; bot row in the guard test                                                                                                                                                        | 46               |
| 18 | Bot SQL sync and hydrate for `BotTransferred`                                                                                                                                                                                 | 46               |
| 19 | Audit label test for the two user merge events                                                                                                                                                                                | 47               |
| 20 | Tests for the three projections                                                                                                                                                                                               | 48               |
| 21 | `MergeUsers` input validation                                                                                                                                                                                                 | 49               |
| 22 | Load survivor and losers through `MergeTarget` and `Retired`; typed rejections for not found, merged, deleted                                                                                                                 | 50               |
| 23 | Third-party external-account conflict naming both rows                                                                                                                                                                        | 51               |
| 24 | Build contributions from the read model                                                                                                                                                                                       | 52               |
| 25 | Dispatch the survivor command                                                                                                                                                                                                 | 53               |
| 26 | Membership move: survivor add or role union, then loser removal; idempotent on a re-run                                                                                                                                       | 54               |
| 27 | Bots: transfer each loser-owned bot                                                                                                                                                                                           | 55               |
| 28 | Sessions: revoke each loser's sessions                                                                                                                                                                                        | 56               |
| 29 | Loser tombstone last                                                                                                                                                                                                          | 57               |
| 30 | Test: the issue's verify scenario, then kill switch K1                                                                                                                                                                        | 58               |
| 31 | Tests: idempotent re-run, conflict, memberships, bots, sessions                                                                                                                                                               | 59               |
| 32 | Tenant deciders: rename, merge, merge-into, create after delete and after merge, wrappers, tests                                                                                                                              | 60 to 66         |
| 33 | Tenant wiring: registrations, postures, read model with the `drop` hook, SQL, `NormalizeTenantName`, tests; bijection gate green; clears the three `unused` warnings                                                          | 67 to 72         |
| 34 | Tenant service: `RenameTenant`, `MergeTenants`, `UniqueTenantNames`, `DuplicateUsers`, `DuplicateTenants`, then kill switch K2                                                                                                 | 73 to 79         |
| 35 | adminui users: merge route, handler, survivor picker, duplicates section, tests (after question 2)                                                                                                                            | 80 to 85         |
| 36 | adminui tenants: rename and merge routes, forms, duplicates section, create-form message, tests (after question 2)                                                                                                            | 86 to 91         |
| 37 | adminui finish: manifest pins, audit labels, templ regeneration, module tests (after question 2)                                                                                                                              | 92 to 94         |
| 38 | systemadapter: decoder, schemas, six registrations, gate declarations, declarative projections, count comments, tests                                                                                                         | 95 to 101        |
| 39 | gfl-usermgmt: regenerate the events artifact and the eventcatalog resources; merge scenarios or a recorded follow-up                                                                                                         | 102 to 104       |
| 40 | Issue #43: make the red `setup` test green, `setup/README.md` row, change entry, true the hold claims, setup tests                                                                                                             | 105 to 109       |
| 41 | Issue #45: `LegalForm` assertions, unknown-value rejection, bill-to aggregate (constants, payloads, commands, state, fold, deciders), tests, README, follow-up row                                                             | 110 to 116       |
| 42 | Docs: domain language, AGENTS and FEATURES counts, module READMEs (incl. the SQL finder semantics D21), skill references, CHANGELOGs, TODO harvest                                                                             | 117 to 124       |
| 43 | ADR for identity merge, gotchas 69 and 70, execution log                                                                                                                                                                      | 125 to 127       |
| 44 | Docs ratchets: measure and re-true `CORPUS_BASELINE` and `TAIL_BASELINE` with the ledger sentences                                                                                                                             | 128 to 130       |
| 45 | Gates battery: domain counts, bijection, re-export policy, module layers, cqrs-lint, route postures, permission surface, coverage, full test, lint                                                                            | 131 to 136       |
| 46 | Flip `TestSnapshotShape_SQLStoreDropsStampToday` as its own message prescribes; close TODO row 90                                                                                                                              | M23              |
| 47 | e2e: admin merge flow, tenant rename flow; battery run or a recorded follow-up                                                                                                                                                 | 137 to 139       |
| 48 | `git status` review, staged commits with detailed messages                                                                                                                                                                    | 140, 141         |
| 49 | Pre-push hook, push, CI watch                                                                                                                                                                                                 | 142              |
| 50 | After the release train: close or comment on the four issues                                                                                                                                                                  | D10, outside the numbers |

## g) Questions

The three questions of the 08:22 report are still open. Each needs your decision; none has an answer in the repo. Until you answer I work with the default named in each.

1. **Release train (asked four times now).** Who cuts the module tags for this wave? May I run `nix run .#wave-tag` myself once every gate is green, or do you cut the train? Default: I stop at pushed and CI-green and hand you a tagging checklist; I tag nothing.
2. **Sharing `adminui` with the sibling session (asked three times).** Its wave changed five `adminui` files again in this window (`handler_audit.go`, `models.go`, `config.go`, `audit_labels.go`, `audit_filters_test.go`). M13 to M15 need new routes there and regenerated `*_templ.go` files, and `nix run .#gen` regenerates every template in the module, including any the sibling has in flight. Two things I cannot decide alone: do I wait for the sibling's wave to finish `adminui` before M13, and who may run the generator meanwhile? Default: new files only, one-line hooks into `routes.go` after a fresh read, and I regenerate only templates I wrote.
3. **Push scope (asked three times).** Master is 45 commits ahead and both waves sit inside the same daemon commits. Three tests are red on the tree: one plan-designed until M16, one foreign in `usermgmt` (M23 flips it), one in `setup` that the sibling wrote for issue #43 and that my M18 would make green. Push master whole, push nothing until every gate is green (K3), or something else? And is making the `setup` test green mine (M18) or the sibling's? Default: push only when every gate is green on the whole tree, treat the `setup` test as mine because issue #43 is in my plan.
