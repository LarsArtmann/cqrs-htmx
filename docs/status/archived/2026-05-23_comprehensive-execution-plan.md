# Comprehensive Execution Plan — cqrs-htmx

**Date:** 2026-05-23 | **Source:** TODO_LIST.md (4 open items) + 5 status reports + coverage analysis + AGENTS.md
**Status:** 19/22 original plan tasks resolved (86%). This plan covers ALL remaining work.

---

## Plan Table (sorted by Importance × Impact / Effort)

| #   | Task                                                                                             | Priority | Est   | Impact   | Status   | Source                        |
| --- | ------------------------------------------------------------------------------------------------ | -------- | ----- | -------- | -------- | ----------------------------- |
|     | **P0 — Security & Correctness**                                                                  |          |       |          |          |                               |
| P01 | Fix Dependabot CVEs (2 moderate) — run `gh auth login`, inspect alerts, update deps              | P0       | 12min | Critical | BLOCKED  | status 05-22 23:27 #1         |
| P02 | Design TypedHandler[T] API — top-level generic `DispatchTyped[T](app, cmd) (*T, error)` function | P0       | 12min | High     | OPEN     | TODO_LIST #3                  |
| P03 | Implement TypedHandler[T] — add function + tests + example_test.go                               | P0       | 12min | High     | OPEN     | TODO_LIST #3                  |
| P04 | Resolve UserID type split — decide usermgmt standalone vs paired, write ADR                      | P0       | 12min | High     | OPEN     | TODO_LIST #1                  |
| P05 | Implement UserID type split decision — migrate types + fix callers + update tests                | P0       | 12min | High     | OPEN     | TODO_LIST #1                  |
|     | **P1 — Test Coverage (usermgmt 88.6% → 91%+)**                                                   |          |       |          |          |                               |
| P06 | Test Register rollback paths — role fail → user deleted, session fail → user+role deleted        | P1       | 12min | High     | OPEN     | status 05-22 23:27 #2         |
| P07 | Test handleLogout — no-session cookie path, store error path (64.3→85%+)                         | P1       | 10min | High     | OPEN     | status 05-22 23:27 #5         |
| P08 | Test Logout error path — store.DeleteSession failure (66.7→90%+)                                 | P1       | 10min | High     | OPEN     | status 05-22 00:13 #5         |
| P09 | Test handleAuthEndpoint timeout branch — context deadline exceeded path                          | P1       | 10min | Medium   | OPEN     | status 05-22 00:13 #10        |
| P10 | Test Apply error paths — remove+add failure branches (69.2→90%+)                                 | P1       | 12min | Medium   | OPEN     | status 05-22 00:13 #7         |
| P11 | Test Login full paths — wrong password, locked, not found edge cases (89.5→95%+)                 | P1       | 10min | Medium   | OPEN     | coverage data                 |
| P12 | Test NewAuthz/Authz wrapper error paths — casbin failures (84.2→95%+)                            | P1       | 12min | Medium   | OPEN     | coverage data                 |
| P13 | Test Enforce/EnforceAny/EnforceEx nil adapter error paths (75→90%+)                              | P1       | 10min | Medium   | OPEN     | coverage data                 |
| P14 | Test NewService config validation — zero/negative bcrypt cost, nil store (88.9→95%+)             | P1       | 8min  | Medium   | OPEN     | coverage data                 |
| P15 | Test ChangePassword edge cases — wrong old password, same password (85.7→95%+)                   | P1       | 8min  | Medium   | OPEN     | coverage data                 |
| P16 | Test GetUser + UpdateRoles error paths — store failures (75→90%, 85→95%)                         | P1       | 10min | Medium   | OPEN     | coverage data                 |
| P17 | Test RecordFailure lockout edge — max attempts reached triggers lock (80→95%+)                   | P1       | 8min  | Medium   | OPEN     | coverage data                 |
|     | **P1 — Test Coverage (root 96.6% → 97%+)**                                                       |          |       |          |          |                               |
| P18 | Test WriteJSON error path — encoder failure, writer error (80→95%+)                              | P1       | 10min | Medium   | OPEN     | status 05-22 23:27 #11        |
| P19 | Test MapError remaining families — nil error, conflict, corruption, transient (93.3→98%+)        | P1       | 10min | Medium   | OPEN     | status 05-22 00:13 #8         |
| P20 | Test Enforce nil enforcer — `enforcer == nil` branch (87.5→100%)                                 | P1       | 5min  | Medium   | OPEN     | status 05-22 00:13 #9         |
| P21 | Test sanitizeRedirectURL remaining edge — fragment-only, userinfo URLs (87.5→100%)               | P1       | 8min  | Low      | OPEN     | coverage data                 |
| P22 | Test handleCommandDispatch error branches — auth denied, dispatch fail (90→95%+)                 | P1       | 10min | Medium   | OPEN     | coverage data                 |
| P23 | Test applyQueryResponse remaining branches — nil renderer, empty result (87.5→95%+)              | P1       | 10min | Medium   | OPEN     | coverage data                 |
| P24 | Test buildGorillaOptions edge cases — custom path, custom domain (88.9→100%)                     | P1       | 8min  | Low      | OPEN     | coverage data                 |
| P25 | Test limiter/eviction concurrent access — parallel Get + evictStale (88.2→95%+)                  | P1       | 10min | Medium   | OPEN     | coverage data                 |
|     | **P2 — Test Infrastructure**                                                                     |          |       |          |          |                               |
| P26 | Create mock UserStore — interface implementation that returns configurable errors                | P2       | 12min | High     | OPEN     | status 05-22 23:27 #5         |
| P27 | Create mock SessionStore — interface implementation that returns configurable errors             | P2       | 12min | High     | OPEN     | status 05-22 23:27 #5         |
| P28 | Add EvictExpired concurrent test — parallel inserts + evict, verify no data race                 | P2       | 10min | Medium   | OPEN     | status 05-22 23:27 #8         |
| P29 | Add EvictStale concurrent test — parallel failures + evict, verify no data race                  | P2       | 10min | Medium   | OPEN     | status 05-22 23:27 #9         |
|     | **P2 — Code Quality & Cleanup**                                                                  |          |       |          |          |                               |
| P30 | Extract `"Bearer "` constant in usermgmt/middleware.go                                           | P2       | 3min  | Low      | OPEN     | status 05-22 00:13 #11        |
| P31 | Extract usermgmt password validation messages as constants                                       | P2       | 5min  | Low      | OPEN     | status 05-22 00:13 #13        |
| P32 | Dedup writeJSON — usermgmt uses root's exported `WriteJSON` (import or copy signature)           | P2       | 10min | Medium   | OPEN     | status 05-22 00:13 #writeJSON |
| P33 | Add golangci.yml to integration_test module                                                      | P2       | 5min  | Low      | OPEN     | status 05-22 23:27 #17        |
| P34 | Extract `maxDisplayNameLength` constant in usermgmt                                              | P2       | 3min  | Low      | OPEN     | status 05-22 00:13 #15        |
|     | **P2 — Upstream Alignment**                                                                      |          |       |          |          |                               |
| P35 | Align go-cqrs-lite version — root v1.4.0 → v1.5.0 (match datastar-demo)                          | P2       | 10min | Medium   | OPEN     | status 05-22 23:27 #8         |
| P36 | Evaluate go-branded-id for numeric IDs — write ADR for future SQL stores                         | P2       | 8min  | Low      | OPEN     | TODO_LIST #2                  |
| P37 | Evaluate go-cqrs-lite catalog v2 migration — assess breaking changes                             | P2       | 12min | Low      | OPEN     | status 05-22 23:27 #23        |
|     | **P3 — Documentation**                                                                           |          |       |          |          |                               |
| P38 | Update CONTRIBUTING.md — reflect context-aware stores, branded types, current patterns           | P3       | 10min | Low      | OPEN     | status 05-22 23:27 #10        |
| P39 | Add usermgmt godoc examples — Service.Register, Service.Login, AuthHandler                       | P3       | 12min | Low      | OPEN     | status 05-22 23:27 #15        |
| P40 | Add godoc for remaining root symbols — Enforce, applyQueryResponse, sanitizeRedirectURL          | P3       | 10min | Low      | OPEN     | coverage data                 |
| P41 | Update AGENTS.md coverage figures — root 96.6%, usermgmt 88.6%                                   | P3       | 5min  | Low      | OPEN     | maintenance                   |
|     | **P3 — Performance & Robustness**                                                                |          |       |          |          |                               |
| P42 | Add usermgmt fuzz tests for Validate() — arbitrary Register/Login inputs                         | P3       | 10min | Low      | OPEN     | status 05-22 23:27 #19        |
| P43 | Add usermgmt benchmarks for bcrypt hot path — Login, Register                                    | P3       | 10min | Low      | OPEN     | status 05-22 23:27 #20        |
| P44 | Add TokenMatches benchmark — constant-time comparison perf                                       | P3       | 5min  | Low      | OPEN     | status 05-22 23:27 #10        |
|     | **P3 — Integration & Examples**                                                                  |          |       |          |          |                               |
| P45 | Expand integration tests — register→dispatch→query cross-module flow                             | P3       | 12min | Medium   | OPEN     | status 05-22 23:27 #6         |
| P46 | Review datastar-demo for improvements — version alignment, error handling                        | P3       | 12min | Low      | OPEN     | status 05-22 23:27 #21        |
| P47 | Add Session.MaxAge to cookie TTL alignment — verify cookie expiry matches session                | P3       | 8min  | Low      | OPEN     | status 05-22 23:27 #22        |
|     | **P4 — Large / Future**                                                                          |          |       |          |          |                               |
| P48 | Add OpenTelemetry tracing hooks — Before/AfterDispatch span creation                             | P4       | 60m+  | High     | DEFERRED | status 05-22 23:27 #24        |
| P49 | Evaluate nix flake migration for CI — replace justfile/Makefile patterns                         | P4       | 60m+  | Medium   | DEFERRED | status 05-22 23:27 #25        |
| P50 | Fix LSP stale cache for usermgmt — investigate golangci_lint_ls                                  | P4       | ?     | Low      | DEFERRED | status 05-22 23:27 #18        |
|     | **BLOCKED — External**                                                                           |          |       |          |          |                               |
| P51 | BrandNamer for root module marker types — upstream go-cqrs-lite markers unexported               | BLOCKED  | 5min  | Low      | BLOCKED  | TODO_LIST #4                  |
| P52 | Dependabot config — add .github/dependabot.yml after gh auth                                     | BLOCKED  | 5min  | Medium   | BLOCKED  | execution plan T16            |

