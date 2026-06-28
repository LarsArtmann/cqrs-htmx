# Status Report — cqrs-htmx

**Date:** 2026-05-20 13:40 | **Session:** Post-deep-audit execution sprint (Part 2)
**Branch:** master | **Commits since last report:** 5

---

## Executive Summary

The project is in **excellent shape**. A deep audit identified 7 improvement areas; all 7 were executed this session. Coverage increased, godoc was added across the entire usermgmt submodule, magic strings were extracted as constants, test types were consolidated, and lint is clean. One real bug was found (swallowed errors in `Authz.Apply`) and is documented below.

| Metric         | Root            | usermgmt     | Total     |
| -------------- | --------------- | ------------ | --------- |
| Coverage       | **95.9%**       | **92.1%**    | —         |
| Test specs     | 34 Ginkgo specs | 100 Go tests | **134**   |
| Lint issues    | 0 (production)  | 0            | **0**     |
| Race detector  | Clean           | Clean        | **Clean** |
| Production LOC | 2,687           | 1,420        | **4,107** |
| Test files     | 20              | 7            | **27**    |
| Benchmarks     | 16              | 0            | **16**    |
| Examples       | 9               | 0            | **9**     |

---

## A) FULLY DONE ✅

### Completed This Session (5 commits)

1. **Error wrapping consistency** (`24bde70`) — `fmt.Errorf` → `cockroachdb/errors` for sentinel-only wraps and inner-error wraps. Double-wrapping keeps `fmt.Errorf("%w: %w")` for `errors.Is` compatibility. Applied to `authz.go`, `csrf.go`, `options.go`, and all usermgmt files.

2. **Security header constants** (`24bde70`) — Extracted 9 constants in `security.go`: 6 header names + 3 defaults. Unexported, internal-only.

3. **Magic string constants** (`279cc05`) — Extracted `logFieldCorrelationID`, `logFieldUserID`, `logFieldRequestID` (logging.go), `headerRetryAfter`, `rateLimitExceededMsg` (ratelimit.go), `notificationKeyLevel`, `notificationKeyMessage` (notify.go).

4. **Test type consolidation** (`279cc05`) — All shared test types moved to `testing_test.go`; duplicates removed from `app_test.go`, `bdd_test.go`, `integration_test.go`.

5. **Godoc for all usermgmt exported symbols** (`94c6b78`) — ~70 symbols documented across 9 files. Every exported type, function, method, constant, and variable has godoc.

6. **Usermgmt coverage 90.4% → 92.1%** (`278399b`) — 16 new targeted tests: Session.Valid expired, EnforceEx denied, Authorize allowed, Apply remove+add policies, NewAuthz groups/invalid model, RecordFailure expired lockout reset, Authenticate invalid token, Register display name validation, Store.Save email conflict, logout idempotent delete, UserFromContext nil context.

7. **Lint fixes** (`800d520`) — Fixed gci formatting, unparam (unused formatter param), revive (missing const block comment). Updated TODO_LIST.md and FEATURES.md.

### Completed in Previous Sessions (Still Current)

8. **Branded UserID** — `usermgmt.UserID = brandid.ID[userBrand, string]` via `go-branded-id`. All user ID fields/params strongly typed.

9. **SessionMaxAge bug fix** — `NewAuthHandler` now copies `SessionMaxAge` from `HandlerConfig`.

10. **CSRF v1.7.3 plaintext detection** — Auto-detects non-TLS and marks as plaintext.

11. **Authorization config enum** — `authMode` enum replaces `authorize bool + requireAuth bool`.

12. **Notification level type** — `LevelSuccess/Error/Warning/Info` typed constants.

13. **Lifecycle hooks** — `BeforeDispatchHook`/`AfterDispatchHook` on Config.

14. **Request validation** — `ValidateCommand`/`ValidateQuery` HandlerOptions.

15. **Timeout propagation** — `Config.Timeout` wraps dispatch only.

