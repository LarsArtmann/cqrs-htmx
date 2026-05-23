# TODO List — cqrs-htmx

**Date:** 2026-05-07 | **Updated:** 2026-05-21 | **Source:** Self-review session, full codebase audit + comprehensive 9-skill review

## Status Legend

- [ ] OPEN
- [x] DONE
- [~] PARTIALLY DONE
- [-] NOT APPLICABLE

---

## P0 — Security & Correctness

- [x] **Use headerTrue constant in Response.Refresh()** — Fixed in commit `98616cb` (response.go:56)
- [x] **Use headerRedirect constant in DefaultErrorHandlerWithRedirect** — Fixed in commit `21c22c9` (errors.go:110)
- [x] **Fix Config.LoginRedirect dead code** — Was stored but never read; now threads into per-App error handler closure. Commit `ab52b05`.
- [x] **Fix README compile-breaking example** — `cqrshtmx.LoginRedirect` didn't exist. Fixed in commit `1a44e8c`.
- [x] **Verify XSS safety in DefaultErrorHandler** — Removed `html.EscapeString` from `text/plain` responses. `text/plain` Content-Type prevents browser HTML rendering; escaping distorted error messages like "foo < bar". Added `//nolint:gosec` with explanation.
- [x] **CSRF v1.7.3 plaintext HTTP detection** — `CSRFMiddleware` and `executeCSRFValidation` detect non-TLS requests and mark as plaintext via `csrf.PlaintextHTTPRequest`. Commit `9dcae5a`.

## P1 — Code Quality

- [x] **Extract `"true"` string constant** — `headerTrue` constant at htmx.go:35, used in all production code
- [x] **Add doc comment to SwapStrategy const block** — htmx.go:40
- [x] **Deduplicate notification test boilerplate** — `testNotificationTrigger` helper + `notifyOption`/`triggerNotification` private helpers
- [x] **Consolidate duplicate test types** — `mockTemplComponent`, `deleteUserCmd`, `listUsersQuery` → use `bdd*` types
- [x] **Extract helper functions** — `hasNoResponse()`, `hasMinimalResponse()`, `decodeJSONBody`, `decodeRequest`, `decodeFormBody`, `notifyOption`, `triggerNotification`
- [x] **Export `HeaderTrue`** — `HeaderTrue` exported constant in htmx.go. All tests use `cqrshtmx.HeaderTrue` instead of hardcoded `"true"`.
- [x] **Add error context to authorization errors** — `ErrForbidden`, `ErrEnforcerNil`, and `ErrUnauthorized` (with Authorize) now include resource/action context for debugging

## P2 — Architecture Improvements

- [x] **Move remaining mutable globals to per-App config** — `DefaultNotificationEvent` is now an unexported constant; exported var is deprecated. Added `NotifyWithEvent` builder for custom event names per-handler.
- [x] **Extract Casbin interface** — `authz.go` now defines `Enforcer interface { Enforce(...any) (bool, error) }`. `*casbin.Enforcer` satisfies it automatically. Enables mock/fake enforcers in consumer tests.
- [x] **Fix AuthorizeMiddleware ghost system** — Was bypassing App's error handler (raw `http.Error`). Now uses `DefaultErrorHandlerWithRedirect` for HTMX-aware auth error handling. Optional `loginRedirect` parameter.
- [x] **Remove dead sentinels** — `ErrNoUserID` and `ErrRendererMissing` removed. `ErrCommandsNil`, `ErrQueriesNil`, `ErrDecoderMissing` unexported. `DefaultNotificationEvent` removed.
- [x] **Extract shared test helpers** — `testing_test.go` with 11 helpers covering decoders, handlers, capture utilities. Reduced clone groups by 48% (27→14 at t=25).

## P3 — Feature Enhancements

