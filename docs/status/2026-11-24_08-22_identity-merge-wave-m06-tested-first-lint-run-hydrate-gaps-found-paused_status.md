# Status Report: Identity merge wave (issues #46, #47, #45, #43): M06 tested, first lint run of the wave, two hydrate gaps found, paused

**Reported at:** 2026-11-24 08:22 CET (session window about 07:58 to 08:22 CET).
**Trigger:** owner told me to finish the whole TODO list without stopping, then interrupted for this report. The session is paused here.
**Tree at report time:** `c517b62e` (master, 38 commits ahead of `origin/master`, working tree clean). Nothing was pushed, tagged, closed or commented on GitHub by this session.
**Plan of record:** `docs/planning/2026-11-24_06-30_SUPERB-identity-merge-open-issue-wave-pareto-plan.md`.
**Predecessor report:** `docs/status/2026-11-24_07-36_identity-merge-wave-m04-tested-m06-landed-untested-tree-repaired-paused_status.md`.

## Scope: what this report verified, and what it did not

VERIFIED by command in this session, on the tree above (every Go command with `GOEXPERIMENT=jsonv2`):

| Check                                                                                                              | Command                                                                                                                                          | Result                                                                                                                                                   |
| ------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Formatting of every Go file in `usermgmt/` and `identity-model/`                                                   | `gofmt -l identity-model/*.go usermgmt/*.go`                                                                                                     | empty, re-checked after the last edit                                                                                                                    |
| usermgmt production build                                                                                          | `go build ./usermgmt/`                                                                                                                           | ok, re-checked after the last edit                                                                                                                       |
| usermgmt vet, test binaries included                                                                               | `go vet ./usermgmt/`                                                                                                                             | ok, re-checked after the last edit                                                                                                                       |
| M06 read-model tests                                                                                               | `go test ./usermgmt/ -run 'TestUserReadModel_Merge\|TestReadModels_HandleEveryDeclaredEventType' -count=1 -v`                                   | 17 PASS lines (11 test functions, 6 subtests), 0 failures. Two of them were FAIL before the two production changes of this session (section "decisions") |
| M04 decider tests, re-run after the lint refactor                                                                  | `go test ./usermgmt/ -run 'MergeUsers\|MergeUserInto\|TransferBot\|UnmergedLosers' -count=1 -v`                                                 | 33 PASS lines, 0 failures (same count as before the refactor)                                                                                            |
| Full usermgmt suite, re-run after the last edit                                                                    | `go test ./usermgmt/ -count=1`                                                                                                                   | one FAIL, `TestSnapshotShape_SQLStoreDropsStampToday` (not this wave, section d6)                                                                        |
| identity-model hermetic build, vet, tests                                                                          | `(cd identity-model && GOWORK=off go build ./... && GOWORK=off go vet ./... && GOWORK=off go test ./... -count=1)`                              | ok, with the three new tests                                                                                                                             |
| golangci-lint for the usermgmt module, through BuildFlow                                                            | `buildflow -s "golangci-lint [usermgmt]" --format finding`                                                                                        | run 1 (08:16): 14 warnings. Run 3 (08:19, after fixes): 5 warnings (section b)                                                                           |

NOT verified, and why:

- **cqrs-lint did not run.** The plan's battery lists it; I only ran golangci-lint.
- **No test ran with `-race`.** The repo's own test command uses it.
- **No test dispatches a merge command through a service.** The three registrations are still proven by count only (bijection output of the earlier session, not re-run today).
- **M07 is research only.** No SQL read-model line was changed. The two gaps in section "Design findings for M07" come from reading `sql_readmodel.go`, `sql_hydrate.go`, `view_cache.go`, the first 200 lines of `sql_readmodel_extra.go` and the two login lookups. I did not read the view store itself (`storage/v4/view/store.go`), so the row order `Scan` returns is unknown.
- The bijection and domain-count gates were not re-run; nothing that feeds them changed.
- `adminui`, `setup`, `systemadapter`, `dashboardui`, `integration_test` were not built or tested today. The sibling session is editing `adminui` while I work.
- No coverage, no `check-modules`, no e2e, no docs gate.
- Only two of the four skills were re-read on resume (`cqrs-htmx`, `buildflow`). The `go-cqrs-lite` and `how-to-golang` skills were read in the 07:18 session, not in this one.

