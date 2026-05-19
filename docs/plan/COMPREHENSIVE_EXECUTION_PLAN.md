# Comprehensive Execution Plan

**Date:** 2026-05-19  
**Status:** 9 tests failing on master (commit `586d24b`)  
**Total Specs:** 249 | **Passing:** 240 | **Failing:** 9 | **Coverage:** ~94%

---

## Executive Summary

Commit `586d24b` introduced 2 bugs that broke the test suite:

1. **Nil decoder panic** — `handleCommandDispatch` no longer checks `cfg.commandDecoder == nil` before calling it
2. **CSRF regression** — 7 CSRF tests fail with 403 after gorilla/csrf v1.7.3 bump

This plan covers fixing these bugs plus all remaining architectural improvements, features, and quality work identified during the comprehensive review.

---

## Phase 1: Critical Fixes (Do First — Blocking)

### P0.1 Fix `handleCommandDispatch` Nil Decoder Panic

**Bug:** Commit `586d24b` moved the `commandDecoder == nil` check to AFTER `cfg.commandDecoder(r)` is called. When no decoder is configured (e.g., `app.Command("CreateUser")` with no `DecodeJSON` option), `cfg.commandDecoder` is nil and calling it panics.

**Repro:** `go test ./...` — 2 specs panic:

- `Validation HandlerOption/ValidateCommand/no-op when decoder is not set`
- `App/Command handler/returns error when decoder is missing`

**Fix:** In `handler.go:handleCommandDispatch`, add nil check BEFORE calling `cfg.commandDecoder(r)`:

```go
if cfg.commandDecoder == nil {
    a.errorHandler(w, r, errDecoderMissing)
    a.afterDispatchHook(ctx, r, errDecoderMissing)
    return
}
```

**Files:** `handler.go`
**Est. Time:** 5 min

---

### P0.2 Fix CSRF Test Regression

**Bug:** 7 CSRF-related specs return 403 instead of 200 after gorilla/csrf v1.7.3 bump. The token validation flow broke.

**Repro:** `go test ./...` — 7 specs fail:

- `CSRF Protection/CSRFMiddleware/allows POST with valid CSRF token in header`
- `CSRF Protection/CSRFMiddleware/allows POST with valid CSRF token in form field`
- `CSRF Protection/CSRFMiddleware/validates PUT, PATCH, and DELETE methods`
- `CSRF Protection/CSRFMiddleware/uses custom header name when configured`
- `CSRF Protection/HMAC-signed tokens with Secret/validates HMAC-signed token correctly`
- `Full Integration/End-to-end CQRS + HTMX + CSRF protection/allows command dispatch with valid CSRF token via HTMX header`
- `Full Integration/End-to-end CQRS + CSRFProtect per-handler option/allows command dispatch with CSRFProtect and valid token`

**Root Cause Hypotheses:**

1. gorilla/csrf v1.7.3 changed token format or validation behavior
2. `executeCSRFValidation` writes to `ResponseWriter` via gorilla/csrf, which may conflict with our error handling
3. The `CSRFTokenFromContext` fallback in template helpers may not match gorilla/csrf's token format

**Fix Strategy:**

1. Downgrade gorilla/csrf to v1.7.2 to verify if bump caused regression
2. If downgrade fixes, investigate v1.7.3 changelog for breaking changes
3. If still broken, the issue is in our integration code, not the bump

**Files:** `csrf.go`, `go.mod`, test files
**Est. Time:** 20-30 min

---

## Phase 2: Security Hardening (High Impact)

### P1.1 Fix `executeCSRFValidation` ResponseWriter Conflict

**Bug:** `executeCSRFValidation` calls `protect(dummy).ServeHTTP(w, r)` which writes to `w` on failure. This conflicts with the caller's error handling.

**Fix:** Use `httptest.ResponseRecorder` to capture gorilla/csrf's response, then translate through our error handler.

**Files:** `csrf.go`
**Est. Time:** 15 min

---

### P1.2 Warn on Empty CSRF Secret

**Bug:** If `CSRFConfig.Secret` is empty, `CSRFMiddleware` generates a random 32-byte key per call. Server restarts invalidate all tokens.

**Fix:** Log a warning at middleware init time if secret is empty.

**Files:** `csrf.go`
**Est. Time:** 10 min

---

### P1.3 Warn on `Secure=false`

**Bug:** gorilla/csrf does NOT auto-detect HTTPS. Consumers behind reverse proxies get `Secure=false` unless explicitly set.

**Fix:** Add `CSRFConfig.Validate()` method that warns/logs when `Secure=false` in production-like environments. Or document this prominently.

