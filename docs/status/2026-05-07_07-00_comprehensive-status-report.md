# Comprehensive Status Report — cqrs-htmx

**Date:** 2026-05-07 07:00  
**Branch:** master (clean working tree)  
**Commit:** 367e160 (9 commits ahead of origin 7ebeb89, now pushed)

---

## Executive Summary

All 18 production features are **FULLY_FUNCTIONAL**. Lint is at **0 issues** (down from 103). All **136 tests pass** with race detector. Coverage is **92.6%**. 9 self-contained commits were pushed in this session addressing split brains, dead code, API asymmetry, and config hygiene. The library is in excellent shape with no regressions.

---

## A) FULLY DONE ✅

### Production Code (10 source files, 1271 lines)

| File | Lines | Purpose | Status |
|------|-------|---------|--------|
| `app.go` | 141 | App builder, Config, Command(), Query(), enrichUserID() | ✅ Complete |
| `handler.go` | 106 | handleCommandDispatch(), handleQueryDispatch() | ✅ Complete |
| `options.go` | 249 | HandlerOption, all decoders, Render/RenderTempl, authz helpers | ✅ Complete |
| `response.go` | 184 | HTMX response builder (fluent API) + notification methods | ✅ Complete |
| `authz.go` | 76 | Casbin authorization (Authorize, Enforce, AuthorizeMiddleware) | ✅ Complete |
| `context.go` | 42 | Context enrichment (user ID → CQRS metadata) | ✅ Complete |
| `errors.go` | 119 | CQRS error → HTTP status mapping, sentinels, LoginRedirect | ✅ Complete |
| `htmx.go` | 171 | HTMXRequest struct, accessors, context storage, RenderPartial | ✅ Complete |
| `middleware.go` | 47 | HTTP middleware (HTMXMiddleware, ContextEnrichmentMiddleware, Chain) | ✅ Complete |
| `notify.go` | 36 | Notification HandlerOptions + DefaultNotificationEvent | ✅ Complete |

### Test Code (9 test files, 2274 lines, 136 specs)

| File | Lines | Specs | Purpose | Status |
|------|-------|-------|---------|--------|
| `suite_test.go` | 14 | 1 | Ginkgo suite entry | ✅ |
| `app_test.go` | 453 | 21 | App builder, command/query, auth, handler options | ✅ |
| `bdd_test.go` | 470 | 16 | Consumer integration scenarios (BDD) | ✅ |
| `htmx_test.go` | 451 | 49 | HTMX accessors, Response builder, SwapStrategy, context | ✅ |
| `coverage_test.go` | 477 | 24 | Coverage gap tests, edge cases, notification helpers | ✅ |
| `integration_test.go` | 265 | 8 | Full E2E CQRS + HTMX + Casbin flow | ✅ |
| `middleware_test.go` | 109 | 4 | ContextEnrichmentMiddleware, Chain | ✅ |
| `context_test.go` | 55 | 6 | WithUserID, UserIDFromContext, EventOptionsFromContext | ✅ |
| `errors_test.go` | 80 | 8 | MapError, DefaultErrorHandler, sentinel errors | ✅ |

### Session Accomplishments (9 commits, this session)

1. **`98616cb`** — Fix `Response.Refresh()` split brain (hardcoded `"true"` → `headerTrue` constant)
2. **`a914f86`** — Remove 3 redundant gocritic disabled-checks causing warnings
3. **`6e9bac8`** — Remove dead `io`/`event` imports + suppressor hack from `app_test.go`
4. **`f56be10`** — Consolidate 3 duplicate test types (`mockTemplComponent`, `deleteUserCmd`, `listUsersQuery`)
5. **`b6bfd07`** — Convert NotifySuccess test to use `testNotificationTrigger` helper
6. **`e703cf8`** — Rename `htmlSb226` → `sb` in bdd_test.go
7. **`8a3fc27`** — Optimize `executeAuthorization` to call `UserIDFromContext` once (was twice)
8. **`8881299`** — Add `DecodeFormQuery` for API symmetry + BDD test
9. **`367e160`** — Update AGENTS.md with all findings

