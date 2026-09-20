# v5 Removal Inventory

**Purpose:** single source of truth for EVERYTHING scheduled to disappear in
the v5 major version, with the removal criterion per item. Complements
[ADR-0047](../adr/0047-re-export-layer-retirement-plan.md) (re-export
retirement), [ADR-0046](../adr/0046-drop-websocket-sse-only.md) (WebSocket
removal, already executed), and the CHANGELOG's `### Deprecated` sections.
**Generated:** 2026-08-30 from a `grep -rn "// Deprecated"` sweep (34 files).

## Removal classes

### 1. Root httputil/SSE re-exports — DONE as migration target, remove in v5

Files: `csrf_reexport.go`, `ratelimit_reexport.go`, `server_timing_reexport.go`
(39 symbols), `security.go` (deprecated alias + delegating wrapper over
`httputil.SecurityHeadersConfig`), `sse_event.go`, `sse_store.go`,
`event_store_sse.go` (go-sse re-export shims).

- **What consumers do instead:** import `github.com/larsartmann/httputil` /
  `github.com/larsartmann/go-sse` directly (migration tables in
  [leveraging-httputil.md](leveraging-httputil.md)).
- **Status:** zero internal callers; zero SA1019 warnings in-repo.
- **Removal criterion for v5:** delete files; nothing else references them.

### 2. Raw-named broadcaster API — superseded by the Hub vocabulary

Files: `sse_broadcaster.go` (`Broadcaster.Raw`, `NewBroadcasterFromRaw`,
`RawBroadcaster` interface), `datastar/broadcaster.go` (`Raw`,
`NewBroadcasterFromRaw`).

- **What consumers do instead:** `Hub()` + `NewBroadcasterFromHub`, or pass
  `*sse.Broadcaster[sse.Event]` directly. Both adapters EMBED the hub, so
  `Subscribe`/`SubscribeFilter`/`Close`/`OnSubscribe` promote automatically.
- **Status:** zero production consumers at deprecation time; test-pinned
  through v4. NOTE (2026-09-17): the datastar `Raw()` METHOD was dropped
  early with the upstream move to `go-datastar/broadcast` (a transparent
  type alias cannot carry methods) — `datastar/broadcaster.go` now contains
  the full deprecated facade (`Broadcaster` alias + constructor shims +
  `NewBroadcasterFromRaw` delegating to `broadcast.NewBroadcasterFromHub`),
  and the whole facade joins this removal class: at v5 delete
  `datastar/broadcaster.go`, point every doc/consumer at
  `github.com/larsartmann/go-datastar/broadcast`, and drop the
  `go-datastar/broadcast` require (dep budget returns to 5).
- **Removal criterion:** delete symbols + their tests; guide
  [sse-and-datastar.md](sse-and-datastar.md) already documents only Hub APIs.

### 3. usermgmt → identity-model re-export layer (~161 symbols, 22 files)

Files: every `es_*.go` event/command/state file plus `authz_types.go`,
`auth_interfaces.go`, `credential.go`, `crypto.go`, `email.go`, `errors.go`,
`external_account.go`, `id.go`, `store_interfaces.go`, `upcaster.go`,
`user.go` in usermgmt — all carrying
`// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.`

- **What consumers do instead:** import identity-model directly (type aliases
  are transparent).
- **Status:** adminui (2026-08-14) and integration_test (2026-08-15) fully
  migrated; no module uses the deprecated aliases anymore.
- **Removal criterion:** delete the alias declarations. NOTE: the constructor
  WRAPPERS (e.g. `usermgmt.NewExternalAccount`) are NOT deprecated — only the
  type/constant/fold aliases are.

### 4. Deprecated constructors with behavior consequences (remove, no alias)

- `identity-model/id.go` `NewUserID(string)` — silently SHA-256-hashes
  non-ULID input (collision risk). Replacements: `ParseUserID` (strict),
  `SyntheticUserID` (explicit), `GenerateUserID` (root). **Removal
  criterion:** v5 signature-clean break; audit guidance in CHANGELOG v4.2.x.