**Files:** `csrf.go`, `README.md`
**Est. Time:** 15 min

---

### P1.4 Extract `isAuthError` Helper

**Code Smell:** `DefaultErrorHandlerWithRedirect` and `JSONErrorHandlerWithRedirect` duplicate the same HTMX auth check:

```go
if IsHTMXRequest(r) && (errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden) || errors.Is(err, ErrCSRFInvalid)) {
    w.Header().Set(headerRedirect, loginRedirect)
    w.WriteHeader(http.StatusSeeOther)
    return
}
```

**Fix:** Extract `func isAuthError(err error) bool` helper.

**Files:** `errors.go`
**Est. Time:** 10 min

---

## Phase 3: Architecture Improvements (Medium Impact)

### P2.1 Split `options.go` (340 lines — over threshold)

**Problem:** `options.go` has too many responsibilities:

- Type definitions (HandlerOption, authMode, handlerConfig, decoders, renderers)
- Handler config methods (hasNoExplicitBody)
- Decoder functions (decodeJSONBody, decodeFormBody, decodeRequest, decodeFormValues)
- Handler option constructors (DecodeJSON, DecodeForm, Render, Redirect, etc.)
- Authorization logic (executeAuthorization)
- Response application (applyHTMXResponse)

**Proposed split:**

- `decoder.go`: `decodeJSONBody`, `decodeFormBody`, `decodeRequest`, `decodeFormValues`
- `handler_config.go`: `handlerConfig`, `authMode`, `hasNoExplicitBody`, `executeAuthorization`
- `options.go`: Public `HandlerOption` constructors only

**Files:** New `decoder.go`, `handler_config.go`; modified `options.go`
**Est. Time:** 20 min

---

### P2.2 Move `registerErrorClassifications` to `init()`

**Problem:** `sync.Once` in `MapError` hot path is wasteful.

**Fix:** Register once at package init time.

**Files:** `errors.go`
**Est. Time:** 5 min

---

### P2.3 Fix `perKeyLimiter` Memory Leak

**Problem:** `map[string]*rate.Limiter` grows unbounded with no cleanup.

**Fix Options:**

1. Add TTL-based eviction (simplest: check last access time on Allow)
2. Use a bounded LRU cache (e.g., `container/list` or `golang.org/x/exp/lru`)
3. Document as known limitation and recommend middleware wrapping

**Recommendation:** Option 1 — add `lastUsed time.Time` to limiter entry, evict on access if stale > 2×window.

**Files:** `ratelimit.go`
**Est. Time:** 30 min

---

### P2.4 Add `//nolint:funlen` to `handleQueryDispatch`

**Problem:** Pre-existing lint issue (61 lines, limit 60).

**Fix:** Add `//nolint:funlen` with justification, or extract hook logic to helper.

**Files:** `handler.go`
**Est. Time:** 2 min

---

## Phase 4: Features (Lower Impact, Higher Value)

### P3.1 Add `RotateCSRFToken()` Helper

**Feature:** Rotate CSRF token on login/logout to prevent fixation attacks.

**API:** `func RotateCSRFToken(w http.ResponseWriter, r *http.Request, cfg CSRFConfig) error`

**Files:** `csrf.go`
**Est. Time:** 20 min

---

### P3.2 Create `example/basic/` Directory

**Feature:** Runnable example showing CQRS + HTMX + CSRF end-to-end.

**Contents:**

- `main.go` — HTTP server with middleware chain
- `templates/index.html` — HTMX form with CSRF token
- `README.md` — How to run

**Files:** New `example/basic/`
**Est. Time:** 30 min

---

### P3.3 Add CSRF Benchmarks

**Feature:** Measure middleware overhead.

**Tests:**

- Benchmark `CSRFMiddleware` with/without gorilla/csrf
- Benchmark token generation
- Benchmark validation

**Files:** `benchmark_test.go`
**Est. Time:** 20 min

---

### P3.4 Write `SECURITY.md`

**Contents:**

- Security features overview
- Responsible disclosure process
- Hardening recommendations
- Known limitations (rate limiter memory, CSRF secret rotation)

**Files:** `SECURITY.md`
**Est. Time:** 15 min

---

### P3.5 Add `govulncheck` to CI

**Feature:** Vulnerability scanning in GitHub Actions.

**Files:** `.github/workflows/ci.yml`
**Est. Time:** 10 min

---

### P3.6 Fix BuildFlow Pre-commit Hook

**Problem:** Hook enforces incompatible rules (root-level files must be in `internal/` or `pkg/`, demands `flake.nix`, treats NOTE comments as TODOs).

