# Comprehensive Status Report — cqrs-htmx

**Date:** 2026-05-20 03:45 | **Branch:** master | **Commits:** 191 total (14 since last push)

---

## Executive Summary

cqrs-htmx is a **Go library** (not an application) that makes it easy to use go-cqrs-lite with HTMX, templ, and Casbin authorization. The project is in **excellent health** after a massive session spanning P5 deduplication, branded UserID migration, bug fixes, and test coverage improvements.

**Overall Health: 🟢 EXCELLENT (Grade: A)**

| Metric             | Root Module      | usermgmt Submodule |
| ------------------ | ---------------- | ------------------ |
| Coverage           | **95.6%**        | **88.5%**          |
| Test specs         | 289              | 80+                |
| Lint issues        | 0                | 0                  |
| Race detector      | ✅ Clean         | ✅ Clean           |
| Prod Go files      | 16 (2,890 lines) | 9 (1,445 lines)    |
| Test Go files      | 19 (5,450 lines) | 7 (1,393 lines)    |
| Build              | ✅ Clean         | ✅ Clean           |
| TODO/FIXME in code | 0                | 0                  |
| Hardcoded secrets  | 0                | 0                  |

---

## a) FULLY DONE ✅

### Core Library (16 production files)

1. **App Builder** — `New(Config)` with validation, per-App LoginRedirect, timeout, maxBodySize
2. **Command/Query Dispatch** — Generic flow: decode → auth → dispatch → HTMX response
3. **JSON/Form Decoders** — `DecodeJSON[T]`, `DecodeJSONQuery[T]`, `DecodeForm[T]`, `DecodeFormQuery[T]` — now backed by `decodeAndSet[T,R]` generic
4. **Casbin Authorization** — `Authorize`, `RequireAuth`, `Enforce`, `AuthorizeMiddleware`
5. **User Identity Propagation** — Branded `UserID`/`CorrelationID`/`RequestID` → context → event metadata
6. **HTMX Request Context** — `HTMXMiddleware` parses all headers → context. Accessors use generic `htmxBoolField`/`htmxStringField`
7. **HTMX Response Builder** — Fluent `Response` with PushURL, Redirect, Refresh, Location, Reswap, Retarget, Reselect, Trigger, Notifications
8. **Notifications** — HandlerOptions + Response methods via `notificationDetail()` shared helper. `NotifyWithEvent` builder
9. **Templ Integration** — `RenderTempl(component)`, `RenderTemplResult[T](mapper)`, duck-typed interface
10. **Error Classification** — `sync.Once` sentinel registration, `MapError` CQRS → HTTP mapping. Shared `handleErrorCore()` for Default + JSON handlers
11. **Lifecycle Hooks** — `BeforeDispatchHook` / `AfterDispatchHook` with context propagation
12. **Correlation ID** — Strongly-typed `id.CorrelationID`, auto-extracted from `X-Correlation-ID` header
13. **Request Validation** — `ValidateCommand`/`ValidateQuery` backed by `validateDispatch[T]` generic
14. **Timeout Propagation** — `Config.Timeout` wraps dispatch only, per-handler override via `WithTimeout`
15. **Request Logging** — `RequestLogging(formatter, writer)`, `JSONLogFormatter`. Shared `contextFields(r)` helper
16. **Rate Limiting** — Token-bucket per key, `MaxKeys` cap with `evictOldestIfAtCapacity()`, `Retry-After` header
17. **CSRF Protection** — `CSRFMiddleware` + `CSRFProtect` (cached instance), template helpers in `csrf_helpers.go`, gorilla/csrf v1.7.3 with auto plaintext HTTP detection
18. **Security Headers** — `SecurityHeadersMiddleware` with configurable CSP, HSTS, Permissions-Policy
19. **Request ID** — `NewRequestID`/`ParseRequestID`/`MustParseUserID` backed by `parseID[T]` generic

### usermgmt Submodule (9 production files)

1. **Service** — Register, Login, Logout, Authenticate, ChangePassword, UpdateRoles with `context.Context`
2. **Branded UserID** — `usermgmt/id.go` defines `UserID = brandid.ID[userBrand, string]` via `go-branded-id v0.1.0`. Compile-time type safety across all 6 files
3. **RBAC Authorization** — Casbin v3 with domain-based RBAC, `EnforceAny`, `AsEnforcer`, full policy CRUD
4. **User/Session Types** — bcrypt hashing with configurable cost, session management
5. **In-Memory Stores** — `UserStore` with email index, `SessionStore` with TTL. All methods error-context-enriched
6. **HTTP Handlers** — `AuthHandlers` with session cookie management, `SessionMiddleware`, configurable `SessionMaxAge`
7. **Account Lockout** — Configurable max attempts + duration with lazy eviction on read. `ErrAccountLocked` (429)
8. **Input Validation** — `RegisterRequest.Validate()`, `LoginRequest.Validate()` (email, password length)
9. **Structured Logging** — `ServiceConfig.Logger` defaults to `slog.Default()`

