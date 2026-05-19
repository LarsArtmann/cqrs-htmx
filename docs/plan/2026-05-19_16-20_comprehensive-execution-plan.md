# Comprehensive Execution Plan — What Was Missed & What Comes Next

**Date:** 2026-05-19 16:20 CEST  
**Source:** Self-critique + deep architecture analysis  
**Constraint:** Each task ≤ 12 minutes

---

## Executive Summary: What Was Forgotten / Could Be Better

### 🔴 Critical Bugs Introduced in This Session

| #   | Bug                                                                                        | Location                                    | Impact                                      |
| --- | ------------------------------------------------------------------------------------------ | ------------------------------------------- | ------------------------------------------- |
| 1   | **Data race**: `entry.lastUsed = time.Now()` writes without lock after `RUnlock()`         | `ratelimit.go:147`                          | Race condition under concurrent load        |
| 2   | **Wrong sentinel**: `CSRFConfig.Validate()` returns `ErrEnforcerNil` for empty CSRF secret | `csrf.go:136`                               | Semantically incorrect error classification |
| 3   | **Coverage dropped 1.2%**: New code has 0% tests                                           | `RotateCSRFToken`, `Validate`, TTL eviction | 93.7% vs 94.9% before                       |

### 🟡 Architecture Gaps Not Addressed

| #   | Gap                                           | Why It Matters                                                       |
| --- | --------------------------------------------- | -------------------------------------------------------------------- |
| 4   | **No body size limits** on JSON/form decoders | DoS vulnerability — attacker sends multi-GB request body             |
| 5   | **decodeFormValues uses JSON round-trip**     | Breaks nested structs, time fields, custom unmarshalers              |
| 6   | **ErrorHandler receives no status code**      | Forces re-derivation of HTTP status from error in every handler      |
| 7   | **No RequestID** — only CorrelationID         | Can't distinguish multiple requests with same correlation ID         |
| 8   | **RequestLogging writes plain strings**       | No structured logging integration (slog, zap, etc.)                  |
| 9   | **No per-handler timeout override**           | Global `Config.Timeout` only — can't set tighter limits per endpoint |
| 10  | **SecurityHeadersMiddleware is static**       | No builder for CSP, HSTS, or custom headers                          |

---

## Phase 0: Critical Bug Fixes (Do First — Blocking)

### P0.1 Fix Data Race in `ratelimit.go`

**Bug:** `entry.lastUsed = time.Now()` on line 147 writes to shared memory after releasing the read lock. Under concurrent load, multiple goroutines race on the same `lastUsed` field.

**Fix Options:**

- A: Remove the write (entries may be evicted even if active, but recreated cheaply)
- B: Use `atomic.Int64` for `lastUsed` (cleanest, no lock contention)
- C: Take full write lock for the update (simplest, adds contention)

**Decision:** Option B — `atomic.Int64` storing Unix nanoseconds. Zero contention, type-safe.

**Files:** `ratelimit.go`  
**Est. Time:** 8 min  
**Impact:** 🔥 Critical — correctness under concurrency

---

### P0.2 Fix `CSRFConfig.Validate()` Sentinel Error

**Bug:** Returns `ErrEnforcerNil` ("casbin enforcer is required for authorization") for an empty CSRF secret. Wrong domain, wrong semantics.

**Fix:** Create `ErrCSRFConfig` sentinel and return it instead.

**Files:** `csrf.go`, `errors.go`  
**Est. Time:** 5 min  
**Impact:** 🔥 Critical — error classification correctness

---

## Phase 1: Test Coverage Recovery (High Impact, Low Effort)

### P1.1 Add Tests for `RotateCSRFToken`

**Coverage:** 0% → target 100%  
**What to test:**

- Cookie has `MaxAge=-1`
- Cookie `Value` is empty
- Cookie `Expires` is in the past
- Cookie uses correct name from config
- Cookie uses correct path, domain, secure, sameSite from config

**Files:** `csrf_test.go`  
**Est. Time:** 10 min  
**Impact:** High — recovers coverage

---

### P1.2 Add Tests for `CSRFConfig.Validate()`

**Coverage:** 0% → target 100%  
**What to test:**

