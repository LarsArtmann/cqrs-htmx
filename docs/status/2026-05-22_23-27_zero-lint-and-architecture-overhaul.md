# Status Report — cqrs-htmx

**Date:** 2026-05-22 23:27 CEST
**Branch:** master (clean)
**Commits today:** `8303075`, `b9af187`, `0ef8242`, `5e12279`

---

## Executive Summary

The project is in **excellent shape**. Two major sessions today brought the usermgmt submodule from 165 lint issues to **zero**, and delivered significant architectural improvements including context-aware store interfaces, compensating transactions, and cleanup methods. All 4 modules build and test green with zero lint warnings.

| Metric | Root | usermgmt | integration_test | datastar-demo |
|--------|------|----------|-----------------|---------------|
| Build | ✅ | ✅ | ✅ | ✅ |
| Tests | ✅ 13 tests | ✅ 122 tests | ✅ 4 tests | — (main pkg) |
| Coverage | 96.6% | 88.6% | — | — |
| Lint | 0 issues | 0 issues | — | — |
| Race | ✅ | ✅ | ✅ | — |

**LOC:** 3,018 root production · 5,276 root tests · 1,582 usermgmt production · 2,031 usermgmt tests

---

## a) FULLY DONE

### Session 1 — Bug Fixes & Security Hardening (`8303075`, `b9af187`)