## What this session did, in order

1. 07:58. Resumed on a clean tree at `8d40da00`, 34 ahead. Read the plan in full and two skills.
2. Read the in-memory user, tenant, bot and membership read models and `es_constants.go`. Found that the tenant list already declares `TenantRenamed`, `TenantMerged`, `TenantMergedInto` and the bot list declares `BotTransferred`, while neither read model has a handler for them; `handleEvent` ignores a type without a handler without any signal.
3. Wrote `usermgmt/es_readmodel_merge_test.go` (619 lines): ten merge tests and one guard test that lists every event type a read model subscribes to without a handler.
4. Two tests were red by design: a pointer cycle did not answer not-found, and a re-delivered registration brought a merged row back. Changed production code for both (decisions D17 and D18 below). All 17 PASS lines green.
5. Added three identity-model tests: `Clone` copies the merge fields, a live user serializes neither merge field, both fields survive a JSON round trip.
6. Ran the full usermgmt suite: one red, the snapshot tripwire (foreign).
7. Ran golangci-lint for the first time in this wave: 14 warnings. Fixed nine, five remain.
8. Read the SQL user read model, the hydrate code, the view cache and the login lookups for M07. Found two gaps the plan does not cover.
9. 08:20. Owner interrupt; this report.

## a) FULLY DONE

| Milestone | What is done | Evidence |
| --------- | ------------ | -------- |
| M01 to M04 | Unchanged since the 07:36 report | see there |
| M06 | Pointer map `mergedInto`; `handleUserMerged`, `handleUserMergedInto`; `repoint` and ownership-scan `evict`; `resolve` walks the chain and answers zero on a cycle; every finder follows the pointer and refuses a nil row; a retired stream stays retired. Micro-tasks 34 to 38 | 17 PASS lines; `es_readmodel_merge_test.go` |
| 39 | `identitymodel.User.MergedInto` is omitted for a live user; `MergedUserIDs` is cloned and round-trips | identity-model hermetic battery ok |
| Lint debt in the wave's files | 14 warnings down to 5 | BuildFlow run 3 |

## b) PARTIALLY DONE

| Milestone | What exists | What is missing |
| --------- | ----------- | --------------- |
| M05 | Registrations for `MergeUsers`, `MergeUserInto`, `TransferBot`; three postures | The bijection gate stays red until M11 registers the three tenant commands. A dispatch through a service is untested. `RegisterCommands` now carries a `maintidx` warning (micro-task 31 or 32 made it worse or created it; not checked against `origin/master`) |
| M07 | Micro-task 39 only | Micro-tasks 40 to 43; the alias rebuild on hydrate; the `tombstoned` filter in `FindByEmailSQL` |
| M08 | Micro-task 47 (audit labels) as code | Untested. Micro-tasks 44 to 46 and 48 |
| M18 | Knob and the sibling's red test, both from earlier sessions | Micro-tasks 105 to 109 |
| Plan file | Rows M01 to M05 | No M06 row; D15 to D18 not recorded; no expected-reds note in section 9; the M18 row still says the test is mine to write; micro-task 107 names `setup/CHANGELOG.md`, which does not exist |
| Lint | 9 of 14 warnings fixed | `errcheck` at `es_decide_merge_test.go:841` (one ignored `*errorfamily.Error`, fix is one line); `maintidx` at `es_dispatch.go:16`; three `unused` warnings for `cmdRenameTenant`, `cmdMergeTenants`, `cmdMergeTenantInto` (expected until M11) |

Micro-tasks: **38 of 142 have code** (numbers 1 to 36, 39, 47). Of those, **36 are verified by their own tests** (1 to 30, 34 to 39). **Three are compile-only** (31, 32, 47). One is half verified (33: postures tested, bijection red by plan). The 07:36 report counted 37; its own list has 38 entries.