### Previous Session Accomplishments

- **Lint zero**: Eliminated all 103 golangci-lint issues (exhaustruct, unused-param, dot-imports, etc.)
- **Code quality**: Extracted constants, fixed ~70 unused parameters, safe type assertions
- **BDD tests**: Added comprehensive consumer scenario tests
- **Architecture**: Extracted helper functions, reduced cyclomatic complexity

---

## B) PARTIALLY DONE 🟡

### TODO_LIST.md is STALE

The TODO_LIST.md was written BEFORE the lint-zero session and the self-review session. Several items are marked `[ ]` but have been completed:

| Item | Marked | Actual Status |
|------|--------|---------------|
| Extract `"true"` string constant | `[ ]` | ✅ Done as `headerTrue` in commit from lint-zero session |
| Add doc comment to SwapStrategy const block | `[ ]` | ✅ Done in lint-zero session |
| Deduplicate notification test boilerplate | `[ ]` | ✅ Done — `testNotificationTrigger` helper + NotifySuccess converted |
| Consistent test naming | `[ ]` | ✅ Done — consolidated to `bdd*` prefix |

### FEATURES.md is STALE

- Missing `DecodeFormQuery` (added this session)
- Coverage shows 92.8% but is now 92.6% (different measurement run)
- Test count shows 121 but is now 136

### README.md is STALE

- Missing `DecodeFormQuery` in the Request Decoding table (line 101)
- `LoginRedirect` example shows package-level mutation pattern which is a race condition concern

---

## C) NOT STARTED ⬜

### P0 — Security & Correctness

- [ ] **Fix XSS in DefaultErrorHandler** — `errors.go:118` writes `err.Error()` unsanitized via `html.EscapeString()`. Actually, it IS escaped now. Need to verify this is truly safe.
- [ ] **Move LoginRedirect to per-App config** — `defaultLoginRedirect` is a package-level var; `Config.LoginRedirect` copies it in `New()` but the global `defaultLoginRedirect` is still mutable. Race condition with multiple Apps.
- [ ] **Move NotificationEvent to per-App config** — `DefaultNotificationEvent` is a mutable global. Same race concern.
- [ ] **Extract Casbin interface** — `authz.go` uses `*casbin.Enforcer` concrete type. Should define interface for testability.

### P1 — Architecture

- [ ] **Add dispatch lifecycle hooks** — `OnBeforeDispatch` / `OnAfterDispatch` for logging/metrics/tracing
- [ ] **Add request validation middleware** — Optional schema validation in decode pipeline
- [ ] **Add observability/logging hooks** — No logging or metrics middleware exists
- [ ] **Add correlation ID propagation** — `WithCorrelationID` / `CorrelationIDFromContext`
- [ ] **Add JSON error response option** — DefaultErrorHandler only returns plain text
- [ ] **Add timeout propagation** — Library doesn't set deadlines on context

### P2 — Polish

- [ ] **Add examples to exported types** — `SwapStrategy`, `HTMXRequest`, `Config` godoc examples
- [ ] **Create CONTRIBUTING.md** — Document lint config, test patterns, naming conventions
- [ ] **Add `golangci-lint` to CI/CD** — Ensure zero-lint enforcement in automation
- [ ] **Add pre-commit hook** — Run `golangci-lint run` and `go test` before commits
- [ ] **Add benchmark tests** — Hot paths: `MapError`, `parseHTMXRequest`, `HTMXMiddleware`
- [ ] **Document `.golangci.yml` decisions** — Add comments explaining why each exclusion exists
- [ ] **Review `sync.Once` pattern** — Verify lazy initialization race conditions
- [ ] **Review LSP vs CLI discrepancy** — LSP shows ~31 stale warnings that CLI does not

---

## D) TOTALLY FUCKED UP 💥

### Nothing is truly broken.

