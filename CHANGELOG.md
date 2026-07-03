# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [v4.1.0] - 2026-07-04

### Added

- **Embedded HTMX extensions** (`htmx_extensions.go`): SSE (htmx-ext-sse 2.2.4), WS (htmx-ext-ws 2.0.4), and idiomorph (0.7.4) JS files embedded via `go:embed`. These are the three extensions with direct server-side counterparts in the library (SSEStream/Broadcaster, WSMessage/WSBroadcaster, morph-swap for SSE partials).
  - `HTMXExtensionHandler(name)`: Serve a single embedded extension. Constants: `HTMXExtSSE`, `HTMXExtWS`, `HTMXExtIdiomorph`.
  - `HTMXExtensionsHandler(names...)`: Serve a concatenated bundle (one HTTP request instead of N).
  - `HTMXExtensionVersion(name)`, `HTMXExtensionNames()`: Introspection helpers.
  - `HTMXExtensionCDNScriptTag(name)`: CDN `<script>` tag fallback for consumers who prefer CDN.
  - Same caching as `HTMXScriptHandler` (ETag, Cache-Control 1yr immutable, 304 Not Modified).

### Changed

- **HTMX core bumped 2.0.9 → 2.0.10**: Updated embedded `htmx.min.js` and all version references.
- **serveJS helper extracted**: `HTMXScriptHandlerWith` now delegates to shared `serveJS(js, etag)` function, eliminating duplication with extension handlers.

## [v4.0.1] - 2026-07-02

### Added

- **Configurable TOTP pending-secret TTL** (`ServiceConfig.TOTPPendingSecretTTL`): Was hardcoded to 5 minutes, now configurable. Defaults to 5 minutes when ≤ 0. Mirrors the `WebAuthnSessionTTL` pattern.
- **Configurable WebAuthn session TTL** (`ServiceConfig.WebAuthnSessionTTL`): Was hardcoded to 5 minutes, now configurable. Already shipped in v4.0.0 but missing from the v4.0.0 changelog.
- **TOTP cross-module integration test** (`integration_test/totp_integration_test.go`): Full Service.EnableTOTP → totp.Provider.GenerateSecret → VerifyTOTPSetup → VerifyTOTP chain through the provider boundary. Includes nil-provider guards.
- **OAuth2 cross-module integration test** (`integration_test/oauth2_integration_test.go`): Full Service.BeginOAuthLogin → oauth2.Provider.BeginLogin (PKCE + redirect URL) flow. Includes unknown-provider and nil-provider guards.
- **JSON serialization boundary benchmarks** (`usermgmt/webauthn_benchmark_test.go`): `BenchmarkMarshalWebAuthnUser` (2 creds: 1.2µs, 4 allocs) and `BenchmarkMarshalWebAuthnUser_NoCreds` (0 creds: 400ns, 3 allocs). Confirms negligible overhead for ceremony paths.
- **Coverage tests for parse helpers** (`usermgmt/coverage_parse_helpers_test.go`): `ParseUserID`, `MustParseUserID`, `ParseActorID` round-trip, `ParseActorID` no-prefix fallback, `MustParseEmail` panic.
- **Pre-generated RSA key fixture** for OAuth2 OIDC tests: `sync.Once` cache eliminates ~50ms RSA key generation per test invocation. Shared across all `fakeOIDCServer` tests.
- **Fuzz tests on JSON boundary** (`usermgmt/webauthn_fuzz_test.go`, `usermgmt/webauthn/provider_fuzz_test.go`): `FuzzMarshalWebAuthnUser`, `FuzzParseUser`, `FuzzParseSession` — crash-tested with 400K+ iterations.

### Changed

- **OAuth2 lint config**: Added `gochecknoglobals` exclusion for test files (RSA key fixture uses package-level `sync.Once` pattern).
- **CONTRIBUTING.md**: Updated module table from 8 to 11 modules. Added auth sub-module structural typing explanation and dependency direction.
- **DOMAIN_LANGUAGE.md**: Added `TOTPProvider`, `WebAuthnProvider`, `OAuth2Provider`, `WebAuthnSessionTTL` terms.
- **ADR-0035**: Updated JSON serialization section with benchmark data (400ns–1.2µs) and fuzz/integration test references.

### Fixed

- **Stale doc references**: AGENTS.md coverage stat (95.4%/80.1% → 94.3%/74.5%), README.md go-cqrs-lite version (v3.1.0 → v3.5.0), README.md file tree (WebAuthnConfig → WebAuthnSessionTTL, added totp/webauthn/oauth2 dirs), TODO_LIST.md coverage stat and deferred items.
- **Migration guide**: Removed stale "pending" references in dependency impact table.

## [v4.0.0] - 2026-07-02

### Breaking Changes — Auth Strategy Extraction (Sollbruchstellen)

All three auth strategies (TOTP, WebAuthn, OAuth2) extracted behind primitive-type interfaces as independent Go modules. Consumers import only the auth strategies they need — zero transitive dependencies on pquerna/otp, go-webauthn, golang.org/x/oauth2, coreos/go-oidc, or go-jose unless they explicitly opt in.

#### New Modules

- **`usermgmt/totp/v4`** — TOTP MFA (pquerna/otp). Implements `TOTPProvider`.
- **`usermgmt/webauthn/v4`** — WebAuthn passkeys (go-webauthn). Implements `WebAuthnProvider`.
- **`usermgmt/oauth2/v4`** — OAuth2/OIDC (golang.org/x/oauth2 + coreos/go-oidc). Implements `OAuth2Provider`.

#### TOTP Changes

- `ServiceConfig.TOTPConfig *TOTPConfig` → `TOTP TOTPProvider`
- `TOTPConfig`, `TOTPTimeStep`, `TOTPDigits`, `TOTPSecretLength` removed from core
- New `TOTPProvider` interface: `GenerateSecret(string) ([]byte, string, string, error)`, `ValidateCode([]byte, string) bool`
- `TOTPVerifier` interface removed (was a ghost — never wired)

#### WebAuthn Changes

- `ServiceConfig.WebAuthnConfig *WebAuthnConfig` → `WebAuthn WebAuthnProvider`
- `WebAuthnConfig` type removed from core (now `webauthn.Config` in the webauthn module)
- `webauthn_adapter.go` moved to webauthn module
- `WebAuthnSessionStore` interface changed: `*webauthn.SessionData` → `[]byte`
- `BeginRegistrationResponse.Options` / `BeginLoginResponse.Options` changed to `json.RawMessage`
- In-memory session store uses TTL-based eviction (5 min default)
- Virtual authenticator test removed from core (needs go-webauthn)

#### OAuth2 Changes

