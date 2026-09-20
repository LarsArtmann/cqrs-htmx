# Comprehensive Status Report — 2026-05-27

**Date:** 2026-05-27_05-39
**Commit:** 664e3ac
**Coverage:** 96.9% root, 91.2% usermgmt
**Lint:** 0 issues (root + usermgmt)
**Tests:** All pass with -race
**Build:** All 4 modules green

---

## a) FULLY DONE

### Documentation Overhaul (this session)

- README.md: Fixed 2 broken code examples (UserIDExtractor signature), rewrote CSRF section for nosurf, removed CSRFConfig.Secret, updated deps table, added missing handler options and helpers, fixed catalog return types, added recovery.go to arch tree, updated go-cqrs-lite version
- SECURITY.md: Full rewrite — removed gorilla/csrf claims, updated deps, removed stale hardening items
- CONTRIBUTING.md: Removed cockroachdb/errors note, added 9 missing files to dir tree, updated errors.go description
- CHANGELOG.md: Added Unreleased entries for nosurf + go-error-family migrations
- AGENTS.md: Restructured from 263 → 198 lines, deduped gotchas from 74 → 21, fixed stale deps
- TODO_LIST.md: Reduced from 171 → 46 lines, archived 168 DONE items, shows only open items
- FEATURES.md: Expanded from 30 → 43 features, categorized by area, updated metrics
- ROADMAP.md: Created from scratch with v1.1/v1.2/v2.0 milestones
- Planning doc: Created docs/planning/2026-05-27_05-16_documentation-quality-plan.md

### Code Quality

- go mod tidy on all 3 modules (clean)
- Added godoc to StatusRecorder.Write

---

## b) PARTIALLY DONE

### None currently. Everything committed is complete.

---

## c) NOT STARTED

### Confirmed Bugs (from brutal self-review)

| #  | Bug                                                                                             | Severity | File                    |
| -- | ----------------------------------------------------------------------------------------------- | -------- | ----------------------- |
| B1 | `GetUser` wraps ALL errors as `Transient` — `ErrUserNotFound` → 500 instead of 404              | High     | usermgmt/service.go:293 |
| B2 | `UpdateRoles` same Transient-wrapping bug for all errors                                        | Medium   | usermgmt/service.go:308 |
| B3 | Rate limiter fast-path doesn't update `lastUsed` — hot keys get evicted under TTL               | Medium   | ratelimit.go:248        |
| B4 | `FindByID`/`FindByEmail` return pointers to stored objects — callers can mutate store internals | Medium   | usermgmt/store.go:49-68 |
| B5 | `CSRFTokenHXHeaders` builds JSON via string concat — malformed if token contains `"`            | Low      | csrf_helpers.go:48      |

### Architecture Improvements (from review)

| #   | Item                                                                             | Impact | Effort |
| --- | -------------------------------------------------------------------------------- | ------ | ------ |
| A1  | Replace hand-rolled `Chain` with `justinas/alice`                                | Low    | 30min  |
| A2  | Replace `decodeFormValues` JSON round-trip with `gorilla/schema`                 | Medium | 1h     |
| A3  | Replace hand-rolled `StatusRecorder` with `go-chi/chi/middleware` or `httpsnoop` | Low    | 30min  |
| A4  | Deduplicate `handleCommandDispatch`/`handleQueryDispatch` into generic           | Medium | 2h     |
| A5  | Deduplicate `Command()`/`Query()` on App                                         | Medium | 1h     |
| A6  | Extract `handlerConfig` into grouped sub-structs                                 | Low    | 1h     |
| A7  | Add `clock` interface for testable time-dependent logic                          | Medium | 2h     |
| A8  | Defensive copy in `FindByID`/`FindByEmail`                                       | Low    | 15min  |
| A9  | Fix error handler double-call risk in `executePreDispatchChecks`                 | Low    | 30min  |
| A10 | Log error in `SessionMiddleware` when auth fails (instead of silent swallow)     | Low    | 15min  |

### usermgmt Improvements

| #   | Item                                                                               | Impact | Effort |
| --- | ---------------------------------------------------------------------------------- | ------ | ------ |
| U1  | Fix `GetUser`/`UpdateRoles` error wrapping — don't wrap domain errors as Transient | High   | 15min  |
| U2  | Add `*bool` or safe-zero-value for `HandlerConfig.Secure`                          | Medium | 30min  |
| U3  | Log auth failures in `SessionMiddleware`                                           | Low    | 15min  |
| U4  | Return defensive copies from store lookups                                         | Medium | 30min  |
| U5  | Add `List`/`FindAll` to `UserStore` interface                                      | Low    | 30min  |
| U6  | Rate limit on registration endpoint                                                | Medium | 1h     |
| U7  | Hash session tokens before storage                                                 | Medium | 1h     |
| U8  | Fix `Register` compensation — log rollback errors instead of `_ =`                 | Low    | 15min  |
| U9  | Fix `Authz.Apply` ordering — add before remove                                     | Medium | 30min  |
| U10 | Extract password validation to shared function (DRY)                               | Low    | 15min  |

---

## d) TOTALLY FUCKED UP

Nothing is catastrophically broken. The codebase compiles, passes all tests, has 0 lint issues, and the library works. The issues above are real but not blocking — they're quality/correctness improvements.

---