## c) NOT STARTED

M09 to M17 and M19 to M25: sixteen milestones, 104 micro-tasks (40 to 46, 48 to 142).

## d) TOTALLY FUCKED UP (or nearly)

**d1. One daemon commit carries a test file that does not compile.** `3e9ada5e` (08:10:33) holds the first version of `es_readmodel_merge_test.go` with two type errors: a `[]UserID` passed where the helper declared `[]*User`, and a `ulid.ULID` value used as a string. `ab3bc960` (08:13:27) fixed it. Production code compiled throughout, because `go build` skips test files. Second occurrence of this class in the wave (`41158979` was the first). Nothing is pushed.

**d2. Lint had not run since M04.** Eleven of the fourteen warnings were in files this wave wrote or extended: `es_decide_merge.go` (complexity 14 against a limit of 12), `es_decide_merge_test.go` (complexity 35 in one test, three uses of a deprecated method, an ignored error, an unnecessary conversion), `es_command_id_test.go` (line length), `es_readmodel_merge_test.go` (line length), `es_constants.go` (three unused variables). The 07:36 report said no linter had run; the debt was larger than its "one exhaustruct" line suggested.

**d3. Two tool mistakes on one file, caught by luck.** I assembled the test file through `write`, a shell script and the `edit` tool. One `write` left a call to a helper that did not exist; the shell step completed the file in the same minute, so no commit carries it. One `edit` then refused with "modified since last read" and did not apply; I noticed only because I grepped the file before the next step.

**d4. A wrong expectation in a test I wrote.** I asserted not-found for the external subject of a retired stream. The designed answer is the survivor (invariant 5). The test run showed it; corrected before commit.

**d5. A type assumed, not checked.** `StreamID.Get()` returns a string; `UserID.Get()` returns a `ulid.ULID`. I wrote for the first and hit the second.

**d6. Foreign reds, unchanged and untouched.**

| Module          | Test                                                 | State           | Origin |
| --------------- | ---------------------------------------------------- | --------------- | ------ |
| `usermgmt`      | `TestSnapshotShape_SQLStoreDropsStampToday`          | red             | its own message prescribes the fix: flip the test to assert preservation, close TODO row 90 (M23) |
| `setup`         | `TestRunWithAppkit_ServerTimingHeaderDefaultAndOptOut` | red, 2 of 3     | sibling session, test-first for issue #43 |
| `systemadapter` | `TestArchitectureCatalog_EventParityWithUsermgmt`    | red (not re-run today) | the six new events are missing from its derivation; cleared by M16 |

**d7. The daemon mixes both sessions' files into one commit.** `c517b62e` holds four files of mine and five `adminui` files of the sibling session. For micro-task 140 ("stage only this wave's files") history can no longer separate the two waves; only the push scope can be decided.

## What I forgot, what I could have done better, what I can still improve

- **Run lint per milestone, not per wave.** The plan's section 9 lists the lint battery once at the end. One `buildflow -s "golangci-lint [usermgmt]"` after M04 would have shown nine findings a day earlier.
- **One editing path per file.** Pick `write` or `edit`, never both with a shell script in between.
- **Read the return type before writing a comparison.** The existing tests compare `UserID` values as arrays for a reason.
- **Write the expectation from the invariant, then the test.** d4 came from writing the assertion before re-reading invariant 5.
- **`-race` at least once per milestone.** The repo's test command does; my loop did not.
- **Re-read every skill that a milestone needs.** M07 and M09 sit on repository, hydrate and checkpoint semantics that the `go-cqrs-lite` skill documents.
- **Count from the list, not from memory.** 38, not 37.

## Decisions made this session (none recorded in the plan yet)