- Empty Secret → returns error
- SameSite=None + Secure=false → returns error
- Valid config → returns nil
- SameSite=None + Secure=true → returns nil

**Files:** `csrf_test.go`  
**Est. Time:** 10 min  
**Impact:** High — recovers coverage

---

### P1.3 Add Tests for TTL Eviction

**Coverage:** ~50% → target 100% for eviction paths  
**What to test:**

- Entry accessed within TTL → not evicted
- Entry not accessed for >TTL → evicted on next access
- Fresh entry created after eviction

**Challenge:** Time-based testing requires either clock injection or short TTL.

**Decision:** Use a short TTL (1ms) in tests, sleep 2ms, then verify eviction. This is reliable and fast.

**Files:** `ratelimit_test.go`  
**Est. Time:** 12 min  
**Impact:** High — validates correctness of fix

---

### P1.4 Add Regression Test for ResponseWriter Conflict Fix

**What to test:** `executeCSRFValidation` with `httptest.ResponseRecorder` correctly:

- Returns `ErrCSRFInvalid` on failure
- Does NOT write to the original `ResponseWriter`

**Files:** `csrf_test.go`  
**Est. Time:** 10 min  
**Impact:** Medium — prevents regression

---

## Phase 2: Security Hardening (High Impact, Medium Effort)

### P2.1 Add Body Size Limits to Decoders

**Gap:** `decodeJSONBody` and `decodeFormBody` read `r.Body` without size limits. A malicious client can send multi-GB payloads causing OOM.

**Fix:** Wrap `r.Body` with `http.MaxBytesReader` before decoding. Add `MaxBodySize int64` to `Config` with a sensible default (1MB).

**Existing code check:** `Config` struct already exists in `app.go`. We can add the field there.

**Library consideration:** None needed — `http.MaxBytesReader` is stdlib.

**Files:** `decoder.go`, `app.go`  
**Est. Time:** 10 min  
**Impact:** 🔥 High — DoS prevention

---

### P2.2 Replace Fragile `decodeFormValues` with Proper Form Decoder

**Gap:** Current implementation:

```go
func decodeFormValues(form url.Values, target any) error {
    jsonMap := make(map[string]any, len(form))
    // ... marshal to JSON, unmarshal into target
}
```

This breaks for: nested structs, time fields, custom unmarshalers, slices with single elements.

**Existing code check:** The function is already extracted to `decoder.go`. We can replace the body.

**Library consideration:** `github.com/go-playground/form/v4` is the de-facto standard for form→struct decoding in Go. It handles nesting, slices, custom types, and is actively maintained.

**Alternative:** Keep JSON round-trip but document limitation. But this is a library — consumers expect form decoding to work for reasonable struct shapes.

**Decision:** Use `github.com/go-playground/form/v4`. It's lightweight, well-tested, and purpose-built.

**Files:** `decoder.go`, `go.mod`  
**Est. Time:** 12 min  
**Impact:** High — robustness

---

## Phase 3: Type Model & API Improvements (Medium Impact, Low Effort)

### P3.1 Create Dedicated `ErrCSRFConfig` Sentinel

**Part of P0.2.** Create the sentinel in `errors.go`:

```go
var ErrCSRFConfig = errors.New("invalid CSRF configuration")
```

**Files:** `errors.go`  
**Est. Time:** 3 min  
**Impact:** Medium — error domain correctness

---

### P3.2 Add `RequestID` Type

**Gap:** Only `CorrelationID` exists. CorrelationID spans multiple requests (distributed tracing). RequestID is per-request (unique to each HTTP request).

**Type model improvement:** Mirror the `CorrelationID` pattern:

```go
type RequestID = id.RequestID  // or id.ULID if RequestID doesn't exist
func NewRequestID() RequestID
func RequestIDFromContext(ctx context.Context) RequestID
func WithRequestID(ctx context.Context, requestID RequestID) context.Context
```

**Existing code check:** `go-cqrs-lite/core/pkg/id` — check if `RequestID` exists. If not, use `id.ULID` or create a branded type locally.

**Files:** `context.go`  
**Est. Time:** 10 min  
**Impact:** Medium — observability

---

### P3.3 Enhance `ErrorHandler` with Status Code

