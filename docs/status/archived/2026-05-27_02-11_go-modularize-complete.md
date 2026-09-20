# Status Report — cqrs-htmx

**Date:** 2026-05-27 02:11\
**Session:** go-modularize + execution mode\
**Coverage:** 96.9% root, 91.1% usermgmt | **Lint:** 0 issues | **Race:** clean

---

## a) FULLY DONE

| #  | Item                                 | Detail                                                                                                                                |
| -- | ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **7-phase modularization analysis**  | Phase 1 (detect) → Phase 7 (reflection). Conclusion: existing 4-module structure is optimal. Root module should NOT be split further. |
| 2  | **datastar-demo migrated to v1.5.1** | `command.Core` → `command.BasicCommand`, `query.Core` → `query.BasicQuery`, struct field references updated                           |
| 3  | **datastar-demo lint fixes**         | `strings.Builder` for HTML concatenation, `strings.Cut` for extractTitle                                                              |
| 4  | **usermgmt version alignment**       | go-cqrs-lite v1.5.0 → v1.5.1-pre, Go 1.26.2 → 1.26.3, go-error-family v0.1.0 → v0.1.1                                                 |
| 5  | **ALL dependency versions aligned**  | All 4 modules: go-cqrs-lite v1.5.1-pre, go-error-family v0.1.1, go-branded-id v0.3.0, Go 1.26.3                                       |
| 6  | **go mod tidy** in all 4 modules     | Clean                                                                                                                                 |
| 7  | **Full test suite passes**           | Root + usermgmt + integration_test with `-race`. Datastar-demo builds.                                                                |
| 8  | **Lint clean**                       | 0 issues in root and usermgmt                                                                                                         |
| 9  | **docs/modularization/ updated**     | PROPOSAL.md, DEPENDENCY_GRAPH.md, EXECUTION_PLAN.md all reflect final state                                                           |
| 10 | **AGENTS.md updated**                | Dependencies table, gotchas, key decisions reflect current state                                                                      |

---

## b) PARTIALLY DONE

| # | Item                               | What's Done                                                        | What's Missing                                                  |
| - | ---------------------------------- | ------------------------------------------------------------------ | --------------------------------------------------------------- |
| 1 | **Root module architecture audit** | Dependency graph, coupling analysis, cycle identification done     | Action items not executed (not in scope for modularize session) |
| 2 | **Test quality audit**             | Identified duplication patterns, coverage gaps, missing edge cases | No test refactoring executed                                    |

---

## c) NOT STARTED

| #  | Item                                                        | Impact                             | Effort                                    |
| -- | ----------------------------------------------------------- | ---------------------------------- | ----------------------------------------- |
| 1  | Email normalization (lowercase) in usermgmt Register/Login  | HIGH — prevents duplicate accounts | LOW — 1 line change                       |
| 2  | RecoveryMiddleware deduplication                            | MEDIUM — reduce copy-paste         | LOW — extract shared panic recovery logic |
| 3  | `isErr` → `errors.Is` replacement                           | LOW — cosmetic                     | LOW — 1 line per call site                |
| 4  | Hardcoded `"application/json"` → `ContentTypeJSON` constant | LOW — consistency                  | LOW — ~5 files                            |
| 5  | SwapStrategy validation (constructor or IsValid)            | LOW — defensive                    | LOW — add validation method               |
| 6  | usermgmt `writeJSON` → use root's `httputil.WriteJSON`      | MEDIUM — dedup                     | MEDIUM — cross-module import concern      |
| 7  | Full E2E middleware chain integration test                  | HIGH — confidence in real usage    | MEDIUM — 1 new test file                  |
| 8  | Lockout email normalization (case-insensitive key)          | MEDIUM — lockout bypass            | LOW — normalize before lookup             |
| 9  | AccountLockout.IsLocked RLock optimization                  | LOW — performance                  | LOW — use RLock, upgrade if expired       |
| 10 | usermgmt HandlerConfig → functional options pattern         | MEDIUM — API consistency           | MEDIUM — refactor all callers             |

---

## d) TOTALLY FUCKED UP

| # | Item                                       | What Went Wrong                                                                                                                                                                                                                          | Fix                                                                |
| - | ------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| 1 | **renderTodoList deleted during lint fix** | The `strings.Builder` edit in handlers.go replaced the function definition but the old_string only matched the body, not the full function signature. Result: function was deleted. Caught during final verification, fixed immediately. | ✅ Fixed — function restored with `strings.Builder` implementation |

Nothing else fucked up. All builds/tests pass.

---

## e) WHAT WE SHOULD IMPROVE

### Type Model

1. **UserID type divergence** — root uses `id.UserID` (go-cqrs-lite, ULID-backed), usermgmt uses `brandid.ID[userBrand, string]` (go-branded-id, string-backed). These are completely different types for the same concept. The integration bridge uses `.Get()` → `ParseUserID()`. This is the **#1 type safety issue**. A unified UserID type (or at minimum a shared interface) would eliminate bridge boilerplate.

2. **authMode as typed string** — currently unexported `int` with `iota`. A typed string (`type authMode string` with constants) would be more debuggable in logs and eliminate the need for the `String()` method.

3. **SwapStrategy unvalidated** — any arbitrary string can be passed. Add `ValidSwapStrategies` set or constructor that validates.