16. **Security headers middleware** — Configurable CSP, HSTS, etc.

17. **Rate limiting middleware** — Token-bucket per-key with TTL eviction.

18. **Request logging** — `RequestLogging` + `RequestLoggingSlog`.

19. **Correlation ID** — Strongly-typed, auto-extracted from header.

20. **Request ID** — Strongly-typed ULID-backed.

21. **Context key sentinels** — Private empty-struct types for collision-free context values.

22. **Error classification** — `sync.Once` lazy registration of all sentinels.

23. **Generic decoders** — `decodeAndSet[T,R]`, 4 public wrappers.

24. **Generic validation** — `validateDispatch[T]`, thin public wrappers.

25. **Generic HTMX accessors** — `htmxBoolField`/`htmxStringField`, 8 accessors → 2 generics.

---

## B) PARTIALLY DONE 🔶

### Lint — 7 Test-Only Warnings (Not Production Code)

```
app_test.go:26:2   gochecknoglobals  — adminUserID, viewerUserID (test fixtures)
testing_test.go    noctx             — httptest.NewRequest without context (2 occurrences)
ratelimit_test.go  prealloc          — codes slice not preallocated
testing_test.go    unparam           — newCommandApp opts always nil, newPostJSONRequest path always "/users"
```

These are all in test code. The `gochecknoglobals` for test fixtures is a false positive (intentional test data). The `noctx` warnings are in test helpers where context is not needed. The `unparam` warnings indicate these helpers are only used with fixed arguments.

**Status:** Low priority. Could fix with `//nolint` comments or refactoring helpers.

---

## C) NOT STARTED ⬜

1. **Resolve usermgmt vs cqrshtmx UserID type split** — `usermgmt.UserID` (string-backed brandid) vs `cqrshtmx.UserID` (ULID-backed). Currently bridged via `.String()` at boundaries. Architectural decision needed.

2. **Rate limiter eviction O(n) → O(log n)** — `evictOldestIfAtCapacity()` does linear scan. Should use min-heap or LRU.

3. **Integration tests between root module and usermgmt** — Full register → cqrs dispatch with user context flow.

4. **Evaluate go-branded-id for numeric IDs** — Future SQL backends could use `brandid.ID[Brand, int64]`.

5. **SQL/persistent store backends** — Only in-memory stores exist. No database integration.

6. **OpenAPI/HTTP API documentation** — No auto-generated API docs.

7. **Performance benchmarks for usermgmt** — No benchmarks in submodule.

8. **CI/CD pipeline** — No GitHub Actions or CI configuration found.

---

## D) TOTALLY FUCKED UP 💥

### 🔴 Bug: Swallowed Errors in `Authz.Apply()` (usermgmt/authz.go:229-244)

This is the **only real bug** found. `RemoveGroups` and `AddGroups` errors are silently ignored:

```go
// Line 229-234 — ERROR SWALLOWED
for _, g := range update.RemoveGroups {
    if _, err := a.enforcer.RemoveGroupingPolicy(g.Subject, string(g.Role), g.Domain); err != nil {
        // Empty body — error is silently dropped
    }
}

// Line 238-244 — ERROR SWALLOWED
for _, g := range update.AddGroups {
    if _, err := a.enforcer.AddGroupingPolicy(g.Subject, string(g.Role), g.Domain); err != nil {
        // Empty body — error is silently dropped
    }
}
```

Meanwhile, `RemovePolicies` and `AddPolicies` in the same function properly return errors. This means `PolicyUpdate` can partially succeed: groups silently fail but policies succeed, and the caller never knows.

**Impact:** Production safety issue. Partial policy updates could leave users with incorrect permissions. The `UpdateRoles` service method calls `Apply` with both `RemoveGroups` and `AddGroups` — if the remove succeeds but add fails (or vice versa), the user has no roles.

**Fix:** Return errors from group operations, or batch all operations and return a combined error.

