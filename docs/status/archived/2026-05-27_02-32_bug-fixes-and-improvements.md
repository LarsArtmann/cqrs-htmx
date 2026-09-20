# Status Report — cqrs-htmx

**Date:** 2026-05-27 02:32\
**Session:** Bug fixes + architectural improvements from go-modularize status report\
**Coverage:** 96.9% root, 91.2% usermgmt | **Lint:** 0 issues | **Race:** clean

---

## a) FULLY DONE

| #  | Item                                           | Detail                                                                                                                                              |
| -- | ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **Email normalization in usermgmt**            | `RegisterRequest.Validate()` and `LoginRequest.Validate()` now lowercase emails via `strings.ToLower`                                               |
| 2  | **Lockout email normalization**                | `AccountLockout` uses `normalizeEmail()` (lowercase + trim) for all keys, preventing case bypass                                                    |
| 3  | **RecoveryMiddleware deduplication**           | Extracted `shouldRePanic()` and `writePanicResponse()`; both `RecoveryMiddleware()` and `App.RecoveryMiddleware()` now delegate to shared helpers   |
| 4  | **Hardcoded `"application/json"` → constants** | `httputil.go` uses `ContentTypeJSON`; `usermgmt/http.go` uses local `contentTypeJSON` (respects zero-cross-import boundary)                         |
| 5  | **`isErr` → direct `errors.Is` calls**         | Removed `isErr()` wrapper entirely; all 7 call sites use `errors.Is` directly                                                                       |
| 6  | **SwapStrategy validation**                    | Added `SwapStrategy.Valid()` method with exhaustive switch over all 8 known strategies                                                              |
| 7  | **AccountLockout.IsLocked RLock optimization** | Changed from `Lock` to `RLock` for reads; upgrades to write lock only when expired lockout needs cleanup                                            |
| 8  | **E2E middleware chain integration test**      | Added 2 tests in `integration_test.go`: full chain dispatch (SecurityHeaders→Recovery→CSRF→HTMX→Context→CQRS) and panic recovery through full chain |
| 9  | **Case-insensitive test coverage**             | Added `TestService_Register_DuplicateEmail_CaseInsensitive`, `TestService_Login_CaseInsensitive`, `TestAccountLockout_CaseInsensitive`              |
| 10 | **SwapStrategy.Valid() test coverage**         | Added validation tests for known/unknown strategies in `htmx_test.go`                                                                               |
| 11 | **Lint clean after all changes**               | 0 issues in root and usermgmt; removed stale `//nolint:contextcheck` directives from recovery.go                                                    |

---

## b) PARTIALLY DONE

| # | Item                              | What's Done                                                   | What's Missing                                                                                                                                                                       |
| - | --------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1 | **Content-Type constant rollout** | Root `httputil.go` and usermgmt `http.go` updated             | Test helper `encodeJSONResult` in `testing_test.go` still hardcodes `"application/json"`; integration test request setup still hardcodes Content-Type header ( cosmetic — test-only) |
| 2 | **Test quality consolidation**    | E2E middleware chain test added; case-insensitive tests added | `assertStatusCode` still duplicated between root and usermgmt; app construction boilerplate still repeated 30+ times; `coverage_test.go` name still suggests coverage chasing        |

---

## c) NOT STARTED

