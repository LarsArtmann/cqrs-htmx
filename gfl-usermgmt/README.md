# usermgmt, rebuilt as GFL executable specs

This folder is a side-by-side experiment: the decider surface of
[`../usermgmt/`](../usermgmt/) — the same real deciders, folds, commands and
projections, untouched — described as one GFL document
([usermgmt.gfl](usermgmt.gfl)) and compiled into an executable test suite.
Compare the artifacts here against the hand-written originals a directory over.

**Headline: 13 prose scenarios → 13 generated tests, green under `-race` in
~1s, against the production deciders wired by `systemadapter.DomainConfig()`.**

## The comparison

| usermgmt hand-rolls | Where | GFL equivalent here | Command |
| --- | --- | --- | --- |
| 23 `scenario.Given(...).When(...).Then(...)` tests, each ~35 lines of marshal + adapter boilerplate | `es_scenario*_test.go` | 13 scenarios in prose + 25 bindings, compiled to `usermgmt_test.go` | `gfl generate tests` (below) |
| `TestEventCatalog_InventoryMatchesEventConstants` (code-vs-code drift police) | `es_event_catalog_test.go` | spec-vs-universe drift gate with nearest-match suggestions + [events_sync_test.go](events_sync_test.go) (universe-vs-code) | `gfl validate --events-file` |
| `ArchitectureCatalog()` (hand-built topology, one flat service) | `es_architecture_catalog.go` | [usermgmt.dot](usermgmt.dot) / [usermgmt.png](usermgmt.png) — the feature graph with typed edges | `gfl parse` |
| `DefaultEventCatalog()` served at `GET /events/catalog` | `es_event_catalog.go` | [eventcatalog/](eventcatalog/) — 24 MDX pages (services, events, flows) | `gfl export eventcatalog` |

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

## Honest coverage gaps (GFL today)

- **Error paths**: 8 of the 23 hand-written tests are `ThenError(...)`
  assertions (duplicate registration, name-required, not-found, ...). GFL
  bindings support exactly `Query+Result` / `Events` / `Commands` Then shapes —
  no `ThenError`. The scenarios above cover every *happy path*; the rejection
  paths stay hand-written until GFL grows an error binding.
- **Service-level cascades**: `DeleteUser` → membership + Casbin cleanup and
  `UpdateRoles` → session revocation live in the service layer, not the event
  journal, so journal-asserting generated specs cannot pin them. The
  saga/deriver shape (see GFL's `examples/saga.gfl`) is the eventual fix:
  promote cascades to derived commands/events first.
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

| File | Role |
| --- | --- |
| `usermgmt.gfl` | The spec: 7 features, 13 scenarios, typed relationships |
| `gfl.bindings.json` | Step-to-Go bindings (25 patterns) |
| `helpers.go` | The only hand-written Go: fixed streams + one helper per binding |
| `usermgmt_test.go` | Generated (DO NOT EDIT) — the executable suite |
| `events_sync_test.go` | Pins `usermgmt-events.json` to identity-model's 27 constants |
| `usermgmt-events.json` | The event universe for the drift gate |
| `usermgmt.dot` / `usermgmt.png` | Feature graph (v2 DOT generator + graphviz) |
| `eventcatalog/` | EventCatalog MDX export |

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
  --bindings gfl-usermgmt/gfl.bindings.json

# 3. run them against the real deciders (same command shape as usermgmt's own suite)
GOEXPERIMENT=jsonv2 go test ./gfl-usermgmt/... -race -count=1

# 4. the graph and the catalog
(cd gfl-usermgmt && gfl parse usermgmt.gfl && mv usermgmt.gfl.dot usermgmt.dot \
  && dot -Tpng usermgmt.dot -o usermgmt.png \
  && gfl export eventcatalog usermgmt.gfl -o eventcatalog)
```

The `--deploy` flag matters: `systemadapter`'s declarative projections need
`RecommendedMemoryDeployment()` (the `system.Standard` instance triple), not
the bare `systemscenario.Memory()` preset.