**Fix Options:**

1. Add `.buildflow.yml` config to disable incompatible rules
2. Remove pre-commit hook, rely on CI only
3. Make hook non-blocking (warn, don't fail)

**Recommendation:** Option 1 — add `.buildflow.yml`.

**Files:** `.buildflow.yml`
**Est. Time:** 15 min

---

## Phase 5: Future Work (Nice to Have)

| #     | Task                                                               | Why                       |
| ----- | ------------------------------------------------------------------ | ------------------------- |
| P4.1  | Add `CSRFConfig` validation (SameSite=None without Secure → error) | Prevent broken configs    |
| P4.2  | Document Secure flag + reverse proxy (`X-Forwarded-Proto`)         | Proxy deployments         |
| P4.3  | Add `CSRFToken` branded type (`type CSRFToken string`)             | Type safety               |
| P4.4  | Functional options for `CSRFConfig`                                | Cleaner API               |
| P4.5  | Extract `gorilla/csrf` adapter into internal package               | Cleaner separation        |
| P4.6  | Support double-submit without cookie (session-backed)              | Alternative pattern       |
| P4.7  | Add CSRF bypass for trusted origins/internal IPs                   | Internal APIs             |
| P4.8  | Integration test with real `httptest.Server`                       | More realistic            |
| P4.9  | Add snapshot testing with `go-snaps`                               | Reduce brittle assertions |
| P4.10 | Add `CSRFMiddleware` warn-only mode                                | Gradual rollout           |

---

## Execution Order

```
Phase 1 (Blocking)
├── P0.1 Fix nil decoder panic        [5 min]
├── P0.2 Fix CSRF regression          [20-30 min]
└── Verify: go test ./... passes      [2 min]

Phase 2 (Security)
├── P1.1 CSRF ResponseWriter fix      [15 min]
├── P1.2 Empty CSRF secret warning    [10 min]
├── P1.3 Secure=false warning         [15 min]
├── P1.4 isAuthError helper           [10 min]
└── Commit all Phase 2                [2 min]

Phase 3 (Architecture)
├── P2.1 Split options.go             [20 min]
├── P2.2 registerErrorClassifications [5 min]
├── P2.3 Rate limiter memory leak     [30 min]
├── P2.4 funlen suppress              [2 min]
└── Commit all Phase 3                [2 min]

Phase 4 (Features)
├── P3.1 RotateCSRFToken              [20 min]
├── P3.2 example/basic/               [30 min]
├── P3.3 CSRF benchmarks              [20 min]
├── P3.4 SECURITY.md                  [15 min]
├── P3.5 govulncheck CI               [10 min]
├── P3.6 BuildFlow config             [15 min]
└── Commit all Phase 4                [2 min]
```

**Total Estimated Time:** ~4.5 hours

---

## Current Test Failures (Master @ `586d24b`)

| Spec                                                             | Error              | Root Cause                                                              |
| ---------------------------------------------------------------- | ------------------ | ----------------------------------------------------------------------- |
| `ValidateCommand/no-op when decoder is not set`                  | PANIC: nil pointer | `handleCommandDispatch` calls `cfg.commandDecoder(r)` without nil check |
| `Command handler/returns error when decoder is missing`          | PANIC: nil pointer | Same as above                                                           |
| `CSRFMiddleware/allows POST with valid CSRF token in header`     | 403, expected 200  | gorilla/csrf v1.7.3 regression or integration bug                       |
| `CSRFMiddleware/allows POST with valid CSRF token in form field` | 403, expected 200  | Same                                                                    |
| `CSRFMiddleware/validates PUT, PATCH, and DELETE methods`        | 403, expected 200  | Same                                                                    |
| `CSRFMiddleware/uses custom header name when configured`         | 403, expected 200  | Same                                                                    |
| `CSRFMiddleware/HMAC-signed token correctly`                     | 403, expected 200  | Same                                                                    |
| `Integration/CSRF token via HTMX header`                         | 403, expected 200  | Same                                                                    |
| `Integration/CSRFProtect with valid token`                       | 403, expected 204  | Same                                                                    |

---

## Success Criteria

- [ ] All 249 specs pass
- [ ] Coverage ≥ 95%
- [ ] golangci-lint clean (except pre-existing funlen)
- [ ] No new dependencies for P0-P2
- [ ] SECURITY.md exists
- [ ] example/basic/ runs end-to-end
- [ ] BuildFlow hook no longer requires `--no-verify`

---

_Plan created: 2026-05-19_  
_Next action: Fix P0.1 nil decoder panic, then P0.2 CSRF regression_
