# Status Report: Identity merge wave (issues #46, #47, #45, #43): M04 and M05 code landed, decider tests still open

**Reported at:** 2026-11-24 07:04 CET (session window about 06:40 to 07:04 CET).
**Trigger:** owner asked for the whole plan to be executed without stopping, then paused the run for this report.
**Tree at report time:** `51e708eb` (master, 21 commits ahead of `origin/master`, working tree clean). Nothing was pushed, tagged, closed or commented on GitHub by this session.
**Plan of record:** `docs/planning/2026-11-24_06-30_SUPERB-identity-merge-open-issue-wave-pareto-plan.md`.
**Predecessor report:** `docs/status/2026-11-24_06-20_open-issue-wave-46-47-45-43-midwave-paused-self-review_status.md`.

## Scope: what this report verified, and what it did not

VERIFIED by command in this session, on the tree above:

| Check                                                                                                            | Command (with `GOEXPERIMENT=jsonv2`)                                                                                                      | Result                                                                                                                                      |
| ---------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Merge symbols exist in identity-model and usermgmt (M01 to M03 landed)                                           | `rg` over `identity-model/` and `usermgmt/`                                                                                               | 33 event payloads, 27 commands, 6 aliases, 6 catalog rows present                                                                            |
| Formatting of the 7 Go files edited this session                                                                 | `gofmt -l <files>`                                                                                                                        | empty                                                                                                                                       |
| usermgmt production build                                                                                       | `go build ./usermgmt/`                                                                                                                    | ok                                                                                                                                          |
| usermgmt vet, test binaries included                                                                            | `go vet ./usermgmt/`                                                                                                                      | ok                                                                                                                                          |
| Filtered usermgmt tests (`Decide\|Merge\|Transfer\|Posture\|Audit\|Bijection\|AllCommands\|CommandID\|OptionApplier\|ProjectionHealth\|EventCatalog`) | `go test ./usermgmt/ -run '...' -count=1`                                                                                           | `ok ... 0.660s` after the fix in section d1; FAIL (2 posture tests) before it                                                               |
| Command to registration bijection                                                                               | `bash scripts/checks/check-command-bijection.sh`                                                                                          | FAIL: 3 lines, all tenant commands; `constants: 27 registrations: 24`                                                                        |
| Domain counts against the living docs                                                                           | `bash scripts/checks/check-domain-counts.sh`                                                                                              | 3 DRIFT lines (`AGENTS.md` 27 events, `FEATURES.md` 27 events and 21 commands)                                                               |
| Live report count under `docs/status/`                                                                          | `ls docs/status/*.md`                                                                                                                     | 19 reports before this one, 20 with it; `TAIL_BASELINE` is 16                                                                               |

NOT verified, and why:

- **Zero tests exist for the three new deciders.** `decideMergeUsers`, `decideMergeUserInto` and `decideTransferBot` are compile-verified and vet-verified only. The `Merge|Transfer` terms in the test filter match nothing, because no such test was written yet.
- The three new `RegisterTyped` registrations are proven to exist by the bijection count (21 to 24) and by nothing else. No test dispatches `MergeUsers`, `MergeUserInto` or `TransferBot` through a repository or a service.
- The two new audit labels are not asserted by any test. The existing audit tests still pass with the map entries present.
- The full usermgmt suite was NOT run. The identity-model hermetic build and tests (M02) were NOT re-run today; they were green in the previous session.
- No lint of any kind: no golangci-lint, no cqrs-lint, no re-export policy, no module layers, no route postures, no coverage, no `check-modules`, no `nix run .#test`, no e2e.
- Nothing outside `usermgmt/` was built or tested today.

## What this session did, in order

1. Resumed from the session summary. Loaded the `cqrs-htmx` skill. Found the tree clean at `814d9607`, 17 ahead.
2. Confirmed M01 to M03 had landed (symbol grep, domain counts 33 / 27).
3. Read the identity-model sources and the usermgmt deciders, read models, SQL read models and hydrate code needed for M04 to M07.
4. Wrote `usermgmt/es_decide_merge.go` (295 lines) and wired it: three exported wrappers, three registrations, three role-gated postures, two audit actions.
5. Ran `gofmt`, `go build` and `go vet` only. All green. The auto-commit daemon committed the result (`f83fe905`, `64806f87`).
6. While verifying for this report, ran the filtered tests and found two red posture tests that I had caused (section d1). Fixed them. The daemon committed the fix (`51e708eb`). Re-ran the same filtered tests: green.