| #   | Decision | Why |
| --- | -------- | --- |
| D15 | `identitymodel.User` carries `MergedUserIDs` next to `MergedInto` (from the 07:18 session) | the admin panel and the duplicates report need "which identities did this one absorb" without replaying the journal |
| D16 | A merged-away row leaves the in-memory `users` map; only `mergedInto` keeps its trace; deleting a user evicts every index entry that resolves to it (from the 07:18 session) | a merged row must not count, list or answer as live; a survivor owns the aliases it inherited |
| D17 | `resolve` answers with the zero stream ID when the walk has used every pointer and is still on a merged stream | a cycle has no live member; handing out a member would return a row the handlers would never have kept. The deciders cannot build a cycle (D13), so this is defence against a corrupt map |
| D18 | A retired stream stays retired: `readModelCore.drop` lets a read model refuse an event before dispatch; the user model refuses everything on a stream in `mergedInto` except `UserMergedInto`, which re-applies as a no-op | after a hydrate the drain can replay events that precede the merge on a stream whose pointer is already known; `handleUserRegistered` builds a new row and brought the merged identity back (the test was red before the change). The tenant read model can reuse the hook in M11 |

## Design findings for M07 (read-only, no code changed)

| #  | Finding | Covered by the plan? |
| -- | ------- | -------------------- |
| G1 | `syncToSQL` calls `FindByID` after the in-memory handler. Since M06 that lookup follows the pointer, so on `UserMergedInto` it would write the survivor's row under the loser's key. | Yes, micro-task 41 (skip) |
| G2 | The tombstone row must be built from the loser's row before the in-memory handler removes it. A loser without a row in memory still needs a minimal tombstone (ID and pointer), or the pointer is lost on the next restart. | Partly, micro-task 40 |
| G3 | `Hydrate` fills `users`, `emails` and `externalAccounts` and never touches `mergedInto`. After a restart `FindByID(loser)` answers not-found instead of the survivor. | Yes, micro-task 42. The tombstone's `Data` JSON carries the pointer through the existing `userAlias` shape; the new identity-model round-trip test shows the field survives JSON |
| G4 | **Inherited email aliases do not survive a restart.** In memory the loser's email entry is re-pointed to the survivor; `Hydrate` rebuilds `emails` from live rows only, and the survivor's row holds one email. `BeginLogin` and `matchVerifiedEmail` resolve by email, so a loser's distinct email that logged in as the survivor before the restart answers "user not found" after it. The live data shares one email across all nine rows and does not hit this; any merge of two different emails does. | **No.** No micro-task names it |
| G5 | **A tombstone's external accounts must not be re-indexed unconditionally.** In memory, unlinking an inherited account deletes the entry. A hydrate that re-indexes the tombstone's accounts would make a subject the survivor dropped resolve to the survivor again, and `matchOrCreateUser` returns whoever `FindByExternalAccount` answers. Gate: re-index only accounts the survivor's live row still holds. | **No.** No micro-task names it |
| G6 | `Hydrate`'s keep-first relies on `Scan` returning rows in registration order. The partial-merge case (the owner of a shared email or subject is merged away, younger duplicates stay live) depends on it for both indexes. | No. I did not read the view store; verify at M07 |
| G7 | `FindByEmailSQL` has no `tombstoned` filter; `CountSQL` has one. Callers not surveyed. | Only as a note in my own list |
| G8 | Writing the tombstone through `m.cache.Set` keeps a cached loser view truthful. `Hydrate` scans the store, not the cache. | Assumed from reading; untested |

My plan for G4 and G5, unless told otherwise: add them to M07 as two micro-tasks (about 45 minutes with tests). Without them a restart breaks what the merge just fixed.

## Files touched this session (all auto-committed by the daemon)

