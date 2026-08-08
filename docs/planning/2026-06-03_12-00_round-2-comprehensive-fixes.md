# Round 2 — Comprehensive Fixes Plan

**Date:** 2026-06-03
**Status:** Post Phase 1+2 execution. Lint clean. Tests pass. Coverage: 96.3% root, 89.8% usermgmt.

---

## What I Forgot (Last Session)

1. **Lint issues from my own changes** — exhaustruct + gofumpt (FIXED in cb72e8b)
2. **Coverage gaps in new code** — nil-enforcer paths in Authz not fully exercised
3. **RequestLoggingSlog completely untested** — entire middleware not covered
4. **Response.Body, WriteString non-StringWriter, JSON error paths** — untested
5. **PtrBool still exists** — gopls flags it everywhere

## What Could Be Better

### Type Safety

- `Enforcer` interface uses `...any` — perpetuates untyped Casbin boundary
- `EventHandler` uses `event any` — complete type erasure
- `Response.JSON(v any)` — could be generic
- `WriteJSON(w, status, v any)` — could be generic

### Architecture

- `EnforceAny` + `enforcerAdapter` + `AsEnforcer` — triad of unnecessary adapters
- `PtrBool` — trivial wrapper, `new(bool)` is idiomatic
- `ClientIP` — dead weight re-export of `httputil.ClientIP`
- `RequestLogging` and `RequestLoggingSlog` — two separate middlewares instead of one

### Coverage

- `UpdateRoles` error paths (authz.Apply and users.Save failures)
- `readBody` error paths (read error + close error)
- `Response` body methods (Body, WriteString non-StringWriter, JSON marshal error)
- `RequestLoggingSlog` (entire middleware)
- `DefaultErrorHandlerWithRequestID` and variants

---

## Execution Plan (Sorted by Impact/Effort)

### Phase 1: Quick Fixes (High Impact, Low Effort)

| # | Task                                             | Impact | Effort | File(s)                       |
| - | ------------------------------------------------ | ------ | ------ | ----------------------------- |
| 1 | Remove PtrBool, use new(bool)                    | Low    | 10m    | usermgmt/http.go, \*\_test.go |
| 2 | Remove ClientIP re-export                        | Low    | 5m     | httputil.go                   |
| 3 | Remove EnforceAny + enforcerAdapter + AsEnforcer | Medium | 20m    | authz.go, authz_test.go       |
| 4 | Add UpdateRoles error path tests                 | Medium | 15m    | usermgmt/service_test.go      |
| 5 | Add readBody error path tests                    | Low    | 10m    | decoder_test.go               |

### Phase 2: Coverage Gaps (Medium Impact, Medium Effort)

| #  | Task                                           | Impact | Effort | File(s)          |
| -- | ---------------------------------------------- | ------ | ------ | ---------------- |
| 6  | Add Response.Body test                         | Low    | 5m     | coverage_test.go |
| 7  | Add Response.WriteString non-StringWriter test | Low    | 10m    | coverage_test.go |
| 8  | Add Response.JSON marshal error test           | Low    | 10m    | coverage_test.go |
| 9  | Add RequestLoggingSlog tests                   | Medium | 20m    | logging_test.go  |
| 10 | Add error handler variant tests                | Low    | 15m    | errors_test.go   |

### Phase 3: Architecture Improvements (High Impact, Medium Effort)

| #  | Task                                     | Impact | Effort | File(s)     |
| -- | ---------------------------------------- | ------ | ------ | ----------- |
| 11 | Dedupe RequestLogging/RequestLoggingSlog | Medium | 30m    | logging.go  |
| 12 | Add typed variant of Response.JSON[T]    | Medium | 15m    | response.go |
| 13 | Add typed variant of WriteJSON[T]        | Medium | 15m    | httputil.go |
| 14 | Use samber/lo for slice operations       | Low    | 20m    | Various     |

---

## Type Model Improvements

### Current

```go
Enforcer interface{ Enforce(rvals ...any) (bool, error) }
EventHandler func(userID UserID, event any)
Response.JSON(v any)
WriteJSON(w http.ResponseWriter, status int, v any)
```

### Target

```go
Enforcer interface{ Enforce(rvals ...any) (bool, error) } // keep for Casbin compat
EventHandler[T any] func(userID UserID, event T) // typed events
Response.JSON(v any) // keep for backward compat
Response.JSONTyped[T any](v T) // new generic variant
WriteJSON(w http.ResponseWriter, status int, v any) // keep
WriteJSONTyped[T any](w http.ResponseWriter, status int, v T) // new
```

---

## Library Leverage

| Library                | Current Use                     | Better Use                             |
| ---------------------- | ------------------------------- | -------------------------------------- |
| samber/lo              | indirect dep                    | Replace manual slice ops, deep copies  |
| samber/ro              | indirect dep (via go-cqrs-lite) | Reactive streams, observable patterns  |
| golang.org/x/time/rate | Custom map+heap                 | Could use sync.Map for less contention |
| justinas/nosurf        | Fragile CSRF                    | Consider double-submit cookie instead  |

---

## Questions

1. **Should we drop EnforceAny/AsEnforcer entirely?** The `Enforcer` interface in root matches `*casbin.Enforcer` directly. `AsEnforcer()` is only needed to bridge usermgmt's `Authz` to the root `Enforcer` interface. Can we make `Authz` satisfy `Enforcer` directly instead?

2. **Should we remove Ginkgo from root tests?** It's a heavy dependency for a library. Standard `testing` is more appropriate.