### Infrastructure & Quality

1. **Generic Deduplication** — 6 generic helpers extracted: `decodeAndSet[T,R]`, `validateDispatch[T]`, `htmxBoolField`/`htmxStringField`, `parseID[T]`, `handleErrorCore()`, `contextFields()`, `notificationDetail()`
2. **CSRF File Split** — Template helpers in `csrf_helpers.go` (65 lines), core middleware in `csrf.go` (386 lines)
3. **ADR: HTMX_GO_DECISION** — Documented explicit rejection of `angelofallars/htmx-go` dependency
4. **CI/CD** — GitHub Actions with golangci-lint, 0 issues
5. **Documentation** — AGENTS.md, FEATURES.md, TODO_LIST.md, CONTRIBUTING.md, 4 D2 architecture diagrams
6. **Test Infrastructure** — Ginkgo/Gomega, shared test helpers, 16 benchmarks, 9 godoc examples

### Bugs Fixed This Session (since `b827417`)

| Bug                                                   | Fix                                 | Commit    |
| ----------------------------------------------------- | ----------------------------------- | --------- |
| gorilla/csrf v1.7.3 breaks httptest (7 test failures) | Auto-detect non-TLS, mark plaintext | `9dcae5a` |
| `applyQueryResponse` bypassed `handleErr` helper      | Route through centralized helper    | `3a7f04b` |
| AccountLockout maps grew unbounded                    | Evict expired entries in `IsLocked` | `b27fe2c` |
| Error messages in usermgmt lost userID context        | Added `userID` to 8 error sites     | `98fd897` |
| Rate limiter could grow unbounded                     | Added `MaxKeys` cap with eviction   | `b41deb5` |
| CSRF `Protect()` allocated per-request                | Cache instance on `handlerConfig`   | `cb50931` |
| `SessionMaxAge` silently ignored in `NewAuthHandlers` | Copy from `HandlerConfig`           | `cb50931` |
| Duplicate code across 8 files (40 clone groups)       | 6 generic helpers extracted         | `d5a272e` |

---

## b) PARTIALLY DONE 🟡

1. **csrf.go still 386 lines** — Template helpers (65 lines) extracted to `csrf_helpers.go`, but the remaining file still has 4 responsibilities: config, context helpers, middleware factory, per-handler validation. Needs further split into `csrf_config.go` + `csrf_middleware.go`.

2. **usermgmt test coverage at 88.5%** — Up from 84.9% but still below root module's 95.6%. `http.go` handler edge cases and some error paths in `service.go` remain uncovered.

3. **UserID type split** — `usermgmt.UserID` is `brandid.ID[userBrand, string]` (string-backed) while `cqrshtmx.UserID` is `id.UserID` (ULID-backed). These are incompatible types. Decision depends on whether usermgmt is always paired with cqrs-htmx or can be standalone.

4. **Rate limiter eviction is O(n)** — `evictOldestIfAtCapacity()` does linear scan of all entries. Works fine for small maps, but for high-cardinality deployments (thousands of keys), a min-heap or LRU would be O(log n).

---

## c) NOT STARTED ⬜

1. **Integration tests between root module and usermgmt** — Full register → cqrs dispatch with user context flow. The two modules have never been tested together.

2. **Fuzz testing** — No `Fuzz*` functions exist. Decoder, CSRF token parsing, form decoding are good candidates.

3. **docs/adr/ directory** — ADR exists as root-level `HTMX_GO_DECISION.md` but no `docs/adr/` directory with numbered ADRs.

4. **flake.nix migration** — justfile is deprecated per AGENTS.md, flake.nix doesn't exist.

5. **WebSocket/SSE support** — Listed as NOT_PLANNED in FEATURES.md.

6. **`minPasswordLength` constant** — Magic number `8` appears in multiple validation sites in usermgmt.

7. **`type Role string` in usermgmt** — Roles are loose `[]string` throughout. No compile-time validation.

8. **`GroupPolicy.User` typed as `UserID`** — Currently raw string.

---

## d) TOTALLY FUCKED UP 💥 → FIXED ✅

### gorilla/csrf v1.7.3 Regression — **FIXED** (commit `9dcae5a`)

