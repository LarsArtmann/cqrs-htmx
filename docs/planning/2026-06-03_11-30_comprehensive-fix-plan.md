# Comprehensive Execution Plan — cqrs-htmx

**Date:** 2026-06-03
**Context:** Post replace-removal self-review. All builds/tests pass. 96.4% root, 90.8% usermgmt coverage.
**Goal:** Fix real bugs, eliminate false-positive tests, close coverage gaps, reduce split brains.

---

## Pareto Analysis

### 1% → 51% Impact (The Vital Few)

| # | Task                                                      | Impact | Why                                                             |
| - | --------------------------------------------------------- | ------ | --------------------------------------------------------------- |
| 1 | Fix false-positive rollback test + empty logout test      | 51%    | False confidence is worse than no tests. These tests lie to us. |
| 2 | Fix UpdateRoles ordering (Casbin before save)             | 15%    | Real data inconsistency bug. Silent authz drift.                |
| 3 | Add missing tests (nil-enforcer, query nil, login errors) | 10%    | Coverage gaps for critical security paths.                      |

### 4% → 64% Impact

| # | Task                               | Impact | Why                                                    |
| - | ---------------------------------- | ------ | ------------------------------------------------------ |
| 4 | Fix rate limiter heap growth       | 5%     | Memory DoS vector. Documented but unaddressed.         |
| 5 | Fix Response.Status() fluent chain | 3%     | API contract violation. Users expect chaining to work. |
| 6 | Remove PtrBool, use new(bool)      | 1%     | Code hygiene. gopls already flags it.                  |
| 7 | Clean stale coverage files         | 1%     | Repo hygiene.                                          |

### 20% → 80% Impact

| #  | Task                                          | Impact | Why                                        |
| -- | --------------------------------------------- | ------ | ------------------------------------------ |
| 8  | Collapse 6 error handlers to 1 with options   | 3%     | Reduces API surface, eliminates confusion. |
| 9  | Fix CSRF proxy bypass                         | 2%     | Security hardening.                        |
| 10 | Deduplicate RequestLogging/RequestLoggingSlog | 2%     | Maintainability.                           |
| 11 | Adopt v2 typed dispatch                       | 2%     | Type safety for consumers.                 |

---

## Medium-Granularity Plan (27 tasks, 30-100 min each)

Sorted by **impact/effort** ratio (highest first):

| #  | Task                                                                                                                                          | Module           | Effort  | Impact   | Type     |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------- | ---------------- | ------- | -------- | -------- |
| 1  | **Fix false-positive TestService_Register_RollbackOnGroupPolicyFailure** — use mock authz that fails AddGroupPolicy, verify user not in store | usermgmt         | 30 min  | Critical | Bugfix   |
| 2  | **Fix empty TestHandlers_Logout_ServiceError** — inject store failure, verify error response                                                  | usermgmt         | 20 min  | Critical | Bugfix   |
| 3  | **Fix UpdateRoles ordering** — Move `authz.Apply()` before `users.Save()`, add rollback on Apply failure                                      | usermgmt         | 30 min  | High     | Bugfix   |
| 4  | **Add nil-enforcer bypass test** — Verify `Authorize` returns 403 (not panic) when enforcer is nil                                            | root             | 30 min  | High     | Test     |
| 5  | **Add query nil panic test** — Verify `Query` handler doesn't panic when query dispatcher is nil                                              | root             | 20 min  | High     | Test     |
| 6  | **Add Login error classification test** — Verify store errors return transient family, not ErrInvalidCredentials                              | usermgmt         | 30 min  | High     | Test     |
| 7  | **Remove PtrBool helper, use new(bool) everywhere**                                                                                           | usermgmt         | 20 min  | Low      | Cleanup  |
| 8  | **Clean stale coverage files** — Delete usermgmt/cov.out, usermgmt/coverage.out, reports/coverage.out                                         | repo             | 10 min  | Low      | Cleanup  |
| 9  | **Update ROADMAP.md** — Update date, mark done items, refresh status                                                                          | docs             | 20 min  | Low      | Docs     |
| 10 | **Fix Response.Status()** — Defer WriteHeader to Apply(), add test for Status+Redirect chain                                                  | root             | 45 min  | Medium   | Bugfix   |
| 11 | **Fix rate limiter unbounded heap growth** — Add heapIndex map, use heap.Fix for in-place updates                                             | root             | 60 min  | Medium   | Bugfix   |
| 12 | **Collapse 6 error handlers to 1** — Create ErrorHandlerOptions, migrate all variants                                                         | root             | 60 min  | Medium   | Refactor |
| 13 | **Fix CSRF proxy bypass** — Add TrustedProxies []string to CSRFConfig, IP-based trust check                                                   | root             | 60 min  | Medium   | Security |
| 14 | **Deduplicate RequestLogging/RequestLoggingSlog** — Extract shared formatter logic                                                            | root             | 45 min  | Low      | Refactor |
| 15 | **Add real integration HTTP test** — Wire usermgmt.AuthHandler into cqrshtmx.App, test register→login→command flow                            | integration_test | 90 min  | High     | Test     |
| 16 | **Adopt v2 typed dispatch** — Add CommandTyped/QueryTyped HandlerOptions                                                                      | root             | 90 min  | High     | Feature  |
| 17 | **Add PaginatedResult[T] support** — Query handler option for typed pagination                                                                | root             | 45 min  | Medium   | Feature  |
| 18 | **Fix decodeFormBody to use PostForm** — Prevent query-string injection                                                                       | root             | 20 min  | Low      | Bugfix   |
| 19 | **Fix \_ = r.Body.Close() in decoder** — Properly handle close error                                                                          | root             | 15 min  | Low      | Bugfix   |
| 20 | **Remove ClientIP re-export** — Dead weight from httputil                                                                                     | root             | 15 min  | Low      | Cleanup  |
| 21 | **Add UpdateRoles rollback test** — Verify state consistency on Casbin failure                                                                | usermgmt         | 45 min  | High     | Test     |
| 22 | **Fix TriggerWithDetail non-determinism** — Use sorted keys for consistent HX-Trigger headers                                                 | root             | 30 min  | Low      | Bugfix   |
| 23 | **Add middleware chaining integration test** — SessionMiddleware + ContextEnrichment + AuthorizeMiddleware                                    | integration_test | 60 min  | Medium   | Test     |
| 24 | **Add OpenTelemetry via upstream middleware** — Use middleware.NewTracing via MessageAdapter                                                  | root             | 90 min  | High     | Feature  |
| 25 | **Expose reactive EventBus helper** — Wrapper for event.EventBus + HTMX SSE integration                                                       | root             | 120 min | High     | Feature  |
| 26 | **Implement SQL UserStore** — PostgreSQL adapter for UserStore interface                                                                      | usermgmt         | 240 min | High     | Feature  |
| 27 | **Fix datastar-demo to use cqrs-htmx** — Or remove it if it's not a real demo                                                                 | examples         | 120 min | Medium   | Cleanup  |