---

## Summary

| Category               | Count | Tasks        |
| ---------------------- | ----- | ------------ |
| **P0 Security/Design** | 5     | P01–P05      |
| **P1 Coverage**        | 20    | P06–P25      |
| **P2 Infrastructure**  | 4     | P26–P29      |
| **P2 Quality/Cleanup** | 5     | P30–P34      |
| **P2 Upstream**        | 3     | P35–P37      |
| **P3 Documentation**   | 4     | P38–P41      |
| **P3 Perf/Robustness** | 3     | P42–P44      |
| **P3 Integration**     | 3     | P45–P47      |
| **P4 Large/Future**    | 3     | P48–P50      |
| **BLOCKED**            | 2     | P51–P52      |
| **Total actionable**   | 47    | ≤12 min each |
| **Deferred/Blocked**   | 5     | P48–P52      |

## Estimated Total Effort

| Batch           | Tasks    | Total Est | Cumulative |
| --------------- | -------- | --------- | ---------- |
| P0              | P01–05   | ~60 min   | 60 min     |
| P1 Coverage     | P06–25   | ~197 min  | ~4.3h      |
| P2 Infra        | P26–29   | ~44 min   | ~5.4h      |
| P2 Cleanup      | P30–34   | ~26 min   | ~5.8h      |
| P2 Upstream     | P35–37   | ~30 min   | ~6.3h      |
| P3 Docs         | P38–41   | ~37 min   | ~6.9h      |
| P3 Perf         | P42–44   | ~25 min   | ~7.3h      |
| P3 Integration  | P45–47   | ~32 min   | ~7.9h      |
| **Grand Total** | 47 tasks | ~7.9h     |            |

