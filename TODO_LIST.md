# TODO List — cqrs-htmx

**Updated:** 2026-06-28 | **Coverage:** 95.4% root, 80.1% usermgmt | **Lint:** 0 issues (all modules) | **Version:** v3.2.0 (go-cqrs-lite v3.2.0)

## Status Legend

- [ ] OPEN
- [x] DONE
- [~] PARTIALLY DONE
- [-] NOT APPLICABLE / BLOCKED

---

## Open Items

### Passwordless Event-Sourced Migration (2026-06-16)

- [x] **Wire AccountLockout into BeginLogin/FinishLogin** — Lockout checked on begin; failures recorded on finish; reset on success.
- [x] **Add credential management HTTP endpoints** — GET /auth/credentials + DELETE /auth/credentials/{id}.
- [x] **WebAuthn session proactive eviction** — Background goroutine with Service.Stop() cleanup.
- [x] **CasbinProjection subscribes to credential events** — Ordering guarantee without policy changes.
- [x] **Eliminate production panics** — marshalPayload and aggIDFromUser return errors.
- [x] **RegisterCommands returns error** — No more silently ignored registration failures.
- [x] **Log bus.Subscribe failures** — Event bridge errors logged at warn level.
- [x] **Log projection runner errors** — Background projection errors logged at error level.
- [x] **Refactor WebAuthn HTTP body parsing** — Query params instead of fragile double body read.
- [x] **Fix goroutine leak in tests** — t.Cleanup(svc.Stop) for all WebAuthn test services.
- [x] **Remove stale password references** — All test helpers updated for passwordless API.
- [x] **Fix lint warnings** — gosec G101, perfsprint, wrapcheck, exhaustruct, gci all clean.
- [x] **Fill DOMAIN_LANGUAGE.md** — All domain terms defined.
- [x] **Update README.md architecture** — Correct file names, remove stale references.
- [x] **Update CHANGELOG.md** — Comprehensive Unreleased entries.

### Verification, TOTP & Import/Export Hardening (2026-06-17)

- [x] **Admin authorization on import/export** — ImportExportAuthorizer defaults to RequireAdminRole. Non-admin users get 403.
- [x] **Per-IP rate limiting on all sensitive endpoints** — HandlerConfig.ImportRateLimit, TOTPRateLimit, VerificationRateLimit. checkRateLimit helper eliminates boilerplate.
- [x] **withTimeout helper** — Extracts duplicated context-timeout boilerplate from 8+ handlers.
- [x] **ImportUser.Validate()** — Email format/length validation, display-name length check. Wired into JSON and CSV paths.
- [x] **Negative-path handler tests** — Invalid TOTP code, setup without pending secret, TOTP disable when not enabled, import invalid email, import empty array, import duplicate email, verify already-verified.
- [x] **pquerna/otp/totp library** — Replaced hand-rolled RFC 6238 implementation with audited library.
- [x] **Require TOTP code to disable** — DisableTOTP now takes a code parameter. Prevents MFA stripping via session hijack.
- [x] **Email value type** — ParseEmail/MustParseEmail with RFC 5322 validation. Used in ExportUser.
- [x] **Rename ExportFormat → UserDataFormat** — Honest naming for dual import/export use.

### Catalog Sub-Package (2026-06-17)