**Gap:** `ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)` receives no status code. Every handler must re-derive it via `MapError(err)`.

**Fix:** Add status code parameter:

```go
type ErrorHandler func(w http.ResponseWriter, r *http.Request, status int, err error)
```

**Backward compatibility concern:** This is a breaking change. Alternative: keep existing signature but provide a helper:

```go
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
// New: enriched error handler with pre-computed status
type ErrorHandlerV2 func(w http.ResponseWriter, r *http.Request, status int, err error)
```

**Decision:** Provide `ErrorHandlerV2` as new type, update internal callers to use it. Keep `ErrorHandler` for backward compat.

**Files:** `errors.go`, `handler.go`, `app.go`  
**Est. Time:** 12 min  
**Impact:** Medium — API ergonomics

---

### P3.4 Add `slog` Integration to `RequestLogging`

**Gap:** `RequestLogging` writes plain strings. Modern Go uses `log/slog` (stdlib since Go 1.21) for structured logging.

**Existing code check:** `RequestLogging` already accepts `LogFormatter` and `LogWriter`. We can add an `SlogLogWriter` adapter.

**Library consideration:** `log/slog` is stdlib — no external dependency.

**Decision:** Add `SlogLogWriter(logger *slog.Logger) LogWriter` function that logs structured key-value pairs.

**Files:** `logging.go`  
**Est. Time:** 10 min  
**Impact:** Medium — modern Go idioms

---

## Phase 4: Features (Lower Impact, Higher Value for Consumers)

### P4.1 Add `MaxBodySize` to `Config`

**Part of P2.1.** Add the field so consumers can configure body size limits.

**Files:** `app.go`  
**Est. Time:** 3 min  
**Impact:** Low — but required for P2.1

---

### P4.2 Add `SecurityHeadersConfig` Builder

**Gap:** `SecurityHeadersMiddleware` is static. Consumers can't add CSP, HSTS, or custom headers.

**Existing code check:** `security.go` has a basic middleware. We can enhance it.

**Decision:** Create `SecurityHeadersConfig` struct with optional CSP, HSTS, and custom headers. `SecurityHeadersMiddleware` accepts an optional config.

**Files:** `security.go`  
**Est. Time:** 12 min  
**Impact:** Medium — consumer flexibility

---

### P4.3 Add `Timeout` Override Per Handler

**Gap:** `Config.Timeout` is global. Individual handlers can't override.

**Existing code check:** `timeoutCtx()` method on `App` uses `a.timeout`. `handlerConfig` doesn't have a timeout field.

**Decision:** Add `timeout time.Duration` to `handlerConfig`. Add `WithTimeout(d time.Duration) HandlerOption`. `timeoutCtx()` checks handler config first, falls back to app config.

**Files:** `options.go`, `app.go`, `handler.go`  
**Est. Time:** 10 min  
**Impact:** Low — consumer flexibility

---

### P4.4 Add `GzipMiddleware`

**Gap:** No response compression.

**Library consideration:** `github.com/klauspost/compress/gzhttp` is the fastest, but adds a dependency. `compress/gzip` in stdlib requires writing a custom middleware.

**Decision:** Write a simple gzip middleware using stdlib. Check `Accept-Encoding: gzip`, wrap `ResponseWriter` with `gzip.Writer`, set `Content-Encoding: gzip`.

**Files:** New `gzip.go`  
**Est. Time:** 12 min  
**Impact:** Low — performance

---

### P4.5 Add Rate Limiter Hooks

**Gap:** `RateLimiterMiddleware` has no callbacks for metrics, logging, or custom rejection responses.

**Existing code check:** `RateLimiterConfig` struct exists. We can add hook fields.

**Decision:** Add to `RateLimiterConfig`:

```go
OnAllowed    func(r *http.Request)
OnRejected   func(r *http.Request, retryAfter string)
RejectionHandler func(w http.ResponseWriter, r *http.Request, retryAfter string)
```

**Files:** `ratelimit.go`  
**Est. Time:** 10 min  
**Impact:** Low — observability

---

## Full Prioritized Table