### Architecture

4. **RecoveryMiddleware duplication** — `RecoveryMiddleware()` and `App.RecoveryMiddleware()` share 90% of logic. Extract a shared `recoverFromPanic(w, r, err, handler)` internal function.

5. **decoder.go form→JSON hack** — `decodeFormValues` converts `url.Values` → JSON → `json.Unmarshal`. This is fragile (no int/bool coercion). Consider `gorilla/schema` or at minimum document the limitation.

6. **usermgmt error→HTTP mapping duplication** — `usermgmt/http.go` has its own `errorStatus` function while root has the full `MapError` + `event.Family` classification system. usermgmt can't use root's system without importing it (circular concern).

### Library Usage

7. **StatusRecorder** — hand-rolled `http.ResponseWriter` wrapper in `logging.go`. Could use `felixge/httpsnoop` for battle-tested implementation. BUT: current implementation works, has tests, and the dependency cost may not be worth it.

8. **Rate limiter** — 150 lines of hand-rolled rate limiter with min-heap eviction. Already uses `golang.org/x/time/rate`. Could use `ulule/limiter` for a more complete solution. BUT: current implementation is well-tested and self-contained.

### Security

9. **Email normalization** — usermgmt doesn't lowercase emails. `Test@Example.com` and `test@example.com` create separate accounts. This is a **real production bug**.

10. **Lockout key normalization** — `AccountLockout` uses raw email as key, so `Test@Example.com` bypasses `test@example.com`'s lockout.

### Test Quality

11. **coverage_test.go naming** — suggests coverage chasing rather than behavior verification. Should be renamed to reflect what it tests.

12. **No E2E middleware chain test** — no test exercises the full middleware → handler → CQRS dispatch pipeline.

13. **Test duplication** — `assertStatusCode` duplicated between root and usermgmt. App construction boilerplate repeated 30+ times.

---

## f) Top 25 Things We Should Get Done Next

Sorted by **Impact × Effort** (highest first):

| #  | Item                                                                                    | Impact | Effort | Category        |
| -- | --------------------------------------------------------------------------------------- | ------ | ------ | --------------- |
| 1  | **Email normalization in usermgmt** (lowercase in Register/Login)                       | HIGH   | 1 line | Bug fix         |
| 2  | **Lockout email normalization** (lowercase key)                                         | HIGH   | 1 line | Bug fix         |
| 3  | **RecoveryMiddleware deduplication** (extract shared recovery function)                 | MEDIUM | 30 min | Architecture    |
| 4  | **Hardcoded `"application/json"` → `ContentTypeJSON`**                                  | LOW    | 10 min | Consistency     |
| 5  | **`isErr` → direct `errors.Is` calls**                                                  | LOW    | 10 min | Code quality    |
| 6  | **E2E middleware chain integration test**                                               | HIGH   | 2 hrs  | Testing         |
| 7  | **Missing edge case tests** (MaxBodySize 413, timeout 503, concurrent lockout)          | MEDIUM | 2 hrs  | Testing         |
| 8  | **SwapStrategy validation** (constructor or IsValid)                                    | LOW    | 15 min | Type safety     |
| 9  | **authMode → typed string**                                                             | LOW    | 15 min | Debuggability   |
| 10 | **usermgmt Register rollback error logging** (currently silently swallowed)             | MEDIUM | 30 min | Error handling  |
| 11 | **decoder.go form decode: document limitation or use gorilla/schema**                   | MEDIUM | 1 hr   | Robustness      |
| 12 | **usermgmt HandlerConfig → functional options**                                         | MEDIUM | 2 hrs  | API consistency |
| 13 | **AccountLockout.IsLocked RLock optimization**                                          | LOW    | 15 min | Performance     |
| 14 | **SessionStore.Find expiration check** (currently returns expired sessions)             | MEDIUM | 30 min | Correctness     |
| 15 | **Test helper consolidation** (extract shared app construction, dedup assertStatusCode) | MEDIUM | 1 hr   | Test quality    |
| 16 | **coverage_test.go rename** (to behavior-based name)                                    | LOW    | 5 min  | Test quality    |
| 17 | **User.MarshalJSON explicit allowlist** (currently hides by exclusion)                  | LOW    | 20 min | Security        |
| 18 | **handleMe public DTO** (don't return full User JSON)                                   | MEDIUM | 30 min | Security        |
| 19 | **HealthHandler deep health check** (verify dispatcher connectivity, not just non-nil)  | LOW    | 30 min | Observability   |
| 20 | **CSRFConfig getters → `cmp.Or`** (simplify trio of identical getters)                  | LOW    | 10 min | Code quality    |
| 21 | **SecurityHeadersConfig getters → `cmp.Or`**                                            | LOW    | 10 min | Code quality    |
| 22 | **Response.Apply() double-sanitize fix** (Redirect already sanitizes)                   | LOW    | 10 min | Performance     |
| 23 | **NewService config validation** (reject negative BcryptCost, zero SessionTTL)          | MEDIUM | 20 min | Robustness      |
| 24 | **DefaultLogFormatter use contextFields()** (like JSONLogFormatter does)                | LOW    | 10 min | Consistency     |
| 25 | **datastar-demo LSP integration** (add to go.work or document stale LSP)                | LOW    | 10 min | DX              |

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
