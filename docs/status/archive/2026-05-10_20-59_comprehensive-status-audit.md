# Status Report — cqrs-htmx Comprehensive Audit

**Date:** 2026-05-10 20:59 | **Session:** Full Comprehensive Status Update

---

## Executive Summary

Library is in excellent shape. 10 production files, 3,827 total lines, 148 tests passing with 95.5% coverage, 0 lint issues, clean build. Strong ID migration complete. Several documentation files are stale (TODO_LIST.md, FEATURES.md reference outdated metrics). Low-hanging coverage gaps in `handleQueryDispatch` (72.7%), `decodeFormValues` (72.7%), and `NewUserID` (0%). No security issues, no banned dependencies, no architecture problems.

---

## Metrics

| Metric      | Value    | Notes                                                                                                                               |
| ----------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Go version  | 1.26.2   |                                                                                                                                     |
| Test specs  | 148      | All passing, race-safe                                                                                                              |
| Coverage    | 95.5%    | 95.5% of statements                                                                                                                 |
| Lint issues | 0        | `golangci-lint run` clean                                                                                                           |
| Build       | ✅ Clean | `go build ./...`                                                                                                                    |
| Prod files  | 10       | `app.go`, `handler.go`, `options.go`, `response.go`, `htmx.go`, `middleware.go`, `errors.go`, `authz.go`, `context.go`, `notify.go` |
| Test files  | 9        | Including `bdd_test.go`, `suite_test.go`                                                                                            |
| Total lines | 3,827    |                                                                                                                                     |
| Sentinels   | 7        | All actively used                                                                                                                   |
| Banned deps | 0        |                                                                                                                                     |

### Coverage by Function (Uncovered Only)

| Function               | Coverage | File              |
| ---------------------- | -------- | ----------------- |
| `NewUserID`            | 0.0%     | `context.go:16`   |
| `decodeFormValues`     | 72.7%    | `options.go:251`  |
| `handleQueryDispatch`  | 72.7%    | `handler.go:63`   |
| `decodeFormBody`       | 80.0%    | `options.go:100`  |
| `Enforce`              | 87.5%    | `authz.go:41`     |
| `setTriggerWithDetail` | 88.2%    | `response.go:153` |
| `enrichUserID`         | 90.9%    | `app.go:118`      |
| `MapError`             | 93.3%    | `errors.go:50`    |
| Everything else        | 100.0%   | —                 |

---

## a) FULLY DONE ✅

### Core Library (10/10 files production-ready)

1. **App Builder** (`app.go`) — `New(Config)` validates, creates per-App error handler with login redirect closure. `Command()`, `Query()`, `Middleware()` all fully functional. 100% coverage on all exported methods.

2. **Command Dispatch** (`handler.go`) — Decodes → authorizes → dispatches → applies HTMX response. Pre-dispatch checks extracted cleanly. 100% on command path.

3. **Query Dispatch** (`handler.go`) — Same flow as commands with render step. 72.7% coverage (some branches uncovered).

4. **Handler Options** (`options.go`) — 4 decoder pairs (`DecodeJSON`/`DecodeJSONQuery`, `DecodeForm`/`DecodeFormQuery`), 3 render options (`Render`, `RenderTempl`, `RenderTemplResult`), 3 response options (`Redirect`, `Trigger`, `PushURL`). Generic decoder helpers. Authorization execution. All at 100% except form decoders.

5. **HTMX Response Builder** (`response.go`) — Fluent API with 18 methods. HTMX-aware redirect. Notification methods. Trigger merging logic. 100% on most functions.

6. **HTMX Context** (`htmx.go`) — `HTMXMiddleware` parses all headers once. `HTMXRequest` struct with 8 fields + `RenderPartial()`. 10 standalone accessors with middleware-or-header fallback. 100% coverage.

7. **Casbin Authorization** (`authz.go`) — `Enforcer` interface (duck-types `*casbin.Enforcer`). `Authorize`, `RequireAuth`, `Enforce`, `AuthorizeMiddleware`. Error context includes resource/action. 87.5% on `Enforce` (casbin error path).

8. **Error Classification** (`errors.go`) — 7 sentinels, `sync.Once` lazy registration. `MapError` translates CQRS families → HTTP status. `DefaultErrorHandlerWithRedirect` is HTMX-aware. 93.3% on `MapError`.