| File | Change |
| ---- | ------ |
| `usermgmt/es_readmodel_merge_test.go` | NEW, 619 lines: `userStream` event builder, ten merge tests, `unhandledEventTypes` and the guard test |
| `usermgmt/es_readmodel.go` | `drop` wiring in `NewUserReadModel`; `dropOnRetiredStream`; `resolve` answers zero on a cycle; comments |
| `usermgmt/es_readmodel_base.go` | `drop` hook on `readModelCore`, consulted under the write lock |
| `usermgmt/es_decide_merge.go` | `requireLiveSurvivor` and `requireOtherContributors` extracted from `decideMergeUsers` |
| `usermgmt/es_decide_merge_test.go` | adoption test split into eight `require` helpers; `StreamID` instead of the deprecated `AggregateID`; one conversion removed; line lengths |
| `usermgmt/es_command_id_test.go` | line lengths only |
| `identity-model/model_test.go` | three tests and the `bytes` import |
| this report | |

Daemon commits carrying them: `3e9ada5e` (08:10, the non-compiling first version), `ab3bc960` (08:13), `ba557eb8` (08:17, one line of mine beside two sibling files), `c517b62e` (08:20).

## e) WHAT WE SHOULD IMPROVE

1. **Lint after every milestone**, written into the plan's battery as a per-milestone line, not only into the final gate.
2. **The plan's M07 needs three more lines** (G4, G5, G6) and micro-task 42 should say "pointer map and inherited aliases".
3. **A raw lookup on the read model for M09.** `FindByUserID` now follows the pointer, so the service cannot see that a loser is already merged; it needs a lookup that reports the pointer without following it. Design the shape before micro-task 50.
4. **The declared-versus-handled guard should cover all four in-memory read models.** Tenant and bot are known-red until their handlers land; add the rows in M08 and M11 so the guard never stays red.
5. **Record decisions in the plan on the day they are made.** D15 and D18 wait since 07:18 and 08:12.
6. **One `-race` run per milestone.**
7. **Honest counting:** list the micro-tasks, then count the list.
8. **The two sessions share one tree and one daemon.** A file-level owner list per wave, kept in each plan, would let micro-task 140 separate them at push time.

## f) Next: 50 work packages (the 104 open micro-tasks plus the gaps found today)

