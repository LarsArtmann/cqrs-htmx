# usermgmt, rebuilt as GFL executable specs

This folder is a side-by-side experiment: the decider surface of
[`../usermgmt/`](../usermgmt/) — the same real deciders, folds, commands and
projections, untouched — described as one GFL document
([usermgmt.gfl](usermgmt.gfl)) and compiled into an executable test suite.
Compare the artifacts here against the hand-written originals a directory over.

**Headline: 32 prose scenarios → 32 generated tests (23/23 parity with the
hand-written suite — 11 happy + 12 rejection — plus 9 read-model and
audit-journal assertions the hand-written suite never had), green under
`-race` in ~1s, against the production deciders wired by
`systemadapter.DomainConfig()`.**

## The comparison

| usermgmt hand-rolls                                                                                 | Where                                                  | GFL equivalent here                                                                                                                                                                                                                                           | Command                                   |
| --------------------------------------------------------------------------------------------------- | ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| 23 `scenario.Given(...).When(...).Then(...)` tests, each ~35 lines of marshal + adapter boilerplate | `es_scenario*_test.go`                                 | 32 scenarios in prose + 40 bindings, compiled to `usermgmt_test.go`                                                                                                                                                                                           | `gfl generate tests` (below)              |
| `DeleteTenant` cascade: the service loop-dispatches `RemoveMember` per member                       | `service_tenant.go` (legacy loop, kept)                | ADR-0154 bus deriver in `systemadapter` + the cascade scenario (emitted events, dispatched commands, read model emptied)                                                                                                                                      | deriver + `gfl generate tests`            |
| `TestEventCatalog_InventoryMatchesEventConstants` (code-vs-code drift police)                       | `es_event_catalog_test.go`                             | spec-vs-universe drift gate with nearest-match suggestions + [events_sync_test.go](events_sync_test.go) (universe-vs-code)                                                                                                                                    | `gfl validate --events-file`              |
| `ArchitectureCatalog()` (hand-built topology, one flat service)                                     | `es_architecture_catalog.go`                           | [usermgmt.dot](usermgmt.dot) / [usermgmt.png](usermgmt.png) — the feature graph with typed edges                                                                                                                                                              | `gfl parse`                               |
| `GET /events/catalog` served from runtime registration (`DefaultEventCatalog()`)                    | `es_event_catalog.go` + `es_event_catalog_artifact.go` | the endpoint now serves the committed generated artifact `usermgmt/event-catalog.json` (byte-identical, ETag-verified); the spec-side gate chains through `usermgmt.gfl` → [usermgmt-events.json](usermgmt-events.json) → identity-model constants → artifact | `gfl validate --events-file` + sync tests |

### One scenario, two authoring styles

Hand-written (`../usermgmt/es_scenario_test.go`, `TestScenario_ChangeEmail_HappyPath`,
~35 lines: construct event, marshal payload, adapter closure, fold, assert):

```go
existing, err := event.NewEvent(eventUserRegistered, aggID, aggregateTypeUser, 1,
    mustMarshalPayload(t, UserRegisteredPayload{...}))
decide := func(state UserState, _ *ChangeEmailCmd) ([]event.Event, error) {
    inner := decideChangeEmail(cmd.StreamID(), cmd.Email())
    return inner(state, 1)
}
scenario.Given[*ChangeEmailCmd, UserState](t, foldUser, UserState{}, existing).
    When(cmd, decide).Then(eventEmailChanged)
```

GFL (`usermgmt.gfl`, and the generated assertion runs the FULL system —
decider, bus, projections — not just the decide function):

```gfl
Scenario: Changing the email un-verifies the account
  Given a registered user with a verified email
  When the email is changed to "new@test.com"
  Then events "EmailChanged" are emitted
    And the read model shows the email unverified
```

The second `Then` line is something the hand-written test does not check: it
asserts the `user_by_id` projection actually flipped `EmailVerified` to false.
Prose scenarios made adding the read-model assertion a one-line edit.

Rejection paths compile too. The hand-written
`TestScenario_RegisterUser_AlreadyExists` ends in
`ThenError(errorfamily.NewConflict("usermgmt.user_already_exists", ""))`; the
GFL equivalent is prose plus a two-line binding:

```gfl
Scenario: Registering a duplicate user is rejected
  Given a registered user
  When a user registers
  Then the command is rejected with conflict "usermgmt.user_already_exists"
```

```json
{ "pattern": "the command is rejected with conflict \"([\\w.]+)\"", "error": "errorfamily.NewConflict(\"$1\", \"\")" }
```

All 12 `ThenError` tests from the hand-written suite (duplicate/required/
not-found across user, tenant, membership, bot) have GFL counterparts.

### Lines of code, honestly counted (`wc -l`)

| Hand-written (usermgmt)           | LOC | GFL rebuild (here)                          |     LOC |
| --------------------------------- | --: | ------------------------------------------- | ------: |
| `es_scenario*_test.go` (23 tests) | 602 | `usermgmt.gfl` (32 scenarios, 7 features)   |     283 |
|                                   |     | `gfl.bindings.json` (40 patterns)           |     177 |
|                                   |     | `helpers.go` (the only hand-written Go)     |     351 |
|                                   |     | **authored total**                          | **811** |
|                                   |     | `usermgmt_test.go` (generated, DO NOT EDIT) |     222 |