1. **Validate() trim bug** — `RegisterRequest.Validate()` and `LoginRequest.Validate()` used value receivers; `strings.TrimSpace` modified a copy. Fixed to pointer receivers. 3 tests added.
2. **HandlerConfig.Timeout propagation** — `NewAuthHandler` never copied `cfg[0].Timeout`. Fixed. Test added.
3. **Max password length (128)** — `maxPasswordLength = 128` in both `RegisterRequest.Validate()` and `Service.ChangePassword()`. Prevents bcrypt CPU abuse. 2 tests.
4. **ErrUserIDExists sentinel** — Replaced untyped `errors.Newf(...)` with typed sentinel, mapped to HTTP 404.
5. **WriteJSON returns error** — Root `httputil.go` `WriteJSON` now returns error from `json.Encoder.Encode` (was silently swallowed).
6. **bdd_test.go writestring** — Fixed inefficient string concatenation.
7. **coverage_test.go SA1012** — Changed `nil` context to `var nilCtx context.Context`.
8. **Documentation** — Production warnings on in-memory stores/lockout, mutation warning on `Service.Authz()`, zero-value caveat on `HandlerConfig`.
9. **AGENTS.md** — Updated with 8 new gotchas (#48-#54).

### Session 2 — Zero Lint + Architecture (`0ef8242`, `5e12279`)

10. **usermgmt/.golangci.yml** — Full lint config with appropriate test exclusions (gosec G104/G124, goconst, paralleltest, unparam, noctx, exhaustruct, wrapcheck). Reduced 165 → 0 issues.
11. **All wrapcheck fixed (29 issues)** — Every casbin method call, every store call, every `json.Unmarshal`/`json.Marshal`/`rand.Read` now wraps errors with `fmt.Errorf("...: %w", err)`.
12. **All errcheck/exhaustruct/gosec/revive/perfsprint fixed** — Nolint directives with explanations for intentional patterns; `defaultCookieName` constant extracted; `_ = json.Encode` pattern for post-commit writes.
13. **Session.TokenMatches()** — Extracted constant-time token comparison from `Valid()`. Test added.
14. **InMemorySessionStore.EvictExpired()** — Periodic cleanup method for stale sessions. Returns eviction count. Test added.
15. **AccountLockout.EvictStale()** — Removes expired lockout entries from internal maps. Test added.
16. **contextKey → empty struct** — `userContextKeyType struct{}` replaces `contextKey string`. Standard Go sentinel pattern.
17. **Timeout before body read** — `handleAuthEndpoint` creates timeout context BEFORE `json.Decode`, ensuring the entire operation is bounded.
18. **context.Context through store interfaces** — All `UserStore` and `SessionStore` methods accept `context.Context` as first param. **Breaking change** for custom implementations. Enables future cancellation, tracing, timeout propagation.
19. **Register compensating transaction** — `Service.Register` rolls back user+role if role assignment fails, and rolls back user+role+session if session creation fails. Best-effort cleanup.
20. **AGENTS.md** — Updated with 10 new gotchas (#54-#63), new Key Decisions, coverage figures.

### Previously Done (Still Valid)

- 30 features fully functional (see FEATURES.md)
- All P0-P4 TODO items completed (see TODO_LIST.md)
- Branded UserID/CorrelationID types
- CSRF protection with gorilla/csrf
- Rate limiting with min-heap eviction
- Security headers middleware
- Request logging (text + JSON)
- Lifecycle hooks (Before/AfterDispatch)
- Zero-cost catalog API integration
- datastar-demo example app

---

## b) PARTIALLY DONE

| Item | Status | Details |
|------|--------|---------|
| usermgmt test coverage | 88.6% (was 91.3%) | Dropped due to new methods (EvictExpired, EvictStale, TokenMatches) and context-threaded compensating paths that are hard to trigger without mocking. Register compensating rollback paths at 66.7%. |
| Register compensating tests | Partial | The happy path is tested, but rollback paths (role assignment failure, session creation failure) are not exercised. Would require mocking store/authz interfaces. |

---

## c) NOT STARTED

| # | Item | Priority | Notes |
|---|------|----------|-------|
| 1 | Dependabot vulnerability fixes | High | 2 moderate vulnerabilities reported by GitHub. Not inspected yet. |
| 2 | Register compensating transaction tests | Medium | Rollback paths need mock store/authz to trigger failures |
| 3 | Usermgmt coverage recovery to 91%+ | Medium | EvictExpired, EvictStale, TokenMatches, and context paths need more tests |
| 4 | Root coverage push to 97%+ | Low | WriteJSON at 80%, Hijack at 60%, heap Push at 75% |
| 5 | Store interface mock/fake implementations | Medium | Would enable testing of error paths in Service methods |
| 6 | Integration test expansion | Low | Only 4 tests — could add more cross-module scenarios |
| 7 | datastar-demo tests | Low | Main package, no tests exist |
| 8 | go-cqrs-lite v1.5.0 upgrade | Low | Root uses v1.4.0, datastar-demo uses v1.5.0. Could unify. |
| 9 | API documentation (godoc) | Low | Most types documented but could be more thorough |
| 10 | CONTRIBUTING.md update | Low | Should reflect current state (context-aware stores, etc.) |

---

## d) TOTALLY FUCKED UP!

Nothing is fucked up. The project is in the best shape it has ever been:

- ✅ Zero lint in both modules
- ✅ All 4 modules build/test green
- ✅ Race detector clean
- ✅ No known bugs
- ✅ No compiler errors

**Minor concerns:**

- **Coverage regression in usermgmt (91.3% → 88.6%)** — Not a regression in functionality, just new untested paths from architectural changes. Acceptable but should be recovered.
- **LSP stale cache** — The golangci_lint_ls LSP shows ~35 stale warnings that don't match CLI output. This is an LSP caching issue, not a real problem. Workaround: ignore LSP diagnostics for usermgmt test files.

---

## e) WHAT WE SHOULD IMPROVE!

### High Priority

1. **Fix Dependabot vulnerabilities** — 2 moderate CVEs. Should inspect and update affected dependencies.
2. **Recover usermgmt coverage to 91%+** — The context threading and new methods need test coverage. Focus on:
   - `Register` rollback paths (66.7% → 90%+)
   - `Logout` error path (66.7%)
   - `handleLogout` (64.3%)
   - `handleRegister` (87.5%)
3. **Add mock/fake store implementations** — Create test doubles for `UserStore` and `SessionStore` to test Service error paths without relying on the in-memory implementation's behavior.

### Medium Priority

4. **Test EvictExpired/EvictStale in concurrent scenarios** — Both use locks but haven't been tested under concurrent load.
5. **Add integration tests for Register compensating transaction** — Verify that a failed role assignment actually deletes the user from the store.
6. **golangci.yml for root module** — Root module's `.golangci.yml` should be reviewed for consistency with usermgmt's config.
7. **go-cqrs-lite version alignment** — Root uses v1.4.0, datastar-demo uses v1.5.0. Evaluate upgrade path.

### Low Priority

8. **Benchmark tests for new usermgmt methods** — EvictExpired, EvictStale, TokenMatches should have benchmarks.
9. **Example tests for usermgmt** — Go doc examples for Service.Register, Service.Login, AuthHandler.RegisterRoutes.
10. **CONTRIBUTING.md refresh** — Reflect current patterns (context-aware stores, branded types, etc.).

---

## f) Top 25 Things We Should Get Done Next!

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 1 | Fix Dependabot CVEs (2 moderate) | Security | Small |
| 2 | Recover usermgmt coverage to 91%+ | Quality | Medium |
| 3 | Add Register rollback path tests | Correctness | Medium |
| 4 | Create mock/fake UserStore + SessionStore | Testability | Medium |
| 5 | Test Logout error path (66.7% coverage) | Quality | Small |
| 6 | Test handleLogout (64.3% coverage) | Quality | Small |
| 7 | Test handleRegister error path (87.5%) | Quality | Small |
| 8 | Add EvictExpired concurrent test | Correctness | Small |
| 9 | Add EvictStale concurrent test | Correctness | Small |
| 10 | Add TokenMatches benchmark | Performance | Small |
| 11 | Root coverage: WriteJSON 80% → 95%+ | Quality | Small |
| 12 | Root coverage: Hijack 60% → 90%+ | Quality | Small |
| 13 | Root coverage: heap Push 75% → 95%+ | Quality | Small |
| 14 | Align go-cqrs-lite version (v1.4→v1.5) | Consistency | Medium |
| 15 | Add usermgmt godoc examples | Docs | Medium |
| 16 | Update CONTRIBUTING.md | Docs | Small |
| 17 | Add golangci.yml to integration_test | Consistency | Small |
| 18 | Fix LSP stale cache for usermgmt | DX | Unknown |
| 19 | Add usermgmt fuzz tests for Validate() | Robustness | Medium |
| 20 | Add usermgmt benchmarks for bcrypt hot path | Performance | Small |
| 21 | Review datastar-demo for improvements | Quality | Medium |
| 22 | Add Session.MaxAge to cookie TTL alignment | Correctness | Small |
| 23 | Consider go-cqrs-lite catalog v2 migration | Future-proofing | Large |
| 24 | Add OpenTelemetry tracing hooks | Observability | Large |
| 25 | Evaluate nix flake migration for CI | DX | Large |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the usermgmt `UserStore` and `SessionStore` interfaces include context-aware error types (e.g., `context.Canceled`, `context.DeadlineExceeded`) in their error contracts?**

Right now the interfaces accept `context.Context` but the in-memory implementations ignore it entirely with `_`. This is correct for the current state, but it raises a design question:

- Should we document that implementations **MUST** respect context cancellation?
- Should `Service` methods check `ctx.Err()` before proceeding with store calls?
- Or should we keep the current "pass-through" design and let future SQL/Redis implementations handle context naturally?

This affects the public API contract and cannot be decided without knowing the intended production store backends. The current "pass context, ignore it in-memory" pattern is pragmatic but may mislead consumers about cancellation guarantees.

---

## Git State

```
HEAD: 5e12279 feat(usermgmt): context-aware stores, compensating transactions, cleanup methods
Branch: master (clean, pushed)
Recent:
  5e12279 feat(usermgmt): context-aware stores, compensating transactions, cleanup methods
  0ef8242 fix(usermgmt): zero lint warnings — add golangci.yml and fix all issues
  b9af187 style: apply gci/goimports formatting from pre-commit hooks
  8303075 fix: trim validation bug, password length limit, timeout propagation, sentinel errors, lint fixes
  700aed8 chore: add modularization status report, stop tracking demo binary
```

---

_Report generated by Crush on 2026-05-22 at 23:27 CEST._