- `ServiceConfig.OAuth2Config *OAuth2Config` → `OAuth2 OAuth2Provider`
- `OAuth2Config`, `OAuth2ProviderConfig` removed from core (now `oauth2.Config`, `oauth2.ProviderConfig`)
- New `OAuth2Provider` interface: `BeginLogin(ctx, provider, state) (redirectURL, pkceVerifier, error)`, `FinishLogin(ctx, provider, code, pkceVerifier) (userInfoJSON, error)`
- `OAuth2UserInfo` exported in core (Service deserializes provider's JSON return)
- `OAuth2StateStore.Save` signature changed: `Save(state, provider, pkceVerifier, ttl) error`
- OIDC discovery moved from Service.initOAuth2 to `oauth2.New()`
- `initOAuth2` and `getOAuth2Provider` methods removed from Service

#### Module Paths

- All import paths changed `/v3` → `/v4` across all modules

See `docs/migrations/v3-to-v4.md` for detailed before/after examples.

## [3.5.0] - 2026-07-01

### Added

#### Error Model Overhaul

- **HTTPStatusCarrier interface** (`errors_status.go`): Errors can now pin a specific HTTP status via `WithHTTPStatus(err, status)`. The wrapper preserves the cause's error family + sentinel identity (`errors.Is` traverses). Replaces the old `errorStatus()` split brain in usermgmt — usermgmt sentinels now use `withStatus()` which delegates to `WithHTTPStatus`. See ADR-0034.
- **ProblemDetailsErrorHandler**: Emits `StructuredError` as `application/problem+json` — unified RFC 7807 shape across all transports (HTTP, SSE, WS). Opt in via `Config.ErrorHandler`.
- **StructuredError enrichment**: Now exposes `Message`, `Why`, `Fix` fields (RFC 7807 extensions from `Family.DefaultMessage/Why/Fix`). Same JSON shape across HTTP/SSE/WS.
- **5xx detail redaction**: `SafeDetail(err, status, includeInternal)` replaces 5xx error text with the family's public-safe default message. 4xx detail (raw error) is preserved. `Config.IncludeInternalDetails` opts back in for dev. `StructuredError.Detail` is also redacted for SSE/WS.
- **Exported auth error codes**: `CodeUnauthorized`/`CodeForbidden` — compile-time-safe shared between root and usermgmt.

#### Architecture Enforcement

- **CI module architecture scripts** (4 scripts): `check-module-isolation.sh` (GOWORK=off build+vet per module), `check-dep-budgets.sh` (per-module max production deps), `check-version-drift.sh` (sibling version consistency), `check-replace-directives.sh` (no absolute paths). Wired as `nix run .#check-modules` and as a `module-architecture` CI job in `.github/workflows/ci.yml`.
- **Auth strategy interfaces** (`usermgmt/auth_interfaces.go`): `TOTPVerifier`, `WebAuthnProvider`, `OAuth2Provider` interfaces. Non-breaking, additive. Compile-time assertion `var _ TOTPVerifier = (*Service)(nil)` proves Service already satisfies the TOTP contract. Prepares the v4 extraction seam.

### Changed

- **go-cqrs-lite upgraded to v3.5.0**: All modules updated. v3.5.0 ships CBOR codec support, stricter encoding validation, storage package restructure into view/relational/ subdirs (with backward-compatible aliases), and lint cleanup across 10 modules.
- **Generic `errors.AsType`**: Replaced manual error chain type assertions with the generic helper from go-error-family.
- **Command decode error wrapping**: Decode failures now wrapped with context and family parity (`event.Wrapf` with `event.Classify(err)`).

### Fixed

- **usermgmt `errorStatus` split brain eliminated**: The old `errorStatus()` function in usermgmt was a separate mapping path from root's `MapError`. Now unified through `HTTPStatusCarrier` — one resolution path across modules.
- **Root module go.sum gap**: Missing `storage/memory/v3` and `snapshot/v3` entries caused `GOWORK=off go vet` to fail. Fixed via targeted `go get` + `go mod tidy`.
- **Root dep budget regression**: v3.5.0 upgrade promoted 7 indirect deps to direct (23 vs budget 18). Fixed via `go mod tidy -e` — properly demoted back to 16 direct deps.
- **Version drift eliminated**: All go-cqrs-lite sibling modules (snapshot, schema, storage/memory, listing, scenario) now pinned to consistent versions across root, usermgmt, adminui, integration_test, and all examples.
- **wrapcheck lint in usermgmt**: `forbidden()` now routes through `withStatus()` like all its siblings; `withStatus` marked as wrapping boundary.

### Rejected

- **ADR-0030 (Phase 2b IndexedDB)**: Marked REJECTED. Client-side persistence for the SharedWorker queue is a fundamentally inconsistent API surface that doesn't belong in a server-side Go library.

## [Unreleased]

### Added

- **Observability wiring guide** (`docs/observability-wiring.md`): Complete OTel tracing + Prometheus `/metrics` + Server-Timing wiring recipes using `BeforeDispatchHook`/`AfterDispatchHook`. References go-cqrs-lite upstream `otel/v3` + `middleware/v3` + `prometheus/v3` modules.
- **v3→v3.3 incremental migration guide** (`docs/MIGRATION-v3-incremental.md`): Documents checkpoint replay, BasicCommand embedding, Server-Timing, SQL read models, stack presets — all opt-in, backward compatible.
- **scenario/v3 BDD for Tenant, Bot, Membership** (3 new test files, 20 tests): Completes scenario/v3 BDD adoption for all 4 usermgmt aggregates. Tenant: 8 tests (create/suspend/reactivate/delete happy + error paths). Bot: 6 tests (register/delete happy + error paths). Membership: 6 tests (add/update-roles/remove happy + error paths).
- **Server-Timing fuzz tests** (`server_timing_fuzz_test.go`): Adversarial metric names/descs/durations, middleware fuzz, nil-receiver no-op verification. Found CRLF injection bug (see Fixed).
- **CI codegen drift guard** (`nix run .#check-codegen`): Verifies adminui `_templ.go` files match `.templ` sources by regenerating + diffing. Prevents silent codegen drift.
- **Templ CLI in devShell** (`flake.nix`): Pins `pkgs.templ` v0.3.1020 (matches go.mod) to eliminate codegen oscillation between CLI versions.
- **`nix run .#gen` app**: One-command templ codegen + gofmt normalization for adminui.
- **Server-Timing API** (`server_timing.go`): W3C Server Timing response header support for debug-mode performance profiling. Emits `Server-Timing: total;desc="Total request";dur=12, db;dur=8` headers visible in browser DevTools and curl. Three entry points: (1) `ServerTimingMiddleware()` — standalone always-on middleware; (2) `ServerTimingMiddlewareWhen(pred)` — predicate-gated (e.g. `?debug=1`, admin role); (3) `Config.ServerTiming` — 1-line integration into `App.Command()`/`App.Query()` handlers. Thread-safe `*ServerTiming` collector uses nil-receiver pattern (disabled=nil=natural no-op). Helpers `MeasureServerTiming(ctx, name)` and `RecordServerTiming(ctx, name, desc, dur)` are nil-safe. Interface preservation: wrapper delegates `Flusher`/`Hijacker`/`Pusher`/`Unwrap` so SSE/WS/HTTP2 work transparently. Benchmark: disabled=3.6ns/0-allocs, enabled Measure=138ns/1-alloc.
- **Checkpoint-based projection replay** (`usermgmt/es_projection_setup.go`): `StartProjections` gains optional `event.CheckpointStore` parameter. When non-nil AND journal implements `SeekableJournal`, replay uses `ReadFrom(checkpoint.EventID, 0)` instead of `ReadAll()` — avoids full journal replay on every restart. Checkpoint saved after each replayed event. Graceful fallback: nil store or non-seekable journal → full replay (backward compatible). `EventSourcedConfig`, `ServiceConfig`, `SQLiteSetupConfig`, `PostgresSetupConfig` all gain `CheckpointStore` field.
- **ADR-0032**: BasicCommand embedding decision — documents the structural fix for the zero-cmdID bug class and the rationale for `mustCommand` panic-on-construction-failure.
- **ADR-0031 status update**: PROPOSED → Accepted. Checkpoint-based replay shipped in StartProjections; CatchUpSubscriber migration deferred.
- **Command ID regression tests** (`es_command_id_test.go`): `TestAllCommandsProduceDifferentIDs` — constructs every command twice (40 total), asserts all IDs are mutually unique. `TestMustCommand_PanicsOnZeroAggregateID` and `TestMustCommand_PanicsOnEmptyCommandType` — verify fail-fast behavior on programming bugs.
- **CheckpointStore usage example** (`es_checkpoint_test.go`): Demonstrates the opt-in checkpoint round-trip (fresh store → save → reload → resume from checkpoint).
- **scenario/v3 BDD on ChangeEmail** (`es_scenario_test.go`): Second decider scenario test demonstrating the BDD DSL adoption beyond RegisterUser.
- **Authz doc comments**: `RemoveAllRolesForUser` and `RemoveAllRolesInDomain` now document why the `subject` parameter is intentionally `string` (Casbin subjects are polymorphic: user IDs, bot IDs, prefixed actors).

### Fixed

- **CRLF injection in Server-Timing `escapeQuotedString`**: CR and LF characters in metric descriptions are now replaced with spaces. Previously only `"` and `\` were escaped, leaving raw newlines that could enable HTTP header splitting via crafted descriptions. Found by `FuzzServerTimingHeaderValue`.

### Changed

- **CONTRIBUTING.md rewritten (root + usermgmt)**: Removed deleted catalog module references, fixed module count (5→8), updated test framework documentation (standard testing + scenario/v3 BDD, not Ginkgo), removed password auth references, synced file tree, added error-family enforcement, added templ codegen instructions, added Nix-first workflow.
- **usermgmt coverage gate raised**: 75% → 78% (actual: 79.3%). Locks in coverage gains.
- **Command constructors embed `command.BasicCommand`**: All 20 usermgmt command structs now embed `*command.BasicCommand`, which promotes `Type()`, `AggregateID()`, `ID()` methods automatically. This structurally eliminates the zero-cmdID bug class (7 constructors previously returned zero command IDs, silently breaking idempotency dedup and Watermill message UUIDs). The `mustCommand` helper panics on construction failure — the only error cases (empty command type, zero aggregate ID) are programming bugs. See ADR-0032.
- **ADR-0015 status table updated**: All 6 remaining "Planned" items (Session struct, Impersonation, Tenant, Bot, upcasters, Roles removal) marked Done with version references.
- **flake.nix package version bumped**: 3.1.0 → 3.3.0.
- **FEATURES.md synced**: Removed catalog column (module merged upstream), updated ClientIP to FULLY_FUNCTIONAL, synced coverage/test counts.
- **ROADMAP.md synced**: Marked scenario/v3 BDD, OTel seam guide, Prometheus seam guide as Done.
- **ROADMAP typo fixed**: "v3.30" → "v3.3.0".
- **TODO_LIST triaged**: All `[~]` (partially done) items resolved — marked `[x]` (BrandNamer wired, TenantState.IsValid added, Phase 2a shipped) or `[-]` (blocked: ActorID split brain, Email type, WebAuthn \*http.Request, snapshot integration).
- **Status reports archived**: 7 reports older than 2 weeks (pre-June-15) moved to `docs/status/archive/`.

## [3.3.0] - 2026-06-29

### Added

- **Regression test for command IDs** (`es_command_id_test.go`): Table-driven test asserting all 20 command constructors produce a non-zero `ID()`. Prevents recurrence of the zero-cmdID bug.
- **Offline command queue — Phase 2a** (ADR-0029): `adminui/assets/sync-worker.js` — a SharedWorker (~80 lines vanilla JS) that queues command IDs when the network is down and tells tabs to retry on reconnect. The worker is a coordinator, not a proxy: it does NOT send HTTP requests (HTMX does), does NOT own SSE (per-tab EventSource), does NOT persist to disk (in-memory). Reactive detection via `htmx:sendError`. Served at `GET /-/sync-worker.js` when `Config.SSEURL` is set. IndexedDB banned; OPFS deferred to Phase 2b. admin.js gains `initSyncWorker()`, `enqueueCommand()`, `retryQueuedCommand()`, and an `htmx:sendError` handler that queues instead of rejecting. CSS adds `[data-sync-queued]` (dimmer, slower pulse) and `.sync-bar[data-sync-status="offline"]` (amber).
- **Production SSEEventStore** (`JournalSSEStore`): Backed by `event.SeekableJournal` for efficient cursor-based replay. Falls back to `ReadAll` + in-memory filter when the journal doesn't support `ReadFrom`. `WithMaxReplay(n)` limits first-connection replay volume (default 1000). Consumer-provided `EventToSSEMapper` function converts domain events to SSE events.
- **ACK protocol** (command confirmation): `CommandAck` struct with `{commandId, status, error}` JSON. `BroadcastOnAck()` / `BroadcastOnAckFunc()` on `Broadcaster` (SSE) and `BroadcastOnAckWS()` / `BroadcastOnAckWSFunc()` on `WSBroadcaster` (WS parity). Opt-in via `X-Command-Id` header.
- **Integration tests**: 6 end-to-end tests prove `JournalSSEStore` + `Broadcaster` + ACK protocol work together in real HTTP handlers (replay, confirmed/rejected ACK, reconnect + live ACK, opt-in guard, concurrent race).
- **ADRs**: 0023 (command-sync — sync commands not events), 0024 (honest UI — never lie about pending state), 0026 (idempotency store), 0027 (decide stays on server), 0029 (SharedWorker Phase 2a).
- **Idempotency store** (`IdempotencyStore` interface + `MemoryIdempotencyStore`): Prevents duplicate command execution on client retry. `CheckAndRecord` interface method is truly atomic (single lock for check+record). `ErrDuplicateCommand` → HTTP 409 Conflict. TTL-based expiration with background sweep goroutine + lazy expiry in `Seen()`. See ADR-0026.
- **admin-demo idempotency wiring**: The admin-demo showcase now rejects duplicate mutations (same `X-Command-Id`) with HTTP 409 before they reach the panel handler. Proves the feature end-to-end.
- **ADR-0027**: Definitive decision that `decide()` stays on the server (Queue-Only client). The library provides the queue/sync/ACK protocol; pre-validation is a consumer concern. Unblocks all Phase 2 work.

### Changed

- **go-cqrs-lite upgraded to v3.4.0** across all 8 modules: command, event, idempotency, query, listing, projection, snapshot, stack, storage → v3.4.0; decider/id/otel/watermill/codec/dispatcher at v3.3.0/v3.3.1 (latest tags). v3.4.0 adds managed projection host, durable scheduling, scenario-testing DSL.
- **SSE delegation lint cleared**: `sse_event.go` vars converted to proper wrapper functions (`gochecknoglobals`), wrapcheck annotated for pure delegation. All modules now report 0 lint issues.
- **Form decoder upgraded** (`decoder.go`): Replaced allocation-heavy JSON round-trip (`url.Values → map[string]any → JSON → struct`) with `go-playground/form/v4` (zero transitive deps, `SetTagName("json")` for backward compat). Form keys normalized to lowercase for case-insensitive field matching.
- **Pagination unified**: Both root `DecodePagination` and usermgmt `credential_http.go` now delegate to `query.NewPagination` from go-cqrs-lite. **BREAKING**: Requesting a page beyond the last page now returns an empty page (standard REST) instead of silently clamping to the last page. The response includes `total_pages` so clients can detect the valid range.
- **go.mod alignment**: All 8 modules aligned to Go 1.26.4.
- **Stdlib modernization**: `slices.Contains`, `min()`, `slices.IndexFunc` replace 5 manual loops across root/usermgmt/adminui/examples.
- **ID types branded** (**BREAKING**): `ActorID`, `ImpersonatorID`, `SSEEventID` changed from `type X string` to `brandid.ID[brand, string]` — phantom-typed with `.Get()`, `.IsZero()`, `.Equal()`. `ImpersonatorID = ActorID` (type alias — an impersonator IS an actor). Use `NewActorID("...")` / `NewSSEEventID("...")` constructors instead of casts. `.String()` now returns brand-prefixed form for debug; use `.Get()` for raw value. See ADR-0028.

### Fixed

- **Command ID minting bug** (CRITICAL): 7 of 20 usermgmt command constructors (`RegisterUserCmd`, `LinkExternalAccountCmd`, `UnlinkExternalAccountCmd`, `AddMemberCmd`, `UpdateMemberRolesCmd`, `RemoveMemberCmd`, `RegisterBotCmd`) returned a zero-value `cmdID`, silently breaking idempotency dedup and Watermill message UUIDs (which derive from `cmd.ID()`). All constructors now call `id.NewCommandID()`. Regression test added.
- **Idempotency `CheckAndRecord` atomicity**: The original free function called `Seen()` then `Record()` as two separate interface calls — a TOCTOU race. Fixed by moving `CheckAndRecord` into the `IdempotencyStore` interface; `MemoryIdempotencyStore` now does check+record under a single write lock. Proven by a 200-goroutine concurrency test (exactly 1 winner).
- **Idempotency memory leak**: `Seen()` now lazily deletes expired entries, preventing unbounded map growth when the sweep goroutine is disabled (`sweepInterval=0`).

## [3.2.0] - 2026-06-28

### Changed

- **catalog/ module merged into go-cqrs-lite** (**BREAKING**): The
  `github.com/larsartmann/cqrs-htmx/catalog/v3` module is deleted. Its
  single-service Builder facade and standalone HTTP handlers (D2, Health,
  EventCatalog) are now part of `go-cqrs-lite/catalog/v3` v3.2.0 (`simple` and
  `docserver` sub-packages). The redundant `OpenAPIHandler`/`AsyncAPIHandler`
  are removed — use the richer upstream `docserver.DocsServer` (HTML UIs,
  static assets, YAML+JSON). Migrate by swapping imports:
  `cataloghtmx.New` → `simple.New`, `cataloghtmx.Command[T]` → `simple.Command[T]`,
  `cataloghtmx.D2Handler` → `docserver.D2Handler`,
  `cataloghtmx.HealthCheckHandler` → `docserver.HealthCheckHandler`,
  `cataloghtmx.GenerateEventCatalog` → `docserver.GenerateEventCatalog`,
  `cataloghtmx.OpenAPIHandler(cat)` → `docserver.NewDocsServer(...).OpenAPISpec()`.
  See ADR 0020 for rationale.

### Fixed

- **ADR number collision** (0015): The catalog-merge ADR shared number 0015 with the identity-model-redesign ADR. Renumbered catalog-merge ADR → 0020. Cross-references in ADR 0008 and CHANGELOG updated.
- **Broken CHANGELOG link**: The v2.2.0 changelog entry linked to `catalog/README.md` (deleted by the merge). Replaced with a valid link to ADR 0008.
- **Stale status reports archived**: 51 status reports >2 weeks old moved to `docs/status/archive/`. Reduces noise from 96 → 45 visible reports.
- **Migration checklist added to AGENTS.md**: New "Module Deletion / Migration" gotchas (#22-26) encode the reference-sweep, ADR-collision-check, breaking-change-CHANGELOG, and write-once-report lessons. Prevents recurrence of the "code green but docs lie" failure mode.

## [3.1.0] - 2026-06-26

### Added

- **SQL-backed persistent read models**: `SQLUserReadModel`, `SQLMembershipReadModel`, `SQLTenantReadModel`, `SQLBotReadModel` — survive process restarts without rebuilding from the event journal. Each stores data as queryable scalar SQL columns + a JSON blob for full reconstruction. Dual SQLite and Postgres constructors (`NewSQLite*ReadModel` / `NewSQL*ReadModel`). Uses go-cqrs-lite `storage.SQLViewStore[V,K]` + `AutoMapperWithTombstone[V]`.
- **One-call stack presets**: `NewSQLiteEventSourcedSetup(cfg)` and `NewPostgresEventSourcedSetup(cfg)` — create event store, bus, repositories, SQL read models, and projections in a single call. Postgres preset supports multi-DB split (`EventDSN` / `QueryDSN`). Each returns a setup struct with `Close()`, `GracefulClose(ctx)`, and `Authz()`.
- **`OptimizeSQLiteDB(ctx, db)`**: Applies WAL mode, synchronous=NORMAL, busy_timeout=5000ms, 64 MB cache, temp_store=MEMORY, 256 MB mmap. 3-10x write throughput improvement. Call before creating stores. No-op for Postgres/MySQL.
- **`Service.Close()` / `Service.GracefulClose(ctx)`**: Full lifecycle shutdown — stops eviction goroutines, then closes bus and store (if they implement `io.Closer`). Idempotent.
- **`EventSourcedSetup.Close()`**: Closes the setup's bus and store. Idempotent.
- **CI gate apps**: `nix run .#coverage-gate` (fails if coverage drops below thresholds: root 90%, usermgmt 75%, catalog 90%) and `nix run .#errorfamily` (verifies zero stdlib error constructors).
- **Restart-survival test**: Proves read model persistence across database close/reopen via journal replay.

### Changed

- **go-cqrs-lite upgraded to v3.1.0** across all 7 modules. Adds SQL-backed view stores (`storage.SQLViewStore`), `AutoMapper`/`AutoMapperWithTombstone`, `stack/sqlite` and `stack/postgres` one-call presets, typed command/query metadata (`event.CustomData[K]`), multi-DB split presets, and SQLite WAL tuning. Zero breaking changes over v3.0.0.
- **go-error-family upgraded to v0.5.1** across all modules. `ErrDispatchFailed` is now natively classified (`event.NewTransient`) — the old `sync.Once` + `RegisterClassification` machinery was removed.
- **Coverage**: root 95.4%, usermgmt 79.5% (was 74.7%), catalog 95.3%. 697 usermgmt tests (was ~500).
- **`flake.nix` package version**: Updated from `2.4.0` → `3.1.0`.

### Fixed

- **Coverage CI gate was silently always passing**: `bc` was missing from `runtimeInputs` and `go test` stdout leaked into the coverage variable, corrupting the `bc` comparison. Fixed: added `pkgs.bc`, fully suppressed stdout.
- **foldUser now errors on unknown events**: Previously silently ignored. Now returns a `Rejection` error, matching upstream's strict event handling.
- **SSEStream race condition**: Fixed concurrent access to stream state during Send/Close.
- **Sweeper double-close panic**: Fixed panic when periodic eviction tried to close an already-closed store.
- **CSRF plaintext HTTP origin bypass**: Hardened — defaults to off unless explicitly trusted via `TrustedProxies`.
- **actorKindFromString validates input**: Previously accepted any string; now rejects unknown actor kinds.

## [3.0.0] - 2026-06-22

### Added

- **`migrations/0001_user_events_to_events.sql`**: Migration script for deployments that used the pre-delegation `user_events` table. Renames `user_events` → `events`, adds the upstream columns (`schema_version`, `payload_encoding`, `created_at`), and backfills them with defaults (`schema_version = 1`, `payload_encoding = 'json'`, `created_at = occurred_at`). Idempotent via `IF NOT EXISTS` on columns and a guard on the old table name.
- **Branded types for `AuditEntry`**: `AggregateID` is now `id.AggregateID`, `UserID` is now `usermgmt.UserID`, and `EventType` is now `event.Type`. Prevents accidental cross-assignment with other string-typed IDs. **Breaking:** `AuditLog.EntriesFor` now takes `id.AggregateID` instead of `string` — convert via `id.ParseAggregateID(user.ID.Get())`. JSON wire format: non-zero IDs still serialize as strings; zero-value `UserID` now serializes as `null` instead of `""`.
- **`SSEEventID` branded type**: A `type SSEEventID string` that wraps the `Last-Event-ID` / `id:` SSE field. Constructed via `NewSSEEventID` (no validation) or `ParseSSEEventID` / `MustParseSSEEventID` (rejects newlines/carriage returns that would corrupt the wire format). Has `String()` and `IsZero()` methods. **Breaking:** `SSEEvent.ID` is now `SSEEventID`, `SSEStream.LastEventID()` returns `SSEEventID`, `LastEventIDFromRequest` returns `SSEEventID`, `ReplayEvents` takes `SSEEventID`. The low-level `SSEEventStore.EventsAfter(string)` interface is unchanged — pass `.String()` at the boundary.
- **`JSONKeyError` / `JSONKeyStatus` constants in root**: Exported JSON map-key constants (`"error"`, `"status"`) for consistent error/status response shapes within the root module. Note: usermgmt and catalog are independent Go modules and cannot import these — they retain their own local `errorKey`/`statusKey` constants with matching values. True cross-module unification would require extracting a shared `jsonkeys` sub-module, which is not worth the added module complexity for two string constants.

### Changed

- **`SQLEventStore` is now a type alias** over `go-cqrs-lite/storage/v3`'s `storage.SQLEventStore`. The hand-rolled 413-LOC store was replaced by a 78-LOC facade. The upstream store is a strict superset: richer schema (`schema_version`, `payload_encoding`, `created_at`), `event.SeekableJournal` / `BackwardsSource` conformance, OpenTelemetry tracing, and `event.WrapInfrastructure` error wrapping.
- **go-error-family upgraded to v0.5.0** across all modules. `MapError` now delegates to upstream `Family.HTTPStatus()` — the hand-rolled `familyStatus()` switch (E04) is deleted. The upstream method returns identical status codes (Rejection→400, Conflict→409, Transient→503, Corruption→500, Infrastructure→503), confirming the ADR-0017 reconciliation was correct. The upgrade is now applied uniformly (root, usermgmt, catalog, integration_test) — the initial upgrade had left usermgmt and catalog stranded on v0.4.0.

### Fixed

- **HTTP status mapping reconciled with upstream `go-error-family`** (ADR-0017). `Corruption` errors now return **500** (was 422) — data integrity breaks are server-side, not client input errors. `Infrastructure` errors now return **503** (was 500) — the correct "service unavailable, retry later" semantic. Panics still return 500 via explicit override (a panic is a server bug, not service-unavailable). **Breaking:** any client checking for 422 on corruption or 500 on infrastructure must update to 500 and 503 respectively.
- **`usermgmt.UserID` unified with `id.UserID`** (ADR-0018, supersedes ADR-0002). `usermgmt.UserID` is now a type alias of `id.UserID` from go-cqrs-lite, backed by `ulid.ULID` — eliminating the string-backed/ULID-backed type split that forced manual `.String()`→`NewUserID()` conversion at every boundary. `NewUserID(s)` now accepts any string: valid ULIDs pass through directly, non-ULID strings are deterministically hashed to valid ULIDs (backward compat for tests). `MustParseUserID` added for strict ULID validation in production code. `.Get()` now returns `ulid.ULID` instead of `string` — use `.Get().String()` at string boundaries (SQL, Casbin, logging). **Breaking:** consumers storing or transmitting UserID values as raw strings must use `.Get().String()`.
- **Broadcaster send-on-closed-channel panic** (`fanout.go`): `Broadcast()` previously snapshted subscriber channels under RLock, released the lock, then iterated — a concurrent `Unsubscribe()` could `close()` a channel mid-iteration, panicking the entire process. Fixed by holding the RLock during the non-blocking fan-out (race-safe: `Unsubscribe()` cannot acquire the write lock to close a channel during iteration). Regression test added.

### Breaking Changes

- **`SQLEventStore.Close()` no longer closes the `*sql.DB`.** The delegated upstream store uses a borrowed DB handle — callers now own its lifecycle. **Migration:** any code that called `store.Close()` to clean up the DB must now call `db.Close()` separately:

  ```go
  // Before (v2.5.0):
  store, _ := usermgmt.NewSQLEventStore(ctx, db, "postgres")
  defer store.Close() // closed db too

  // After (v3.0.0):
  store, _ := usermgmt.NewSQLEventStore(ctx, db, "postgres")
  defer store.Close() // marks store closed; db stays open
  defer db.Close()    // caller owns the DB lifecycle
  ```

  Rationale: shared-DB usage (event store + session store on one `*sql.DB`) was already broken under the old behavior — `store.Close()` would close the DB out from under the session store. The upstream library's "borrowed handle, caller owns lifecycle" philosophy is `database/sql` best practice and is correct for shared-DB scenarios. No backward-compat wrapper is provided because silently closing the DB was itself a bug for shared-DB consumers.

- **`SQLEventStore.Load()` on empty aggregate** now returns `event.ErrAggregateNotFound` instead of an empty slice. The decider's `Repository` handles this transparently (returns `Initial` state), so consumers going through `Service` are unaffected. Direct callers of `store.Load()` should check `errors.Is(err, event.ErrAggregateNotFound)` if they need to distinguish "no events yet" from a real error.
- **MySQL dropped for event store.** `NewSQLEventStore(ctx, db, "mysql")` now returns an error — `go-cqrs-lite/storage` has no MySQL dialect. `SQLSessionStore` retains MySQL support (it manages its own simpler schema). Migration path: switch to Postgres or SQLite for the event store.

## [2.5.0] - 2026-06-18

### Added

- **StructuredError**: RFC 7807-shaped transport-agnostic error payload (`type`, `title`, `status`, `detail`, `instance`). `NewStructuredError(err, r)` maps via `MapError` and extracts request ID. `JSON()` method for SSE/WS data serialization. Used uniformly across SSE and WS error channels.
- **BroadcastOnError / BroadcastOnErrorFunc**: Symmetric to `BroadcastOnSuccess`. `AfterDispatchHook` factories that fire when `err != nil`, broadcasting a `StructuredError` JSON event. Closes the SSE error-reporting gap — clients now learn when commands fail.
- **SSEStream.Heartbeat(ctx, interval)**: Sends SSE comment-frame pings (`: keepalive\n\n`) on a ticker. Prevents reverse proxies (Nginx, Cloudflare, AWS ALB) from killing idle SSE connections after 30–60s of silence.
- **SSEStream.OnDisconnect(fn)**: Registers cleanup callbacks fired on `Close()`. Enables metrics, logging, session deregistration.
- **WriteWSMessage / WriteWSMessageInto[T]**: Outbound WebSocket message encoders. Counterparts to `ParseWSMessage` / `ParseWSMessageInto[T]`. Round-trip verified.
- **WSBroadcaster**: Thread-safe fan-out for WebSocket messages. Mirrors SSE `Broadcaster` API — `Subscribe` / `Unsubscribe` / `Broadcast` / `SubscriberCount`. O(1) unsubscribe via channel pointer identity. Buffered channels (64).
- **BroadcastOnSuccessWS / BroadcastOnErrorWS**: `AfterDispatchHook` factories for `WSBroadcaster`. WebSocket equivalents of SSE `BroadcastOnSuccess` / `BroadcastOnError`.
- **DispatchWSCommand / DispatchWSQuery**: WebSocket→CQRS dispatch bridge. `App.DispatchWSCommand(r, type, decoder, data)` decodes raw WS message bytes, runs lifecycle hooks, dispatches the command, and returns the error. `DispatchWSQuery` returns `(result, error)`. Does NOT handle auth/CSRF/response-writing — WS is authenticated at upgrade time. `DecodeWSJSON[T]` / `DecodeWSJSONQuery[T]` decoder factories mirror HTTP `DecodeJSON[T]` but operate on `[]byte`.
- **ADR 0010**: Documents transport parity decisions — StructuredError (RFC 7807), BroadcastOnError symmetry, WSBroadcaster mirroring SSE Broadcaster, and WS dispatch bridge design. Closes the SSE↔WebSocket feature gap.
- **WSBroadcaster benchmarks**: `BenchmarkWSBroadcasterBroadcastStress` and `BenchmarkWSBroadcasterConcurrentSubscribe` mirror the SSE Broadcaster benchmarks. Validated fan-out scaling (1 sub = 1.5μs, 1000 subs = 174μs).
- **Generic fanOut[T]**: Extracted the shared subscriber hub from SSE `Broadcaster` and WS `WSBroadcaster` into a single generic `fanOut[T any]` type. Both broadcasters embed it, gaining Subscribe/Unsubscribe/Broadcast/SubscriberCount via method promotion. Eliminates ~100 lines of copy-pasted code. Zero breaking changes — identical public API. `channelPtr` helper moved to `fanout.go` (was misplaced in `sse_broadcaster.go` despite being transport-agnostic).
- **WS Func variants**: Added `BroadcastOnSuccessWSFunc(fn)` and `BroadcastOnErrorWSFunc(fn)` for dynamic message generation, closing the parity gap with SSE's `BroadcastOnSuccessFunc` / `BroadcastOnErrorFunc`.
- **Unified timeout**: Removed `dispatchTimeout` split brain. WS dispatch now calls `timeoutCtx(ctx, nil)` — the same method the HTTP path uses, made nil-safe for `*handlerConfig`.
- **catalog sub-package** (5th Go module): Automatic API documentation generation from Go CQRS types. Produces OpenAPI 3.0, AsyncAPI 3.0, D2 diagrams, and EventCatalog MDX file trees from a single registration. Zero dependency on root or usermgmt modules — consumers opt in via `go get github.com/larsartmann/cqrs-htmx/catalog/v3`. Includes HTTP handlers (`OpenAPIHandler`, `AsyncAPIHandler`, `D2Handler`, `GenerateEventCatalog`), JSON/YAML output via `WithFormat()`, schema auto-derivation from struct tags, and catalog validation (`Build()` panics, `BuildValid()` returns violations). See [ADR 0008](docs/adr/0008-catalog-sub-package.md). _(Note: this module was later merged upstream into `go-cqrs-lite/catalog/v3` — see ADR 0020.)_
- **ADR 0009**: Documents the rationale for which go-cqrs-lite modules are used (8 direct) vs not used (13 excluded), with specific reasons for each exclusion. Cross-references the middleware integration guide.
- **Middleware integration guide** (`docs/integrations/go-cqrs-lite-middleware.md`): Documents how go-cqrs-lite dispatch middleware (retry, circuit breaker, metrics, tracing) composes with cqrs-htmx HTTP middleware. Different layers, no conflict.
- **`ForbiddenErrorHandler`**: Exported named CSRF error handler for `CSRFConfig.ErrorHandler` — writes HTTP 403 with no body. Convenient default for consumers wiring CSRF protection without rendering nosurf's default error page.
- **catalog-demo example**: `examples/catalog-demo/` — standalone server generating live OpenAPI, AsyncAPI, and D2 docs from Go struct tags via the catalog module. Includes graceful shutdown and `ReadHeaderTimeout` hardening.
- **`test-race` / `test-fuzz` nix apps**: `nix run .#test-race` runs the race detector across all four Go modules; `nix run .#test-fuzz` discovers and runs every `Fuzz*` target (default 30s each, overridable via `FUZZTIME`).

### Fixed

- **`applyConfigDefaults` silent field loss**: The `HandlerConfig` defaults logic created a fresh config and manually copied each field — any field not copied was silently zeroed. `WebAuthnRateLimit` was missing, silently disabling rate limiting on all 4 WebAuthn endpoints. Refactored to start from the caller's config and override only fields needing non-zero defaults, eliminating the entire class of "forgot to copy a field" bugs.
- **Removed lying `EventCatalogHandler`**: The handler set `Content-Type: application/zip` but served a JSON file listing — a critical honesty violation. Removed entirely; kept `GenerateEventCatalog()` (build-time/startup-time file generation, which is the correct EventCatalog model).
- **Fixed `flake.nix` split brain**: `nix run .#test` (and `.#lint`/`.#build`/`.#coverage`) silently skipped the catalog module. Catalog is now included in all four multi-module nix apps.
- **Removed self-referential `replace` directive** in `catalog/go.mod` (`replace github.com/larsartmann/cqrs-htmx/catalog/v3 => ./`) — pointed the module at itself.
- **Bumped `flake.nix` `packages.default` version** from stale `2.3.0` to `2.4.0` to match the current release.
- **Fixed `errors.Join` anti-pattern**: Replaced `errors.Join(errors.New("..."), err)` with idiomatic `fmt.Errorf("...: %w", err)` for proper error wrapping.
- **Fixed 7 lint issues** in the catalog module: errchkjson, exhaustive, exhaustruct×3, forcetypeassert, gci. All modules report 0 lint issues.
- **Removed duplicate import** in `catalog/example_test.go` (same package imported twice with different aliases).

## [2.4.0] - 2026-06-17

### Changed

- **go-cqrs-lite v2.4.0**: All modules upgraded to the latest per-module tags (command, event, id, query, codec, decider, dispatcher, memory, projection, snapshot, otel).
- **go-error-family v0.4.0**: Upgraded with idiomatic `event.Wrapf` / `event.WrapTransient` replacing all `NewTransient(fmt.Sprintf(...)).WithCause(err)` patterns.
- **go-branded-id v0.3.1**: Latest branded ID library.
- **ginkgo v2.31.0 + gomega v1.42.0**: Latest BDD test framework.
- **Lint: 0 issues**: Fixed all 16 pre-existing lint warnings (errorlint, exhaustruct, contextcheck false positives).
- **Email validation consolidated**: `ImportUser.Validate()` now delegates to `ParseEmail()` instead of reimplementing identical logic.

### Security

- **Import/export authorization**: Endpoints now require admin role by default. Previously any authenticated user could export all user data or create accounts.
- **TOTP disable protection**: `DisableTOTP` now requires a valid code, preventing MFA stripping if a session is hijacked.
- **Per-endpoint rate limiting**: `HandlerConfig.ImportRateLimit`, `TOTPRateLimit`, `VerificationRateLimit` for per-IP rate limiting.

### Changed

- **TOTP library**: Replaced hand-rolled HMAC-SHA1 with `pquerna/otp/totp`.
- **DisableTOTP signature**: Now takes a code parameter.
- **ExportFormat → UserDataFormat**: Honest naming for dual import/export use.
- **Email value type**: `ParseEmail`/`MustParseEmail` for centralized validation. Used in `ExportUser`.
- **withTimeout helper**: Eliminates duplicated context-timeout boilerplate.

### Added

- **TOTP multi-factor authentication**: RFC 6238 TOTP via `pquerna/otp/totp` library. Events: `TOTPEnabled`/`TOTPDisabled`. Two-phase setup: `EnableTOTP` → pending secret → `VerifyTOTPSetup` confirms. Generates `otpauth://` URIs for QR codes. `Service.VerifyTOTP` validates codes as second factor. `DisableTOTP` requires a valid code to prevent MFA stripping.
- **Email verification flow**: Token-based email confirmation with `EmailVerified` event. `SendVerificationEmail` generates token (configurable TTL, default 24h) with optional SMTP callback. `VerifyEmail` consumes single-use token. Email change resets verification status.
- **User import/export**: `Service.ImportUsersFromJSON`/`ImportUsersFromCSV` for batch user creation (skips existing emails). Input validation via `ImportUser.Validate()`. `Service.ExportUsersToJSON`/`ExportUsersToCSV` for data export. Admin-only by default via `ImportExportAuthorizer`. Per-IP rate limiting via `HandlerConfig.ImportRateLimit`.
- **Virtual authenticator integration test**: Full WebAuthn registration → login ceremony using W3C spec test vectors. Tests: successful full flow, registration replay rejection, login with wrong challenge.
- **SQL event store**: `SQLEventStore` implements `event.Store` + `event.Journal` for Postgres/SQLite/MySQL. Auto-migrates schema, supports optimistic concurrency, parameterized queries per dialect.
- **Audit log projection**: `AuditLog` records all user events as `AuditEntry` structs. Queryable by aggregate ID, recent N, or total count. Optional via `ServiceConfig.AuditLog`.
- **Rate-limited registration**: Per-IP fixed-window rate limiting on the registration endpoint. Configurable via `HandlerConfig.RegistrationRateLimitConfig`.
- **Session rotation on privilege change**: `UpdateRoles` deletes all user sessions after role change, forcing re-authentication.
- **Credential listing pagination**: `GET /auth/credentials?page=N&page_size=N` with configurable page size (default 20, max 100).
- **Structured logging in WebAuthn ceremonies**: All 4 ceremony methods log at debug/info/warn with user_id, email, error context.
- **CI pipeline (GitHub Actions)**: Multi-module CI with separate lint jobs for root and usermgmt, race detector, `GONOSUMCHECK` env vars, concurrency group.
- **doc.go**: Comprehensive root package documentation with Quick Start, middleware stack, response builder, error mapping, SSE, and submodule reference.
- **Property-based testing**: 8 rapid-based property tests for `foldUser` invariants.
- **Benchmarks**: 5 benchmarks for `foldUser`, `ReadModel.Handle`, `BeginRegistration`, `BeginLogin`.
- **Fuzz tests**: 3 fuzz tests for WebAuthn ceremony inputs and credential ID decoding.
- **Coverage tests**: 15 tests covering error paths, schema version, RegisterCommands, DefaultEventSourcedSetup, writeJSON, HTTP handlers.

- **AccountLockout wired into WebAuthn login**: `BeginLogin` checks lockout status; `FinishLogin` records failures and resets on success.
- **Credential management HTTP endpoints**: `GET /auth/credentials` (list), `DELETE /auth/credentials/{id}` (remove by base64url ID).
- **WebAuthn session proactive eviction**: Background goroutine (`webauthnEvictionInterval`) periodically removes expired challenge sessions. `Service.Stop()` terminates the goroutine.
- **CasbinProjection subscribes to CredentialAdded/Removed**: Ensures projection ordering without affecting policies.
- **`Service.Stop()` method**: Gracefully shuts down background resources (WebAuthn eviction goroutine). Safe to call multiple times.
- **`Service.Authz()` accessor**: Returns the underlying `*Authz` for direct policy queries.
- **Service source propagation**: `Config.ServiceName` (string) and `App.EventOptions(ctx)` inject `event.WithSource` into event options. `EventOptionsFromContextWithSource(ctx, name)` is the free-function equivalent for callers that don't hold an `*App`.
- **Typed-query examples in root**: `ExampleApp_Query_typedRegister` and `ExampleApp_Query_typedDispatch` demonstrate the `query.RegisterTyped[T]` / `query.DispatchTyped[T]` pattern crossing module boundaries.
- **`ExampleBroadcaster_BroadcastOnSuccessFunc`**: Documents the dynamic-event `AfterDispatchHook` factory.
- **`ExampleConfig_BeforeDispatch` / `ExampleChain` (tracing)**: Show timing/span integration via the lifecycle hooks; no OpenTelemetry dependency required.
- **Per-module nix apps**: `nix run .#test-root`, `nix run .#test-usermgmt`, `nix run .#test-integration`, `nix run .#build-datastar-demo` set `GOWORK=off` automatically — no more per-module `cd` dance.
- **UpdateTodo command in datastar-demo**: `UpdateTodoCmd` + `TodoUpdatedPayload` + `handleUpdateTodo` + `POST /api/todos/update` route + inline edit input in the UI. The read-model `Projector.Apply` now handles `TodoUpdated` and the broadcaster renders it as `todo_updated`.
- **`GetStatsQry` typed query in datastar-demo**: `Stats` struct + `GetStatsQry` + `renderStatsFromQuery` helper. Wired into `handleListTodos` and `handleEventStream` so the stats read goes through the typed query dispatcher (caching/authorization-friendly).
- **SSE replay endpoint in datastar-demo**: `GET /api/events/replay` reads `Last-Event-ID` and replays missed events from the in-memory event store.
- **`FuzzEventOptionsFromContext`**: Fuzzes `EventOptionsFromContext` with arbitrary combinations of (deadline, cancel, user ID, correlation ID, request ID) — verifies the bounded-options invariant (≤ 4 options).
- **Real-server integration tests**:
  - `TestRateLimiter_RealServer_AllowsThenBlocks` + `TestRateLimiter_RealServer_ConcurrentRequests`: end-to-end rate limiting over `httptest.NewServer` with a real `http.Client`.
  - `TestSSE_RealServer_ReconnectionWithLastEventID` + `TestSSE_RealServer_ReconnectionNoLastID`: end-to-end SSE reconnection with real `Last-Event-ID` header.
  - `TestTypedQueryDispatch_CrossModule` + `TestCrossModule_PaginationFlow`: cross-module CQRS + pagination.
- **Benchmarks**:
  - `BenchmarkDecodePagination` (8 URL shapes: no params, with page, with size, with both, with extras, invalid, zero, huge).
  - `BenchmarkCommandRegisterTypedVsRegister` (typed ≈ manual: 57.4 vs 56.9 ns/op).
  - `BenchmarkQueryDispatchTypedVsDispatch` (typed ≈ manual: 85.7 vs 81.0 ns/op).
- **usermgmt service transient-error context**: `withUserIDContext` helper annotates every `event.NewTransient` in the service with `user_id` context (no PII — only the user ID, not email/display name). New `transientErr(userID, msg, cause)` helper keeps service methods concise.
- **ADR 0005**: Documents the go-cqrs-lite v2.3.0 adoption decisions (IsZero validation, FromContext deadline propagation, RegisterTyped/DispatchTyped, replace-directive removal).

### Changed

- **Eliminated production panics**: `marshalPayload` and `aggIDFromUser` now return errors instead of panicking. All callers updated to handle errors gracefully.
- **RegisterCommands returns error**: Silently ignored command registration failures now propagate to `NewService`.
- **bus.Subscribe failures logged**: Event bridge subscription errors now logged at warn level instead of silently ignored.
- **Projection runner errors logged**: Background projection goroutine errors now logged at error level.
- **WebAuthn HTTP refactor**: Finish endpoints use query params (`?user_id=X`) instead of fragile double body read pattern.
- **Coverage**: 86.2% usermgmt, 96%+ root, zero 0% functions.

### Fixed

- **Goroutine leak**: All WebAuthn test services now call `t.Cleanup(svc.Stop)` to terminate eviction goroutines.
- **Stale password references**: Removed all `"password":"secret12"` from test helpers (passwordless migration).
- **gosec G101**: Credential event/command type constants now have proper nolint annotations.
- **wrapcheck**: `marshalPayload` and `Dispatch` errors are now properly wrapped.

## [2.3.0] - 2026-06-13

### Changed

- **go-cqrs-lite upgraded to v2.3.0** across all modules (root, usermgmt, integration_test, datastar-demo). Per-module tags now published — no go.work replace directives needed.
- **Empty command/query type validation**: `App.Command("")` and `App.Query("")` panic at handler registration time using `command.Type.IsZero()` / `query.Type.IsZero()` — fail-fast instead of silent empty dispatch.
- **Deadline propagation**: `EventOptionsFromContext` now propagates context deadline to event options via `event.FromContext(ctx)` when present. Downstream events inherit request timeouts.
- **Type-safe handlers in datastar-demo**: Refactored all 3 command handlers from manual type assertions to `command.RegisterTyped[T]` — eliminates `cmd.(*CreateTodoCmd)` boilerplate.
- **Local MustParse wrappers**: Reimplemented `MustParseUserID`, `MustParseCorrelationID`, `MustParseRequestID` locally (go-cqrs-lite removed `id.MustParse[T]`). Preserves API compatibility for consumers.
- **query.MustNew removed upstream**: Updated `datastar-demo` to use `query.New()` + error check.
- **Code deduplication**: Eliminated all clone groups at threshold 50 (industry standard). At threshold 30, remaining groups are all 2–7 line spans in test code. −104 lines net across 14 source files.
- **goconst warnings eliminated**: Extracted test constants in `sse_test.go` (`eventTodoCreated`, `eventUpdate`, `eventItem`, `dataFirst`) and `coverage_test.go` (`aliceName` usage). Added `goconst` exclusion for `example_test.go` (self-contained examples should not reference test constants).
- **nestif warning fixed**: Extracted `parseWSHeaders` helper from `ParseWSMessageInto` in `ws.go`, reducing nesting complexity from 7 to 1.
- **Test deduplication**: Deleted 6 duplicate ClientIP tests from `coverage_test.go` (already covered by `httputil_test.go`). Merged 3 `sanitizeRedirectURL` DescribeTables into one. Extracted `queryNamedResultHandler` helper for query-result fixtures.
- **`ws_test.go` coverage improved**: `ParseWSMessageInto[T]` from 86.7% → 93.3% by exercising the type-assertion failure path and non-string header values.
- **`usermgmt.UpdateRoles` refactor**: Extracted `transientErr` helper to keep the function under the 60-line `funlen` limit. Replaces inline `withUserIDContext(event.NewTransient(...).WithCause(err), userID)` boilerplate.

## [2.1.0] - 2026-06-08

### Added

- **SSE (Server-Sent Events) support**: Full SSE implementation for the HTMX SSE extension (`hx-ext="sse"`). `SSEStream` manages a single connection with correct headers, flush, and context-aware lifecycle. `SSEEvent` struct with `Event`, `Data`, `ID`, `Retry` fields. `WriteSSEEvent` for writing events to any `io.Writer`.
- **SSE Broadcaster**: Thread-safe fan-out `Broadcaster` with O(1) unsubscribe via channel identity, buffered channels (64), and non-blocking broadcast (drops to slow consumers). `SubscriberCount()` for monitoring.
- **SSE reconnection**: `LastEventIDFromRequest(r)` extracts `Last-Event-ID`. `SSEEventStore` interface and `ReplayEvents(stream, store, lastID)` for full SSE spec reconnection support.
- **SSE + CQRS bridge**: `BroadcastOnSuccess(event, data)` and `BroadcastOnSuccessFunc(fn)` — `AfterDispatchHook` factories that broadcast SSE events on successful command dispatch.
- **WebSocket message parser**: `ParseWSMessage(data)` parses HTMX WebSocket JSON into `WSMessage` with separated `Headers` and `Body` fields. `StringBody(key)` for typed string access.
- **Typed WebSocket parser**: `ParseWSMessageInto[T](data)` — generic typed parser that deserializes body into struct T while separating HEADERS. Compile-time safe.
- **WebSocket OOB HTML**: `WSOOBHTML(id, html, strategy)` wraps HTML with `hx-swap-oob` attributes for out-of-band swaps. Uses `SwapStrategy` type.
- **Pagination**: `DecodePagination(r)` extracts page/page_size from query params, delegates to `query.NewPagination` for defaults (page=1, page_size=20, max=100). `RenderPaginatedJSON[T]()` renders `query.PaginatedResult[T]` as JSON 200.
- **Embedded HTMX JavaScript**: `HTMXScriptHandler()` serves embedded HTMX v2.0.9 (minified, ~49KB) with `Content-Type`, long-lived `Cache-Control` (1 year, immutable), `ETag`, and `If-None-Match` support for 304 responses. `HTMXScriptTag(path)` generates `<script>` tags. `HTMXVersion()` returns `"2.0.9"`. Opt-in, zero CDN dependency.

### Changed

- **go-cqrs-lite upgraded to v2.2.0** across all modules. Adopts `query.Pagination` and `query.PaginatedResult[T]` from upstream.
- **httputil upgraded to v0.1.0** (→ v0.1.1 pending).
- Modernized Go idioms and normalized table formatting across codebase.

## [2.0.0] - 2026-05-27

### Changed

- **CSRF: replaced `gorilla/csrf` with `justinas/nosurf`**: Simpler API, no secret management required, no HMAC/securecookie. Token generation uses `crypto/rand` with origin validation instead. `CSRFConfig.Secret` field removed. `RotateCSRFToken` removed (nosurf rotates automatically). Custom header/field translation via `translateCSRFHeaders`.
- **Error handling: replaced `cockroachdb/errors` with `go-error-family`**: Error classification now uses `github.com/larsartmann/go-error-family` via `go-cqrs-lite/core/event`. `sync.Once` registers sentinels with `event.RegisterClassification`.
- **`ClientIP` delegated to `larsartmann/httputil`**: `httputil.ClientIP(r)` replaces the local implementation.
- **go-cqrs-lite/core upgraded to v1.5.1**: `CatalogEntry` → `HandlerMeta` for API compatibility.

### Added

- **Panic recovery middleware**: `RecoveryMiddleware` recovers from panics, logs stack traces via `slog.ErrorContext`, and writes 500. `App.RecoveryMiddleware()` uses the App's configured `ErrorHandler` for consistent error formatting (JSON, redirects, request ID correlation). `http.ErrAbortHandler` is re-raised without recovery (Go net/http convention).
- **JSON result rendering**: `RenderJSON[T]()` renders query results as JSON with 200 OK. `RenderJSONStatus[T](status)` renders with a custom status code (e.g., 201 Created). Both include runtime type assertion for compile-time documentation.
- **Request ID in error responses**: `JSONErrorHandlerWithRedirect` now includes `"request_id"` field when a `RequestID` is present in context. `DefaultErrorHandlerWithRequestID` and `DefaultErrorHandlerWithRedirectAndRequestID` prefix plain-text errors with `[request_id: RID]`.
- **Config field**: `IncludeRequestIDInErrors` — when `true` and no custom `ErrorHandler` is set, the default handler automatically includes request IDs in error responses.
- `CorrelationID` type alias (`type CorrelationID = id.CorrelationID`) with `NewCorrelationID()`, `ParseCorrelationID()`, `MustParseCorrelationID()` helpers
- E2E test verifying event.NewEvent accepts options from EventOptionsFromContext
- `ContextEnrichmentMiddleware` now validates `X-Correlation-ID` as ULID; non-ULID values are silently dropped

### Changed

- **`WithCorrelationID` / `CorrelationIDFromContext`**: context now stores strongly-typed `id.CorrelationID` (ULID-backed branded type) instead of `string`. **Breaking change for consumers** passing raw strings — use `MustParseCorrelationID()` in tests, `NewCorrelationID()` to generate, or `ParseCorrelationID()` in production.
- `AuthorizeMiddleware` now prefers branded `UserID` from context over raw extractor string; falls back to extractor + `ParseUserID()` validation. Unparseable ULIDs now return 401 instead of passing raw strings to Casbin.
- Context keys are now empty-struct sentinel types (`userIDKey{}`, `correlationIDKey{}`, `htmxKey{}`) instead of string-based types — standard Go pattern for collision-free context values.

### Fixed

- `CorrelationIDFromContext` → `EventOptionsFromContext` → `event.WithCorrelationID()` pipe now fully wired with branded `id.CorrelationID` type
- Dead test: "returns error when casbin enforce fails" now actually tests failure (asserts error instead of asserting success)
- `EventOptionsFromContext` now propagates `CorrelationID` alongside `UserID` into event metadata

## [1.0.0] - 2026-05-16

### Added

- BDD test suite using Ginkgo/Gomega (`bdd_test.go`)
- `DecodeFormQuery` handler option for query parameter form decoding (symmetry with `DecodeForm`)
- `docs/` directory with architecture reviews, planning docs, and status reports
- `NewUserID()`, `ParseUserID()`, `MustParseUserID()` helpers; `type UserID = id.UserID` re-export
- `NotificationLevel` type with `LevelSuccess`, `LevelError`, `LevelWarning`, `LevelInfo` constants
- `JSONErrorHandlerWithRedirect` for JSON error responses with custom login redirect
- Dispatch lifecycle hooks: `BeforeDispatchHook` / `AfterDispatchHook` on `Config`
- Request validation: `ValidateCommand` / `ValidateQuery` HandlerOptions with `ErrValidationFailed`
- Correlation ID propagation: `WithCorrelationID` / `CorrelationIDFromContext`
- Timeout propagation: `Config.Timeout` wraps dispatch with `context.WithTimeout`
- Godoc examples (7 `Example*` functions)
- Benchmark tests (10 sub-benchmarks)
- `CONTRIBUTING.md` contribution guide
- GitHub Actions CI pipeline (build + test + lint + coverage gate)
- `authMode` typed enum for handler authorization (makes impossible states unrepresentable)

### Changed

- **`WithUserID` / `UserIDFromContext`**: context now stores strongly-typed `id.UserID` (ULID-backed branded type) instead of `string`. `UserIDExtractor` still returns `string`; middleware parses to `UserID`. **Breaking change for consumers** passing string literals or plain strings — use `MustParseUserID()` in tests, `ParseUserID()` in production.
- Extract helper functions (`hasNoResponse`, `hasMinimalResponse`, `decodeJSONBody`, `decodeRequest`, `decodeFormBody`, `notifyOption`, `triggerNotification`) to reduce duplication
- Extract notification helpers to dedicated `notify.go` module
- Consolidate duplicate test types across test files into shared helpers
- Remove local-path `replace` directives from `go.mod` — resolve from GitHub
- Notification levels now use `NotificationLevel` type instead of magic strings
- Remove `headerTrue` alias — use `HeaderTrue` everywhere
- `"X-Correlation-ID"` header extracted to `headerCorrelationID` constant
- `JSONErrorHandler` now delegates to `JSONErrorHandlerWithRedirect` with default redirect
- Authorization config consolidated from 4 fields (`authorize bool` + `requireAuth bool` + `resource` + `action`) to typed `authMode` enum + `resource` + `action`

### Fixed

- Use `headerRedirect` constant instead of hardcoded `"HX-Redirect"` string in `DefaultErrorHandlerWithRedirect`
- Thread `Config.LoginRedirect` into per-App error handler (was dead code — `New()` now creates a closure that captures the resolved loginRedirect)
- Use `headerTrue` constant in `Response.Refresh()` instead of hardcoded `"true"`
- Fix README compile-breaking example (`cqrshtmx.LoginRedirect` → `Config.LoginRedirect`)
- Fix error wrapping: `errors.Wrapf` with `%s` on sentinels → `fmt.Errorf("%w: ...")` throughout
- `Enforce(nil, ...)` error now includes all three fields (subject, resource, action) — was missing subject
- `JSONErrorHandler` now respects `Config.LoginRedirect` (was hardcoded to `/login`)

### Removed

- Remove dead `enrichContext()` no-op stub
- Remove redundant gocritic `disabled-checks` entries (`dupImport`, `octalLiteral`, `whyNoLint` — already disabled by default)
- Remove unused `io` and `event` imports from test files
- Remove dead sentinels `ErrNoUserID` and `ErrRendererMissing` (exported but never returned by any code path)
- Remove deprecated `DefaultNotificationEvent` var (race risk, unexported constant used internally)
- Unexport internal sentinels: `ErrCommandsNil` → `errCommandsNil`, `ErrQueriesNil` → `errQueriesNil`, `ErrDecoderMissing` → `errDecoderMissing`

## [0.2.0] - 2026-05-07

### Added

- Eliminate all 103 golangci-lint issues → 0 issues, project is lint-clean
- `.golangci.yml` v2 format with proper exclusion rules
- Comprehensive test coverage at 93.5% (138 specs)

## [0.1.0] - 2026-05-04

### Added

- **App builder**: `App` struct with `Config`, `Command()`, `Query()`, per-App `ErrorHandler` and `LoginRedirect`
- **CQRS dispatch**: `handleCommandDispatch()` and `handleQueryDispatch()` with automatic error handling
- **Handler options**: `DecodeJSON`, `DecodeForm`, `Render`, `RenderTempl`, `RenderTemplResult`, `Authorize`, `Enforce`, `UserIDExtractor`
- **HTMX response builder**: Fluent API with `Response` struct — `StatusCode()`, `Header()`, `Redirect()`, `Refresh()`, `Retarget()`, `Reswap()`, `Trigger()`, `TriggerAfterSettle()`, `TriggerAfterSwap()`
- **HTMX middleware**: `HTMXMiddleware` parses `HX-*` headers once, stores `HTMXRequest` in context
- **HTMX context accessors**: `IsHTMXRequest()`, `GetHTMXPrompt()`, `GetHTMXTarget()`, `GetHTMXTrigger()`, `GetHTMXTriggerName()`, `RenderPartial()`
- **Notification system**: `NotifySuccess`, `NotifyError`, `NotifyWarning`, `NotifyInfo` — standard `{level, message}` trigger pattern via `notify.go`
- **Casbin authorization**: `Authorize()`, `Enforce()`, `AuthorizeMiddleware()` using `casbin/casbin/v3`
- **Context enrichment**: `ContextEnrichmentMiddleware` + `UserIDExtractor` → context → event metadata
- **Error classification**: CQRS error → HTTP status mapping with `RegisterClassification`, sentinel errors (`ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrBadRequest`, `ErrConflict`, `ErrInternal`), and `LoginRedirect` support
- **templ integration**: `TemplComponent` duck-typed interface (no `a-h/templ` import dependency) with `RenderTempl` and `RenderTemplResult` options
- **Middleware chain**: `Chain()` utility for composing `net/http` middleware
- **Git Town integration**: `git-town.toml` configuration

### Changed

- Deduplicate `UserIDExtractor` calls — handlers check context first, skip if middleware already set user ID