- Storage view API call sites in usermgmt (`sql_*.go`) — currently
  nolint-justified as the supported runtime path until the v5 migration;
  migrate to the record/listing view APIs at the same cut (see the
  ADR-0123 nolint comments).
- `systemadapter.NewProjectionLayer` (marked `// Deprecated:` 2026-09-10) —
  superseded by the declarative projections path (`DomainConfig.Projections` +
  `ProjectionTypeDecoder`, equivalence-tested by
  `TestDeclarative_EquivalenceWithProjectionLayer`). **Removal criterion:** no
  consumer examples left on the legacy path (`examples/system-demo` migrates
  with the same cut); the module's own equivalence tests keep exercising it
  until then via a scoped staticcheck exclusion in
  `systemadapter/.golangci.yml`. **Accepted limitation (recorded 2026-09-20):**
  the declarative path has no custom checkpoint/DLQ store injection —
  `system.New` uses an internal in-memory checkpoint store — so consumers with
  durable-checkpoint or dead-letter requirements stay on `NewProjectionLayer`
  (with `WithCheckpointStore`/`WithDeadLetterStore`) until go-cqrs-lite ships a
  `system.New` option for both. If no such option lands before the v5 cut,
  removal proceeds with this documented in the migration guide
  (`docs/guides/declarative-projections.md` §checkpoint-store caveat).
- Root in-memory idempotency default — nolint-justified (library principle,
  consumer opts into a durable store); revisit at v5.

### 5. Module renames: discovery fix (proposed, needs a decision)

`adminui` vs `dashboardui` does not say "identity ops" vs "event-store
observability"; newcomers routinely land on the wrong module. Docs mitigation
landed 2026-09-17: cross-linked contrast sections in both module READMEs, both
doc.go package docs, the root README UI-modules table, and the loginpage
README. v5 is the only breaking window for module-path renames:

- `adminui/v4` → `identityadmin/v4` (alt: `useradmin/v4`): identity
  operations over a `*usermgmt.Service`.
- `dashboardui/v4` → `esdashboard/v4` (alt: `observui/v4`): CQRS/ES
  observability over go-cqrs-lite introspection interfaces.
- **Removal criterion:** pick final names, cut the new module paths in the v5
  train via the verify-tag protocol, keep old paths as deprecated doc-only
  shims for one cycle, then sweep README/AGENTS/guides/examples for old names
  (`check-docs-links` catches stale relative links).
- **Decision needed:** final name choice per module. If rejected, the docs
  mitigation becomes the permanent fix.

### 6. Already removed (for completeness — do NOT re-plan)

- **WebSocket transport** — ADR-0046 (executed): `WSBroadcaster`,
  `WSMessage`, `WSOOBHTML`, dispatch/encoder helpers, `HTMXExtWS`, all WS
  tests. Library is SSE-only.
- **Aggregate→Stream API family** — migrated 2026-08-14 (`id.AggregateID`→
  `id.StreamID`, `evt.AggregateType`→`evt.StreamType`,
  `event.ErrAggregateNotFound`→`event.ErrStreamNotFound`).
- `metaengine.OnTyped` — migrated to `OnRecordTyped` 2026-08-14
  (systemadapter).

## Sweep protocol for the v5 cut

1. Delete classes 1-3 files/symbols (mechanical — the aliases are transparent;
   no consumer-invisible behavior changes).
2. Apply class 4 decisions per item (each has a written rationale).
3. Run: `nix run .#build`, `.#test`, `.#lint`, `bash
   scripts/check-module-isolation.sh`, `nix run .#check-release-train`,
   `nix run .#check-docs-links` — then a docs sweep for
   `cqrshtmx.CSRF|cqrshtmx.SecurityHeaders|Broadcaster.Raw|usermgmt.User.*Payload`
   references in guides/examples.
4. Bump via the coordinated family train (runbook
   `docs/runbooks/release-next-train-prep.md` + verify-tag protocol).

## See Also

- [ADR-0047](../adr/0047-re-export-layer-retirement-plan.md) — why the
  re-export layers exist and the retirement plan
- [ADR-0046](../adr/0046-drop-websocket-sse-only.md) — WebSocket removal
- [sse-and-datastar.md](sse-and-datastar.md) — Hub-first broadcaster vocabulary