- v1.7.3 defaults to HTTPS, enforces strict Origin/Referer checks
- httptest requests have no TLS, no Origin/Referer → 7 tests fail with 403
- Fix: `r.TLS == nil` → `csrf.PlaintextHTTPRequest(r)` in both `CSRFMiddleware` and `executeCSRFValidation`
- Previous downgrade to v1.7.2 (commit `ffd748e`) was a workaround; now properly fixed with v1.7.3

### SessionMaxAge silently ignored — **FIXED** (commit `cb50931`)

- `NewAuthHandlers` did not copy `SessionMaxAge` from `HandlerConfig`
- All sessions always got 86400s TTL regardless of config
- Simple one-line fix: copy the field in constructor

### applyQueryResponse error inconsistency — **FIXED** (commit `3a7f04b`)

- `applyQueryResponse` called `a.errorHandler` + `a.afterDispatch` directly instead of `handleErr`
- Any future error-handling changes would silently skip the render error path
- Now routes through centralized `handleErr` helper

### AccountLockout memory leak — **FIXED** (commit `b27fe2c`)

- `IsLocked` used `RLock` and returned false for expired entries without cleanup
- Both `attempts` and `lockedAt` maps grew unbounded
- Now uses full `Lock` to evict expired entries on read

---

## e) WHAT WE SHOULD IMPROVE

### High Priority

1. **Split `csrf.go` (386 lines)** — Extract config methods + context helpers to `csrf_config.go`, validation logic to `csrf_validation.go`. Target: each file under 200 lines.

2. **Raise usermgmt coverage to 92%+** — Focus on `http.go` handler error paths and `service.go` edge cases (empty userID, concurrent access).

3. **`type Role string` with constants** — Replace loose `[]string` roles with a branded type. Prevents typos like `"admim"` being stored and enforced.

4. **`minPasswordLength` constant** — Extract magic number `8` to a named constant used in both `RegisterRequest.Validate()` and `ChangePassword()`.

5. **Integration tests root ↔ usermgmt** — Full flow: register → login → cqrs dispatch with user context → HTMX response.

### Medium Priority

6. **Rate limiter eviction optimization** — Replace O(n) scan with O(log n) min-heap or LRU for high-cardinality deployments.

7. **Resolve UserID type split** — Decide: should `usermgmt.UserID` use ULID backing (breaking change but unified types) or stay string-backed (flexible but fragmented)?

8. **Move `HTMX_GO_DECISION.md` to `docs/adr/`** — Create proper ADR directory with numbered decisions.

9. **Fuzz tests** — Decoder, form parsing, CSRF token parsing, URL sanitization.

10. **Extract `newStatusRecorder(w)` helper** — `logging.go` creates `statusRecorder` inline; could be shared if other middlewares need status capture.

### Low Priority

11. **Naming cleanup** — `TriggerID` field in `HTMXRequest` could be confusing vs `TriggerName`. Document the distinction clearly.
12. **`GroupPolicy.User` → `UserID` type** — Currently raw `string`.
13. **`RateLimiterConfig.Limit/Burst` → `uint`** — Currently `int` but negative values make no sense.
14. **`LockoutConfig.MaxAttempts` → `uint`** — Same.
15. **Coverage file organization** — Move `coverage.out` to `coverage/` directory.
16. **Status report cleanup** — 42 status reports in `docs/status/`, many redundant. Archive old ones.

### Already Excellent (Don't Touch)

- Error classification system (`sync.Once` sentinel registration)
- Generic deduplication (6 helpers, clean codebase)
- Branded UserID in usermgmt via `go-branded-id`
- CSRF v1.7.3 plaintext HTTP auto-detection with cached Protect instance
- Enforcer interface (composition over concrete Casbin)
- Test infrastructure (Ginkgo/Gomega, shared helpers, BDD tests)

---

## f) Top 25 Things We Should Get Done Next

### P0 — This Session (High Impact, Low Effort)

| #   | Item                                               | Effort | Impact |
| --- | -------------------------------------------------- | ------ | ------ |
| 1   | Split `csrf.go` → config + middleware + validation | 15m    | HIGH   |
| 2   | Extract `minPasswordLength` constant in usermgmt   | 3m     | MED    |
| 3   | `type Role string` with constants in usermgmt      | 15m    | HIGH   |
| 4   | `GroupPolicy.User` → `UserID` type                 | 5m     | MED    |
| 5   | `RateLimiterConfig.Limit/Burst` → `uint`           | 10m    | LOW    |

### P1 — This Week

| #   | Item                                        | Effort | Impact |
| --- | ------------------------------------------- | ------ | ------ |
| 6   | Raise usermgmt coverage to 92%+             | 2h     | HIGH   |
| 7   | Integration tests root ↔ usermgmt           | 2h     | HIGH   |
| 8   | Create `docs/adr/` with numbered ADRs       | 1h     | MED    |
| 9   | Add fuzz tests for decoder and form parsing | 1h     | MED    |
| 10  | Resolve UserID type split decision          | 30m    | HIGH   |