- [x] **Add dispatch lifecycle hooks** — `BeforeDispatchHook` / `AfterDispatchHook` on `Config`. Tested in `hooks_test.go` with 9 specs.
- [x] **Add request validation middleware** — `ValidateCommand` / `ValidateQuery` HandlerOptions with `ErrValidationFailed` sentinel. Tested in `validation_test.go` with 6 specs.
- [x] **Add JSON error response option** — `JSONErrorHandler` writes `{error, status}` JSON. HTMX auth errors redirect via HX-Redirect.
- [x] **Add correlation ID propagation** — `WithCorrelationID` / `CorrelationIDFromContext` (branded `id.CorrelationID`). Auto-extracted from `X-Correlation-ID` header in `ContextEnrichmentMiddleware`.
- [x] **Fix CorrelationID event metadata pipeline** — `EventOptionsFromContext` now propagates branded `id.CorrelationID` into event metadata.
- [x] **Fix AuthorizeMiddleware identity parsing** — now uses branded `UserID` from context, falls back to extractor + `ParseUserID()` validation.
- [x] **Add Request Logging middleware** — `RequestLogging(formatter, writer)` with `DefaultLogFormatter`, captures correlation/user IDs.
- [x] **Add Rate Limiting middleware** — `RateLimiterMiddleware` with token-bucket per-key via `golang.org/x/time/rate`.
- [x] **Add timeout propagation** — `Config.Timeout time.Duration` wraps dispatch with `context.WithTimeout`. Zero/negative = no timeout. Tested in `timeout_test.go` with 7 specs.
- [x] **Add Request ID** — `RequestID` strongly-typed identifier, `NewRequestID`/`ParseRequestID`/`MustParseRequestID`, `WithRequestID`/`RequestIDFromContext`. Propagated into event metadata.
- [x] **Add Security Headers middleware** — `SecurityHeadersMiddleware` / `SecurityHeadersMiddlewareWithConfig` with configurable CSP, HSTS, etc.
- [x] **Add CSRF Protection** — `CSRFMiddleware`, `CSRFProtect`, template helpers, `CSRFResponseHeaderMiddleware`, `RotateCSRFToken`. Uses gorilla/csrf internally.

## P4 — Polish

- [x] **Add godoc examples** — 9 `Example*` functions in `example_test.go`: `New`, `App_Command`, `App_Query`, `NewResponse`, `SwapStrategy`, `HTMXMiddleware`, `RequestLogging`, `RateLimiterMiddleware`.
- [x] **Add benchmark tests** — 16 sub-benchmarks in `benchmark_test.go`: `MapError` (6), `ParseHTMXRequest` (2), `CommandDispatch`, `QueryDispatch`, etc.
- [x] **Create CONTRIBUTING.md** — Document lint config, test patterns, naming conventions
- [x] **Add `golangci-lint` to CI/CD** — GitHub Actions enforcement (.github/workflows/ci.yml)
- [x] **Document `.golangci.yml` decisions** — Inline comments explaining exclusions

## Already Done

- [x] **App builder with validation** — `New(Config)` validates at least one dispatcher
- [x] **Generic decoders** — `DecodeJSON[T]`, `DecodeJSONQuery[T]`, `DecodeForm[T]`, `DecodeFormQuery[T]`
- [x] **Casbin authorization** — `Authorize`, `RequireAuth`, `Enforce`, `AuthorizeMiddleware`
- [x] **HTMX request context** — `HTMXMiddleware`, `HTMXRequest` struct, all accessors with fallback
- [x] **HTMX response builder** — Fluent `Response` with all HTMX headers supported
- [x] **Notifications** — Both HandlerOptions and Response methods via shared helpers; `NotifyWithEvent` builder for custom events
- [x] **Error classification** — `sync.Once` registers all sentinels. `MapError` maps families to HTTP status
- [x] **Context propagation** — User ID → context → event metadata. Dedup in handlers
- [x] **Templ duck-typing** — `RenderTempl`, `RenderTemplResult[T]` without importing templ
- [x] **Middleware chain** — `Chain` composes middleware left-to-right
- [x] **94.8% test coverage** — 289 specs, race-safe, usermgmt 95.6%
- [x] **0 lint issues** — golangci-lint clean
- [x] **All header constants consolidated** — No hardcoded HTMX header strings in production code
- [x] **Per-App LoginRedirect** — Config field now actually works via closure
- [x] **Enforcer interface** — Enables testability without concrete Casbin dependency

## P5 — Open Items from 2026-05-19 Comprehensive Review

### Correctness

- [x] **Fix context mismatch in `applyQueryResponse`** — Verified already correct: caller at `handler.go:166` passes `r.WithContext(ctx)`, so `r.Context()` inside `applyQueryResponse` IS the enriched context. No bug.
- [x] **Fix nil context in usermgmt tests** — Replaced `nil` with `context.TODO()` in `handler_test.go`. Silences SA1012.

### Production Safety

- [x] **Add max-keys cap to rate limiter** — `MaxKeys int` added to `RateLimiterConfig` with O(n) eviction. `evictOldestIfAtCapacity()` extracted to keep complexity under cyclop threshold.
- [x] **Cache CSRF Protect instance** — `handlerConfig.csrfProtect` field caches the middleware. `CSRFProtect()` builds it once. `executeCSRFValidation` reuses the cached instance.

### Deduplication (High Impact)

- [x] **Generic HTMX accessor** — `htmxBoolField`/`htmxStringField` generic helpers. 8 accessors delegate to 2 generics.
- [x] **Generic decoder** — `decodeAndSet[T,R]` generic decoder. 4 public functions remain as ergonomic wrappers.
- [x] **Generic validation** — `validateDispatch[T]` generic validator. `ValidateCommand`/`ValidateQuery` are thin wrappers.
- [x] **Unified notification implementation** — `notificationDetail()` shared helper extracted.