| # | Item | Impact | Effort |
| --- | --------------------------------------------------------------------------------------- | ------------------------ | --------------- | -------------- |
| 1 | **Missing edge case tests** (MaxBodySize 413, timeout 503, concurrent lockout) | MEDIUM | 2 hrs | Testing |
| 2 | **authMode → typed string** | LOW — debuggability | LOW — 15 min |
| 3 | **usermgmt Register rollback error logging** (currently silently swallowed) | MEDIUM | 30 min | Error handling |
| 4 | **decoder.go form decode: document limitation or use gorilla/schema** | MEDIUM | 1 hr | Robustness |
| 5 | **usermgmt HandlerConfig → functional options** | MEDIUM — API consistency | MEDIUM — 2 hrs |
| 6 | **AccountLockout.IsLocked RLock optimization** | LOW — performance | LOW — 15 min |
| 7 | **SessionStore.Find expiration check** (currently returns expired sessions) | MEDIUM | 30 min | Correctness |
| 8 | **Test helper consolidation** (extract shared app construction, dedup assertStatusCode) | MEDIUM | 1 hr | Test quality |
| 9 | **coverage_test.go rename** (to behavior-based name) | LOW — 5 min | LOW — 5 min |
| 10 | **User.MarshalJSON explicit allowlist** (currently hides by exclusion) | LOW | 20 min | Security |
| 11 | **handleMe public DTO** (don't return full User JSON) | MEDIUM — 30 min | MEDIUM — 30 min |
| 12 | **HealthHandler deep health check** (verify dispatcher connectivity, not just non-nil) | LOW | 30 min | Observability |
| 13 | **CSRFConfig getters → `cmp.Or`** (simplify trio of identical getters) | LOW | 10 min | Code quality |
| 14 | **SecurityHeadersConfig getters → `cmp.Or`** | LOW — 10 min | LOW — 10 min |
| 15 | **Response.Apply() double-sanitize fix** (Redirect already sanitizes) | LOW | 10 min | Performance |
| 16 | **NewService config validation** (reject negative BcryptCost, zero SessionTTL) | MEDIUM | 20 min | Robustness |
| 17 | **DefaultLogFormatter use contextFields()** (like JSONLogFormatter does) | LOW | 10 min | Consistency |
| 18 | **datastar-demo LSP integration** (add to go.work or document stale LSP) | LOW | 10 min | DX |

---

## d) TOTALLY FUCKED UP

Nothing. All builds/tests pass. No regressions introduced.

---

## e) WHAT WE SHOULD IMPROVE

### Type Model

1. **UserID type divergence** — root uses `id.UserID` (go-cqrs-lite, ULID-backed), usermgmt uses `brandid.ID[userBrand, string]` (go-branded-id, string-backed). These are completely different types for the same concept. The integration bridge uses `.Get()` → `ParseUserID()`. Unifying these would eliminate boilerplate but requires architectural decision with consumer impact.

2. **authMode as typed string** — currently unexported `int` with `iota`. A typed string (`type authMode string` with constants) would be more debuggable in logs and eliminate the need for the `String()` method.

3. **GroupPolicy.User/Domain remain string** — Casbin boundary types. Intentional but could be documented more explicitly.

### Architecture

4. **decoder.go form→JSON hack** — `decodeFormValues` converts `url.Values` → JSON → `json.Unmarshal`. This is fragile (no int/bool coercion). Consider `gorilla/schema` or at minimum document the limitation.

5. **usermgmt error→HTTP mapping duplication** — `usermgmt/http.go` has its own `errorStatus` function while root has the full `MapError` + `event.Family` classification system. usermgmt can't use root's system without importing it (cross-module concern).

### Security

6. **Email normalization done** — ✅ `RegisterRequest.Validate()` and `LoginRequest.Validate()` now lowercase. `InMemoryUserStore` still stores whatever email is passed (normalized by Validate), but a persistent store must apply the same normalization.

7. **Lockout normalization done** — ✅ `AccountLockout` normalizes all keys. `EvictStale` still iterates raw keys (they're already normalized on insert).

### Test Quality

8. **coverage_test.go naming** — still suggests coverage chasing rather than behavior verification. Should be renamed to reflect what it actually tests (edge cases, error paths, integration scenarios).

9. **No test for concurrent lockout** — `AccountLockout` has RLock optimization but no explicit test exercises concurrent `RecordFailure` + `IsLocked` + `Reset` races.

10. **Test duplication** — `assertStatusCode` and app construction patterns duplicated between root and usermgmt. Extracting to shared test helpers would reduce boilerplate significantly.

### API Consistency

11. **usermgmt HandlerConfig → functional options** — `HandlerConfig` struct with variadic `cfg ...HandlerConfig` is inconsistent with root module's `HandlerOption` pattern. Refactoring to `func NewAuthHandler(service *Service, opts ...HandlerOption)` would align APIs.

---

## f) Top 25 Things We Should Get Done Next

Sorted by **Impact × Effort** (highest first):

| #  | Item                                                                                    | Impact | Effort  | Category        |
| -- | --------------------------------------------------------------------------------------- | ------ | ------- | --------------- |
| 1  | **Missing edge case tests** (MaxBodySize 413, timeout 503, concurrent lockout)          | MEDIUM | 2 hrs   | Testing         |
| 2  | **E2E middleware chain integration test**                                               | HIGH   | ✅ DONE | Testing         |
| 3  | **usermgmt Register rollback error logging** (currently silently swallowed)             | MEDIUM | 30 min  | Error handling  |
| 4  | **decoder.go form decode: document limitation or use gorilla/schema**                   | MEDIUM | 1 hr    | Robustness      |
| 5  | **usermgmt HandlerConfig → functional options**                                         | MEDIUM | 2 hrs   | API consistency |
| 6  | **SessionStore.Find expiration check** (currently returns expired sessions)             | MEDIUM | 30 min  | Correctness     |
| 7  | **Test helper consolidation** (extract shared app construction, dedup assertStatusCode) | MEDIUM | 1 hr    | Test quality    |
| 8  | **handleMe public DTO** (don't return full User JSON)                                   | MEDIUM | 30 min  | Security        |
| 9  | **NewService config validation** (reject negative BcryptCost, zero SessionTTL)          | MEDIUM | 20 min  | Robustness      |
| 10 | **SwapStrategy validation**                                                             | LOW    | ✅ DONE | Type safety     |
| 11 | **authMode → typed string**                                                             | LOW    | 15 min  | Debuggability   |
| 12 | **AccountLockout.IsLocked RLock optimization**                                          | LOW    | ✅ DONE | Performance     |
| 13 | **coverage_test.go rename** (to behavior-based name)                                    | LOW    | 5 min   | Test quality    |
| 14 | **User.MarshalJSON explicit allowlist** (currently hides by exclusion)                  | LOW    | 20 min  | Security        |
| 15 | **HealthHandler deep health check** (verify dispatcher connectivity, not just non-nil)  | LOW    | 30 min  | Observability   |
| 16 | **CSRFConfig getters → `cmp.Or`** (simplify trio of identical getters)                  | LOW    | 10 min  | Code quality    |
| 17 | **SecurityHeadersConfig getters → `cmp.Or`**                                            | LOW    | 10 min  | Code quality    |
| 18 | **Response.Apply() double-sanitize fix** (Redirect already sanitizes)                   | LOW    | 10 min  | Performance     |
| 19 | **DefaultLogFormatter use contextFields()** (like JSONLogFormatter does)                | LOW    | 10 min  | Consistency     |
| 20 | **datastar-demo LSP integration** (add to go.work or document stale LSP)                | LOW    | 10 min  | DX              |
| 21 | **UserID type unification** (root vs usermgmt divergence)                               | HIGH   | 2 hrs   | Architecture    |
| 22 | **SQL store backend for usermgmt**                                                      | HIGH   | 4 hrs   | Production      |
| 23 | **OpenTelemetry integration**                                                           | MEDIUM | 2 hrs   | Observability   |
| 24 | **WebSocket/SSE helpers**                                                               | MEDIUM | 4 hrs   | Feature         |
| 25 | **BrandNamer for root module marker types**                                             | LOW    | BLOCKED | Upstream        |

---

## g) Top #1 Question

**Should we unify the UserID type across root and usermgmt, and if so, where does it live?**

The root module uses `id.UserID` from `go-cqrs-lite/core/pkg/id` (ULID-backed). The usermgmt submodule uses `brandid.ID[userBrand, string]` from `go-branded-id` (string-backed). They're incompatible — the integration bridge manually converts via `.Get()` → `ParseUserID()`.

Options:

- A) **usermgmt adopts root's `id.UserID`** — breaks `go-branded-id` usage, ties usermgmt to go-cqrs-lite's ID system
- B) **Root adopts `brandid.ID`** — breaks ULID guarantee, affects context.go, errors.go, all consumers
- C) **Create shared `types/` module** — new go.mod with shared UserID type, both import it — adds a 5th module for 1 type
- D) **Keep as-is with better bridge documentation** — accept the divergence as intentional (different backing stores: ULID vs string), improve bridge ergonomics

This is an architectural decision with consumer impact. I cannot decide this alone.
