# Performance Optimization Session — cqrs-htmx

**Date:** 2026-05-24 20:09 | **Session:** Performance review + optimization pass

---

## Project Health

| Metric        | Root              | usermgmt  | Integration |
| ------------- | ----------------- | --------- | ----------- |
| Coverage      | **97.1%**         | **91.1%** | N/A         |
| Test cases    | 51                | 170       | 2           |
| Lint issues   | 0                 | 0         | 0           |
| `go vet`      | clean             | clean     | clean       |
| Race detector | clean             | clean     | clean       |
| Lines of Go   | ~9,500            | ~4,400    | ~200        |
| Benchmarks    | 16 sub-benchmarks | 0         | 0           |
| Prod files    | 17                | 9         | 1           |

**Branch:** `master` | **Commits ahead of origin:** 1 | **Total Go code:** ~14,000 lines

---

## a) FULLY DONE — This Session

### 7 Performance Optimizations Applied

| #   | Optimization                                     | File(s)           | Allocs Saved                    | Impact                                                                                               |
| --- | ------------------------------------------------ | ----------------- | ------------------------------- | ---------------------------------------------------------------------------------------------------- |
| 1   | Pre-allocated `noopCancel`                       | `app.go:230`      | 1 closure/dispatch (no timeout) | Eliminates heap alloc on every command/query dispatch when no timeout configured                     |
| 2   | `Response.WriteString` → `io.StringWriter`       | `response.go:196` | 1 `[]byte(string)`/write        | Net/http ResponseWriter implements `io.StringWriter` in real deployments — avoids string→[]byte copy |
| 3   | `ClientIP` → `strings.Cut`                       | `httputil.go:33`  | 1 `[]string`/call               | Eliminates `strings.Split` alloc when X-Forwarded-For present                                        |
| 4   | `sanitizeRedirectURL` → inline `pathEscapesRoot` | `response.go:266` | 1 `[]string`/redirect           | Eliminates `strings.Split` + removed `strings` import from response.go                               |
| 5   | `DefaultLogFormatter` inline context lookups     | `logging.go:52`   | 1 `map[string]string`/log       | Eliminates intermediate map allocation per logged request                                            |
| 6   | `applyHTMXResponse` early return                 | `options.go:316`  | 1-2 (Response + context lookup) | Skips Response struct allocation when handler has no HTMX options (most common case)                 |
| 7   | `setTriggerWithDetail` reorder                   | `response.go:303` | Minor                           | Moves `Header.Get` before `json.Marshal` to avoid marshal cost when no existing trigger              |

### Measured Benchmark Improvements (allocs/op)

| Benchmark                       | Before | After | Change  |
| ------------------------------- | ------ | ----- | ------- |
| CommandDispatch                 | 25     | 24    | **-4%** |
| QueryDispatch                   | 31     | 30    | **-3%** |
| RequestLogging/DefaultFormatter | 16     | 15    | **-6%** |
| RequestLogging/WithContextIDs   | 30     | 28    | **-7%** |

### Also Previously Completed (Earlier Sessions, All Still Green)

- **50+ improvements** from previous comprehensive session (commit `593c5a3`)
- Full security hardening: CSRF, rate limiting, security headers, redirect sanitization
- Branded types for UserID, CorrelationID, RequestID
- `usermgmt` submodule: RBAC, sessions, lockout, compensating transactions
- Integration test module with cross-module bridge tests
- Zero lint across all modules
- 97%+ root coverage, 91%+ usermgmt coverage

---

## b) PARTIALLY DONE

Nothing partially done — all 7 optimizations are complete, tested, and committed.

---

## c) NOT STARTED

- **Real-world HTTP benchmarking** — `httptest.NewRecorder` doesn't implement `io.StringWriter`, so the `WriteString` optimization won't show in synthetic benchmarks. Need a real `net/http` server benchmark to measure actual impact.
- **usermgmt benchmarks** — No benchmarks exist for the submodule (service registration, login, bcrypt, lockout, session management).
- **Memory pressure testing** — No test simulating high-concurrency rate limiter map growth/eviction patterns.
- **pprof CPU/memory profiling** — No continuous profiling setup or production profile data.
- **go-cqrs-lite dispatcher benchmarks** — The upstream dispatch accounts for ~30% of CommandDispatch latency but isn't benchmarked here.

---

## d) TOTALLY FUCKED UP!

**Nothing is broken.** All green:

- `go test ./... -count=1 -race` → PASS (root, usermgmt, integration_test)
- `golangci-lint run` → 0 issues (root, usermgmt)
- `go vet ./...` → clean
- `go build ./...` → clean

The only notable thing: benchmark numbers show high variance on this machine (AMD RYZEN AI MAX+ 395), likely due to turbo boost and thermal throttling. The `allocs/op` numbers are stable and reliable; the `ns/op` numbers should be taken with a grain of salt.

---

## e) WHAT WE SHOULD IMPROVE!

### Architecture

1. **`decodeFormBody` JSON round-trip** — Form values → `map[string]any` → `json.Marshal` → `json.Unmarshal` is 2 allocations minimum. A schema-aware form decoder (like `gorilla/schema` or a custom reflect-based decoder) would eliminate both the intermediate map and the marshal/unmarshal pair. This is the **single biggest remaining allocation hot path**.
2. **`setTriggerWithDetail` still allocates** — `json.Marshal(map[string]any{name: detail})` allocates a map per call. Could use a pooled encoder or pre-allocated buffer for simple cases.
3. **`HealthHandler` allocates** — `[]byte(`{"status":"ok"}`)` is allocated per request. Should be a pre-allocated constant.
4. **`JSONLogFormatter` allocates** — `json.Marshal(map[string]any{...})` per log line. Could use `json.Encoder` with pooled buffer.
5. **Error handler response bodies** — `[]byte(err.Error())` allocates per error. For hot paths, could write directly to ResponseWriter.
6. **`RateLimiter` map-of-struct-pointers** — `perKeyLimiter.limiters` stores `*limiterEntry` (heap alloc per key). Could use a value-based map with `rate.Limiter` embedded directly, though `rate.Limiter` isn't move-safe.