## e) WHAT WE SHOULD IMPROVE

### Type Model Improvements

1. **Error types**: The `Rejection` vs `Transient` split is good but inconsistently applied. `GetUser` wrapping `ErrUserNotFound` as `Transient` defeats the classification. The pattern should be: domain errors (not found, validation, conflict) → `Rejection`; infrastructure errors (database down, timeout) → `Transient`.

2. **`handlerConfig` god struct**: 14 fields mixing auth, decode, response, CSRF, rate limiting. Group into `authConfig`, `decodeConfig`, `responseConfig` sub-structs. Or use functional options that set typed config groups.

3. **Return types from store**: `*User` aliases stored state. Either return value copies or define read-only interface. This is a correctness issue for concurrent access.

4. **`any` overuse**: `Enforce(rvals ...any)`, `RenderFunc(result any)`, `TriggerWithDetail(detail any)`. These defeat compile-time safety. Consider typed alternatives or generics where feasible.

### Library Usage Opportunities

| Current                            | Replacement                                               | Why                                       |
| ---------------------------------- | --------------------------------------------------------- | ----------------------------------------- |
| Hand-rolled `Chain()`              | `justinas/alice`                                          | Battle-tested, same API                   |
| `decodeFormValues` JSON round-trip | `gorilla/schema`                                          | Purpose-built, type-safe                  |
| Hand-rolled `StatusRecorder`       | `go-chi/chi/middleware.WrapResponseWriter` or `httpsnoop` | Covers all http.ResponseWriter interfaces |
| `time.Now()` everywhere            | `clock.Clock` interface (or `benbjohnson/clock`)          | Testable time-dependent logic             |

---

## f) Top 25 Things We Should Get Done Next

Sorted by impact × effort (highest first):

| #  | Task                                                                      | Impact | Effort | Category      |
| -- | ------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Fix `GetUser` error wrapping (Transient → let domain errors pass through) | High   | 15min  | Bug           |
| 2  | Fix `UpdateRoles` error wrapping (same bug)                               | High   | 10min  | Bug           |
| 3  | Fix rate limiter `lastUsed` not updated on fast path                      | Medium | 20min  | Bug           |
| 4  | Fix `CSRFTokenHXHeaders` JSON concat → use `json.Marshal`                 | Low    | 10min  | Bug           |
| 5  | Defensive copy in `FindByID`/`FindByEmail`                                | Medium | 15min  | Correctness   |
| 6  | Log auth failures in `SessionMiddleware`                                  | Low    | 15min  | Observability |
| 7  | Fix `Register` compensation — log rollback errors                         | Low    | 15min  | Correctness   |
| 8  | Extract password validation to shared function                            | Low    | 15min  | DRY           |
| 9  | Fix `Authz.Apply` ordering — add before remove                            | Medium | 30min  | Correctness   |
| 10 | Fix error handler double-call risk in pre-dispatch checks                 | Low    | 30min  | Correctness   |
| 11 | Add rate limiting on registration endpoint                                | Medium | 1h     | Security      |
| 12 | Fix `HandlerConfig.Secure` zero-value trap                                | Medium | 30min  | Security      |
| 13 | Deduplicate `handleCommandDispatch`/`handleQueryDispatch`                 | Medium | 2h     | Architecture  |
| 14 | Deduplicate `Command()`/`Query()` on App                                  | Medium | 1h     | Architecture  |
| 15 | Replace `decodeFormValues` with `gorilla/schema`                          | Medium | 1h     | Perf/Quality  |
| 16 | Add `clock` abstraction for testable time logic                           | Medium | 2h     | Testability   |
| 17 | Replace hand-rolled `Chain` with `justinas/alice`                         | Low    | 30min  | Quality       |
| 18 | Replace hand-rolled `StatusRecorder` with `httpsnoop`                     | Low    | 30min  | Quality       |
| 19 | Group `handlerConfig` into sub-structs                                    | Low    | 1h     | Architecture  |
| 20 | Return value types from store (or read-only interface)                    | Medium | 30min  | Correctness   |
| 21 | Fix `WriteJSON` to encode to buffer before WriteHeader                    | Low    | 15min  | Correctness   |
| 22 | Add structured logging for dispatch failures                              | Medium | 1h     | Observability |
| 23 | Reduce `any` usage in public API (typed generics)                         | Low    | 2h     | Type safety   |
| 24 | Fix `Response.JSON` to propagate marshal errors                           | Low    | 30min  | Correctness   |
| 25 | Hash session tokens before storage                                        | Medium | 1h     | Security      |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Is the `handlerConfig` deduplication (items 13-14) worth the complexity cost?**

The `handleCommandDispatch` and `handleQueryDispatch` are structural twins (~70 lines duplicated). A generic `dispatch[Req any]` would eliminate the duplication but would also make the dispatch path harder to understand for contributors — they'd need to understand how generics interact with the `command.Command`/`query.Query` interfaces, the error handling flow, and the timeout/auth hooks.

Similarly, `Command()` and `Query()` on `App` are near-identical. But unifying them means consumers lose the type-safe distinction between "I'm registering a command handler" vs "I'm registering a query handler."

**My recommendation**: Do the dedup — the duplication is a maintenance hazard (bugs fixed in one path but not the other). Use clear naming and good comments. But I want your call before committing to this refactor scope.