### Deduplication (Medium Impact)

- [x] **Shared logging context extraction** — `contextFields(r)` helper extracts correlation/user/request IDs. Used by 3 formatters.
- [x] **Shared error handler core** — `handleErrorCore(w, r, err, redirect, writeBody)` extracted. Used by both Default and JSON error handlers.
- [x] **Generic ID helpers** — `parseID[T]` generic helper. `ParseUserID`/`ParseCorrelationID`/`ParseRequestID` delegate to 1 generic.

### File Organization

- [x] **Split `csrf.go`** — Template helpers moved to `csrf_helpers.go`.

### New Items from 2026-05-20 Session

- [x] **Branded UserID migration** — `usermgmt/id.go` with `UserID = brandid.ID[userBrand, string]` via `go-branded-id`. All user ID fields/params strongly typed. 17/18 violations fixed (TriggerID intentionally skipped — DOM element ID, not domain entity).
- [x] **Fix SessionMaxAge not copied in NewAuthHandlers** — `http.go:NewAuthHandlers` did not copy `SessionMaxAge` from `HandlerConfig`, always defaulting to 86400.
- [x] **usermgmt test coverage 85% → 95.6%** — Added tests for untested authz methods (ImplicitRoles, ImplicitPermissions, Policies, GroupPolicies, AddPolicy, RemovePolicy, RemoveGroupPolicy), store paths (duplicate ID, WithTTL), http config (SessionMaxAge, custom CookieName), and UserID type operations.

### New Items from 2026-05-20 Session (Part 2)

- [x] **Error wrapping consistency** — `fmt.Errorf` → `cockroachdb/errors` for sentinel-only wraps (`errors.WithMessagef`) and inner-error wraps (`errors.Wrapf`). Double-wrapping keeps `fmt.Errorf("%w: %w")` for `errors.Is` compatibility.
- [x] **Extract security header constants** — 9 unexported constants in `security.go` (6 header names + 3 defaults).
- [x] **Extract magic string constants** — `logFieldCorrelationID`, `logFieldUserID`, `logFieldRequestID` (logging.go), `headerRetryAfter`, `rateLimitExceededMsg` (ratelimit.go), `notificationKeyLevel`, `notificationKeyMessage` (notify.go).
- [x] **Consolidate test type declarations** — All shared test types moved to `testing_test.go`; duplicates removed from `app_test.go`, `bdd_test.go`, `integration_test.go`.
- [x] **Add godoc to all usermgmt exported symbols** — ~70 symbols across 9 files.
- [x] **Raise usermgmt coverage to 92.1%** — Added 16 targeted tests for uncovered paths.
- [x] **Fix UserIDFromRequest godoc** — Updated to clarify it returns `string`, not `cqrshtmx.UserID`.

### Open Items

- [x] **Resolve usermgmt vs cqrshtmx UserID type split** — Decided: intentionally separate types (ADR 0002). `usermgmt.UserID` is string-backed for standalone use; `cqrshtmx.UserID` is ULID-backed via go-cqrs-lite. Bridge via `.Get()` for string conversion. No migration needed.
- [x] **Evaluate go-branded-id for numeric IDs** — Evaluated (ADR 0003). Future SQL stores should use `brandid.ID[Brand, int64]` for auto-increment PKs. Documented pattern ready for implementation.
- [ ] **Adopt TypedHandler[T] from go-cqrs-lite v1.5.0** — Upstream has `command.RegisterTyped[T]` and `query.RegisterTyped[T]`. cqrs-htmx needs `DispatchTyped[T]` wrapper for type-safe query results. API design decision: top-level generic function vs method on generic App.
- [ ] **BrandNamer for root module marker types** — BLOCKED: upstream `go-cqrs-lite/core/pkg/id` marker types (`userMarker`, `correlationMarker`) are unexported. Requires upstream change.

### Items Completed in 2026-05-23 Session