| #  | Package | Micro-task |
| -- | ------- | ---------- |
| 1  | Fix the ignored `*errorfamily.Error` at `es_decide_merge_test.go:841`; re-lint | 30 |
| 2  | `RegisterCommands` in `es_dispatch.go`: check against `origin/master` whether the `maintidx` warning is new, then extract the two merge registrations | 31, 32 |
| 3  | Plan file: rows D15 to D18, the M06 row, the expected-reds note in section 9, the M18 correction, the micro-task 107 path, the count fix | bookkeeping |
| 4  | Plan file: decisions for G4 to G6 and the two new M07 micro-tasks | before M07 |
| 5  | Read `storage/v4/view/store.go`: `Scan` order, `Set` with `Tombstoned`, `Delete` | G6 |
| 6  | Read the rest of `sql_readmodel_extra.go` (lines 200 to 353) | M07 |
| 7  | Survey the callers of `FindByEmailSQL`, `FindByIDSQL`, `CountSQL` | G7 |
| 8  | SQL `Handle` writes the soft tombstone before the in-memory removal; pointer in `Data` | 40 |
| 9  | `syncToSQL` skips `UserMergedInto` | 41 |
| 10 | `Hydrate` rebuilds `mergedInto` from tombstoned rows | 42 |
| 11 | `Hydrate` rebuilds inherited email aliases | G4 |
| 12 | `Hydrate` re-indexes a tombstone's external accounts only while the survivor holds them | G5 |
| 13 | `FindByEmailSQL` filters `tombstoned = false` | G7 |
| 14 | Tests: SQL live path, tombstone view readable by ID, pointer in `Data` | 43 |
| 15 | Tests: fresh hydrate, pointers, aliases, unlink case | 43, G4, G5 |
| 16 | Tests: `CountSQL` and `FindByEmailSQL` exclude tombstones | 43 |
| 17 | Tests: view cache enabled, the cached loser view becomes the tombstone | G8 |
| 18 | Tests: re-delivered pre-merge events after a hydrate do not resurrect a row, SQL model | D18 |
| 19 | Lint and cqrs-lint for M07; one full usermgmt run with `-race` | M07 |
| 20 | Casbin: `UserMergedInto` removes every role of the user | 44 |
| 21 | Casbin: `TenantMergedInto` clears the domain | 45 |
| 22 | Bot read model handler for `BotTransferred`; bot row in the guard test | 46 |
| 23 | Bot SQL sync and hydrate for `BotTransferred` | 46 |
| 24 | Audit label test for the two user merge events | 47 |
| 25 | Tests for the three projections | 48 |
| 26 | `MergeUsers` input validation | 49 |
| 27 | Raw pointer lookup on the read model; typed rejections for not found, merged, deleted | 50 |
| 28 | Third-party external-account conflict naming both rows | 51 |
| 29 | Build contributions from the read model | 52 |
| 30 | Dispatch the survivor command | 53 |
| 31 | Membership move: survivor add or role union, then loser removal; idempotent on re-run | 54 |
| 32 | Bots: transfer each loser-owned bot | 55 |
| 33 | Sessions: revoke each loser's sessions | 56 |
| 34 | Loser tombstone last | 57 |
| 35 | Test: the issue's verify scenario, then kill switch K1 | 58 |
| 36 | Tests: idempotent re-run, conflict, memberships, bots, sessions | 59 |
| 37 | Tenant deciders: rename, merge, merge-into, create after delete and after merge, wrappers, tests | 60 to 66 |
| 38 | Tenant wiring: registrations, postures, read model with the `drop` hook, SQL, `NormalizeTenantName`, tests; bijection gate green; clears the three `unused` warnings | 67 to 72 |
| 39 | Tenant service: `RenameTenant`, `MergeTenants`, `UniqueTenantNames`, `DuplicateUsers`, `DuplicateTenants`, then kill switch K2 | 73 to 79 |
| 40 | adminui users: merge route, handler, survivor picker, duplicates section, tests (after question 2) | 80 to 85 |
| 41 | adminui tenants: rename and merge routes, forms, duplicates section, create-form message, tests (after question 2) | 86 to 91 |
| 42 | adminui finish: manifest pins, templ regeneration, module tests | 92 to 94 |
| 43 | systemadapter: decoder, schemas, registrations, gate declarations, projections, tests; clears the parity red | 95 to 101 |
| 44 | gfl-usermgmt: events artifact, catalog resources, merge scenarios or a recorded follow-up | 102 to 104 |
| 45 | Issue #43: read what the red setup test assumes against what appkit emits, repair it, README, hold claims | 105 to 109 |
| 46 | Issue #45: `LegalForm` assertions, bill-to aggregate, fold, deciders, tests, README | 110 to 116 |
| 47 | Docs: domain language, count trues, `FEATURES.md`, module READMEs, skill references | 117 to 122 |
| 48 | CHANGELOG entries for six modules; `TODO_LIST.md` harvest | 123, 124 |
| 49 | ADR for identity merge, gotchas 69 and 70, plan execution log | 125 to 127 |
| 50 | Docs ratchets (`TAIL_BASELINE` is 16, 22 reports are live), gates battery, snapshot tripwire flip, e2e, commits, push and CI watch (after questions 1 and 3) | 128 to 142 |

## g) Three questions I cannot answer myself

**Question 1 (asked three times, still open).** Who cuts the release train? May I run `nix run .#wave-tag` for identity-model, usermgmt and the rest? My default until answered: stop at pushed and CI-green and hand over a checklist.

**Question 2 (asked once, still open).** The sibling session is editing `adminui` right now (`c517b62e` carries its `audit.templ`, `handler_audit.go`, `routes.go`, `routes_manifest_test.go`, `users.templ`). M13 to M15 need new routes, new templates and a regenerated `users_templ.go` in that package. How do we share it? My default: new files only, one-line hooks after a fresh read, never regenerate a template I did not write.

**Question 3.** Master is 38 commits ahead of origin and mixes both waves in single commits; two foreign tests are red on the tree. What is the push scope when the wave is green? My default: push master whole only if every gate is green on the whole tree, fix a foreign red only as its own message prescribes, otherwise hold and report.