The library compiles, passes all tests, has zero lint issues, and all features work. However, there are two architectural concerns that could bite consumers:

1. **Mutable globals** — `defaultLoginRedirect` and `DefaultNotificationEvent` are package-level `var`s that any consumer can mutate, causing race conditions when multiple `App` instances exist in the same process. This is a design smell, not a bug — yet.

2. **LSP stale cache** — The `golangci_lint_ls` LSP server shows ~31 warnings (unused-parameter, dupl, goconst, etc.) that `golangci-lint run` CLI does not report. This is an unresolved LSP cache issue that creates confusion during development. The `.golangci.yml` test-file exclusions under `linters.exclusions.rules` seem to not be picked up by the LSP.

---

## E) WHAT WE SHOULD IMPROVE 📈

### Architecture

1. **Eliminate mutable globals** — Move `defaultLoginRedirect` and `DefaultNotificationEvent` to per-App config. This is the single biggest architectural improvement possible.
2. **Extract Casbin interface** — `*casbin.Enforcer` is a concrete dependency. An interface would make testing easier and decouple from Casbin's API evolution.
3. **Add lifecycle hooks** — Even simple `OnBeforeDispatch`/`OnAfterDispatch` callbacks would enable logging/metrics without coupling.

### Type Models

4. **`SwapStrategy` should be a type-safe enum** — Currently a `string` alias. Could use Go 1.26's `iter` + sealed interface pattern for exhaustive matching.
5. **`handlerConfig` could be split** — It mixes auth config (authorize, requireAuth, resource, action) with response config (redirect, trigger, pushURL) with decoder config. Three focused structs would be clearer.
6. **`Config` struct has too many optionals** — `Enforcer`, `UserIDExtractor`, `ErrorHandler`, `LoginRedirect` are all optional. Consider functional options pattern or builder pattern.

### Testing

7. **Test helper builder** — Many tests repeat the same `command.NewDispatcher() → register → cqrshtmx.New → handler` pattern. Extract a test builder.
8. **`testNotificationTrigger` has unused first param** — Takes a `string` name that's never used. Remove it.
9. **Coverage is 92.6%, not 93.1%** — Some edge cases in trigger merging (`setTriggerWithDetail` error paths) may not be covered.

### Documentation

10. **README.md missing `DecodeFormQuery`** — Table at line 101 only shows `DecodeForm[T]`, not the new `DecodeFormQuery[T]`.
11. **TODO_LIST.md stale** — 4 items marked `[ ]` are actually done.
12. **FEATURES.md stale** — Missing `DecodeFormQuery`, wrong coverage/test count.

---

## F) Top 25 Things to Do Next (Priority Order)

### Tier 1: Critical — Do Immediately (1-2 hours total)

| # | Task | Impact | Effort | Rationale |
|---|------|--------|--------|-----------|
| 1 | **Update TODO_LIST.md** — Mark 4 stale items as done | High | 5min | Current state is misleading |
| 2 | **Update FEATURES.md** — Add `DecodeFormQuery`, fix coverage/test count | High | 5min | Public-facing doc is wrong |
| 3 | **Update README.md** — Add `DecodeFormQuery` to decoder table | Medium | 5min | API docs are incomplete |
| 4 | **Remove unused first param from `testNotificationTrigger`** | Low | 2min | Dead code in helper |
| 5 | **Verify XSS safety in DefaultErrorHandler** | High | 5min | Security: confirm `html.EscapeString` is sufficient |

### Tier 2: Important — Do This Week (4-8 hours)

| # | Task | Impact | Effort | Rationale |
|---|------|--------|--------|-----------|
| 6 | **Move LoginRedirect to per-App** — Eliminate `defaultLoginRedirect` global | High | 1h | Race condition risk |
| 7 | **Move NotificationEvent to per-App** — Eliminate `DefaultNotificationEvent` global | High | 1h | Race condition risk |
| 8 | **Extract Casbin interface** — `Enforcer` interface with `Enforce(...)` | Medium | 2h | Testability + decoupling |
| 9 | **Add godoc examples** — `SwapStrategy`, `Config`, `Response`, `HTMXRequest` | Medium | 2h | GoDoc quality |
| 10 | **Add CI/CD lint enforcement** — GitHub Actions with `golangci-lint` | High | 1h | Prevent regression |

