# Comprehensive Status Report — cqrs-htmx

**Date:** 2026-05-19_23-19
**Session:** 9-Skill Comprehensive Review + Execution Plan

---

## Work Status

### FULLY DONE

| #   | Item                       | Details                                                                                                                                                                  |
| --- | -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | BDD Testing Analysis       | 289 Ginkgo specs, 94.7% coverage, race-safe. No significant gaps.                                                                                                        |
| 2   | Features Audit             | Updated `FEATURES.md` — added Security Headers (#27), Request ID (#28), corrected metrics. 29 features total, all FULLY_FUNCTIONAL.                                      |
| 3   | Code Quality Scan          | `docs/status/2026-05-19_22-39_code-quality-scan.md` — 0 build/lint/vet issues. 11 duplication patterns identified (4 high, 4 medium, 3 low).                             |
| 4   | Full Code Review           | `docs/status/2026-05-19_22-39_full-code-review.md` — Grade A-. 0 critical findings. 3 high (context mismatch, rate limiter bounds, CSRF allocation), 6 medium, 4 low.    |
| 5   | Architecture Deepening     | `docs/status/2026-05-19_22-39_improve-codebase-architecture.md` — 7 deepening opportunities identified using module/depth/seam vocabulary.                               |
| 6   | Go Modularize Assessment   | `docs/status/2026-05-19_22-39_go-modularize-assessment.md` — **Recommend NOT splitting further**. Flat package is right for ~5K LOC library. usermgmt split was correct. |
| 7   | Architecture Review        | `docs/status/2026-05-19_22-39_architecture-review.md` — Scalability 8/10, Modularity 7/10, Composability 9/10. Overall 8/10.                                             |
| 8   | Architecture Visualization | 2 D2 diagrams + SVGs at `docs/architecture-understanding/2026-05-19_22-39_*` — current and improved architecture.                                                        |
| 9   | TODO List Builder          | Updated `TODO_LIST.md` — P0-P4 all DONE. Added P5 with 13 open items.                                                                                                    |
| 10  | P5 Execution Plan          | `docs/planning/2026-05-19_23-03_p5-comprehensive-deduplication-plan.md` — 25 tasks, ≤12min each, Pareto-sorted, with D2 execution graph.                                 |
| 11  | usermgmt Branded UserID    | Fixed broken build — `usermgmt.UserID` branded type migration completed. All tests pass.                                                                                 |

### PARTIALLY DONE

None.

### NOT STARTED

| #   | P5 Task                                             | Effort |
| --- | --------------------------------------------------- | ------ |
| 1   | Fix context mismatch in `applyQueryResponse`        | 5min   |
| 2   | Test for context timeout during query render error  | 10min  |
| 3   | Add `MaxKeys` to `RateLimiterConfig` + eviction     | 10min  |
| 4   | Test for rate limiter max-keys eviction             | 10min  |
| 5   | Extract generic `htmxBoolField` + `htmxStringField` | 8min   |
| 6   | Rewrite 8 HTMX accessors to use generics            | 8min   |
| 7   | Verify HTMX accessor refactor                       | 3min   |
| 8   | Extract `decodeAndSet[T,R]` generic decoder         | 10min  |
| 9   | Rewrite 4 Decode functions to use generic           | 8min   |
| 10  | Verify decoder refactor                             | 3min   |
| 11  | Extract `validateDispatch[T]` generic               | 10min  |
| 12  | Rewrite ValidateCommand/Query to use generic        | 5min   |
| 13  | Verify validation refactor                          | 3min   |
| 14  | Unify `notifyOption` + `triggerNotification`        | 10min  |
| 15  | Extract `contextFields(r)` helper from logging      | 8min   |
| 16  | Rewrite 3 logging formatters to use helper          | 10min  |
| 17  | Verify logging refactor                             | 3min   |
| 18  | Extract `handleErrorCore` from error handlers       | 10min  |
| 19  | Rewrite both error handlers to delegate             | 8min   |
| 20  | Verify error handler refactor                       | 3min   |
| 21  | Extract `parseID[T]` generic                        | 8min   |
| 22  | Rewrite 3 Parse functions to use generic            | 5min   |
| 23  | Split `csrf.go` → `csrf.go` + `csrf_helpers.go`     | 10min  |
| 24  | Fix nil context in usermgmt tests                   | 5min   |
| 25  | Full test suite + lint verification                 | 5min   |

### TOTALLY FUCKED UP (Fixed This Session)

- **usermgmt broken build** — Prior partial `UserID` branded type migration left 3 compilation errors. Fixed: `middleware.go` (`.String()`), `service.go` (`.String()` at casbin boundary, keep `UserID` at `RolesForUser`).

---

## Top 25 Things to Get Done Next

Sorted by importance/impact/effort:

| #   | Task                                                                             | Impact                | Effort | Priority |
| --- | -------------------------------------------------------------------------------- | --------------------- | ------ | -------- |
| 1   | Fix context mismatch in `applyQueryResponse` (handler.go:124)                    | Critical correctness  | 5min   | P0       |
| 2   | Add test for context timeout during render error                                 | Regression prevention | 10min  | P0       |
| 3   | Add `MaxKeys` cap to rate limiter                                                | Production safety     | 10min  | P0       |
| 4   | Add test for rate limiter max-keys eviction                                      | Regression prevention | 10min  | P0       |
| 5   | Generic HTMX accessor (8→2 functions)                                            | Maintainability       | 16min  | P1       |
| 6   | Generic decoder (4→1 + 4 wrappers)                                               | Maintainability       | 18min  | P1       |
| 7   | Generic validation (2→1 + 2 wrappers)                                            | Maintainability       | 15min  | P1       |
| 8   | Unified notification implementation                                              | Maintainability       | 10min  | P1       |
| 9   | Shared logging context extraction                                                | Maintainability       | 18min  | P2       |
| 10  | Shared error handler core                                                        | Maintainability       | 18min  | P2       |
| 11  | Generic `parseID[T]` helper                                                      | Maintainability       | 13min  | P2       |
| 12  | Split csrf.go (445→~280+~165 lines)                                              | File readability      | 10min  | P3       |
| 13  | Fix nil context in usermgmt tests                                                | Lint clean            | 5min   | P3       |
| 14  | Consider making `GroupPolicy.User` typed as `UserID`                             | Type safety           | 15min  | Future   |
| 15  | Consider generic `withID[T]`/`fromContext[T]` for context                        | Type safety           | 20min  | Future   |
| 16  | Add `Response.RedirectWithStatus(code int)`                                      | Composability         | 5min   | Future   |
| 17  | Consider `WithDecoder(d CommandDecoder)` for fully custom decoders               | Extensibility         | 10min  | Future   |
| 18  | Add max-keys eviction to rate limiter docs                                       | Documentation         | 3min   | Future   |
| 19  | Verify AGENTS.md reflects usermgmt UserID branded type                           | Documentation         | 5min   | Future   |
| 20  | Add godoc example for `CSRFProtect` per-handler usage                            | Documentation         | 10min  | Future   |
| 21  | Add godoc example for `SecurityHeadersMiddlewareWithConfig`                      | Documentation         | 10min  | Future   |
| 22  | Consider extracting `csrf_helpers.go` template helpers as package-level tests    | Test coverage         | 10min  | Future   |
| 23  | Consider `RateLimiterConfig.KeyExtractor` returning structured key with metadata | Observability         | 15min  | Future   |
| 24  | Add integration test covering full middleware chain (CSRF+HTMX+Context)          | Test coverage         | 15min  | Future   |
| 25  | Consider `Config.DisableCSRF` opt-out for API-only routes                        | Usability             | 10min  | Future   |

---

## Metrics

| Metric                 | Value                       |
| ---------------------- | --------------------------- |
| Test specs             | 289 (root) + ~40 (usermgmt) |
| Coverage               | 94.7%                       |
| Lint issues            | 0                           |
| Build errors           | 0                           |
| Prod files             | 15 (root) + 9 (usermgmt)    |
| Test files             | 20 (root) + 7 (usermgmt)    |
| Open TODOs             | 25 tasks (P5 plan)          |
| Total estimated effort | ~185 min                    |

## My #1 Blocking Question

**The branded `usermgmt.UserID` type is `brandid.ID[userBrand, string]` (arbitrary string), while the root module's `cqrshtmx.UserID` is `id.UserID` (ULID-backed).** These are completely different types. The bridge function `usermgmt.UserIDFromRequest` returns `string` because it can't return `cqrshtmx.UserID` (no cross-module import). Should `usermgmt.UserID` be changed to use `id.UserID` (ULID-backed) for consistency, or is the current design (different ID types at module boundaries with string conversion) intentional? This affects the `GroupPolicy.User` field and all casbin-boundary conversions.