### P2 — Next Sprint

| #   | Item                                                  | Effort | Impact |
| --- | ----------------------------------------------------- | ------ | ------ |
| 11  | Rate limiter O(log n) eviction (min-heap)             | 2h     | MED    |
| 12  | Migrate to flake.nix build system                     | 2h     | MED    |
| 13  | Expand benchmark suite (CSRF, rate limit, middleware) | 1h     | MED    |
| 14  | Fix `RenderPartial` split brain (context vs header)   | 30m    | LOW    |
| 15  | Add cookie session store (not just in-memory)         | 2h     | HIGH   |
| 16  | Add password reset flow to usermgmt                   | 2h     | MED    |

### P3 — Backlog

| #   | Item                                               | Effort | Impact |
| --- | -------------------------------------------------- | ------ | ------ |
| 17  | Add email verification flow to usermgmt            | 2h     | MED    |
| 18  | Add SSE/EventStream helper for real-time updates   | 3h     | HIGH   |
| 19  | Add OAuth2/OIDC integration hooks in usermgmt      | 3h     | HIGH   |
| 20  | Move `coverage.out` to `coverage/` directory       | 5m     | LOW    |
| 21  | Archive old `docs/status/` reports (42 → 10)       | 30m    | LOW    |
| 22  | Add 100% godoc coverage for all exported types     | 2h     | MED    |
| 23  | Performance profiling and optimization pass        | 2h     | LOW    |
| 24  | Add multi-tenancy support via Casbin domains       | 2h     | MED    |
| 25  | Create visual architecture diagram (D2) for README | 1h     | MED    |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `usermgmt.UserID` switch from `brandid.ID[userBrand, string]` to `id.UserID` (ULID-backed) to unify with the root module?**

Current state:

- `cqrshtmx.UserID` = `id.UserID` (ULID-backed, 26 chars, from `go-cqrs-lite`)
- `usermgmt.UserID` = `brandid.ID[userBrand, string]` (string-backed, any format)

**For unification (switch to ULID):**

- Single type across the ecosystem — no conversion at the boundary
- Stronger guarantee: all user IDs are valid ULIDs
- But: `usermgmt` becomes dependent on `go-cqrs-lite/core/pkg/id`, breaking its independence

**Against (keep string-backed):**

- `usermgmt` stays self-contained — no `go-cqrs-lite` dependency
- More flexible: consumers can use UUIDs, integers, or any format
- But: requires conversion when bridging between modules, and the type mismatch is confusing

**My recommendation:** Keep string-backed in usermgmt. The submodule is designed to be standalone. The `AsEnforcer()` bridge pattern already exists for crossing module boundaries — a similar `ResolveUserID()` bridge function at the integration layer would handle the conversion cleanly without coupling the modules.

---

## Dependency Status

| Dependency         | Version | Status                 |
| ------------------ | ------- | ---------------------- |
| go-cqrs-lite/core  | v1.2.0  | ✅ Current             |
| casbin/casbin/v3   | v3.10.0 | ✅ Current             |
| gorilla/csrf       | v1.7.3  | ✅ Fixed               |
| cockroachdb/errors | v1.13.0 | ✅ Current             |
| golang.org/x/time  | v0.15.0 | ✅ Current             |
| go-branded-id      | v0.1.0  | ✅ New (usermgmt only) |

## Recent Commits (since last push)

```
cb50931 feat(usermgmt): branded UserID, SessionMaxAge bugfix, CSRF caching, test coverage
d1f30fd docs(adr): add HTMX_GO_DECISION.md
5f4cd1c docs: reformat status report tables + fix Go type parameter syntax
8b5fb6c docs: post-P5 comprehensive review — status report with top 25 next tasks
d5a272e refactor: deduplicate code across 8 files using generics and shared helpers
b41deb5 feat(ratelimit): add MaxKeys cap to prevent unbounded map growth
1b8de1e feat(usermgmt): complete branded UserID migration — tests and docs
eddf90d docs: comprehensive status report post-9-skill review
191c559 feat(usermgmt): introduce branded UserID type for compile-time safety
f4fca56 docs: comprehensive 9-skill review, audit reports, and P5 execution plan
5368e1e style(usermgmt): fix SA1012 warnings and modernize rangeint
98fd897 fix(usermgmt): add userID context to error returns
82b8ddf docs(ratelimit): fix misleading unbounded growth warning
b27fe2c fix(usermgmt): evict expired lockout entries in IsLocked
```

---

_Generated at 2026-05-20 03:45 by Crush_