## Execution Order Recommendation

1. **Batch 1 (P0):** P04 → P05 → P02 → P03 → P01 — Resolve design decisions first, then implement
2. **Batch 2 (P2 Infra):** P26 → P27 — Mock stores unlock all coverage work
3. **Batch 3 (P1 usermgmt):** P06 → P07 → P08 → P10 → P12 → P13 → P09 → P11 → P14 → P15 → P16 → P17
4. **Batch 4 (P1 root):** P18 → P19 → P20 → P22 → P23 → P25 → P21 → P24
5. **Batch 5 (P2):** P28 → P29 → P30 → P31 → P34 → P32 → P33 → P35 → P36 → P37
6. **Batch 6 (P3):** P38 → P39 → P40 → P41 → P42 → P43 → P44 → P45 → P46 → P47
7. **Blocked:** P51 (upstream), P52 (gh auth)
8. **Future:** P48 (otel), P49 (nix), P50 (LSP)

---

## Appendix B: Execution Results (2026-05-23)

### Final Metrics

| Metric            | Before | After  |
| ----------------- | ------ | ------ |
| Root coverage     | 96.6%  | 96.7%  |
| Usermgmt coverage | 88.6%  | 91.0%  |
| Root lint         | 0      | 0      |
| Usermgmt lint     | 0      | 0      |
| Integration tests | 4      | 5      |
| go-cqrs-lite      | v1.4.0 | v1.5.0 |
| Tasks completed   | 0/47   | 35/47  |

