# TODO List — cqrs-htmx

**Updated:** 2026-06-14 | **Coverage:** 96.0% root, 90.0% usermgmt | **Lint:** 0 issues (all modules) | **Version:** v2.1.0

## Status Legend

- [ ] OPEN
- [x] DONE
- [~] PARTIALLY DONE
- [-] NOT APPLICABLE / BLOCKED

---

## Open Items

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

### Upstream-Blocked

- [ ] **BrandNamer for root module marker types** — BLOCKED: upstream `go-cqrs-lite/core/pkg/id` marker types (`userMarker`, `correlationMarker`) are unexported. Requires upstream change to expose them or provide BrandNamer integration.
- [x] **Remove local replace directives** — go-cqrs-lite v2.0.0 tags are published upstream. All go-cqrs-lite replace directives removed from all 4 go.mod files. Only `integration_test` retains cqrs-htmx local replaces (library not yet published).

### Future Enhancements (Not Started)

- [x] **Upgrade to go-cqrs-lite v2.0.0** — All 4 modules migrated to v2 import paths (`/v2` suffix). CatalogEntries removed (dead upstream code). go-error-family v0.3.0. Replace directives removed (v2.0.0 tags published).
- [x] **SQL store backend for usermgmt** — Pattern documented in `usermgmt/docs/SQL_STORES.md` (Postgres schema + adapter skeleton). Library principle: no SQL driver dep in `usermgmt` core; consumer implements `UserStore`/`SessionStore` (matches Casbin/CQRS pattern). ADR 0003 numeric-ID strategy (BIGSERIAL + public_id TEXT UNIQUE) recorded.
- [x] **OpenTelemetry integration** — `example_otel_test.go` documents the hook-based pattern (OtelBeforeDispatch/OtelAfterDispatch). Library principle: no OTel SDK dep in `cqrs-htmx`; consumers pass hooks into `Config`. Tests show wiring with a fakeTracer; real `otel.Tracer("cqrs-htmx")` swap-in commented in code.
- [x] **Adopt v2 typed dispatch** — `command.RegisterTyped[T]` and `query.RegisterTyped[T]`/`query.DispatchTyped[T]` used in `datastar-demo/domain_cqrs.go` (4 commands + 2 queries), `integration_test/typed_query_test.go` (3 cross-module tests), and root `example_app_test.go` (`ExampleApp_Query_typedRegister`, `ExampleApp_Query_typedDispatch`, `ExampleRegisterTyped`, `ExampleApp_Command`).
- [x] **Adopt PaginatedResult[T]** — `DecodePagination(r)` + `RenderPaginatedJSON[T]()` implemented using `query.Pagination`/`query.PaginatedResult[T]` from go-cqrs-lite v2.2.0.
- [x] **Upgrade to go-cqrs-lite v2.2.0** — All 4 modules upgraded to v2.2.0. Adopted `PaginatedResult[T]` and `query.Pagination` from upstream. Added `DecodePagination` and `RenderPaginatedJSON[T]`.
- [x] **Reactive event streams** — SSE Broadcaster, SSEStream, SSEEventStore, ReplayEvents, CQRS bridge (BroadcastOnSuccess/BroadcastOnSuccessFunc). WebSocket message parser (ParseWSMessage, ParseWSMessageInto[T], WSOOBHTML).
- [x] **Embedded HTMX JS** — HTMXScriptHandler serves embedded HTMX v2.0.9 (minified, ~49KB) with ETag/caching. HTMXScriptTag, HTMXVersion helpers.

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