9. **Context Propagation** (`context.go`) — Strongly-typed `UserID = id.UserID` (ULID-backed branded type). `WithUserID`/`UserIDFromContext` typed. `ParseUserID`/`MustParseUserID`/`NewUserID` exported. `EventOptionsFromContext` builds event options. 100% coverage except `NewUserID`.

10. **Notifications** (`notify.go`) — `NotifySuccess/Error/Warning/Info` as both HandlerOptions and Response methods. `NotifyWithEvent` builder for custom event names. Immutable `defaultNotificationEvent` constant. Deprecated mutable `DefaultNotificationEvent` var. 100% coverage.

### Infrastructure

11. **Middleware Chain** (`middleware.go`) — `ContextEnrichmentMiddleware`, `HTMXMiddleware`, `Chain()` composer. All 100% coverage.

12. **Test Suite** — Ginkgo/Gomega BDD. 148 specs, all passing. Race detector clean. `bdd_test.go` uses consolidated `bdd*` test types.

13. **Lint Configuration** — `.golangci.yml` v2 format. 0 issues. Only `ifElseChain` needs explicit gocritic disable.

14. **Documentation** — `README.md` with comprehensive examples. `CHANGELOG.md` with full history. `AGENTS.md` with architecture, decisions, gotchas.

---

## b) PARTIALLY DONE 🔧

### 1. Test Coverage — 95.5% (target: 95%+)

8 functions below 100%. Most are minor branch gaps:

- `handleQueryDispatch` at 72.7% — multiple uncovered query handler branches
- `decodeFormValues` at 72.7% — error branches
- `NewUserID` at 0% — trivial wrapper, never tested directly

### 2. Documentation Freshness

| Doc            | Status        | Issue                                                                                                                          |
| -------------- | ------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `AGENTS.md`    | ✅ Current    | Updated 2026-05-07                                                                                                             |
| `CHANGELOG.md` | ✅ Current    | [Unreleased] section up to date                                                                                                |
| `README.md`    | ✅ Current    | Examples compile, use typed UserID                                                                                             |
| `TODO_LIST.md` | ⚠️ Stale      | Dead sentinel removal still marked TODO (done in `3ec00f8`), coverage listed as 95.7% (now 95.5%)                              |
| `FEATURES.md`  | ⚠️ Very stale | Metrics say 92.6% coverage, 137 specs — actually 95.5% and 148 specs. User Identity description doesn't mention strong typing. |

### 3. Export `HeaderTrue` — P1 TODO

`headerTrue` is unexported. Tests hardcode `"true"` in 42 places across 7 test files. Production code uses the constant. This is a test quality issue, not a correctness issue.

---

## c) NOT STARTED 📋

### From TODO_LIST.md P1–P4

| #   | Item                                                            | Priority | Impact | Effort |
| --- | --------------------------------------------------------------- | -------- | ------ | ------ |
| 1   | Dispatch lifecycle hooks (`OnBeforeDispatch`/`OnAfterDispatch`) | P3       | High   | Medium |
| 2   | Request validation middleware                                   | P3       | Medium | Medium |
| 3   | JSON error response option                                      | P3       | Medium | Low    |
| 4   | Correlation ID propagation                                      | P3       | High   | Low    |
| 5   | Timeout propagation                                             | P3       | Medium | Low    |
| 6   | Godoc examples                                                  | P4       | Medium | Low    |
| 7   | CONTRIBUTING.md                                                 | P4       | Low    | Low    |
| 8   | golangci-lint CI/CD                                             | P4       | Medium | Low    |
| 9   | Benchmark tests                                                 | P4       | Low    | Medium |
| 10  | Document `.golangci.yml` decisions                              | P4       | Low    | Low    |

---

## d) TOTALLY FUCKED UP 💥

### 1. FEATURES.md Metrics Are Wrong

Last updated 2026-05-07 but metrics table says:

- Coverage: `92.6%` → actually `95.5%`
- Test specs: `137` → actually `148`
- User Identity description doesn't mention strongly-typed UserID migration

### 2. TODO_LIST.md Dead Sentinel Item Still Marked TODO

`3ec00f8` removed `ErrNoUserID` and `ErrRendererMissing` on 2026-05-07, but TODO_LIST.md still shows:

```
- [ ] Remove dead sentinels — ErrNoUserID and ErrRendererMissing...
```

### 3. `NewUserID()` Has 0% Coverage

Exported function at `context.go:16` is a trivial wrapper around `id.NewUserID()` but has zero test coverage. Easy fix, but it's a gap.