| Phase | #   | Task                                                 | Effort (min) | Impact      | Customer Value | Why This Priority                 |
| ----- | --- | ---------------------------------------------------- | ------------ | ----------- | -------------- | --------------------------------- |
| P0    | 1   | Fix data race in `ratelimit.go`                      | 8            | 🔥 Critical | 🔥 Critical    | Correctness bug under concurrency |
| P0    | 2   | Fix `CSRFConfig.Validate()` sentinel                 | 5            | 🔥 Critical | 🔥 Critical    | Wrong error domain                |
| P1    | 3   | Tests for `RotateCSRFToken`                          | 10           | High        | High           | Coverage recovery                 |
| P1    | 4   | Tests for `CSRFConfig.Validate()`                    | 10           | High        | High           | Coverage recovery                 |
| P1    | 5   | Tests for TTL eviction                               | 12           | High        | High           | Validates race fix                |
| P1    | 6   | Regression test for ResponseWriter fix               | 10           | Medium      | Medium         | Prevents regression               |
| P2    | 7   | Body size limits on decoders                         | 10           | 🔥 High     | 🔥 High        | DoS prevention                    |
| P2    | 8   | Replace `decodeFormValues` with `go-playground/form` | 12           | High        | High           | Robustness                        |
| P3    | 9   | Create `ErrCSRFConfig` sentinel                      | 3            | Medium      | Low            | Error correctness                 |
| P3    | 10  | Add `RequestID` type                                 | 10           | Medium      | Medium         | Observability                     |
| P3    | 11  | Enhance `ErrorHandler` with status code              | 12           | Medium      | Medium         | API ergonomics                    |
| P3    | 12  | Add `slog` integration to logging                    | 10           | Medium      | Medium         | Modern idioms                     |
| P4    | 13  | `MaxBodySize` in `Config`                            | 3            | Low         | Low            | Required for #7                   |
| P4    | 14  | `SecurityHeadersConfig` builder                      | 12           | Medium      | Medium         | Consumer flexibility              |
| P4    | 15  | Per-handler timeout override                         | 10           | Low         | Low            | Consumer flexibility              |
| P4    | 16  | `GzipMiddleware`                                     | 12           | Low         | Low            | Performance                       |
| P4    | 17  | Rate limiter hooks                                   | 10           | Low         | Low            | Observability                     |

**Total estimated time:** ~159 minutes (~2.5 hours of focused work)

---

## Library Recommendations

| Gap                | Library                              | Why                                                                            |
| ------------------ | ------------------------------------ | ------------------------------------------------------------------------------ |
| Form decoding      | `github.com/go-playground/form/v4`   | De-facto standard, handles nesting/slices/custom types                         |
| Structured logging | `log/slog` (stdlib)                  | Zero dependency, modern Go, structured key-value                               |
| Compression        | `compress/gzip` (stdlib)             | Custom middleware; no dep needed for basic gzip                                |
| Request validation | `github.com/go-playground/validator` | Could add `ValidateCommand`/`ValidateQuery` tags, but overkill for current API |

**Libraries to AVOID adding:**

- `github.com/ulule/limiter` — We already have `golang.org/x/time/rate`; replacing adds churn
- `github.com/rs/zerolog` — `slog` is stdlib and sufficient
- `github.com/go-chi/chi` — Library is framework-agnostic by design

---

## What Existing Code Already Fits

| Requirement        | Existing Code                             | Gap                                      |
| ------------------ | ----------------------------------------- | ---------------------------------------- |
| Rate limiting      | `RateLimiterMiddleware` + `perKeyLimiter` | Missing hooks, had race (fixed in P0.1)  |
| Request logging    | `RequestLogging`                          | String-based; missing structured/slog    |
| Security headers   | `SecurityHeadersMiddleware`               | Static; missing builder/config           |
| Error handling     | `ErrorHandler` + `MapError`               | Missing status code pre-computation      |
| CSRF protection    | `CSRFMiddleware` + `CSRFConfig`           | Missing validation tests, wrong sentinel |
| Context enrichment | `ContextEnrichmentMiddleware`             | Missing RequestID                        |
| Response builder   | `Response` struct                         | Already fluent and comprehensive         |

---

_Plan created: 2026-05-19 16:20 CEST_  
_Next action: Execute P0.1 (fix data race)_