## a) FULLY DONE

Verified in an earlier session of this wave, not re-run today:

| Milestone | Content                                                                                                                                                                              |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| M01       | `tenantStateEqual` comparator in `usermgmt/property_sequences_more_test.go`; gofmt on two identity-model files                                                                        |
| M02       | `identity-model/merge_test.go`, 11 test functions: fold cases, state helpers, payload round trips, command immutability, `UserState.HasCredential`                                   |
| M03       | `usermgmt/es_constants.go` aliases and event-type lists; 6 catalog rows; `event-catalog.json` regenerated; projection-health pin 27 to 33; `event-catalog.json` sync tests |

Done in this session, verified by the filtered test run or by build and vet:

| Milestone | Content                                                                                                                                                                              | Evidence                                                    |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------- |
| M04 code  | `decideMergeUsers` (guards, credential, external-account, TOTP and email-verification adoption, one `UserMerged` last), `decideMergeUserInto`, `decideTransferBot`, the `mergeAdoption` accumulator, `unmergedLosers` | `go build`, `go vet`, gofmt clean; no tests                  |
| M04 wrappers | `DecideMergeUsers`, `DecideMergeUserInto`, `DecideTransferBot` in `usermgmt/system_exports.go`                                                                                     | build                                                       |
| M05 registrations | `MergeUsers` and `MergeUserInto` in `usermgmt/es_dispatch.go`; `TransferBot` in `usermgmt/es_bot_dispatch.go`                                                                    | bijection count 21 to 24                                     |
| M05 postures | Three role-gated postures in `usermgmt/command_postures.go`; the test-side list `everyDispatchedCommand` extended to 24                                                          | `TestCommandPostures_*` green                                |
| M08 audit labels (pulled forward) | `AuditActionUserMerged`, `AuditActionUserMergedInto` and their two map entries in `usermgmt/audit_log.go`                                                                          | existing audit tests green; no dedicated assertion           |

## b) PARTIALLY DONE

| Item            | What is there                                                                                              | What is missing                                                                                                   |
| --------------- | ---------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| M04             | All production code for the three deciders                                                                 | Micro-tasks 28, 29 and 30: adoption order, versions and codecs; retry no-op and duplicate skipping; every guard   |
| M05             | Registrations, postures, posture test list                                                                 | "Bijection gate green" cannot hold before the three tenant commands register (M11)                                 |
| M08             | The two audit action labels                                                                                | Casbin handlers, bot read model handler, SQL sync for bots, projection tests                                       |
| Plan §10 log    | Row for M01 only                                                                                           | Rows for M02 and M03 (done earlier) and for this session's M04 and M05 work                                        |
| Plan §4         | Decisions D1 to D10                                                                                        | The four deviations this session made (section "Design deviations"), not yet recorded as decisions                 |

By the numbers:

- Milestones: **3 of 25 done**, 3 partially done (M04, M05, M08), 19 not started.
- Micro-tasks: **31 of 142 have code** (#1 to #27, #31 to #33, #47). Of those, 17 were verified by tests in an earlier session, 1 by a test today (#33, and its bijection half is open), and **13 are compile-only** (#18 to #27, #31, #32, #47).

## c) NOT STARTED

| Milestone | Scope                                                                                     |
| --------- | ----------------------------------------------------------------------------------------- |
| M06       | In-memory user read model: pointer map, `UserMerged` / `UserMergedInto` handlers, finders that follow the pointer |
| M07       | SQL user read model: `User.MergedInto`, soft tombstone, `syncToSQL` skip, hydrate of the pointer map |
| M09       | `Service.MergeUsers` and the verify scenario from issue #46                                |
| M10       | Tenant deciders: rename, merge, merge-into, the recreate rule in `decideCreateTenant`     |
| M11       | Tenant registrations, postures, tenant read model, normalized-name lookup                 |
| M12       | `RenameTenant`, `MergeTenants`, `DuplicateUsers`, `DuplicateTenants`, `UniqueTenantNames` |
| M13, M14, M15 | adminui: user merge, tenant rename and merge, duplicates sections, manifest pins, templ regeneration |
| M16       | systemadapter: decoder, schemas, six registrations, gate declarations, declarative projections |
| M17       | gfl-usermgmt artifacts and scenarios                                                     |
| M18       | Issue #43: knob tests, docs, CHANGELOG, six stale hold claims                             |
| M19       | Issue #45: `LegalForm` assertions, decode-unknown test, the bill-to aggregate             |
| M20, M21, M22 | Docs, ADR and gotchas, docs ratchets                                                    |
| M23, M24, M25 | Gates battery, e2e, commit and push                                                     |

## d) TOTALLY FUCKED UP (or nearly)

**d1. I broke two tests and did not notice for several minutes. This one is on me.**

- What: after wiring the postures I added all SIX new postures, including the three tenant commands. The plan puts the tenant postures in M11, together with their registrations. `TestCommandPostures_CoverEveryDispatchedCommand` and `TestCommandPostures_AreUsableAsDeclarations` compare the posture table against the registered command set, so both went red.
- Why I did not see it: I ran `gofmt`, `go build` and `go vet` after the edit and moved on. No test. That breaks the "test after changes" rule and invariant 1 of the plan ("the usermgmt test binary compiles and passes after every milestone").
- Exposure: the daemon committed the red state (`f83fe905`, `64806f87`). It existed only on local master. Nothing was pushed.
- Fix: removed the three tenant postures (they return in M11); added `MergeUsers`, `MergeUserInto` and `TransferBot` to the test-side list. Filtered tests green. Committed by the daemon as `51e708eb`.
- Lesson: the posture drift guard worked exactly as designed and caught the error on the first test run. The failure was mine for not running it.

**d2. The tree carries two red gates on purpose, and will until M11 and M20.**

- `check-command-bijection.sh`: 3 tenant commands have no registration (expected until M11).
- `check-domain-counts.sh`: `AGENTS.md` and `FEATURES.md` still say 27 events and 21 commands (expected until M20).
- Consequence: `check-modules` and CI would be red if this tree were pushed today. It is not pushable, by design of the milestone order.

**d3. Dangerous interim state on this tree (not shipped, must not be tagged).**

- `MergeUserInto` is now dispatchable, but `NewUserReadModel()` has no handler for `UserMerged` or `UserMergedInto`. `UserReadModel.EventTypes()` declares both. `handleEvent` returns nil for a declared type with no handler, so the read model would drop them silently.
- Effect if anyone dispatched a merge on this tree: the loser's stream would be tombstoned in the journal, and the loser would stay a live row in every read model, in Casbin and in SQL.
- No released version can reach this state (nothing is tagged). M06 and M07 close it. Do not tag identity-model or usermgmt before M07 is green.

**d4. Changes in the same daemon commits that I did not make.** `f83fe905` (31 files) also carries edits to `usermgmt/oauth2/go.sum`, `usermgmt/totp/go.sum` and `usermgmt/webauthn/go.sum` (+1, +1, +6 lines). They are not mine. I did not touch, revert or inspect them.

**d5. Plan document defects I noticed while reading it.**

- §9 has `nix run .#check-reexport-policy -- -- && ...`: the double `--` makes the command fail as written.
- §10 stops at "M02+ open".
- §7 micro-task 43 names `CountSQL`, which is right, but no micro-task covers `FindByEmailSQL` returning merged tombstone rows once M07 writes them. I would add one.

## What I forgot, what I could have done better, what I can still improve

**Forgot**

1. Tests for the deciders. I wrote 295 lines of decision logic and zero assertions for it. Micro-tasks 28 to 30 are the M04 definition of done.
2. To run tests after the wiring edit (d1).
3. To append the execution-log rows for M02 and M03 to the plan, again. The summary told me to do it after each verified milestone.
4. To record my design deviations in the plan's decision table the moment I made them.
5. To update the task list when M01 to M03 were confirmed. It is accurate now.

**Could have done better**

1. Test-first for deciders. The guard table and the adoption-order table are obvious before the code; writing them first would also have pinned the event order and versions before I chose them.
2. About 25 read calls before the first write. Most were needed because the context was summarized, but the plan should carry file and line pointers for the next resumption, so a session does not re-derive them.
3. I pulled the tenant postures forward "because I was in the file". Doing one milestone's work inside another is what caused d1.

**Can still improve (concrete, small, in scope)**

1. **A drift guard for declared-but-unhandled event types.** A test asserting that every type in `allUserEventTypes` (and the tenant and bot lists) has a handler would have flagged the M04 to M06 gap before the deciders landed. Without it, d3 is invisible.
2. **Hand-maintained command lists.** `usermgmt/es_command_id_test.go` lists 21 commands under comments "User commands (11)", "Tenant commands (4)", "Bot commands (2)"; `audit_context_test.go` says "all 20". The six new commands are in neither. Extend both and correct the comments.
3. **`FindByEmailSQL` and tombstones.** Once merged losers are stored as tombstoned rows, the by-email SQL query returns them unless it filters `tombstoned = false`. Decide and test in M07.
4. **Hydrate order for pointers.** The pointer map must be rebuilt from tombstoned rows, whose survivor rows may be scanned later. Resolve at lookup time, not at hydrate time, so order does not matter.
5. **Plan hygiene.** Fix §9, fill §10, add the deviations as D11 to D14.

## Design deviations this session (all reversible, none recorded in the plan yet)

The session summary carried a concrete design for `es_decide_merge.go`. I followed it with four differences:

| #    | Summary said                                                                                                   | I implemented                                                                                                          | Why                                                                                                                              |
| ---- | -------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| D11  | Adopted `CredentialAdded` sets exactly the fields `decideAddCredential` sets                                   | The same fields plus `UserPresent` and `UserVerified` from the loser's credential                                     | Lossless. The payload struct has both fields and `NewCredentialFromPayload` already folds them. The clone-detection counter was already included |
| D12  | Email gate: `EqualFold(TrimSpace(...))`                                                                        | Exact string equality `contribution.Email == state.Email`                                                               | `ParseEmail` lowercases and trims on the canonical paths, and D3 says "equals". Exact is the conservative reading of a verification claim |
| D13  | (no such guard)                                                                                                | A survivor that is itself merged away answers Conflict `usermgmt.merge_users.survivor_merged` with `merged_into` context | Terminal identities stay terminal, and the admin gets a message that names the real survivor instead of "deleted"                |
| D14  | Plain-deleted loser: typed rejection                                                                           | Conflict `usermgmt.merge_user_into.deleted`                                                                             | A deleted user is a terminal state, not malformed input; conflicts are how the module spells "already in that state"              |

Error codes introduced (all prefixed `usermgmt.`):

| Decider             | Code                                        | Family     |
| ------------------- | ------------------------------------------- | ---------- |
| `merge_users`       | `not_found`, `deleted`, `no_contributions`, `invalid_contribution`, `self_merge` | Rejection  |
| `merge_users`       | `survivor_merged`                           | Conflict   |
| `merge_users`       | `event_failed`                              | Infrastructure |
| `merge_user_into`   | `not_found`, `survivor_required`, `self_merge` | Rejection  |
| `merge_user_into`   | `already_merged`, `deleted`                 | Conflict   |
| `merge_user_into`   | `event_failed`                              | Infrastructure |
| `bot_transfer`      | `not_found`, `owner_required`               | Rejection  |
| `bot_transfer`      | `event_failed`                              | Infrastructure |

Behaviour as implemented, so the owner can object before tests pin it:

- Adoption emits `CredentialAdded` (CBOR) per new credential, `ExternalAccountLinked` (JSON) per new provider+subject pair, at most one `TOTPEnabled` (CBOR, first non-empty secret, only when the survivor has none), then at most one `EmailVerified` (JSON), then exactly one `UserMerged` (JSON) listing only the losers the survivor has not recorded yet.
- Versions are `v+1, v+2, ...` on the survivor stream, saved and published in one append.
- A retry with the same contributions emits nothing.
- A credential the loser gained between two attempts is adopted on the retry, with no second `UserMerged`.
- `MergeUserInto` repeated for the same survivor emits nothing; for another survivor it is a conflict naming the first survivor.

## Files touched this session

| File                                 | Change                                                        |
| ------------------------------------ | ------------------------------------------------------------- |
| `usermgmt/es_decide_merge.go`        | new, 295 lines                                                |
| `usermgmt/system_exports.go`         | 3 wrappers, identity-model import                             |
| `usermgmt/es_dispatch.go`            | 2 registrations, identity-model import                        |
| `usermgmt/es_bot_dispatch.go`        | 1 registration, identity-model import                         |
| `usermgmt/command_postures.go`       | 3 postures (6 added, 3 removed again in the d1 fix)            |
| `usermgmt/command_postures_test.go`  | 3 names in the dispatched-command list                        |
| `usermgmt/audit_log.go`              | 2 action constants, 2 map entries                             |

Commits: `f83fe905`, `64806f87` (my files plus foreign go.sum edits), `51e708eb` (the d1 fix). All authored by the auto-commit daemon, all unpushed.

## e) WHAT WE SHOULD IMPROVE

1. **Never land a decider without its tests in the same step.** The next session starts with `usermgmt/es_decide_merge_test.go`, before any M06 work.
2. **Run the filtered test set after every wiring edit**, not only build and vet. The filter used today costs under one second.
3. **Add the declared-versus-handled event-type guard** (improvement 1 above) in M06, so "decided but not projected" is a test failure instead of a silent drop.
4. **Keep the milestone order.** The bijection and posture gates assume it: constants, deciders, registrations and postures for one aggregate land together, or the gates are red.
5. **Record deviations where they are decided.** The plan's decision table is the place a future session reads first.
6. **Derive the hand-written command lists**, or at least add the six new commands to the two tests that enumerate commands.
7. **Close the M04 to M07 window quickly.** Until the read models follow the pointer, a dispatchable merge command is a loaded foot-gun on this tree.

## f) Next: 50 work packages (covering all 111 open micro-tasks, #28 to #142 minus the done ones)

| #  | Package                                                                                         | Micro-tasks     |
| -- | ----------------------------------------------------------------------------------------------- | --------------- |
| 1  | M04 tests: adoption happy path, event order, versions, codecs                                   | 28              |
| 2  | M04 tests: retry no-op, duplicates skipped, late credential adopted without a second `UserMerged` | 29              |
| 3  | M04 tests: every guard, merge-into, transfer                                                    | 30              |
| 4  | M04 test: adoption events replay through `FoldUser` to the expected survivor state               | new             |
| 5  | Full usermgmt test run (workspace) and identity-model hermetic run                              | M04 verify      |
| 6  | golangci-lint and cqrs-lint on the new decider file                                             | M04 verify      |
| 7  | Plan: add D11 to D14, fill the §10 execution log, fix the §9 command                             | 127             |
| 8  | Extend `es_command_id_test.go` and `audit_context_test.go` command lists                        | new             |
| 9  | M06 pointer map and chain resolver with a cycle guard                                            | 34              |
| 10 | M06 `User.MergedInto` on the read-model row                                                      | 39 (pulled forward) |
| 11 | M06 handlers for `UserMerged` and `UserMergedInto` (index re-pointing, row removal, pointer)     | 35              |
| 12 | M06 `FindByID`, `FindByEmail`, `FindByExternalAccount` follow the pointer                         | 36              |
| 13 | M06 tests: live path, keep-first re-pointing, transitive chain                                   | 37              |
| 14 | M06 tests: `Count` and `AllUsers` exclude merged rows; a loser ID resolves to the survivor       | 38              |
| 15 | M06 test: every declared event type has a handler (new guard)                                    | new             |
| 16 | M07 SQL `Handle` writes the soft tombstone before the in-memory removal                           | 40              |
| 17 | M07 `syncToSQL` skips `UserMergedInto`                                                           | 41              |
| 18 | M07 `Hydrate` rebuilds the pointer map from tombstoned rows                                      | 42              |
| 19 | M07 `FindByEmailSQL` and `CountSQL` exclude tombstoned rows                                      | new, 43         |
| 20 | M07 tests: SQL live path, fresh hydrate, tombstone readable by ID                                | 43              |
| 21 | M08 Casbin: `UserMergedInto` removes every role of the user                                      | 44              |
| 22 | M08 Casbin: `TenantMergedInto` clears the domain                                                 | 45              |
| 23 | M08 bot read model handler for `BotTransferred`, SQL sync and hydrate                            | 46              |
| 24 | M08 tests for the three projections                                                              | 48              |
| 25 | M09 `MergeUsers` input validation; load survivor and losers; typed rejections                    | 49, 50          |
| 26 | M09 third-party external-account conflict naming both rows                                       | 51              |
| 27 | M09 build contributions from the read model; dispatch the survivor command                        | 52, 53          |
| 28 | M09 membership move (role union), bot transfer, session revoke, loser tombstone last              | 54 to 57        |
| 29 | M09 the verify scenario from issue #46; idempotent re-run, conflict, memberships, bots, sessions  | 58, 59          |
| 30 | **Kill switch 1**: issue #46 scenario green                                                       | K1              |
| 31 | M10 tenant deciders: rename, merge, merge-into                                                   | 60 to 62        |
| 32 | M10 recreate rule in `decideCreateTenant`; four wrappers                                         | 63, 64          |
| 33 | M10 tests: rename, merge, merge-into; create after delete, after merge, duplicate live create    | 65, 66          |
| 34 | M11 three tenant registrations and postures, bijection gate green                                | 67, 72          |
| 35 | M11 tenant read model: pointer, three handlers, SQL sync and hydrate                             | 68, 69          |
| 36 | M11 `NormalizeTenantName` and the normalized-name lookup; tests                                  | 70, 71          |
| 37 | M12 `RenameTenant`; `MergeTenants` validation, roster move, loser tombstone                      | 73 to 75        |
| 38 | M12 `UniqueTenantNames`; `DuplicateUsers` and `DuplicateTenants`                                  | 76, 77          |
| 39 | M12 tests: rename, merge, retry, uniqueness on and off                                           | 78, 79          |
| 40 | **Kill switch 2**: issue #47 scenario green                                                       | K2              |
| 41 | M13 adminui users: route, handler, list duplicates, two templates, tests                         | 80 to 85        |
| 42 | M14 adminui tenants: routes, handlers, templates, create-form message, tests                     | 86 to 91        |
| 43 | M15 adminui finish: manifest pins, templ regeneration, full module tests                         | 92 to 94        |
| 44 | M16 systemadapter: decoder, schemas, registrations, gate declarations, projections, tests        | 95 to 101       |
| 45 | M17 gfl-usermgmt artifacts and scenarios                                                         | 102 to 104      |
| 46 | M18 issue #43: knob tests, README, CHANGELOG, six stale hold claims, setup tests                 | 105 to 109      |
| 47 | M19 issue #45: `LegalForm` assertion, decode-unknown test, bill-to aggregate, README, follow-up  | 110 to 116      |
| 48 | M20 and M21: docs, counts, CHANGELOGs, TODO harvest, ADR, two gotchas                            | 117 to 127      |
| 49 | M22 and M23: docs ratchets, then the full gates battery                                          | 128 to 136      |
| 50 | **Kill switch 3**, then M24 e2e and M25 commit, push, CI watch                                   | 137 to 142      |

The next concrete step on "continue" is package 1 to 4: one new file, `usermgmt/es_decide_merge_test.go`, then package 5 and 6.

## g) Three questions I cannot answer myself

**Q1. Who cuts the release train?** Nothing can close #46, #47, #45 or #43 until identity-model, usermgmt, adminui, systemadapter and setup are tagged, and tags cannot be withdrawn from the module proxy. Plan D10 only says "no GitHub state change before the train". Do you want me to run `nix run .#wave-tag` myself once M23 is green, or do I stop at "pushed, CI green" and hand you a ready-to-run train checklist? My default is the second.

**Q2. What exactly may be pushed at M25?** Master is 21 commits ahead and carries work I did not author: the dashboardui time-travel and adminui cross-link sessions, the `go-sqlitestore` v0.1.2 and `go-legalparty` v0.1.1 bumps, and three `usermgmt/*/go.sum` edits. They sit below my commits on the same branch, so a push of this wave pushes them too. Is that intended? My default: push master whole, but only if every gate is green on the whole tree, foreign commits included; otherwise hold and report.

**Q3. May I archive other sessions' status reports to re-true the docs ratchets?** `docs/status/` holds 20 live reports (this one included) against a tail baseline of 16, so the tail ratchet is already red before this wave adds its M21 and M22 documents. `docs/CORPUS_BASELINE` reads 67696 now; it was 66291 at 06:30 and I did not change it. To get both ratchets green in M22 I would need to archive finished reports from other sessions through `nix run .#archive-status` (which moves their files) and re-pin both baselines. Is that allowed, or do I only bump the corpus baseline and leave their reports alone? My default: archive through the ritual only reports that have a named successor, and write the ledger sentence for every number I move.

The three questions from the 06:20 report were never answered. They are decided for now as D2 (a plain-deleted tenant may be recreated, a merged one is terminal), D4 (the merged user row stays as a soft tombstone, no DDL) and D9 (bill-to lives inside `legalparty-model`, the retention test waits for its first tag). Each is reversible until its first tag.
