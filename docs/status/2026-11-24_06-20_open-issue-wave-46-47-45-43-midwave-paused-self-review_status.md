# Status Report: Open-Issue Wave (#46 user merge, #47 tenant rename/merge, #45 legal party, #43 appkit Server-Timing), mid-wave, paused for owner instructions

**Reported at:** 2026-11-24 06:20 CET (session window 05:15 to 06:20 CET).
**Trigger:** owner asked for the four open GitHub issues to be broken down, executed and verified, then paused the run for this report.
**Tree at report time:** `da705a89` (master, auto-commit daemon; nothing was pushed, tagged, closed or commented on GitHub by this session).

## Scope: what this report verified, and what it did not

VERIFIED by command in this session, on the tree above:

| Check                                                                                                       | Command (all with `GOEXPERIMENT=jsonv2`)                                                                                            | Result                                                                                                                                 |
| ----------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| identity-model production build, vet, tests (hermetic, published deps for everything else)                  | `cd identity-model && GOWORK=off go build ./... ; go vet ./... ; go test ./...`                                                    | all exit 0 (`ok ... 0.015s`)                                                                                                           |
| Production build of every direct consumer (workspace mode)                                                  | `go build ./identity-model/... ./usermgmt/... ./systemadapter/... ./adminui/... ./setup/...`                                      | exit 0                                                                                                                                 |
| Test-binary type-check of integration_test, setup, e2e/server, usermgmt                                     | `go vet ./integration_test/ ./setup/ ./e2e/server/ ./usermgmt/`                                                                    | exit 1, exactly one error: `usermgmt/property_sequences_more_test.go:125:6` (section d1); the other three clean                       |
| Count pins in living docs                                                                                  | `bash scripts/checks/check-domain-counts.sh`                                                                                       | 3 DRIFT lines (section d3); source counts are now 33 events / 27 commands                                                            |
| Command to registration bijection                                                                           | `bash scripts/checks/check-command-bijection.sh`                                                                                   | 6 FAIL lines, `constants: 27  registrations: 21` (section d3)                                                                           |
| Formatting of the four identity-model files I edited                                                       | `gofmt -l`                                                                                                                        | `constants.go` and `commands.go` listed (section d4)                                                                                   |
| Filtered pin tests, workspace mode                                                                          | `go test ./usermgmt/ ./systemadapter/ ./gfl-usermgmt/ -run 'Catalog|Inventory|Decoder|Schema|EventsSync|ProjectionHealth|Fidelity|Volume|Posture|Architecture'` | usermgmt: build failed (d1); systemadapter: ok; gfl-usermgmt: no tests matched the filter                                             |

NOT verified, and why:

- No full test suite was run for any module (usermgmt's test binary does not compile; everything else was cut off by the pause).
- `./examples/*` test binaries were NOT type-checked: `go vet ./examples/...` matched no packages from the workspace root.
- No lint at all: no golangci-lint, no cqrs-lint, no `nix run .#test`, no `check-modules`, no coverage gate, no e2e battery.
- Zero tests exist for anything written this session. Every new event, command and fold case is compile-verified only.
- The new setup knob `AppkitServerTiming` was compiled, never executed.
- `HEAD:identity-model/fold.go` was verified by build, vet, tests and a symbol grep, not by a line-by-line review; half of it was written by a sibling session (section d2).

## Context

| Issue | Title (short)                                            | State found                                                                                                                     |
| ----- | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| #46   | usermgmt: no merge for duplicate user identities         | Nothing implemented. Live deployment has 9 user rows for one human.                                                              |
| #47   | usermgmt: no tenant merge or rename, delete is terminal  | Nothing implemented.                                                                                                            |
| #45   | identity-model: add a legal party type                   | `legalparty-model` module landed 2026-11-20 (prior session); one proposal checkbox and two verify items open.                    |
| #43   | Heads-up: go-appkit v0.10.0 emits Server-Timing          | Never tracked. Repo already on go-appkit v0.13.0; the default header is undocumented and cannot be switched off through setup. |

Design decisions taken this session (all reversible, none implemented past identity-model):

- D1: a merge is resumable and every step is idempotent. Order: survivor adoption (one command, one atomic append), memberships through the existing `AddMember` / `UpdateMemberRoles` / `RemoveMember` commands, bots, loser sessions, loser tombstone LAST (until the tombstone lands the loser is still live, so a failed merge is retried, never left half-dead).
- D2: a merged-away user or tenant is terminal (no recreate); a plain-deleted tenant stream may be recreated because `FoldTenant` already resets on `TenantCreated` (the decider was the only blocker). Owner confirmation requested, question 1.
- D3: adopted assets are emitted as ordinary `CredentialAdded` / `ExternalAccountLinked` / `TOTPEnabled` / `EmailVerified` events on the survivor, followed by one `UserMerged`; `EmailVerified` is adopted only when the loser's verified email equals the survivor's email.
- D4: read model keeps the merged-away row as a soft tombstone (`users_view.tombstoned = 1`, the pointer inside the existing `data` JSON blob), so no DDL and no operator migration; every lookup follows the pointer. Owner confirmation requested, question 2.
- D5: normalized tenant-name uniqueness is opt-in (`ServiceConfig.UniqueTenantNames`, trim plus casefold); the library enforces no default a consumer could disagree with.
- D6: new symbols live only in identity-model. No new usermgmt aliases: ADR-0062 and the re-export ratchet forbid them.
- D7: GitHub issues are not closed or commented until the release train lands, because nothing is tagged or pushed.

## a) FULLY DONE (as far as the verification above reaches)

1. **Triage of all four issues.** Bodies read in full (`gh issue view N --json`), dispatch queue read (`~/.local/state/nextd/NEXT.md`): none of the four is P0 or external.
2. **Implementation map** for the merge wave, read from source: decider, dispatch, read-model (in-memory and SQL), hydrate, Casbin projection, event catalog table, command postures, systemadapter registration path, the gates that pin counts (list in section d3).
3. **identity-model foundation for #46 and #47**, committed by the daemon in `501262fd` (commands.go, constants.go, events.go) and `2a8835ff` (fold.go). Build, vet and the module's existing tests are green.

   | Kind     | User                                     | Tenant                                                                 | Bot             |
   | -------- | ---------------------------------------- | ---------------------------------------------------------------------- | --------------- |
   | Events   | `UserMerged`, `UserMergedInto`           | `TenantRenamed`, `TenantMerged`, `TenantMergedInto`                    | `BotTransferred` |
   | Commands | `MergeUsersCmd`, `MergeUserIntoCmd`      | `RenameTenantCmd`, `MergeTenantsCmd`, `MergeTenantIntoCmd`             | `TransferBotCmd` |
   | State    | `UserState.MergedInto`, `MergedUserIDs`, `IsMerged()` | `TenantState.MergedInto`, `MergedTenantIDs`, `IsMerged()`; `IsValid` gains merged-implies-deleted | `BotState.OwnerID` now movable |
   | Fold     | both new cases                           | three new cases; `TenantCreated` now resets the whole state             | `BotTransferred` |

   Also added: `MergeContribution` (what one loser hands to the survivor, with a deep-copying `Clone`) and the generic helper `withAppendedUnique`.
4. **Issue #43 fact verification** (subagent, read-only): all five claims of the issue are true against the pinned source; every go.mod pins go-appkit v0.13.0 or an indirect v0.13.0; no test in the repo asserts an exact header set; `RunWithAppkit` mounts no second Server-Timing middleware, so there is no double emission.
5. **Issue #45 fact verification** (subagent, read-only): 4 of 5 proposal checkboxes done with tests, see section b.
6. **Opt-out seam for the appkit header**: `setup.Config.AppkitServerTiming *bool` forwarded to `appkit.ServiceConfig.ServerTiming` (commit `db73976b`). Compile-verified only.

## b) PARTIALLY DONE

### Issue #46 (user merge)

| Design checkbox from the issue                                             | Status      | What exists / what is missing                                                                                                   |
| -------------------------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `MergeUsers(ctx, survivor, losers, reason)` dispatching `UserMerged` + loser tombstone | PARTIAL | Event and command types plus fold cases exist. No decider, no registration, no service method.                                   |
| Preserve on survivor: credentials, external accounts, TOTP, `EmailVerified` | NOT STARTED | Rules designed only (D3).                                                                                                       |
| Move memberships through `AddMember` / `RemoveMember`                      | NOT STARTED |                                                                                                                                |
| Re-point bots, revoke loser sessions                                       | PARTIAL     | `BotTransferred` event, command and fold case exist. No decider, no read-model handler, no session step.                        |
| Read model: `MergedInto` on `users_view`, lookups follow the pointer       | NOT STARTED | Design D4 only.                                                                                                                |
| Publish `UserMerged`                                                       | PARTIAL     | Payload struct exists. Not in `eventCatalogSpecs`, `event-catalog.json`, or the systemadapter type decoder.                      |
| Admin UI: `POST /users/{id}/merge`, survivor picker, duplicates report     | NOT STARTED |                                                                                                                                |
| The issue's verify scenario                                                | NOT STARTED |                                                                                                                                |

### Issue #47 (tenant rename and merge)

| Design checkbox from the issue                                             | Status      | What exists / what is missing                                                                                                   |
| -------------------------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `RenameTenant` dispatching `TenantRenamed`                                 | PARTIAL     | Event, command type and fold case exist. No decider, registration or service method.                                             |
| `MergeTenants` dispatching `TenantMerged` + tombstone with `MergedInto`    | PARTIAL     | Same.                                                                                                                          |
| Publish `TenantMerged`                                                     | PARTIAL     | Same as `UserMerged`.                                                                                                          |
| Optional normalized-name uniqueness on `CreateTenant`                      | NOT STARTED | Decision D5 only.                                                                                                              |
| Admin UI: merge action, duplicates report, create form naming the tenant   | NOT STARTED |                                                                                                                                |
| Decide the recreate rule and make both halves agree                        | DECIDED, NOT BUILT | D2. Fold half changed (full reset on `TenantCreated`); decider still answers `usermgmt.tenant.already_exists` for a tombstoned stream. |

### Issue #45 (legal party), nothing changed this session

| Item                                                                                   | Status              | Evidence (from the verification subagent)                                                            |
| -------------------------------------------------------------------------------------- | ------------------- | ---------------------------------------------------------------------------------------------------- |
| (1) `LegalForm` typed enum with jurisdiction                                           | DONE (prior session) | `legalparty-model/aliases.go`, `profile.go`                                                           |
| (2) `LegalParty` aggregate, opened / updated / closed, never deleted                   | DONE (prior session) | `constants.go`, `decider.go`, `fold.go`                                                              |
| (3) VAT ID and Steuernummer as separate typed value objects                            | DONE (prior session) | `identifiers_test.go`                                                                                |
| (4) `Tenant --bill_to--> LegalParty` link                                              | MISSING             | Zero hits for `bill_to` in Go code; deferred in the 2026-11-20 status report, no TODO_LIST row        |
| (5a) Not constructible from a bot actor                                                | DONE (prior session) | `ErrBotActorForbidden`, three decider tests                                                          |
| (5b) Invoice references `LegalParty`, not `User`                                       | PARTIAL, by design  | No invoice aggregate exists; contract recorded in the module README                                  |
| Verify (a) `LegalForm` round-trips and rejects an unknown value                        | PARTIAL             | Round-trip test does not assert `LegalForm`; unknown values are rejected at `Validate()`, no decode-time test |
| Verify (b) survives tenant deletion and user deletion                                  | PARTIAL             | Closure-retention test exists; no cross-aggregate test                                              |
| Verify (c) natural person versus GmbH give different outcomes                         | DONE               | `TestProfileValidate_SameFactsDifferentOutcomes`                                                     |

### Issue #43 (appkit Server-Timing)

| Item                                                                                     | Status      | What is missing                                                                 |
| ---------------------------------------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------- |
| Claims of the issue verified against the pinned source                                   | DONE        |                                                                                 |
| Opt-out seam on `setup.Config`                                                           | PARTIAL     | No test; not in the `setup/README.md` config table; no CHANGELOG entry          |
| Document that the `RunWithAppkit` path emits the header by default                       | NOT STARTED |                                                                                 |
| True the stale "go-appkit v0.13.0 hold" claims                                          | NOT STARTED | `TODO_LIST.md:21`, `AGENTS.md:94`, `docs/review/libs/go-appkit.md:10,41-45`, `.github/workflows/ci.yml:1195-1203`, `flake.nix:5-7,41` |

## c) NOT STARTED

Everything below the identity-model type layer:

- usermgmt: event and command aliases in `es_constants.go`, catalog rows and `event-catalog.json`, all six deciders and their exported wrappers, the three `RegisterTyped` blocks, six command postures, user / bot / tenant read-model handlers (in-memory and SQL), hydrate of the merge pointer, Casbin projection handling, audit log, the service methods (`MergeUsers`, `RenameTenant`, `MergeTenants`), the two duplicates-report APIs, `UniqueTenantNames`.
- systemadapter: type decoder, event schemas, command registrations, gate declarations, declarative projections, count comments and tests.
- gfl-usermgmt: `usermgmt-events.json` and the eventcatalog resources.
- adminui: both merge routes, the rename route, both duplicates reports, the create-form message, manifest pins, templ regeneration.
- Every test for the new behaviour.
- Docs and records: `docs/DOMAIN_LANGUAGE.md`, module READMEs, the cqrs-htmx skill references, six CHANGELOGs, `TODO_LIST.md` harvest, an ADR for identity merge, the plan artifact under `docs/planning/` that the pareto-planning skill prescribes.
- Issue #43 remainder and issue #45 remainder as listed in section b.

## d) TOTALLY FUCKED UP (ranked by damage)

1. **The usermgmt test binary does not compile, caused by my change.** Adding `MergedTenantIDs []TenantID` to `TenantState` made the struct non-comparable; `usermgmt/property_sequences_more_test.go:125:6` compares two states with `!=` and now fails with `invalid operation: whole != chunked (struct containing []identitymodel.TenantID cannot be compared)`. Production code builds. `go test ./usermgmt/` is red until fixed. I did not check for by-value comparisons before adding a slice field.
2. **Shared-worktree corruption incident.** My `fold.go` edit batch carried a garbage prefix (`arg_value><b88a6f17>`) in one match string. The tool reported one of eight edits as failed; I assumed it was the last one and moved on without re-reading the file or building. The token reached the file while `UserState` was still missing its new fields, so the whole workspace stopped compiling. A sibling session hit it when its e2e webserver failed to build, removed the token, added `MergedInto`, `MergedUserIDs` and `IsMerged()` to `UserState`, and the daemon committed the result as `2a8835ff` (their account: `docs/status/2026-11-24_06-09_time-travel-real-data-and-foreign-merge-wip-unblock_status.md`). Consequences: `identity-model/fold.go` in HEAD has two authors; `UserState.HasCredential`, which my decider design relies on, is absent from the file; the breakage cost the sibling session two failed e2e launches.
3. **Gates red from my in-flight change.**
   - `check-domain-counts`: `AGENTS.md` says 27 events (source 33); `FEATURES.md` says 27 event payload structs and 21 command structs (source 33 and 27).
   - `check-command-bijection`: `CmdMergeUsers`, `CmdMergeUserInto`, `CmdTransferBot`, `CmdRenameTenant`, `CmdMergeTenants`, `CmdMergeTenantInto` have no `RegisterTyped` registration.
   - Read but not run, expected red: `usermgmt/es_projection_health_test.go:15` (`want exactly 27` catalog events), `TestEventCatalog_InventoryMatchesEventConstants`, the `event-catalog.json` sync test, the gfl events sync test.
4. **Formatting:** `identity-model/constants.go` and `identity-model/commands.go` are not gofmt-clean (alignment in the const block and in literals). Already committed.
5. **Subagent fan-out collapsed.** Five of six dispatched agents died with `too many requests: Rate limit reached`: three research agents, the blast-radius agent, and the issue-43 implementer mid-task. The implementer had already edited `setup/config.go` and `setup/run_appkit.go` and left the knob without a test or a doc line. Only the read-only verifier for #45 and #43 completed. I ran four agents in parallel on top of my own request traffic.
6. **No tests for anything written.** Six fold cases, two `IsMerged` methods, the `IsValid` rule, the full reset on `TenantCreated` and `withAppendedUnique` are unpinned. The sibling session flagged the same gap from its side.
7. **No plan artifact.** The design lives in `/tmp/cqrs-merge-plan.md`, which is outside the repo and will not survive a reboot, and in this report.
8. **This report trips two repo ratchets.** `docs/status/` held 17 live reports against `TAIL_BASELINE` 16 before this file (the sibling's 06:09 report was already over), 18 with it; `CORPUS_BASELINE` is not re-trued for either report. I archived nothing and bumped nothing: archiving other sessions' reports and pinning baselines while a sibling session writes in the same tree is an owner or docs-health decision. Count by `ls`; the gate itself was not run.
9. **Order of work.** I opened the largest wave first. After about one hour the repo has a half-landed type layer, five red gates and none of the small, self-contained items (#43 knob test and docs, #45 verify tests) banked.

## e) WHAT WE SHOULD IMPROVE (self-review)

What I forgot:

- Build and re-read after every edit. One unchecked edit batch broke the workspace for another session.
- My own failure rule: when an edit fails, read the file again. I guessed which edit had failed.
- Tests first for the fold cases. Six new fold cases with zero tests is the opposite of the repo's testing mandate.
- The count pins. I had them written down from the research and still added symbols without updating AGENTS.md, FEATURES.md and the catalog table in the same step.
- Comparability. `TenantState` was a comparable struct and tests relied on it.
- The plan file the pareto-planning skill requires.
- A receipt for the two-session `fold.go`: no CHANGELOG or agents-notes line explains the intervention.

What I could have done better:

- Sequence by risk: land #43 and the #45 verify tests first (small, isolated, each green on its own), then open the merge wave with its pins updated in the same commit-sized step as the symbols.
- Run one or two subagents at a time, never four; after any agent failure, check `git status` for partial edits before doing anything else.
- Spend context on the core files only. The systemadapter declaration files could have been grepped for names instead of read.
- Keep design decisions in the repo (plan or `CURRENT_FOCUS.md`), not in `/tmp`.
- Re-check the working tree before pausing: the daemon's commits made a half-finished state look clean in `git status`.

What to improve in the design itself:

- The survivor decider should reuse the validation of the existing single-purpose deciders instead of restating credential and external-account rules.
- `EmailVerified` adoption needs the email-equality gate (D3); the issue text does not have it.
- The membership move must be pre-validated with the pure membership deciders so the write phase can only fail on infrastructure errors.
- One duplicates-report API shape for users and tenants, rather than two unrelated ones.
- A guard test that fails when an identity-model event constant has no catalog row, no fold case and no read-model subscription would have caught most of section d3 at the first symbol.

## f) Up to 50 things to get done next (ordered)

| #   | Item                                                                                                                                                      |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Fix the usermgmt test compile break at `property_sequences_more_test.go:125` (field-wise comparator instead of `!=`); grep the repo for other by-value comparisons of `TenantState` |
| 2   | `gofmt -w identity-model/constants.go identity-model/commands.go`                                                                                          |
| 3   | Line-review HEAD `identity-model/fold.go` end to end; restore a credential lookup on `UserState` if the decider needs it                                  |
| 4   | identity-model tests: fold cases for the six events, `IsMerged`, merged-implies-deleted in `IsValid`, full reset on `TenantCreated`, purity of `withAppendedUnique`, accessor deep copies, payload JSON round trips |
| 5   | True the count pins: `AGENTS.md` identity-model bullet (33 events, 27 commands, subset counts) and the two `FEATURES.md` rows                             |
| 6   | Record the `fold.go` intervention receipt (CHANGELOG or `docs/agents-notes.md`)                                                                            |
| 7   | Get the owner answers to section g, then freeze D2 and D4                                                                                                  |
| 8   | usermgmt `es_constants.go`: event and command aliases, extend `allUserEventTypes`, `allTenantEventTypes`, `allBotEventTypes`                               |
| 9   | usermgmt catalog rows for the six events; regenerate `event-catalog.json`; replace the `want exactly 27` pin                                              |
| 10  | usermgmt deciders: `decideMergeUsers`, `decideMergeUserInto`, `decideTransferBot`, `decideRenameTenant`, `decideMergeTenants`, `decideMergeTenantInto`; the recreate rule in `decideCreateTenant`; exported wrappers in `system_exports.go` |
| 11  | `RegisterTyped` blocks in `es_dispatch.go`, `es_tenant_dispatch.go`, `es_bot_dispatch.go` (bijection gate green)                                          |
| 12  | `CommandPostures()`: six role-gated declarations with reasons                                                                                              |
| 13  | In-memory user read model: merge pointer map, `handleUserMergedInto`, pointer-following in `FindByID`, `FindByEmail`, `FindByExternalAccount`              |
| 14  | SQL user read model: soft tombstone on `UserMergedInto`, skip in `syncToSQL`, hydrate the pointer map; `User.MergedInto`                                   |
| 15  | Casbin projection: `UserMergedInto` treated like `UserDeleted`, `TenantMergedInto` like `TenantDeleted`                                                    |
| 16  | Bot read model and its SQL sync: `handleBotTransferred`                                                                                                    |
| 17  | Tenant read model: `TenantRenamed`, `TenantMergedInto`, `Tenant.MergedInto`, normalized-name lookup                                                        |
| 18  | Audit log and adminui audit labels for the six events                                                                                                      |
| 19  | Service: `MergeUsers` (validate, plan, survivor adoption, memberships, bots, sessions, loser tombstone; resumable)                                         |
| 20  | Service: `RenameTenant`, `MergeTenants`, `UniqueTenantNames`                                                                                               |
| 21  | Service: duplicates reports for users (same email or same external subject) and tenants (normalized name)                                                  |
| 22  | usermgmt tests for the issue's verify scenario: two users, one email, separate credentials and external accounts; one live row; loser carries the pointer; both credentials on the survivor; count drops by one; exactly one `UserMerged` |
| 23  | usermgmt tests: idempotent retry, conflict naming both rows, sessions revoked, bots re-pointed, SQL hydrate path, keep-first index re-pointing              |
| 24  | usermgmt architecture-catalog inventory tests                                                                                                             |
| 25  | systemadapter: `type_decoder.go`, `schema.go`, `domain_config.go` registrations, `declarations_*.go`, declarative projections, count comments, fidelity and volume tests |
| 26  | systemadapter: decide whether a tenant merge needs a deriver like the tenant-deletion cascade, or document that the service loop is the only orchestration |
| 27  | gfl-usermgmt: regenerate `usermgmt-events.json` and the eventcatalog resources; add merge scenarios if the `gfl` CLI is available                           |
| 28  | adminui: `POST /users/{id}/merge` with survivor picker, users duplicates report, `POST /tenants/{id}/merge`, `POST /tenants/{id}/rename`, tenants duplicates report, create-form message naming the existing tenant; route rows, manifest pins, templ regeneration, tests |
| 29  | Issue #43: tests for `AppkitServerTiming` (default emits `Server-Timing: total;dur=`, false omits it)                                                        |
| 30  | Issue #43: `setup/README.md` config row and `RunWithAppkit` paragraph; `setup/CHANGELOG.md` entry                                                            |
| 31  | Issue #43: true the stale hold claims in `TODO_LIST.md`, `AGENTS.md`, `docs/review/libs/go-appkit.md`, the `ci.yml` comment and the `flake.nix` comments (after verifying which Go the pinned nixpkgs ships) |
| 32  | Issue #45: assert `LegalForm` in the round-trip test; add a decode-unknown-value rejection test                                                              |
| 33  | Issue #45: cross-aggregate retention test in `integration_test/`, or an explicit dated decision to defer it                                                 |
| 34  | Issue #45: settle the bill-to link (question 2) and add a `TODO_LIST.md` row either way                                                                     |
| 35  | Issue #45: `FEATURES.md` row for `legalparty-model`                                                                                                        |
| 36  | Docs: `docs/DOMAIN_LANGUAGE.md` (six events, six commands, survivor, loser, merge pointer), module READMEs, the cqrs-htmx skill references                  |
| 37  | CHANGELOG entries: identity-model, usermgmt, adminui, systemadapter, setup, root                                                                            |
| 38  | ADR for identity merge: multi-stream orchestration, resumability, pointer-following read models, tombstone semantics                                       |
| 39  | Gates: `nix run .#check-modules`, `check-cqrs-lint`, `check-route-postures`, `check-reexport-policy -- --strict`, `check-skill-claims`, `check-permission-surface` |
| 40  | Full battery: `nix run .#test`, lint, coverage gate for the touched modules                                                                                 |
| 41  | e2e battery for the admin merge and rename flows                                                                                                           |
| 42  | Docs-health pass: archive settled reports, re-true `TAIL_BASELINE` and `CORPUS_BASELINE`                                                                    |
| 43  | Harvest this section into `TODO_LIST.md` and `ROADMAP.md`                                                                                                  |
| 44  | Write the plan artifact the pareto-planning skill expects, or record that this report replaces it                                                            |
| 45  | Two new gotchas: re-read a file after a failed batch edit instead of guessing; a slice field removes struct comparability                                  |
| 46  | Guard test: every identity-model event constant has a catalog row, a fold case and a subscription                                                           |
| 47  | Release train once green: identity-model first, then usermgmt, adminui, systemadapter, setup                                                                |
| 48  | Close #43, #45, #46 and #47 with receipts after the train lands                                                                                            |
| 49  | Owner decision on the docs-gates `audit_max` backlog this wave adds to                                                                                     |
| 50  | Re-run the browser e2e battery the sibling session had to abort twice                                                                                      |

Items 1 to 6 unbreak the tree and should run before anything else.

## g) Questions I can not figure out myself

1. **Recreate rule for tenants (issue #47).** I picked: a plain-deleted tenant stream may be recreated under the same ID (the fold already resets on `TenantCreated`; only the decider blocks it), and a merged-away tenant stays terminal. The alternative is to keep every deletion terminal and only surface the state in the panel. Recreating re-binds every stale `tenant_id` reference from the old incarnation to the new one. Which do you want?
2. **Where the merge pointer lives, and the bill-to link.** (a) For `users_view` I chose a soft-tombstoned row with the pointer inside the existing `data` JSON blob, which needs no DDL; the alternative is a real `merged_into` column with an automatic `ALTER TABLE` on upgrade, like `migrateOriginColumns`. Which do you prefer? (b) Issue #45 checkbox 4, the `Tenant --bill_to--> LegalParty` link, is still unmodelled: implement it now as a small aggregate in `legalparty-model`, or defer it to the billing lane and close #45 on the other four checkboxes?
3. **Ownership of the shared tree.** The sibling session that repaired `fold.go` asked whether my session is alive. Is it still running, and may I keep editing `identity-model/`, `usermgmt/`, `adminui/` and `systemadapter/` in this working tree, or should the rest of the wave run in a separate worktree so two sessions stop writing the same files?