- [x] **Mock stores for usermgmt testing** — `mock_test.go` with `mockUserStore` and `mockSessionStore` using function-field pattern. Enables testing error paths without Casbin/SQL.
- [x] **Usermgmt coverage 88.6% → 91.0%** — 30+ new tests for Register rollback, Logout error, ChangePassword edges, GetUser/UpdateRoles errors, Login lockout, handler validation, session expiry.
- [x] **Root coverage 96.6% → 96.7%** — WriteJSON error, MapError nil/unknown, Enforce nil enforcer, sanitizeRedirectURL edges, handleCommandDispatch auth denied, rate limiter eviction, ClientIP edges.
- [x] **go-cqrs-lite v1.4.0 → v1.5.0** — All 3 modules upgraded. Clean upgrade, no breaking changes.
- [x] **Extract constants in usermgmt** — `bearerPrefix`, `errMsgPasswordTooShort`, `errMsgPasswordTooLong`, `maxDisplayNameLength` constants extracted from magic strings.
- [x] **Usermgmt fuzz tests** — `FuzzRegisterRequest_Validate`, `FuzzLoginRequest_Validate` for arbitrary input validation.
- [x] **Usermgmt benchmarks** — `BenchmarkService_Login`, `BenchmarkService_Register`, `BenchmarkSession_TokenMatches`.
- [x] **Integration test expansion** — `TestUsermgmtBridge_FullRegisterAuthCycle` cross-module flow.
- [x] **Integration test lint config** — `.golangci.yml` for integration_test module.
- [x] **ADR 0003: numeric IDs for SQL stores** — Documented pattern for future SQL store backends.
- [x] **CONTRIBUTING.md update** — Multi-module structure, GOWORK=off commands, 4-module table.

### Items Completed in 2026-05-22 Session

- [x] **Integration tests between root module and usermgmt** — `integration_test/` as separate Go module. Tests `AsEnforcer()` bridge and `UserIDFromRequest` bridge with `.Get()` for cross-module ID conversion.
- [x] **Rate limiter eviction O(n) → O(log n)** — Replaced linear scan with `container/heap` min-heap. `evictionHeap` tracks entries by `lastUsed`. Eviction is now O(log n).
- [x] **CSRFConfig.Secure warning** — `CSRFMiddleware` now emits `slog.Warn` when `Secure=false`, with hint to set `Secure=true` in production.
- [x] **RateLimiterConfig signedness unify** — Changed `perKeyLimiter.burst` and `perKeyLimiter.maxKeys` from `int` to `uint` to match `RateLimiterConfig` fields. Conversion to `int` only at `rate.NewLimiter` boundary.
- [x] **Usermgmt HTTP timeout** — `HandlerConfig.Timeout` adds `context.WithTimeout` to `handleAuthEndpoint` and `handleLogout`. Zero means no timeout (default).
- [x] **CSRF fuzz tests** — `FuzzCSRFConfigValidation` added to `fuzz_test.go`. Tests all CSRFConfig methods with arbitrary inputs.
- [x] **policyWrapErr coverage** — Now at 100%. Added `TestPolicyWrapErr` in `usermgmt/coverage_test.go`.
- [x] **usermgmt handler/authz/service coverage** — Added 20+ tests covering `handleLogout` success/no-session, `handleMe` with/without user, `handleLogin` success/wrong-password, `handleRegister` success, `EnforceEx` denied result fields, `Apply` remove policies, `Login` account locked, `Login` user not found, `Register` duplicate user ID, `SessionMiddleware` cookie/bearer/invalid token paths, `UserFromContextOr`, `UserIDFromRequest`, `errorStatus` all cases, `NewAuthHandler` defaults. Usermgmt coverage: 91.2% → 92.4%.
- [x] **Logging Push/Hijack coverage** — Added `mockPusher`, `pusherRecorder`, `hijackRecorder` test helpers. Tests verify Push delegation to underlying `http.Pusher` and `ErrNotSupported` fallback, plus Hijack delegation.
- [x] **Root coverage** — 96.1% → 97.0%.
- [x] **errorStatus dedup** — NOT RECOMMENDED: would couple usermgmt to go-cqrs-lite's event classification system. The modules serve different purposes.
- [x] **ValidateID adoption** — NOT NEEDED: `ParseUserID` already validates ULID format via `id.ParseUserID`. `ValidateID` only checks non-zero, which is already done at usage sites via `IsZero()`.
- [x] **Publisher/Subscriber ISP** — NOT APPLICABLE: cqrs-htmx dispatches commands/queries, doesn't publish events. The Publisher/Subscriber interfaces are for event sourcing infrastructure.

### New Items from 2026-05-21 Session

- [x] **Fix std/errors import in usermgmt/http.go** — Replaced with `cockroachdb/errors` to eliminate split brain.
- [x] **Eliminate all 7 lint warnings** — gochecknoglobals (excluded), noctx (NewRequestWithContext), prealloc (make), unparam (simplified helpers). golangci-lint now reports 0 issues.
- [x] **Expose CatalogEntries on App** — `CommandCatalogEntries()` and `QueryCatalogEntries()` delegate to go-cqrs-lite v1.4.0's embedded `CatalogDispatcher`.
- [x] **Fix CI pipeline** — Added `GOWORK=off`, removed broken `GOFLAGS=-insecure`, added usermgmt build/test jobs.
- [x] **CSRF helper coverage** — Added tests for `fieldName()`, `sameSite()`, `CSRFTokenFromContext` edge cases.
- [x] **Redirect URL coverage** — Added `data:`, scheme-relative, and unparseable URL test cases.