### 4. 42 Test Hardcodes of `"true"`

Across 7 test files, the string `"true"` is hardcoded 42 times instead of using a shared constant or test helper. This is a maintenance risk — if HTMX ever changes the header value (unlikely but possible), every test would need manual updates.

---

## e) WHAT WE SHOULD IMPROVE 🎯

### Critical (Do Now)

1. **Fix TODO_LIST.md** — Mark dead sentinel removal as done, update coverage to 95.5%, update spec count to 148
2. **Fix FEATURES.md** — Update metrics table, update User Identity description for strong typing, update spec count
3. **Add `NewUserID()` test** — 0% → 100% in one test, brings total coverage up

### High Impact

4. **Cover `handleQueryDispatch` branches** — 72.7% → 90%+, largest coverage gap in production code
5. **Cover `decodeFormValues` error paths** — 72.7%, form parsing undertested
6. **Cover `Enforce` casbin error path** — 87.5%, one test for when enforcer returns error
7. **Export test helper for `"true"`** — Or export `HeaderTrue`, reduce 42 hardcodes

### Architecture

8. **Consider `UserIDExtractor` v2 API** — Currently returns `string`, library parses to `UserID`. Could accept `(UserID, error)` for fail-fast. Needs consumer research.
9. **Middleware silently drops invalid IDs** — Intentional but could surprise consumers. Consider logging or error callback.
10. **`ParseUserID` is a thin wrapper** — Just wraps `id.ParseUserID` with extra `fmt.Errorf`. Consider `//nolint:wrapcheck` instead.

### Design

11. **Add JSON error response option** — Default is plain text only. Many modern APIs expect JSON errors.
12. **Add correlation ID propagation** — `WithCorrelationID`/`CorrelationIDFromContext` for distributed tracing.
13. **Add timeout propagation** — Library doesn't set deadlines on context.

### Test Quality

14. **Split `coverage_test.go`** — 565 lines, largest file. Mix of coverage and behavioral tests.
15. **Test enforcer uses hardcoded ULID subjects** — Correct but unusual. Most Casbin setups use role names.

### Polish

16. **Add godoc examples** — `SwapStrategy`, `Config`, `Response`, `HTMXRequest` would benefit from runnable examples
17. **Add CONTRIBUTING.md** — Document lint config, test patterns, naming conventions
18. **Add golangci-lint to CI/CD** — GitHub Actions enforcement
19. **Benchmark tests** — `MapError`, `parseHTMXRequest`, `HTMXMiddleware`
20. **Document `.golangci.yml` decisions** — Inline comments explaining exclusions

---

## f) Top 25 Things to Do Next

Sorted by impact × effort (Pareto order):

### High Impact, Low Effort (Do First)

1. **Fix TODO_LIST.md** — Mark dead sentinels done, update metrics (5 min)
2. **Fix FEATURES.md** — Update metrics, strong typing description (10 min)
3. **Add `NewUserID()` test** — 0% → 100%, trivial (2 min)
4. **Test `Enforce` casbin error path** — `authz.go:47`, one test (5 min)
5. **Test `handleQueryDispatch` uncovered branches** — 72.7% → 90%+ (30 min)
6. **Test `decodeFormValues` error paths** — 72.7% (20 min)
7. **Test `decodeFormBody` error paths** — 80.0% (15 min)
8. **Test `setTriggerWithDetail` branches** — 88.2% (15 min)
9. **Test `MapError` remaining families** — 93.3% (10 min)
10. **Test `enrichUserID` parse-failure branch** — 90.9% (10 min)

### High Impact, Medium Effort

11. **Export `HeaderTrue` or add test helper** — Replace 42 test hardcodes (30 min)
12. **Add JSON error response option** — `JSONErrorHandler` alternative (1 hr)
13. **Add correlation ID propagation** — `WithCorrelationID`/`CorrelationIDFromContext` (1 hr)
14. **Split `coverage_test.go`** — 565 lines → focused files (45 min)
15. **Add dispatch lifecycle hooks** — `OnBeforeDispatch`/`OnAfterDispatch` (2 hr)

### Medium Impact, Medium Effort

16. **Add timeout propagation** — Context deadline from request (1 hr)
17. **Add request validation middleware** — Schema validation in decode pipeline (2 hr)
18. **Add godoc examples** — `SwapStrategy`, `Config`, `Response`, `HTMXRequest` (1 hr)
19. **Add benchmark tests** — `MapError`, `parseHTMXRequest`, `HTMXMiddleware` (1 hr)
20. **CONTRIBUTING.md** — Document lint config, test patterns (45 min)

