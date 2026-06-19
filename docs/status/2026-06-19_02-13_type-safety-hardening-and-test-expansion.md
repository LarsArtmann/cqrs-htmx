# Status Report — 2026-06-19 02:13

**Session:** type safety hardening, test coverage expansion, migration tooling, documentation

---

## a) FULLY DONE

### New Features

- **`SSEEventID` branded type** (`sse_event.go`): `type SSEEventID string` with `ParseSSEEventID`/`MustParseSSEEventID` (rejects newlines/CR that would corrupt the SSE wire format) and `NewSSEEventID` (no validation). `SSEEvent.ID`, `SSEStream.LastEventID()`, `LastEventIDFromRequest`, and `ReplayEvents` all use the branded type. Prevents accidental cross-assignment with CorrelationID, RequestID, UserID, etc. The low-level `SSEEventStore.EventsAfter(string)` interface is unchanged for backward compatibility — pass `.String()` at the boundary. 10 new test cases (DescribeTable for valid/invalid IDs, IsZero, String, NewSSEEventID).
- **`JSONKeyError` / `JSONKeyStatus` constants** (`response.go`): Exported JSON map-key constants (`"error"`, `"status"`) in the root package. `JSONErrorHandlerWithRedirect` now uses them instead of string literals. Note: usermgmt and catalog are independent Go modules and cannot import these — they retain their own local `errorKey`/`statusKey` with matching values.
- **`AuditEntry` branded types** (`usermgmt/audit_log.go`): `AggregateID` is now `id.AggregateID`, `UserID` is now `usermgmt.UserID`, `EventType` is now `event.Type`. `EntriesFor` takes `id.AggregateID`. `UserID` uses `json:"user_id,omitzero"` (Go 1.24+ `omitzero` tag — leverages `IsZero()` method on branded type). Prevents cross-assignment with other string IDs.
- **Migration SQL** (`usermgmt/migrations/0001_user_events_to_events.sql`): Idempotent Postgres migration for deployments upgrading from `usermgmt < v2.5.0`. Renames `user_events` → `events`, renames `event_id` → `id`, adds `schema_version`/`payload_encoding`/`created_at` columns with backfill, creates upstream indexes. Includes `migrations/README.md` explaining when/how to run.

### New Tests

