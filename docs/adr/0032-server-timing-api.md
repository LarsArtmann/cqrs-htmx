# ADR-0032: Server-Timing API

**Date:** 2026-06-29
**Status:** Accepted

## Context

We need a way for developers to see per-request performance breakdowns
(dispatch, decode, auth, database) directly in their browser DevTools or
curl output, without setting up a full metrics pipeline. The [W3C
Server-Timing API](https://w3c.github.io/server-timing/) provides a
standardized HTTP response header for exactly this purpose.

### The core constraint

The `Server-Timing` header — like all HTTP headers — must be set **before**
the response body is committed (`WriteHeader`/`Write`). This means:

1. `AfterDispatchHook` **cannot** set the header — it receives no
   `http.ResponseWriter`, and the response is already committed by the
   time it runs.
2. The only reliable injection point is wrapping the
   `http.ResponseWriter` and setting the header at the **first**
   `WriteHeader`/`Write` call.

### Why not AfterDispatchHook?

`AfterDispatchHook(ctx, r, err)` was the natural place to "finalize"
metrics, but it has no access to the response writer. Even if we changed
its signature to include `w`, the response would already be committed
(headers flushed) by the time the hook runs.

## Decision

### 1. ResponseWriter wrapper (same pattern as StatusRecorder)

`serverTimingWriter` wraps `http.ResponseWriter` and injects the
`Server-Timing` header at the first `WriteHeader`/`Write` call. This is
the same pattern `StatusRecorder` uses in `logging.go` to capture the
status code.

### 2. Nil-receiver pattern (no `enabled` field)

A disabled `*ServerTiming` is simply `nil` — stored as nil in the request
context. Every method on `*ServerTiming` checks `st == nil` and returns
immediately. This eliminates:

- An `enabled bool` field scattered through every method
- An `Enabled()` check at every call site
- A separate "disabled collector" type

Handlers call `MeasureServerTiming(ctx, "db")` unconditionally. When the
middleware isn't active, the context has no collector, the nil-receiver
methods are no-ops. **Benchmark: 3.6 ns/op, 0 allocations when disabled.**

### 3. Three entry points

| Entry point                        | When to use                                                |
| ---------------------------------- | ---------------------------------------------------------- |
| `ServerTimingMiddleware()`         | Always-on (standalone middleware)                          |
| `ServerTimingMiddlewareWhen(pred)` | Debug-gated (e.g. `?debug=1`, admin role)                  |
| `Config.ServerTiming`              | 1-line App integration — wraps every `Command()`/`Query()` |

### 4. Interface preservation is critical

The wrapper delegates `Flusher`, `Hijacker`, `Pusher`, and `Unwrap()`:

- SSE streams do `w.(flusher).Flush()` (`sse_stream.go:63`)
- WebSocket upgrades need `http.Hijacker`
- HTTP/2 push needs `http.Pusher`
- `http.ResponseController` (Go 1.20+) uses `Unwrap()` to find interfaces

Losing any of these silently breaks SSE/WS.

### 5. TTFB semantics for `total`

The `total` metric is captured at flush time (time-to-first-byte), not at
request end. This is the standard Server-Timing convention — it measures
how long the server took to **start** responding.

### 6. TTFB footgun (documented, not guarded)

`defer MeasureServerTiming(ctx, "render")()` records at function return —
**after** `Write` commits the header. The metric is silently lost for
non-streaming handlers. This is documented in the `MeasureServerTiming`
godoc and tested by `TestServerTimingMiddleware_DeferredMeasureMissesHeader`.

We chose **not** to guard against this in the API because:

- The `defer` idiom IS correct for streaming handlers (SSE)
- A guard would add complexity to the hot path
- The godoc warning is sufficient for a developer-facing API

## Consequences

- Server-Timing is **opt-in** (library principle: never enforce defaults)
- Zero overhead when disabled (nil-receiver + no writer wrapping)
- Consumers compose it via `Chain()` or `Config.ServerTiming`
- The wrapper pattern can be DRY'd with a shared `delegatingWriter` if a
  third consumer emerges (rule of three)