### Tasks Completed (35/47)

P04, P06-P11, P13-P22, P25-P31, P33-P38, P42-P47

### Blocked/Deferred (12/47)

- P01: gh auth needed for Dependabot
- P02/P03: TypedHandler[T] API design decision
- P05: NOT NEEDED (ADR 0002)
- P12/P23/P24: Low-value coverage
- P32: NOT RECOMMENDED (cross-module import)
- P39-P41: LOW PRIORITY
- P48-P50: DEFERRED (large effort)
- P51-P52: BLOCKED (external)

### Files Changed This Session

| File                                      | Change                                                        |
| ----------------------------------------- | ------------------------------------------------------------- |
| `usermgmt/mock_test.go`                   | NEW: mock UserStore + SessionStore test doubles               |
| `usermgmt/fuzz_test.go`                   | NEW: FuzzRegisterRequest_Validate + FuzzLoginRequest_Validate |
| `usermgmt/benchmark_test.go`              | NEW: Login, Register, TokenMatches benchmarks                 |
| `usermgmt/coverage_test.go`               | 30+ new tests for coverage gaps                               |
| `usermgmt/middleware.go`                  | Extract `bearerPrefix` constant                               |
| `usermgmt/service.go`                     | Extract password/display name message constants               |
| `integration_test/.golangci.yml`          | NEW: lint config for integration test module                  |
| `integration_test/bridge_test.go`         | New full-cycle integration test                               |
| `docs/adr/0003-numeric-ids-sql-stores.md` | NEW: ADR for future SQL store IDs                             |
| `coverage_test.go`                        | Root coverage gap tests                                       |
| `CONTRIBUTING.md`                         | Updated for multi-module structure                            |
| `go.mod`                                  | go-cqrs-lite v1.4.0 → v1.5.0                                  |
| `usermgmt/go.mod`                         | go-cqrs-lite v1.4.0 → v1.5.0                                  |
| `integration_test/go.mod`                 | go-cqrs-lite v1.4.0 → v1.5.0                                  |