- **`FuzzSQLSessionStore_CreateFindRoundTrip`**: Fuzz target exercising Create→Find→Delete round-trip with arbitrary userID strings. Seeds include SQL injection (`'); DROP TABLE--`), unicode (Cyrillic, emoji), null bytes, whitespace, 1KB strings. Validates token/userID/timestamp integrity + idempotent Delete + ErrSessionNotFound after deletion. Validated across 14k+ executions.
- **`FuzzSQLSessionStore_DeleteByUserID`**: Fuzz target for batch delete by user. Seeds include injection attempts and edge cases. Validates both sessions removed + idempotent re-call. Validated across 40k+ executions.
- **6 `BenchmarkSQLSessionStore_*`**: Create (15μs), Find (5.6μs), FindMiss (4.4μs), Delete (11.3μs), DeleteByUserID (11.7μs), EvictExpired (63.6μs for 50 rows). All on SQLite in-memory.
- **`TestService_StoreWrapper_TransformationRoundTrip`**: End-to-end proof that a transforming `StoreWrapper` (XOR stand-in for encryption) round-trips correctly. Verifies: (1) projections see plaintext, (2) inner store holds ciphertext (payload bytes don't start with `{`), (3) user data matches after round-trip. This is the critical test for the encryption-at-rest seam — it proves the wrapper genuinely transforms, not just passes through.
- **6 WebSocket end-to-end integration tests** (`ws_end_to_end_integration_test.go`): Multi-subscriber broadcast on success, StructuredError fan-out on failure (5 subscribers), rapid sequential dispatch (20 commands), mid-stream subscriber joining (late subscriber only gets subsequent), concurrent dispatch (10 goroutines × 10 dispatches = 100), request context propagation (UserID through hooks).
- **`TestOAuth2StateStore_Consume_ConcurrentOneTimeUse`**: 50 goroutines race to consume the same state token. Asserts exactly 1 succeeds, 49 get `ErrOAuthInvalidState`. This is the CSRF replay-prevention guarantee under concurrent load.

### Code Quality Polish

- **`goto` removed** from WS concurrent test — replaced with generic `drainChannel[T]` helper.
- **ginkgolinter type-mismatch fixed** — `Equal(SSEEventID("x"))` → `BeEquivalentTo("x")` in `sse_reconnect_test.go`.
- **go.sum cleaned** — removed 46 lines of unused indirect deps (test reporting tools: `gkampitakis/go-snaps`, `mfridman/tparse`, `joshdk/go-junit`, etc.) via `go mod tidy`.

### Documentation

- **CHANGELOG `[Unreleased]`**: Full breaking-change documentation with migration code samples for `SQLEventStore.Close()`, `Load()` → `ErrAggregateNotFound`, MySQL drop, branded types, migration SQL, JSONKey constants.
- **AGENTS.md updated**: SSEEventID documented in SSE section, migration tooling + fuzz/bench coverage noted in SQL store delegation entry.

### Verification

| Metric                      | Value                                               |
| --------------------------- | --------------------------------------------------- |
| Test modules passing        | **4/4** (root, catalog, usermgmt, integration_test) |
| Race detector               | **Clean** across root + usermgmt + integration_test |
| Lint issues                 | **0** across all 3 linted modules                   |
| Root coverage               | **95.2%**                                           |
| usermgmt coverage           | **85.5%**                                           |
| Total tests/benchmarks/fuzz | **972**                                             |
| Commits this session        | **12** (all on master, all pushed)                  |
| LOC change                  | **+1101 / -77** (20 files changed, 5 new files)     |

---

## b) PARTIALLY DONE

### Branching-Flow Remediation (632 findings → unchanged from prior session)

All remaining findings are triaged as opinionated or false positives. No action taken this session. The metrics are stable.

### OAuth2/OIDC Integration Testing

- Existing: mock-provider happy path tests, state store CRUD tests, concurrent Consume test (new this session).
- **Still missing**: Full integration test with a real OIDC provider (Google/GitHub test credentials). Only mock providers are tested. The PKCE flow, OIDC discovery, and ID token verification paths need real-provider validation.

---

## c) NOT STARTED

### From prior sessions / open decisions

1. **`usermgmt.Service` has 27 fields** — flagged as a large struct. Should be split into focused sub-structs (auth config, store handles, webauthn config, oauth2 config). High impact for maintainability, medium effort.
2. **`handlerConfig` has 16 fields** — same issue. Consider grouping related options.
3. **No schema migration versioning** — the migration script (`0001_user_events_to_events.sql`) is manual. No `_schema_version` table tracking, no `migrate up/down` tooling.
4. **Catalog module** (`catalog/`) — OpenAPI/AsyncAPI/D2 doc generation. Has builder + serve handlers but no real consumer beyond the example.
5. **`event.SeekableJournal` usage** — the delegated `SQLEventStore` now supports seeking; not exposed for replay/backup.
6. **Strong-id for `TriggerID`** — `htmx.go:70` uses `string` for HTMX trigger IDs. Could be branded.
7. **`ExternalAccount.ClientID` is `string`** — could be a branded type. Flagged by strong-id.
8. **`usermgmt.Service` split brain check** — both `NewService` and `NewEventSourcedSetup` support the same hooks. Verify no divergence has crept in since the original split-brain fix.

---

## d) TOTALLY FUCKED UP

### Nothing currently broken.

- All tests pass (4/4 modules with `-race`).
- Lint is 0 across all modules.
- Build is clean.
- Working tree is clean. All 12 commits pushed.

### Risk areas (not broken, but fragile)

1. **`SQLEventStore.Close()` behavior change** — documented in CHANGELOG with migration code. No backward-compat wrapper. Consumers upgrading from `< v2.5.0` who call `store.Close()` expecting DB cleanup will leak connections. Migration script + CHANGELOG entry are the mitigation.
2. **`AuditEntry` JSON wire format change** — zero-value `UserID` now serializes as `null` instead of `""` (due to `omitzero` tag). Consumers parsing audit JSON with strict string expectations for `user_id` may break. Documented in CHANGELOG.
3. **`SSEEventID` type change is breaking** — `SSEEvent.ID` is now `SSEEventID` not `string`. Consumers constructing `SSEEvent{ID: "42"}` must now write `SSEEvent{ID: cqrshtmx.NewSSEEventID("42")}` or `SSEEvent{ID: "42"}` (implicit conversion works for string literals). Documented in CHANGELOG.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **`usermgmt.Service` (27 fields) needs splitting** — group into `authConfig`, `storeConfig`, `webauthnConfig`, `oauth2Config` sub-structs. This is the #1 architectural debt item.
2. **`handlerConfig` (16 fields) needs grouping** — same pattern. Rate limiter configs could be a sub-struct.
3. **No WebSocket library integration test** — current WS tests simulate the transport layer (channels, not real WS connections). A real WS integration test (using `coder/websocket` or `gorilla/websocket`) would validate the full stack including connection upgrade, ping/pong, and close handling.
4. **OAuth2 code is new and undertested with real providers** — only mock provider tests exist.

### Type Safety

5. **`TriggerID` is `string`** (`htmx.go:70`) — should be branded. Low effort, prevents confusion.
6. **`ExternalAccount.ClientID` is `string`** — should be branded. Part of the OAuth2 config surface.
7. **SSE `Last-Event-ID` is now branded** — done. Apply the same pattern to remaining string IDs.

### Testing

8. **No fuzz tests for `SQLEventStore`** — the session store has fuzz tests; the event store does not. Event payload round-trips through SQL should be fuzzed.
9. **No benchmarks for `OAuth2StateStore`** — Save/Consume/EvictExpired throughput unknown.
10. **No real OIDC provider test** — only mocks. PKCE + discovery + ID token verification untested with real endpoints.
11. **Coverage gap: usermgmt at 85.5%** — could be pushed to 90%+ with targeted tests on error paths.

### Dependencies

12. **`go-cqrs-lite/storage/v2` pulled in `listing/v2`** — verify if `listing` is actually needed or can be trimmed.
13. **`modernc.org/sqlite` upgraded as transitive dep** — verify no behavioral changes in the SQLite driver.

### Documentation

14. **No architecture diagram** — D2 diagram of the full CQRS + HTMX + SSE/WS flow would help onboarding.
15. **Catalog consumer example** — show a real app generating OpenAPI from CQRS handlers.

---

## f) Top 25 Things to Get Done Next

### High Impact, Low Effort

1. **Brand `TriggerID`** (`htmx.go:70`) — `type TriggerID string` with parser. Same pattern as `SSEEventID`.
2. **Brand `ExternalAccount.ClientID`** — prevent confusion with other string IDs in OAuth2 config.
3. **Add `errors.Is(err, event.ErrAggregateNotFound)` documentation** — already handled by decider, but document the pattern for direct `store.Load()` callers.
4. **Fuzz test `SQLEventStore`** — event payload round-trips through SQL (Save→Load). Mirror session store fuzz pattern.
5. **Add `OAuth2StateStore` benchmarks** — Save/Consume/EvictExpired throughput.

### High Impact, Medium Effort

6. **Split `usermgmt.Service` (27 fields)** into focused config sub-structs. Biggest architectural debt.
7. **Split `handlerConfig` (16 fields)** — group rate limiter configs into a sub-struct.
8. **Real OIDC provider integration test** — test PKCE + discovery + ID token with a test OIDC server (e.g., `go-oidc` mock provider with real JWT signing).
9. **WebSocket real-connection integration test** — use `coder/websocket` for a full upgrade→dispatch→broadcast→close cycle.
10. **Schema migration versioning** — `_schema_version` table + `migrate up` command.
11. **Security audit of CSRF + OAuth2 interaction** — verify state token can't be replayed across providers.
12. **Coverage push: usermgmt 85.5% → 90%+** — target error paths in service_misc.go, oauth2.go.

### Medium Impact, Low Effort

13. **Document `placeholderFunc` as session-store-only** — add comment explaining why it's not using `sqlpkg.Dialect`.
14. **Add `gosec` to CI verification** — the `//nolint:gosec` directives suggest security linting; verify it runs.
15. **Run `branching-flow` in CI** — prevent regressions on the metrics we improved.
16. **Expose `event.SeekableJournal`** — the delegated store supports seeking; expose for replay/backup.
17. **Document the `omitzero` tag decision** — add a note in AGENTS.md about Go 1.24+ `omitzero` usage on branded types.

### Medium Impact, Medium Effort

18. **Catalog consumer example** — show a real app generating OpenAPI from CQRS handlers.
19. **Datastar demo enhancement** — add multi-user real-time updates.
20. **Architecture D2 diagram** — full CQRS + HTMX + SSE/WS flow.
21. **`listing/v2` dependency audit** — verify if it's needed or can be trimmed.
22. **`modernc.org/sqlite` behavioral verification** — run the test suite against the new version.

### Lower Priority

23. **Phantom type audit** — review the 324 findings for genuine safety improvements (most are noise).
24. **Mixin consolidation** — review the 27 mixin suggestions for high-confidence structural deduplication.
25. **Error-family adoption** — review the 174 `fmt.Errorf` sites for any that should be sentinels.

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `usermgmt.Service` (27 fields) be split into sub-structs, or is the current flat struct the right design?**

The `Service` struct aggregates: repository, dispatcher, read model, casbin projection, authz, sessions, logger, lockout, event handler, bus, store, webauthn config, webauthn sessions, eviction goroutines (3), audit log, verification tokens, TOTP config, OAuth2 providers, OAuth2 state store. All 27 fields are used by the service methods.

**Why I can't resolve this alone:**

- **Arguments for splitting**: 27 fields is a code smell. Grouping into `authConfig`, `storeConfig`, `webauthnConfig`, `oauth2Config` would make the struct scannable and clarify which methods touch which concerns. The branching-flow linter flags it.
- **Arguments against splitting**: The `Service` is a cohesive aggregate root — every method crosses multiple concerns (e.g., `Register` touches the repository, read model, sessions, audit log, event handler). Splitting into sub-structs would require either (a) passing sub-structs between methods (breaking encapsulation), or (b) keeping them as fields on `Service` (which is what we have now, just with better naming). The Go idiom for aggregate services is flat structs (see `database/sql.DB`, `http.Server`).
- **The real question**: Is this a _naming_ problem (group fields with comments/spacing) or a _structural_ problem (extract sub-structs)? If it's just naming, a few blank lines and section comments would fix it without changing the type. If it's structural, the sub-struct extraction is worth the churn.

This depends on whether future features will add more fields (structural problem → split now) or whether the struct is stable (naming problem → just organize visually).

---

## Commits This Session

| #   | Hash      | Message                                                        |
| --- | --------- | -------------------------------------------------------------- |
| 1   | `df5c02f` | feat: add SSEEventID branded type for SSE event identifiers    |
| 2   | `bb2b0c7` | feat: add JSONKeyError/JSONKeyStatus constants to root package |
| 3   | `de7c3a9` | refactor: use branded types for AuditEntry fields              |
| 4   | `a7d02ff` | feat: add migration SQL for user_events to events table rename |
| 5   | `1652562` | docs: update CHANGELOG with breaking changes and new features  |
| 6   | `f5b6c51` | test: add SQLSessionStore fuzz tests for round-trip integrity  |
| 7   | `edbf9dc` | test: add SQLSessionStore benchmarks                           |
| 8   | `31e097e` | test: verify StoreWrapper transformation round-trip end-to-end |
| 9   | `80c14ee` | test: add WebSocket end-to-end integration tests               |
| 10  | `c962ba7` | test: add concurrent Consume test for OAuth2 state store       |
| 11  | `d07b82e` | chore: clean up go.sum and update AGENTS.md with new types     |
| 12  | `f2bc1ae` | style: apply gofumpt formatting to SSEEventID test table       |

All commits pushed to `origin/master`.