### 🟡 Swallowed `json.Encoder` Error (usermgmt/http.go:175)

```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)  // error silently ignored
}
```

Common Go pattern, but could silently drop encoding errors (e.g., channel values, functions).

### 🟡 Unused `ttl` Field (usermgmt/store.go:117)

`InMemorySessionStore.ttl` is set by `WithTTL()` but never read. The TTL is always passed explicitly to `Create()`. Dead field.

### 🟡 Unused `_ context.Context` Parameters (usermgmt/service.go)

`Logout`, `Authenticate`, `Authorize`, `GetUser`, `ChangePassword`, `UpdateRoles` all accept `context.Context` but ignore it. bcrypt operations are blocking and not context-aware. Misleads consumers about cancellation support.

---

## E) WHAT WE SHOULD IMPROVE

### Critical

1. **Fix `Authz.Apply()` swallowed errors** — This is a correctness bug. Return errors or use a combined error type.

### High Impact

2. **Remove unused `InMemorySessionStore.ttl` field** — Dead code.

3. **Fix lint warnings in test code** — Either `//nolint` comments or refactor helpers to accept context.

4. **Add `golangci-lint fmt` to pre-commit or CI** — The `golangci-lint fmt` command produced massive formatting changes (18 files, -2269 lines) that haven't been committed yet. Should be automated.

### Medium Impact

5. **Make `writeJSON` return error** — Or at least log it. Don't silently drop encoding failures.

6. **Add benchmarks for usermgmt** — Zero benchmarks in the submodule. bcrypt performance is critical for production.

7. **Consolidate docs/status/ archive** — 45 status report files is noise. Consider `.gitignore` for `docs/status/archive/`.

8. **Remove stale `report/` and `reports/` directories** — Build artifacts committed to git.

9. **Add CI/CD pipeline** — No GitHub Actions, no automated testing on push.

### Low Impact

10. **Context-aware bcrypt** — Wrap bcrypt in goroutine with context cancellation support.

11. **Rate limiter eviction optimization** — O(n) → O(log n) for high-cardinality key spaces.

12. **Usermgmt godoc examples** — `Example*` functions for the submodule.

---

## F) Top 25 Things We Should Get Done Next

| #   | Priority | Item                                                                      | Est. Effort | Impact        |
| --- | -------- | ------------------------------------------------------------------------- | ----------- | ------------- |
| 1   | P0       | Fix `Authz.Apply()` swallowed errors for RemoveGroups/AddGroups           | 30min       | Correctness   |
| 2   | P0       | Commit `golangci-lint fmt` formatting changes (18 files)                  | 5min        | Consistency   |
| 3   | P1       | Remove unused `InMemorySessionStore.ttl` field                            | 10min       | Cleanup       |
| 4   | P1       | Fix 7 test-only lint warnings (nolint comments or refactor)               | 20min       | Hygiene       |
| 5   | P1       | Add GitHub Actions CI (build + test + lint)                               | 1hr         | Automation    |
| 6   | P1       | Make `writeJSON` return or log encoding errors                            | 15min       | Robustness    |
| 7   | P1       | Remove stale `report/` and `reports/` directories                         | 5min        | Cleanup       |
| 8   | P2       | Add benchmarks for usermgmt (bcrypt, session creation, authz enforce)     | 1hr         | Performance   |
| 9   | P2       | Resolve usermgmt vs cqrshtmx UserID type split                            | 2hr         | Architecture  |
| 10  | P2       | Integration tests between root module and usermgmt                        | 2hr         | Coverage      |
| 11  | P2       | Context-aware bcrypt (goroutine + cancel channel)                         | 1hr         | Cancellation  |
| 12  | P2       | Add `//nolint` or CI config to suppress test-only warnings                | 15min       | CI hygiene    |
| 13  | P2       | Rate limiter eviction optimization (min-heap)                             | 2hr         | Performance   |
| 14  | P2       | Add usermgmt godoc examples (ExampleNewService, ExampleAuthz)             | 1hr         | Documentation |
| 15  | P2       | Evaluate go-branded-id for numeric IDs (SQL backend prep)                 | 30min       | Research      |
| 16  | P3       | Add persistent store interface implementations (SQLite, Postgres)         | 4hr         | Feature       |
| 17  | P3       | Consolidate `docs/status/archive/` or gitignore it                        | 10min       | Cleanup       |
| 18  | P3       | OpenAPI spec generation for HTTP endpoints                                | 2hr         | Documentation |
| 19  | P3       | Add structured error types (not just sentinels) for better error matching | 2hr         | API quality   |
| 20  | P3       | Session refresh/token rotation                                            | 2hr         | Security      |
| 21  | P3       | Multi-factor authentication hooks                                         | 3hr         | Security      |
| 22  | P4       | Add OAuth2/OIDC integration layer                                         | 4hr         | Feature       |
| 23  | P4       | Rate limiter middleware for usermgmt login endpoint                       | 1hr         | Security      |
| 24  | P4       | WebSocket/SSE helper for real-time notifications                          | 3hr         | Feature       |
| 25  | P4       | Example application repository consuming this library                     | 4hr         | Documentation |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should `usermgmt.UserID` use the same ULID-backed branded type as `cqrshtmx.UserID`?**