The authored total is ~35% MORE raw lines — and buys 9 extra scenarios the
hand-written suite never had (audit-journal entries, the tenant-events-out
asymmetry, membership/bot-token/email-lookup read models) on top of 23/23
parity. The win is what the lines ARE: 283 lines are prose a product owner
can read and a graph/DOT/catalog are generated from, the bindings are 40
reusable patterns (not per-test boilerplate), and every regeneration picks
up parser/generator fixes for free. The hand-written side additionally
carries per-test event marshaling and adapter closures that the generated
side never needs. See [VALUE-MAP.md](VALUE-MAP.md) for where else in
cqrs-htmx this trade pays off — and where it deliberately does not.

## Honest coverage gaps (GFL today)

- **DeleteTenant cascade: CLOSED (2026-11-15, T04).** The cleanup loop is
  now an ADR-0154 bus deriver in `systemadapter` (`TenantDeleted` → one
  `RemoveMember` per member, projection catch-up awaited before the query —
  the stale-view race is A/B-proven and pinned), and the scenario "Deleting
  a tenant with a member removes the member" pins it end to end: emitted
  events, the full dispatched-command list (`DeleteTenant`, `RemoveMember`),
  and the emptied membership read model.
- **Remaining service-level cascades**: `DeleteUser` → membership + Casbin
  cleanup and `UpdateRoles` → session revocation live in the service layer,
  not the event journal, so journal-asserting generated specs cannot pin
  them — the same deriver-promotion shape closes them when they matter.
- **AuthAudit**: LoginSucceeded/LoginFailed/Session* are emitted by the
  service layer around HTTP flows, not by a decider — modeled as a feature
  with relationships only, no executable scenarios.

## Findings the rebuild surfaced

1. **TOTP read-model divergence**: `identitymodel.FoldUser` resets
   `TOTPEnabled` and clears `TOTPSecret` on `EmailChanged`
   (identity-model/fold.go:149-150), but the declarative `UserView`
   projection's `updateEmailChanged` (systemadapter/declarations_user.go:70)
   resets only `EmailVerified`. After an email change on a TOTP-enabled user,
   aggregate state says TOTP-off while `user_by_id` says `totp_enabled: true`.
   Worth an owner look.
2. The drift gate failure mode is genuinely better than a sync test's: rename
   an event in the spec and you get position + nearest match, not a diff dump:

   ```
   $ gfl validate usermgmt.gfl --events-file usermgmt-events.json   # (with "UserDeleted" misspelled "UserDelete")
   Error: event drift ... unknown event in relationship constraint:
   "UserDelete" in ListensFrom on feature AuditJournal (nearest: "UserDeleted")
   ```

## Files

| File                            | Role                                                                                            |
| ------------------------------- | ----------------------------------------------------------------------------------------------- |
| `usermgmt.gfl`                  | The spec: 7 features, 32 scenarios, typed relationships                                         |
| `gfl.bindings.json`             | Step-to-Go bindings (40 patterns)                                                               |
| `helpers.go`                    | The only hand-written Go: fixed streams + one helper per binding                                |
| `usermgmt_test.go`              | Generated (DO NOT EDIT) — the executable suite                                                  |
| `events_sync_test.go`           | Pins `usermgmt-events.json` to identity-model's 27 constants                                    |
| `usermgmt-events.json`          | The event universe for the drift gate                                                           |
| `usermgmt.dot` / `usermgmt.png` | Feature graph (v2 DOT generator + graphviz)                                                     |
| `eventcatalog/`                 | EventCatalog MDX export (56 resources: 24 services/events + 32 scenario flows via `--bindings`) |

## Reproduce

```bash
# from the cqrs-htmx workspace root; gfl built from ../graph-features-language

# 1. validate: syntax + semantics + drift gate
gfl validate gfl-usermgmt/usermgmt.gfl --events-file gfl-usermgmt/usermgmt-events.json

# 2. compile the prose into executable specs
gfl generate tests gfl-usermgmt/usermgmt.gfl -o gfl-usermgmt -p gflusermgmt \
  --domain 'systemadapter.DomainConfig()' \
  --deploy 'systemadapter.RecommendedMemoryDeployment()' \
  --import github.com/larsartmann/cqrs-htmx/systemadapter/v4 \
  --import github.com/larsartmann/go-error-family \
  --bindings gfl-usermgmt/gfl.bindings.json

# 3. run them against the real deciders (same command shape as usermgmt's own suite)
GOEXPERIMENT=jsonv2 go test ./gfl-usermgmt/... -race -count=1

# 4. the graph and the catalog (scenario→flow export needs --bindings)
(cd gfl-usermgmt && gfl parse usermgmt.gfl && mv usermgmt.gfl.dot usermgmt.dot \
  && dot -Tpng usermgmt.dot -o usermgmt.png \
  && gfl export eventcatalog usermgmt.gfl --bindings gfl.bindings.json -o eventcatalog)
```

The `--deploy` flag matters: `systemadapter`'s declarative projections need
`RecommendedMemoryDeployment()` (the `system.Standard` instance triple), not
the bare `systemscenario.Memory()` preset.
