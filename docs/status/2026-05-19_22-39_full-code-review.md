# Full Code Review — cqrs-htmx

**Date:** 2026-05-19_22-39
**Reviewer:** Senior Software Architect
**Files reviewed:** 15 production files, 20 test files

## Overall Assessment

**Grade: A-**

The codebase is well-architected with strong type safety, clean interfaces, and excellent test coverage (94.7%, 289 Ginkgo specs). The library API is clean and composable. Below are findings organized by severity.

## Critical Findings

None.

## High-Priority Findings

### H1: Rate limiter map grows unbounded under pathological key patterns
**File:** `ratelimit.go:131-207`
**Problem:** `perKeyLimiter.limiters` evicts stale entries only on cache misses. If many unique keys are accessed exactly once (DDoS with random IPs), the map grows until the next eviction cycle. The TTL-based cleanup only runs when a new key needs creation.
**Recommendation:** Add periodic background cleanup or cap the max number of entries. Consider `sync.Map` for read-heavy workloads or a LRU-eviction strategy.

### H2: CSRF per-handler validation creates a new gorilla/csrf Protect instance per request
**File:** `csrf.go:413-444`
**Problem:** `executeCSRFValidation` creates `csrf.Protect()` + `httptest.ResponseRecorder` for every request when `CSRFProtect` is used. This is allocation-heavy per-request.
**Recommendation:** Cache the Protect instance per CSRFConfig at handler creation time rather than per-request.

### H3: `applyQueryResponse` uses `r.Context()` instead of the enriched context
**File:** `handler.go:124`
**Problem:** `a.handleErr(w, r, r.Context(), err)` passes `r.Context()` but the query dispatch used a context with timeout (`ctx`). If the render fails due to timeout, the error handler gets the original context, not the timed-out one.
**Recommendation:** Pass `ctx` from the caller. Note: `handleQueryDispatch` does `r.WithContext(ctx)` at line 166 but `applyQueryResponse` gets `r` without the enriched context.

## Medium-Priority Findings

### M1: HTMX accessor functions follow identical pattern (8 functions)
**File:** `htmx.go:96-170`
**Problem:** `IsHTMXRequest`, `IsBoosted`, `IsHistoryRestore`, `HTMXTarget`, `HTMXTrigger`, `HTMXTriggerName`, `HTMXPrompt`, `HTMXCurrentURL` all follow the same check-context-then-fall-back-to-header pattern.
**Recommendation:** Generic helper: `htmxField[T](r, extractFunc, headerName) T`.

### M2: ValidateCommand/Query are structurally identical
**File:** `options.go:201-250`
**Problem:** Both functions have the same structure: check nil decoder, wrap with validation, short-circuit on decode error. Only differs in `command.Command` vs `query.Query`.
**Recommendation:** Consider a shared generic validation wrapper.

### M3: Logging context extraction repeated 3 times
**File:** `logging.go:33-37, 62-67, 153-163`
**Problem:** `CorrelationIDFromContext` + `UserIDFromContext` extraction block is copy-pasted across `DefaultLogFormatter`, `JSONLogFormatter`, and `RequestLoggingSlog`.
**Recommendation:** Extract `appendContextAttrs(attrs, r)` helper.

### M4: `statusRecorder` setup duplicated
**File:** `logging.go:95, 136`
**Problem:** Both `RequestLogging` and `RequestLoggingSlog` create `statusRecorder` with identical setup/teardown.
**Recommendation:** Extract `measureRequest(next, w, r, callback)` helper.

### M5: `SanitizeRedirectURL` blocks root `/` but not `/./`
**File:** `response.go:187-205`
**Problem:** While `path.Clean` normalizes `..`, the function allows paths like `/./foo` which clean to `/foo` (valid). This is correct behavior but worth documenting that `path.Clean` is the security boundary.

### M6: Error handlers share login redirect default logic
**File:** `errors.go:127-183`
**Problem:** `DefaultErrorHandlerWithRedirect` and `JSONErrorHandlerWithRedirect` both duplicate `if loginRedirect == "" { loginRedirect = defaultLoginRedirect }` and `writeHTMXAuthRedirect`.
**Recommendation:** Extract shared error handler core.