---

## Execution Order

**Phase 1: Trust (Fix the lies)**

1. Fix false-positive rollback test
2. Fix empty logout test
3. Add missing security tests (nil-enforcer, query nil, login errors)
4. Fix UpdateRoles ordering
5. Add UpdateRoles rollback test

**Phase 2: Safety (Fix the bugs)** 6. Fix Response.Status() fluent chain 7. Fix rate limiter heap growth 8. Fix CSRF proxy bypass 9. Fix decodeFormBody PostForm 10. Fix r.Body.Close() error handling

**Phase 3: Clarity (Reduce complexity)** 11. Collapse 6 error handlers to 1 12. Remove PtrBool 13. Deduplicate RequestLogging 14. Remove ClientIP re-export 15. Clean stale files

**Phase 4: Power (Add value)** 16. Adopt v2 typed dispatch 17. Add PaginatedResult[T] support 18. Add real integration HTTP tests 19. Add middleware chaining tests 20. Add OpenTelemetry

**Phase 5: Future (Big items)** 21. Reactive event streams 22. SQL store backend 23. Fix datastar-demo

---

## D2 Execution Graph

```d2
direction: down

# Phase 1: Trust
p1: Phase 1: Trust {
  t1: Fix false rollback test
  t2: Fix empty logout test
  t3: Add missing security tests
  t4: Fix UpdateRoles ordering
  t5: Add UpdateRoles rollback test
}

# Phase 2: Safety
p2: Phase 2: Safety {
  t6: Fix Response.Status()
  t7: Fix rate limiter heap
  t8: Fix CSRF proxy bypass
  t9: Fix decodeFormBody
  t10: Fix Body.Close()
}

# Phase 3: Clarity
p3: Phase 3: Clarity {
  t11: Collapse error handlers
  t12: Remove PtrBool
  t13: Dedup RequestLogging
  t14: Remove ClientIP
  t15: Clean stale files
}

# Phase 4: Power
p4: Phase 4: Power {
  t16: Typed dispatch
  t17: PaginatedResult
  t18: Integration HTTP tests
  t19: Middleware chaining tests
  t20: OpenTelemetry
}

# Phase 5: Future
p5: Phase 5: Future {
  t21: Reactive streams
  t22: SQL store
  t23: Fix datastar-demo
}

# Dependencies
p1 -> p2: After trust
p2 -> p3: After safety
p3 -> p4: After clarity
p4 -> p5: After power
```

---

## Type Model Improvements

### Current State

- `id.UserID`, `id.CorrelationID`, `id.RequestID` — ULID-backed branded types (good)
- `usermgmt.UserID` — string-backed branded type (different from root, needs `.Get()` bridge)
- `any` returned from query dispatch — type safety gap
- `CommandDecoder`, `QueryDecoder` exported but unused by consumers

### Proposed Improvements

1. **Typed dispatch** — `command.RegisterTyped[T]` eliminates `any` at handler registration
2. **PaginatedResult[T]** — `query.PaginatedResult[T]` for typed query responses
3. **Unify UserID** — When upstream exposes marker types, use `go-composable-business-types`
4. **Remove dead exports** — `CommandDecoder`, `QueryDecoder` are implementation details

### Library Leverage

- **go-cqrs-lite middleware/** — `NewTracing[M]`, `NewRecovery[M]`, `NewMetrics[M]` — use via `MessageAdapter`
- **samber/ro** — Already a transitive dep via event/v2. Use for reactive streams.
- **go-error-family** — Already used. Could extend classification.

---

## Notes

- **DO NOT fix datastar-demo lint issues** — These are 39 pre-existing issues in an example module. Not part of this library's core.
- **DO NOT restructure root package** — Intentionally flat per AGENTS.md. Fight tooling, not design.
- **Commit after each task** — Per skill instructions.
- **Push when Phase complete** — Not after every commit.