### Testing & Observability

7. **No usermgmt benchmarks** — bcrypt cost, session store operations, lockout checks are all unmeasured.
8. **No pprof integration** — No `net/http/pprof` handlers in examples or test utilities.
9. **No memory pressure tests** — Rate limiter under 10K+ concurrent keys is untested.
10. **No real-server benchmarks** — All benchmarks use `httptest.NewRecorder` which doesn't implement `io.StringWriter` or `http.Flusher` or `http.Pusher`, hiding real-world allocation behavior.

### Code Quality

11. **`contextFields()` in logging.go is now only used by `RequestLoggingSlog`** — Could inline it there too to eliminate the map allocation for slog logging as well.
12. **`splitPath` was removed** — Good, but the new `pathEscapesRoot` function has its own cyclomatic complexity. Extracted to a separate function to keep `sanitizeRedirectURL` under 12, but the logic is still dense.

---

## f) Top #25 Things We Should Get Done Next

### Performance (High Impact)

| #   | Task                                                                                | Impact                                            | Effort  |
| --- | ----------------------------------------------------------------------------------- | ------------------------------------------------- | ------- |
| 1   | Replace form decode JSON round-trip with `gorilla/schema` or custom reflect decoder | **Very High** (eliminates 2 allocs per form POST) | Medium  |
| 2   | Pre-allocate `HealthHandler` response body as package-level `[]byte`                | Medium                                            | Trivial |
| 3   | Inline `contextFields()` in `RequestLoggingSlog` to eliminate map alloc             | Medium                                            | Low     |
| 4   | Add `sync.Pool` for JSON encoding buffers in `JSONLogFormatter`                     | Medium                                            | Low     |
| 5   | Add real `net/http` server benchmarks (not just `httptest`)                         | High (visibility)                                 | Low     |
| 6   | Add usermgmt benchmarks (bcrypt, session store, lockout)                            | High (visibility)                                 | Low     |
| 7   | Pool `map[string]any` in `setTriggerWithDetail` for common case                     | Low-Medium                                        | Low     |

### Testing & Coverage

| #   | Task                                                             | Impact | Effort |
| --- | ---------------------------------------------------------------- | ------ | ------ |
| 8   | Add memory pressure test for rate limiter (10K+ keys)            | Medium | Medium |
| 9   | Add integration benchmark for full middleware chain              | Medium | Low    |
| 10  | Add fuzz targets for `sanitizeRedirectURL` edge cases            | Medium | Low    |
| 11  | Cover remaining uncovered branches in root module (2.9% to 100%) | Low    | Medium |
| 12  | Cover remaining uncovered branches in usermgmt (8.9% to 100%)    | Low    | Medium |

### Developer Experience

| #   | Task                                                                          | Impact | Effort  |
| --- | ----------------------------------------------------------------------------- | ------ | ------- |
| 13  | Add `pprof` handler to examples for production profiling                      | Medium | Trivial |
| 14  | Add example app with all middleware wired (CSRF → HTMX → Context → Security)  | High   | Medium  |
| 15  | Add README benchmark table with ns/op and allocs/op                           | Medium | Low     |
| 16  | Document performance characteristics in godoc (allocs per call site)          | Medium | Low     |
| 17  | Add `Response.HTMXHeaders()` method to introspect current HTMX response state | Low    | Low     |

### Architecture & Correctness

| #   | Task                                                                                           | Impact               | Effort  |
| --- | ---------------------------------------------------------------------------------------------- | -------------------- | ------- |
| 18  | Extract `decodeFormBody` to use `encoding` package interface                                   | High (extensibility) | Medium  |
| 19  | Add OpenTelemetry span creation in lifecycle hooks example                                     | Medium               | Low     |
| 20  | Review `enrichUserID` — could combine `slog.Warn` into structured logging with request context | Low                  | Trivial |
| 21  | Add `Response.Clone()` for testing patterns where response is inspected                        | Low                  | Low     |

### Maintenance

| #   | Task                                                                         | Impact | Effort  |
| --- | ---------------------------------------------------------------------------- | ------ | ------- |
| 22  | Update `FEATURES.md` metrics (coverage is now 97.1%/91.1%, not 95.9%/92.1%)  | Low    | Trivial |
| 23  | Update `AGENTS.md` with noopCancel, io.StringWriter, pathEscapesRoot gotchas | Low    | Low     |
| 24  | Remove or archive stale `docs/status/` reports (29 files)                    | Low    | Trivial |
| 25  | Add `just` or `task` runner for common benchmark commands                    | Low    | Trivial |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should we replace the form-decode JSON round-trip (`decodeFormValues` in `decoder.go`) with `gorilla/schema` or a custom reflect-based decoder?**

This is the **single biggest remaining allocation hot path** — every form POST goes through `map[string]any` → `json.Marshal` → `json.Unmarshal`. But:

- `gorilla/schema` is archived/sunset (no longer maintained)
- Custom reflect decoder is more code to maintain
- The JSON round-trip handles edge cases (arrays, nested objects) that form decoders typically don't
- Consumer impact: anyone relying on the JSON round-trip behavior might see subtle differences

**I cannot decide the tradeoff between allocation savings vs. maintenance burden vs. behavioral compatibility without knowing your stance on external dependencies and backward compatibility.**