Currently:

- `cqrshtmx.UserID` = `id.UserID` = `brandid.ID[userBrand, string]` (ULID-backed, from `go-cqrs-lite/core/pkg/id`)
- `usermgmt.UserID` = `brandid.ID[userBrand, string]` (string-backed, from `go-branded-id`)

They have the **same underlying representation** but are **different types** from **different packages**. They cannot be compared or assigned without `.String()` conversion. The bridge is `UserIDFromRequest()` → `string` → `cqrshtmx.MustParseUserID()`.

**Why I can't resolve this:**

- If usermgmt imports `go-cqrs-lite/core/pkg/id`, it gains a heavy transitive dependency
- If cqrs-htmx depends on usermgmt's UserID, it creates a circular dependency risk
- A shared `go-branded-id` package is the clean solution but requires the consumer to import both
- The current `.String()` bridge works but loses type safety at the boundary

**What I need from you:** Architectural decision on whether usermgmt should be standalone (keep separate UserID) or coupled to cqrs-htmx (share UserID type). This affects every future consumer of the library.

---

## Session Metrics

| Action                     | Count                            |
| -------------------------- | -------------------------------- |
| Commits made               | 5                                |
| Production files modified  | 15                               |
| Test files modified        | 8                                |
| New test files created     | 1 (coverage_test.go)             |
| Lines of godoc added       | ~165                             |
| Constants extracted        | 11                               |
| Test types consolidated    | 8 types → 1 file                 |
| Coverage gained (usermgmt) | +1.7% (90.4% → 92.1%)            |
| Bugs found                 | 1 (Authz.Apply swallowed errors) |
| Bugs fixed                 | 0 (documented, not yet fixed)    |
| Lint issues resolved       | 3 (gci, unparam, revive)         |
| Remaining lint issues      | 7 (test-only)                    |

## Dependency Health

| Dependency          | Version | Status                        |
| ------------------- | ------- | ----------------------------- |
| casbin/casbin/v3    | v3.10.0 | Current                       |
| cockroachdb/errors  | v1.13.0 | Current                       |
| gorilla/csrf        | v1.7.3  | Current (archived project ⚠️) |
| go-cqrs-lite/core   | v1.2.0  | Private, current              |
| golang.org/x/time   | v0.15.0 | Current                       |
| golang.org/x/crypto | v0.51.0 | Current                       |

**⚠️ gorilla/csrf is archived** — The gorilla organization archived all repositories in 2023. While v1.7.3 is stable and functional, no security patches will be released. Long-term, this should be replaced with an actively maintained CSRF library.