- [x] **catalog/ merged into go-cqrs-lite** — The `catalog/` Go module was merged upstream into `go-cqrs-lite/catalog/v3` (v3.2.0). The single-service Builder facade lives at `catalog/v3/simple`, and the standalone HTTP handlers (D2, Health, GenerateEventCatalog) live at `catalog/v3/docserver`. The `OpenAPIHandler`/`AsyncAPIHandler` were replaced by the richer upstream `docserver.DocsServer`. The `catalog/` directory in cqrs-htmx is deleted.
- [x] **Builder API** — `New(title, version)`, `Command[T](b, id)`, `Query[T](b, id)`, `Event[T](b, id, dir)` using standalone generic functions (Go doesn't allow generic methods).
- [x] **HTTP handlers** — `OpenAPIHandler`, `AsyncAPIHandler`, `D2Handler`, `GenerateEventCatalog` (file generation, not HTTP), `HealthCheckHandler`.
- [x] **YAML output** — `WithFormat(FormatYAML)` for all JSON handlers.
- [x] **Schema reflection** — Auto-derives JSON Schema from struct tags (json, doc, format, enum, default).
- [x] **Catalog validation** — `Build()` panics on invalid catalogs. `BuildValid()` returns violations.
- [x] **ADR 0008** — Documents catalog sub-package decision (separate module, no root dep, generic functions).
- [x] **ADR 0009** — Documents go-cqrs-lite module selection rationale (8 used, 13 skipped, each with reason).
- [x] **Middleware integration doc** — Documents how go-cqrs-lite dispatch middleware composes with cqrs-htmx HTTP middleware.

### Future Work

- [x] **SQL event store** — Postgres/SQLite/MySQL event persistence for production use (SQLEventStore added)
- [x] **OAuth2/OIDC integration** — Social login as alternative to WebAuthn (ADR-0014, 2026-06-18)
- [x] **Event schema versioning** — Version field on events for future migrations (ADR-0013, upcaster registry)
- [x] **go-cqrs-lite v3.0.0 migration** — All 7 modules migrated from v2.6.0 to v3.0.0 (ADR-0016, 2026-06-22)
- [x] **Identity model redesign** — Actor, Tenant, Membership, Bot, Impersonation (ADR-0015, 2026-06-21)
- [x] **CSRF protection on WebAuthn endpoints** — Documented wiring recipe in `integration_test/csrf_webauthn_test.go` (CSRFMiddleware composes with SessionMiddleware via Chain)
- [x] **Rate limiting on WebAuthn endpoints** — `HandlerConfig.WebAuthnRateLimit` wired into all 4 WebAuthn handlers, matching the existing per-endpoint rate-limit pattern
- [x] **Property-based testing for foldUser** — 8 rapid-based property tests verify fold invariants
- [x] **Integration test: full WebAuthn flow** — End-to-end via virtual authenticator (W3C test vectors)

### Security & Correctness (Pre-v2.2.0)

- [x] **Fix rate limiter unbounded heap growth** — Fixed: limiterEntry now stores heapRef back-pointer. Refresh uses heap.Fix for in-place updates instead of pushing duplicate entries. No more ghost entries.
- [x] **Fix CSRF proxy bypass** — Added `CSRFConfig.TrustedProxies` (single IP or CIDR) and refactored `setPlaintextHTTPOrigin` into `shouldBypassPlaintextOrigin`/`isTrustedProxy` helpers. The plaintext-HTTP origin bypass is now restricted to loopback OR configured trusted proxies; empty config logs a warning but allows it (back-compat). Cyclop complexity reduced from 16 → ≤8. 6 new tests cover loopback, single-IP, CIDR, and untrusted-remote rejection.
- [x] **Fix Response.Status() fluent chain** — Fixed: Status() stores code in Response.statusCode. Apply() writes it at the end. Fluent chains like Status(201).Redirect("/x").Apply() now work.
- [x] **Add tests for nil-enforcer + query nil check** — Tests already existed in coverage_test.go (verified). Added ErrEnforcerNotInitialized sentinel to all Authz methods for defensive nil-checking.
- [x] **Add Login error classification tests** — TestService_Login_StoreError added. Verifies store errors return transient, not ErrInvalidCredentials.
- [x] **Add UpdateRoles rollback tests** — TestService_UpdateRoles_AuthzFailurePreservesUser added. Verifies user roles remain unchanged when Casbin Apply fails. Also fixed UpdateRoles ordering (Casbin before user save).
- [x] **Fix rate limiter data race** — Fixed: `perKeyLimiter.limiter()` read `entry.lastUsed` after releasing RLock while a concurrent goroutine wrote it under the write lock. Moved the freshness check inside the RLock-held region. Verified with 10/10 clean race-detector runs (was ~20% failure rate).
- [x] **Fix doc comment split brains** — Consolidated orphaned/truncated doc comments for `CSRFMiddleware` (split across csrf_middleware.go/csrf_context.go), `RateLimiter` struct (copy-paste leftover), `Apply` method (split across authz_types.go/authz_policies.go), `Broadcaster` type (orphaned in sse_store.go). Moved `splitSSELines` to sse_event.go next to its sole caller.
- [x] **Fix incomplete TestUserRegisteredEvent_JSON** — Test was named `_JSON` but never marshaled to JSON; only checked email field. Now actually tests JSON serialization output.
- [x] **Fix orphaned RenderJSON/SSE doc swaps** — RenderJSON doc orphaned in options_htmx.go, SSEStream doc orphaned in sse_event.go, SSEEventStore doc orphaned in sse_stream.go. All moved to their actual declarations. Added SSEEvent type doc.
- [x] **Remove redundant map copy in ParseWSMessageInto** — After `delete(raw, "HEADERS")`, the code copied raw into a new bodyMap via maps.Copy before marshaling. Redundant — marshal raw directly.
- [x] **SSEStream.Context() returns context.Context** — Changed from anonymous `interface{ Done() <-chan struct{} }` to `context.Context`. Consumers expect full context from a method named Context().
- [x] **Add missing exported godoc comments** — Render, NotificationLevel.String(), LoginRequest, RegisterRequest, GetUser.

### Upstream-Blocked

- [~] **BrandNamer for root module marker types** — PARTIALLY UNBLOCKED: go-cqrs-lite v3.1.0 exports marker types. Needs verification and wiring.
- [x] **Remove local replace directives** — go-cqrs-lite v2.0.0 tags are published upstream. All go-cqrs-lite replace directives removed from all 4 go.mod files. Only `integration_test` retains cqrs-htmx local replaces (library not yet published).

### Comprehensive Review Session (2026-06-25)

_Bugs found and fixed by code-quality-scan + full-code-review + architecture-review + data-model-review skills._

- [x] **Fix SSEStream.OnDisconnect data race** — `OnDisconnect()` appended to slice without mutex; `Close()` iterated without mutex. Fixed: both now acquire `s.mu`. Close snapshots under lock, iterates outside.
- [x] **Fix StartCleanupSweeper double-close panic** — `close(done)` in stop function panicked on second call. Fixed: wrapped in `sync.Once.Do()`.
- [x] **Fix 3 lint issues** — exhaustruct on sse_stream.go (add `mu: sync.Mutex{}`), gochecknoglobals on sql_session_store.go (replace `var zeroActorID` with inline `ActorID{}`), forcetypeassert on session_store_contract_test.go (use ok-pattern).
- [x] **Fix 3 stale AGENTS.md claims** — "19 files" → "40 files", "15 event + 15 command constants" → "21 event + 20 command constants", "7 events/7 commands" → "12 events/11 commands" with updated lists.

### Type Safety Improvements (from data-model-review)

- [~] **Type ActorID/ImpersonatorID in context** (CRITICAL) — **DESIGN DECISION DEFERRED**: Root module defines `type ActorID string` while usermgmt defines `type ActorID struct{ kind ActorKind; raw string }`. They're intentionally different: root needs a simple context value (prefixed string like "user:01JX..."), usermgmt needs kind/raw separation for domain logic. Making root depend on usermgmt would reverse the dependency direction. The real fix requires extracting a shared identity module that both can import — a significant architectural change. Current approach works because `string(actorID)` and `ParseActorID` bridge the gap.
- [x] **Make foldUser return error on unknown events** (HIGH) — `es_state.go:166` now returns `event.NewRejection` for unknown types. Other folds (Membership, Tenant, Bot) also correctly return errors.
- [x] **Use UserID for BotState.OwnerID** (HIGH) — `es_bot_state.go:11`, `service_bot.go:15` now use `UserID`. Other call sites updated.
- [~] **Use TenantID in Authz domain parameters** (HIGH) — All domain parameters now use `TenantID` across RolesForUser, ImplicitRolesForUser, ImplicitPermissionsForUser, DomainsForUser (returns `[]TenantID`), UsersForRole, RolesForActor, ImplicitRolesForActor. Remaining: `RemoveAllRolesForUser(subject string)` and `RemoveAllRolesInDomain(subject string, ...)` use raw `string` for subject — intentional, as Casbin subjects can be various ID types.
- [x] **Unexport or validate NewActorID** (HIGH) — `id.go:120` now panics on invalid ActorKind; type-safe constructors `ActorIDFromUser`/`ActorIDFromBot` provided for safe construction.
- [~] **Use Email branded type in domain models** (MEDIUM) — **BLOCKED BY EVENT SERIALIZATION**: The `Email` branded type exists but 10+ structs (UserState, UserRegisteredPayload, ExternalAccount, etc.) use raw `string`. Changing them to `Email` would break JSON event serialization (existing events in event stores have `"email": "user@example.com"` as a JSON string — changing the Go type requires custom UnmarshalJSON or a migration). This is safe for new fields but unsafe for existing event payload structs. Defer to next major version with an upcaster.
- [x] **Fix duplicate sentinel errors across packages** (MEDIUM) — **BY DESIGN, NOT A BUG**: Root and usermgmt define separate `ErrUnauthorized` sentinels with the same code `"unauthorized"` because usermgmt can't import root (module independence). Cross-module matching uses `authStatusFromErrorCode()` which compares by `.Code()` string, not sentinel identity. This is the correct pattern for independent Go modules that need to share error semantics without sharing dependencies.
- [~] **Prevent impossible TenantState** (MEDIUM) — `es_tenant_state.go` event handlers now clear `Suspended` on delete (struct-level invariant). Remaining gap: the struct fields themselves are still independent booleans; a struct-level `IsValid()` method would fully close this.
- [x] **Validate actorKindFromString** (MEDIUM) — `es_membership_state.go:76` now returns `event.NewRejection` for unknown strings instead of defaulting to ActorUser.

### Architecture Improvements (from architecture-review)

- [~] **Extract \*http.Request from WebAuthn service** (HIGH) — **BLOCKED BY UPSTREAM API**: The `go-webauthn/webauthn` library's `FinishRegistration()` and `FinishLogin()` methods take `*http.Request` directly to parse the attestation/assertion response from the request body. We can't pass pre-parsed bytes because the library API doesn't accept them. Fixing this requires either (1) an upstream PR to go-webauthn to accept parsed responses, or (2) wrapping `*http.Request` reconstruction in the service layer (which defeats the purpose). The HTTP leak is inherent to the webauthn library's design.
- [x] **Add interfaces for ephemeral stores** (HIGH) — **DONE**: All 6 stores have interfaces: `SessionStore` (store.go:10), `WebAuthnSessionStore` (store_interfaces.go:12), `VerificationTokenStore` (store_interfaces.go:21), `LockoutStore` (store_interfaces.go:29), `PendingTOTPStore` (store_interfaces.go:38), `OAuth2StateStore` (oauth2.go:262). In-memory implementations exist alongside SQL alternatives. Multi-instance deployment is possible by implementing these interfaces.
- [~] **Consider snapshot integration** (MEDIUM) — **EVALUATED, DEFERRED**: The `go-cqrs-lite/snapshot/v3` package is available as an indirect dependency. Current replay loads all events on startup (service_core.go), which works for moderate event volumes. Snapshot integration would add complexity (snapshot store, snapshot frequency, snapshot invalidation on upcasters) for benefit only at very high event volumes (>10K events per aggregate). Defer until a production deployment hits performance limits — premature optimization now.
- [x] **LastEventIDFromRequest should delegate** (LOW) — Already correct: `SSEStream.LastEventID()` delegates to `LastEventIDFromRequest(s.r)`. No duplication.

### Future Enhancements (Not Started)

- [x] **Upgrade to go-cqrs-lite v2.0.0** — All 4 modules migrated to v2 import paths (`/v2` suffix). CatalogEntries removed (dead upstream code). go-error-family v0.3.0. Replace directives removed (v2.0.0 tags published).
- [x] **SQL store backend for usermgmt** — Pattern documented in `usermgmt/docs/SQL_STORES.md` (Postgres schema + adapter skeleton). Library principle: no SQL driver dep in `usermgmt` core; consumer implements `UserStore`/`SessionStore` (matches Casbin/CQRS pattern). ADR 0003 numeric-ID strategy (BIGSERIAL + public_id TEXT UNIQUE) recorded.
- [x] **OpenTelemetry integration** — `example_otel_test.go` documents the hook-based pattern (OtelBeforeDispatch/OtelAfterDispatch). Library principle: no OTel SDK dep in `cqrs-htmx`; consumers pass hooks into `Config`. Tests show wiring with a fakeTracer; real `otel.Tracer("cqrs-htmx")` swap-in commented in code.
- [x] **Adopt v2 typed dispatch** — `command.RegisterTyped[T]` and `query.RegisterTyped[T]`/`query.DispatchTyped[T]` used in `datastar-demo/domain_cqrs.go` (4 commands + 2 queries), `integration_test/typed_query_test.go` (3 cross-module tests), and root `example_app_test.go` (`ExampleApp_Query_typedRegister`, `ExampleApp_Query_typedDispatch`, `ExampleRegisterTyped`, `ExampleApp_Command`).
- [x] **Adopt PaginatedResult[T]** — `DecodePagination(r)` + `RenderPaginatedJSON[T]()` implemented using `query.Pagination`/`query.PaginatedResult[T]` from go-cqrs-lite v2.2.0.
- [x] **Upgrade to go-cqrs-lite v2.2.0** — All 4 modules upgraded to v2.2.0. Adopted `PaginatedResult[T]` and `query.Pagination` from upstream. Added `DecodePagination` and `RenderPaginatedJSON[T]`.
- [x] **Reactive event streams** — SSE Broadcaster, SSEStream, SSEEventStore, ReplayEvents, CQRS bridge (BroadcastOnSuccess/BroadcastOnSuccessFunc). WebSocket message parser (ParseWSMessage, ParseWSMessageInto[T], WSOOBHTML).
- [x] **Embedded HTMX JS** — HTMXScriptHandler serves embedded HTMX v2.0.9 (minified, ~49KB) with ETag/caching. HTMXScriptTag, HTMXVersion helpers.

### Offline-First Command Sync (2026-06-28)

_Phase 0 + Phase 1 server-side code DONE. See [execution plan](docs/planning/2026-06-28_10-14_offline-first-command-sync-execution.html) and [ADRs 0023/0024](docs/adr/)._

- [x] **Production SSEEventStore** (`JournalSSEStore`) — Backed by `event.SeekableJournal` for cursor-based replay. `WithMaxReplay(n)` limits first-connection volume. `EventToSSEMapper` consumer-provided function. Falls back to `ReadAll` for non-seekable journals.
- [x] **ACK protocol** — `CommandAck` struct, `BroadcastOnAck`/`BroadcastOnAckFunc` (SSE), `BroadcastOnAckWS`/`BroadcastOnAckWSFunc` (WS). Opt-in via `X-Command-Id` header.
- [x] **Integration tests** — 6 end-to-end tests prove SSEEventStore + Broadcaster + ACK protocol work together in real HTTP handlers.
- [x] **ADRs** — 0023 (command-sync: sync commands not events), 0024 (honest UI: never lie about pending state).
- [x] **Honest UI CSS** — `data-sync-state` attribute, `.sync-pending`/`.sync-confirmed`/`.sync-rejected` classes, global sync indicator. (T08-T14)
- [x] **Honest UI JS** — SSE EventSource manager, `sync:ack` listener, `handleSyncAck` DOM flip, optimistic render on `htmx:beforeRequest`, never-silent rollback. (T15-T21)
- [x] **Honest UI templ + admin-demo** — Layout indicator, `data-sync-state` on rows, demo wiring. (T22-T30)
- [x] **Idempotency store** — `IdempotencyStore` interface + `MemoryIdempotencyStore` with TTL sweep. `CheckAndRecord` helper for atomic check-and-record. `ErrDuplicateCommand` → HTTP 409. ADR-0026. (2026-06-28)
- [x] **Form decoder upgrade** — Replaced JSON round-trip with `go-playground/form/v4` (zero transitive deps, `json` tag mode for backward compat). (2026-06-28)
- [x] **Pagination unification** — Both root `DecodePagination` and usermgmt `parseUintQueryParam` now delegate to `query.NewPagination`. Same defaults. Standard REST behavior (no silent page clamping). (2026-06-28)
- [x] **Go.mod version alignment** — All modules aligned to Go 1.26.4 + cqrs-htmx v3.2.0. ETag bumped to `adminui-v3.2.0`. (2026-06-28)
- [x] **Stdlib modernization** — `slices.Contains`, `min()`, `slices.IndexFunc` replace manual loops. goconst warnings fixed. `context.TODO()` → `context.Background()`. (2026-06-28)
- [x] **CI umbrella verified** — `nix run .#test` + `nix run .#lint` + `nix run .#errorfamily` all green. All 4 modules pass with -race. (2026-06-28)
- [x] **go.yaml.in/yaml/v3 investigated** — Confirmed as official Canonical Ltd successor to `gopkg.in/yaml.v3` (same codebase, new import path). NOT a typo-squat. No action needed. (2026-06-28)
- [ ] **Phase 2 implementation** — Q1 ANSWERED (ADR-0027: Queue-Only — decide() stays on server). Q2 still open: must writes survive closed tabs? (SharedWorker / Service Worker + Background Sync). (T31-T35)
- [x] **Idempotency store wired into admin-demo** — ackMiddleware now rejects duplicate X-Command-Id mutations with 409. Ghost system killed. (2026-06-28)
- [x] **CheckAndRecord atomicity fixed** — Moved from racy free function to IdempotencyStore interface method. Proven by 200-goroutine concurrency test. (2026-06-28)
- [x] **Idempotency memory leak fixed** — Seen() lazily deletes expired entries. (2026-06-28)

---

## Completed (2026-05-07 → 2026-06-14)

_170 items completed. See [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master) for full history._

### Highlights by Session

| Session     | Key Accomplishments                                                                                                                                                                                                                                                                                                 |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-05-07  | Initial lint zero (103→0), test coverage 93.5%                                                                                                                                                                                                                                                                      |
| 2026-05-16  | v1.0.0 release: lifecycle hooks, validation, timeout, benchmarks                                                                                                                                                                                                                                                    |
| 2026-05-19  | CSRF protection (gorilla/csrf), error context, deduplication                                                                                                                                                                                                                                                        |
| 2026-05-20  | Branded UserID migration, SessionMaxAge fix, usermgmt 85%→95.6%                                                                                                                                                                                                                                                     |
| 2026-05-21  | CatalogEntries exposure, CI fix, lint elimination, error wrapping                                                                                                                                                                                                                                                   |
| 2026-05-22  | Integration tests, O(log n) eviction, HTTP timeout, fuzz tests                                                                                                                                                                                                                                                      |
| 2026-05-23  | Mock stores, coverage 88.6%→91%, go-cqrs-lite v1.5.0 upgrade                                                                                                                                                                                                                                                        |
| 2026-05-24  | Perf optimizations (7 alloc reductions), security hardening                                                                                                                                                                                                                                                         |
| 2026-05-25+ | gorilla/csrf→nosurf, cockroachdb/errors→go-error-family, httputil delegation                                                                                                                                                                                                                                        |
| 2026-05-27  | RecoveryMiddleware, RenderJSON, request ID in errors, benchmarks                                                                                                                                                                                                                                                    |
| 2026-05-27b | 10 bug fixes: GetUser 404, rate limiter TTL, CSRF JSON, store copies, authz ordering, WriteJSON buffer, password DRY, rollback logging, SessionMiddleware logging                                                                                                                                                   |
| 2026-05-27c | HandlerConfig.Secure \*bool, CSRFConfig.Validate(), Response.JSON 500, correlation ID logging, RecoverHandler rename, go-cqrs-lite v1.6.0, dispatch logging, usermgmt writeJSON buffer, tests                                                                                                                       |
| 2026-05-28  | Domain model enrichment: SetRoles, ChangePassword, SetEmail, SetDisplayName, IsPasswordSet, touch(). Domain events: 4 event types with optional EventHandler. Fuzz + benchmarks. CRUD eliminated.                                                                                                                   |
| 2026-06-02  | v2.0.0 migration (42 files). Pre-release fixes: nil-enforcer bypass, query nil panic, Login error classification, UpdateRoles ordering, store clone, query param logging removed, defaultLoginRedirect const.                                                                                                       |
| 2026-06-08  | SSE/WebSocket polish: SSEEventStore interface, ReplayEvents, LastEventIDFromRequest. WebSocket: ParseWSMessage, ParseWSMessageInto[T], WSOOBHTML. PaginatedResult[T] adoption. v2.2.0.                                                                                                                              |
| 2026-06-12  | v2.3.0 adoption: TypedHandler, deadline propagation, empty type validation, per-module go-cqrs-lite tags.                                                                                                                                                                                                           |
| 2026-06-14  | **TODO sweep**: CSRF proxy bypass (TrustedProxies, cyclop refactor, 6 tests), SQL stores pattern doc, OTel hook example, typed dispatch examples. Lint 67→0 across all 3 modules (exhaustruct /v2 regex, nilnil/goconst/noctx, sse_reconnect_integration_test noctx, integration_test unconvert/wrapcheck/goconst). |