## Low-Priority Findings

### L1: `cockroachdb/errors` vs `fmt.Errorf` inconsistency
**Files:** `authz.go` uses `errors.Wrapf` while `handler.go` uses `fmt.Errorf("%w: ...")`.
**Note:** The AGENTS.md documents this as intentional (gotcha #2), but consistency within the same package would be cleaner.

### L2: `csrf.go` is 445 lines — the largest file
**File:** `csrf.go`
**Problem:** At 445 lines, this is above the 350-line soft limit. The CSRFConfig defaults, template helpers, and middleware could be split.
**Recommendation:** Extract template helpers (`CSRFTokenHTMLMeta`, `CSRFTokenHXHeaders`, `CSRFTokenFormField`, `RotateCSRFToken`) to `csrf_helpers.go`.

### L3: `response.go` content type constant inconsistency
**File:** `response.go:11-15, 179`
**Note:** `ContentTypeHTML`, `ContentTypePlain`, `ContentTypeJSON` are defined in `response.go` but used in `errors.go`. Consider moving to a shared constants file or keeping as-is since `response.go` is the HTTP response center.

### L4: usermgmt `handler_test.go` uses `nil` context
**File:** `usermgmt/handler_test.go:314, 322`
**Problem:** `gopls SA1012` warns about nil context. Should use `context.TODO()` or `context.Background()`.
**Severity:** Test-only, but `staticcheck` flags it.

## Positive Observations

1. **Strong typing**: `authMode` enum, branded `UserID`/`CorrelationID`/`RequestID` types — impossible states are unrepresentable
2. **Clean interfaces**: `Enforcer` interface matches Casbin perfectly, `TemplComponent` duck-typing avoids templ import
3. **Comprehensive godoc**: Every exported function has documentation with usage examples
4. **Error classification**: `sync.Once` lazy-registration, proper sentinel error hierarchy
5. **Security**: Redirect URL sanitization, CSRF with BREACH mitigation, plaintext HTTP detection for gorilla/csrf v1.7.3
6. **Middleware composition**: `Chain()` with `slices.Backward` is clean and correct
7. **Context dedup**: `enrichUserID` skips if middleware already set the ID — prevents double-extraction
8. **Timeout design**: Timeout wraps dispatch only (not decode/auth) — correct separation of concerns
9. **Benchmarks**: 16 benchmarks covering hot paths

## File-by-File Summary

| File | Lines | Assessment | Key Notes |
|------|-------|-----------|-----------|
| `app.go` | 213 | Clean | Builder pattern, hooks, timeout delegation |
| `handler.go` | 169 | Good | H3: context mismatch in applyQueryResponse |
| `options.go` | 282 | Good | M2: validation duplication |
| `response.go` | 244 | Clean | Fluent API, URL sanitization |
| `authz.go` | 126 | Clean | authMode enum, AuthorizeMiddleware |
| `context.go` | 153 | Clean | Branded types, EventOptionsFromContext |
| `errors.go` | 184 | Good | Sentinel hierarchy, HTMX-aware error handling |
| `htmx.go` | 171 | Good | M1: accessor pattern duplication |
| `middleware.go` | 75 | Clean | Chain with slices.Backward |
| `notify.go` | 79 | Clean | NotificationLevel type, NotifyWithEvent builder |
| `decoder.go` | 110 | Clean | Generic decodeJSONBody/decodeFormBody |
| `logging.go` | 193 | Good | M3/M4: extraction duplication |
| `security.go` | 113 | Clean | Configurable security headers |
| `ratelimit.go` | 208 | Good | H1: unbounded map growth |
| `csrf.go` | 445 | Good | L2: file size, H2: per-request allocation |

## Recommendations (Pareto Priority)

1. **1% → 51%**: Fix H3 (context mismatch) — correctness issue
2. **4% → 64%**: Address H1 (rate limiter bounds) — production safety
3. **20% → 80%**: Refactor M1-M4 (deduplication) — maintainability
4. **Remaining**: L2 (split csrf.go), L4 (fix test nil context)