### Lower Impact (Polish)

21. **golangci-lint CI/CD** — GitHub Actions enforcement (30 min)
22. **Document `.golangci.yml` decisions** — Inline comments (15 min)
23. **Consider `//nolint:wrapcheck` on `ParseUserID`** — Thin wrapper (5 min)
24. **Add logging/callback for dropped invalid IDs** — Middleware visibility (30 min)
25. **Review `UserIDExtractor` API for v2** — Could accept `UserID` directly (design discussion)

---

## g) Top Question I Cannot Figure Out

**Should `handleQueryDispatch` and `handleCommandDispatch` share a common pre-dispatch pipeline?**

Currently:

- `handleCommandDispatch` calls `executePreDispatchChecks()` (auth + decoder check) then decodes → dispatches → applies response
- `handleQueryDispatch` inlines the same auth check and decoder check directly

Both follow the same pattern: auth → decoder check → decode → dispatch → response. But they differ in:

- Command has `executePreDispatchChecks` helper, query doesn't
- Query has a render step between dispatch and response
- Query checks `a.queries == nil` inline; command checks `a.commands == nil` inline
- Error wrapping format is identical

**Why I can't decide:** The duplication is ~15 lines, not egregious. Extracting a shared pipeline would reduce it but adds indirection. The question is whether the benefit (single place to modify pre-dispatch logic) outweighs the cost (extra abstraction layer, harder to follow control flow). This is a judgment call that depends on how much these two paths are expected to diverge in the future.

---

## Health Check

| Check                 | Status          |
| --------------------- | --------------- |
| `go test ./... -race` | ✅ 148/148 pass |
| `golangci-lint run`   | ✅ 0 issues     |
| `go build ./...`      | ✅ Clean        |
| `go vet ./...`        | ✅ Clean        |
| Coverage              | 95.5%           |
| Race detector         | ✅ No races     |
| Banned deps           | 0               |

## Project File Map

```
Production (10 files):
  app.go (147 lines)        — App builder, Config, Command(), Query(), enrichUserID()
  handler.go (106 lines)    — handleCommandDispatch(), handleQueryDispatch()
  options.go (271 lines)    — HandlerOption, decoders, Render/RenderTempl, authz helpers
  response.go (179 lines)   — HTMX response builder (fluent API) + notification methods
  authz.go (90 lines)       — Enforcer interface, Authorize, Enforce, AuthorizeMiddleware
  context.go (62 lines)     — UserID type, ParseUserID/MustParseUserID, context enrichment
  errors.go (114 lines)     — CQRS error → HTTP status mapping, sentinels, error handler
  htmx.go (171 lines)       — HTMXRequest struct, accessors, context storage, header constants
  middleware.go (50 lines)  — HTTP middleware (HTMX, ContextEnrichment, Chain)
  notify.go (72 lines)      — Notification HandlerOptions + NotifyWithEvent builder

Tests (9 files):
  app_test.go (521 lines)        — App builder and handler tests
  bdd_test.go (475 lines)        — BDD test types and shared helpers
  htmx_test.go (459 lines)       — HTMX middleware and accessor tests
  coverage_test.go (565 lines)   — Coverage gap tests
  integration_test.go (266 lines)— End-to-end integration tests
  middleware_test.go (127 lines)  — Middleware tests
  errors_test.go (80 lines)      — Error mapping and handler tests
  context_test.go (58 lines)     — Context propagation tests
  suite_test.go (14 lines)       — Ginkgo test suite entry point
```

## Commit History (Last 10 Commits)

```
d94ebd6 docs: fix markdown table alignment in documentation files
bf8ec00 docs: comprehensive status report for strong ID migration session
7612b1b docs: update AGENTS.md and CHANGELOG for dead sentinel removal
e576b30 test: add coverage for middleware dropping unparseable user IDs
3ec00f8 refactor: remove dead sentinels ErrNoUserID and ErrRendererMissing
764f70c feat: strongly-typed UserID (ULID-backed branded type)
04233cb docs: comprehensive self-review status report
c83ce91 docs: update TODO_LIST with completed items from self-review
ad088c0 docs: update AGENTS.md with Enforcer interface, notification builder, error context, and all new gotchas
145dfbf fix: remove unnecessary html.EscapeString from text/plain error handler
```
