# cqrs-htmx — SDK Feedback from Overview

**Consumer:** [Overview](https://github.com/larsartmann/overview) — local project dashboard (read-only HTMX + SSE, no CQRS/auth)
**Date:** 2026-07-05
**Version used:** v4.1.1
**Session:** SSE-driven discovery, embedded htmx.js, middleware chain, SecurityHeaders, Broadcaster

---

## What worked superbly

### 1. `HTMXScriptHandler()` + `HTMXExtensionHandler()` — embedded assets

This is the standout feature for this session. Overview was loading htmx from CDN; switching to `cqrshtmx.HTMXScriptHandler()` for htmx 2.0.10 and `cqrshtmx.HTMXExtensionHandler(cqrshtmx.HTMXExtSSE)` for the SSE extension eliminated two CDN dependencies in one shot. Both set `ETag`, `Cache-Control: 1yr immutable`, and handle 304s. Zero config, correct caching. This is how embedded assets should work.

### 2. `Chain()` middleware composition

`cqrshtmx.Chain(mw1, mw2, ...)` with outer-first ordering is the right abstraction. Nine middlewares composed in one call, readable top-to-bottom. No surprises.

### 3. `NewSSEStream(w, r)` — correct SSE wire format

Handles `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`, and context cancellation. The `stream.Send(SSEEvent{...})` API is clean. The `stream.Context()` for connection-lifecycle detection works correctly with client disconnects.

### 4. `Broadcaster` — simple fan-out

`NewBroadcaster()` → `Subscribe()` → `Broadcast(event)` → `Unsubscribe(ch)`. Three methods, correct semantics. Used for `discovery-complete` fan-out: when `DiscoveryCache.storeCacheSnapshot()` finishes, one `Broadcast()` call notifies all connected SSE clients instantly. Replaced 2-second polling.

### 5. `IsHTMXRequest(r)` and `RenderPartial(r)`

Correct abstractions. `IsHTMXRequest` for "is this an HTMX request at all" (loading skeleton vs full page), `RenderPartial` for "should I render just the partial" (filters HTMX history-restore requests). The distinction matters and is handled correctly.

### 6. `RecoveryMiddleware` + `RequestLoggingSlog(logger)`

Drop-in, works with `*slog.Logger`. The request logging emits structured attrs (method, path, duration, request_id) that integrate with our `charm.land/log/v2` handler. No config needed.

### 7. `RequestIDFromContext(ctx)` — ULID-based request IDs

`ContextEnrichmentMiddleware(nil)` auto-generates ULID request IDs and sets `X-Request-ID`. The `nil` argument means "use default ULID generator." Correlates logs across the middleware chain.

---

## Pain points

### 1. `SecurityHeadersConfig` — cannot suppress `withDefault` headers

**Severity:** Medium (boilerplate)

The three `withDefault` headers (`ContentTypeOptions`, `FrameOptions`, `ReferrerPolicy`) fall back to secure defaults when set to `""`. There is **no way to suppress them entirely** — empty string means "use the default," not "skip this header."

This means our config looks like this:

```go
cqrshtmx.SecurityHeadersMiddlewareWithConfig(cqrshtmx.SecurityHeadersConfig{
    ContentTypeOptions:      "", // → still sets "nosniff"
    FrameOptions:            "", // → still sets "DENY"
    ReferrerPolicy:          "", // → still sets "strict-origin-when-cross-origin"
    ContentSecurityPolicy:   contentSecurityPolicy,
    StrictTransportSecurity: "", // → skipped (guarded by != "")
    PermissionsPolicy:       "", // → skipped (guarded by != "")
    Custom:                  nil,
})
```

Every field must be explicitly listed just to say "only set CSP." The inconsistency (`withDefault` vs guarded) makes this worse — you can't predict which fields are skippable without reading source.

**Suggestion:** Add a sentinel value (e.g., `SecurityHeaderSkip = "-"` matching the header convention) or change the three `withDefault` fields to also be guarded by `!= ""`, documenting the secure defaults in `SecurityHeadersMiddleware` (the zero-config variant).

### 2. SSE handler boilerplate — `Subscribe`/`Unsubscribe`/`for-select` is verbose

**Severity:** Low-medium (ergonomic)

The canonical SSE handler pattern is ~30 lines of boilerplate:

```go
func (s *Server) sseHandler(w http.ResponseWriter, r *http.Request) {
    stream := cqrshtmx.NewSSEStream(w, r)
    defer stream.Close()
    broadcaster := s.cache.Broadcaster()
    eventsCh := broadcaster.Subscribe()
    defer broadcaster.Unsubscribe(eventsCh)
    // send "connected" event
    _ = stream.Send(cqrshtmx.SSEEvent{Event: "connected", Data: "ok", ID: cqrshtmx.NewSSEEventID("")})
    heartbeat := time.NewTicker(30 * time.Second)
    defer heartbeat.Stop()
    for {
        select {
        case <-stream.Context().Done():
            return
        case event := <-eventsCh:
            if stream.Send(event) != nil { return }
        case <-heartbeat.C:
            if stream.Send(cqrshtmx.SSEEvent{...}) != nil { return }
        }
    }
}
```

This pattern (subscribe, defer unsubscribe, for-select with context/heartbeat) is identical across every consumer. A higher-level helper would reduce this:

```go
// Suggestion: a callback or handler-func based API
broadcaster.ServeSSE(w, r, cqrshtmx.SSEHandlerConfig{
    OnConnect: func(stream *cqrshtmx.SSEStream) { stream.Send(...) },
    Heartbeat: 30 * time.Second,
})
```

### 3. `NewSSEEventID("")` — empty string = auto-generate is non-obvious

**Severity:** Low (discoverability)

Every SSE event needs an ID, and `NewSSEEventID("")` auto-generates a ULID. This is a common Go pattern but the empty-string sentinel isn't documented in the function signature or name. A `NewSSEEventIDAuto()` or making the field auto-generate when zero would be clearer.

### 4. `ContextEnrichmentMiddleware(nil)` — nil = auto-generate is under-documented

**Severity:** Low (discoverability)

`nil` means "use the default ULID generator." This works but isn't obvious from the signature `ContextEnrichmentMiddleware(idgen func() string)`. A zero-arg variant (`ContextEnrichmentMiddlewareAuto()`) or a package-level default would be more discoverable.

### 5. `RenderPartial` vs `IsHTMXRequest` — naming doesn't convey the distinction

**Severity:** Low (naming)

`IsHTMXRequest` checks `HX-Request: true`. `RenderPartial` checks `HX-Request: true AND HX-History-Restore-Request != true`. The names don't communicate this — `RenderPartial` sounds like a rendering decision, not a request-classification function. `IsHTMXNavigation` or `IsHTMXPartialRequest` would be clearer.

### 6. No exported SSE event name constants

**Severity:** Low (ergonomic)

`connected`, `heartbeat`, `ping`, `ok` — every consumer defines these strings themselves. Exporting `SSEEventConnected = "connected"`, `SSEEventHeartbeat = "heartbeat"` etc. would standardize the vocabulary and reduce magic strings.

---

## Summary

The root module is **excellent for read-only HTMX apps** — you don't need CQRS, auth, or usermgmt to get tremendous value from the middleware chain, embedded assets, SSE building blocks, and Broadcaster.

The biggest wins this session:

- Embedded htmx.js + SSE extension eliminated CDN dependencies
- Broadcaster + SSEStream enabled instant discovery-complete notifications (replaced 2s polling)
- Chain() middleware composition is clean and correct

The main gap is SSE handler boilerplate — the subscribe/unsubscribe/for-select pattern is identical everywhere and ripe for a higher-level helper.

---

## Resolution Status (2026-07-05)

### Pain Points — Resolutions

| # | Suggestion                                                                   | Status         | Notes                                                                                                                                                                                                                                                                 |
| - | ---------------------------------------------------------------------------- | -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | `SecurityHeadersConfig` — cannot suppress `withDefault` headers              | **DONE**       | `SecurityHeaderSkip` sentinel (`"-"`) added. Set any of `ContentTypeOptions`/`FrameOptions`/`ReferrerPolicy` to `"-"` to suppress. Tested (3 tests). Documented in SKILL.md + core-api.md                                                                             |
| 2 | SSE handler boilerplate — add higher-level helper                            | **NOT DONE**   | Deferred — design decision. The library's philosophy is "building blocks, not a server." Two consumers asked for it; the maintainer needs to decide if `broadcaster.ServeSSE()` crosses the abstraction line. Documented the canonical pattern in realtime.md instead |
| 3 | `NewSSEEventID("")` — empty string = auto-generate is non-obvious            | **DOCUMENTED** | Added to SKILL.md discoverability notes: "empty string auto-generates a ULID"                                                                                                                                                                                         |
| 4 | `ContextEnrichmentMiddleware(nil)` — nil = auto-generate is under-documented | **DOCUMENTED** | Added to SKILL.md discoverability notes: "nil extractor = auto-generate ULID request IDs"                                                                                                                                                                             |
| 5 | `RenderPartial` vs `IsHTMXRequest` — naming doesn't convey distinction       | **DOCUMENTED** | Added to SKILL.md discoverability notes explaining the difference: `IsHTMXRequest` = any HTMX request; `RenderPartial` = HTMX navigation excluding history-restore                                                                                                    |
| 6 | No exported SSE event name constants                                         | **DONE**       | `SSEEventConnected` (`"connected"`) and `SSEEventHeartbeat` (`"heartbeat"`) added to `sse_event.go`. Documented in SKILL.md + realtime.md + core-api.md                                                                                                               |

### Summary Update

The main gap (SSE handler boilerplate) remains open as a design question. All other pain points have been addressed. The `SecurityHeaderSkip` sentinel directly solves the "cannot suppress headers" complaint, and the SSE event name constants reduce the magic strings this consumer noted.