### Tier 3: Nice to Have — Do Eventually

| # | Task | Impact | Effort | Rationale |
|---|------|--------|--------|-----------|
| 11 | **Add dispatch lifecycle hooks** — `OnBeforeDispatch`/`OnAfterDispatch` | High | 3h | Observability |
| 12 | **Add JSON error response option** — Alternative to plain text | Medium | 1h | API flexibility |
| 13 | **Add correlation ID propagation** — `WithCorrelationID`/`CorrelationIDFromContext` | Medium | 2h | Distributed tracing |
| 14 | **Extract test builder** — Reduce boilerplate across test files | Low | 2h | Maintainability |
| 15 | **Add benchmark tests** — `MapError`, `parseHTMXRequest`, `HTMXMiddleware` | Medium | 1h | Performance baseline |
| 16 | **Create CONTRIBUTING.md** — Document conventions | Low | 1h | Open source readiness |
| 17 | **Document `.golangci.yml` decisions** — Inline comments | Low | 15min | Config readability |
| 18 | **Review `sync.Once` race conditions** — Verify concurrent first-call safety | Medium | 30min | Correctness |
| 19 | **Add pre-commit hook** — `golangci-lint run` + `go test` | Low | 30min | Developer experience |
| 20 | **Fix LSP stale cache** — Investigate why exclusions aren't picked up | Low | 1h | DX: removes false warnings |

### Tier 4: Future — Consider for v2

| # | Task | Impact | Effort | Rationale |
|---|------|--------|--------|-----------|
| 21 | **Add timeout propagation** — Context deadlines for dispatch | Medium | 2h | Production hardening |
| 22 | **Add request validation middleware** — Schema validation in decode pipeline | Medium | 3h | Consumer convenience |
| 23 | **Refactor `handlerConfig`** — Split auth/response/decoder into focused structs | Low | 3h | Clarity |
| 24 | **Add `SwapStrategy` type-safe enum** — Sealed interface pattern | Low | 1h | Type safety |
| 25 | **Consider functional options for `Config`** — Replace optional struct fields | Low | 2h | Builder pattern |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Is `DefaultErrorHandler`'s XSS protection actually correct?**

`errors.go:118` does:
```go
_, _ = w.Write([]byte(html.EscapeString(err.Error())))
```

This HTML-escapes the error message before writing. But:
- The `Content-Type` is set to `text/plain; charset=utf-8` — for plain text, HTML escaping is unnecessary and distorts error messages (e.g., `"foo < bar"` becomes `"foo &lt; bar"`).
- If the content type were `text/html`, HTML escaping would be correct.
- The real XSS risk is if a consumer passes user input into an error that ends up in an HTML-rendered HTMX response. But the default handler writes plain text, not HTML.

**The question is:** Should we remove the `html.EscapeString` since the content type is `text/plain`? Or should we keep it as a defense-in-depth measure because HTMX responses might be injected into HTML? I cannot determine the intended threat model without understanding how consumers render error responses in their HTMX UIs.

---

## Metrics Dashboard

| Metric | Value | Trend |
|--------|-------|-------|
| Production files | 10 | Stable |
| Production lines | 1271 | Stable |
| Test files | 9 | Stable |
| Test lines | 2274 | Stable |
| Test specs (It()) | 136 | +1 (DecodeFormQuery) |
| golangci-lint issues | **0** | ↓ from 103 |
| Coverage | **92.6%** | Stable |
| Banned dependencies | 0 | Clean |
| `go vet` issues | 0 | Clean |
| Race detector | Clean | Clean |
| Open TODO items | 25 | Prioritized above |
| Stale docs | 3 (TODO_LIST, FEATURES, README) | Needs update |
