# Recipe: Offline Command Sync (SSE + ACK + Honest UI + SharedWorker)

This recipe shows how to wire the full offline-first command sync stack in a
real consumer application. It covers SSE reconnection, command ACK, honest UI,
idempotency, and the offline SharedWorker command queue.

## Prerequisites

- A cqrs-htmx `App` with commands registered
- An SSE endpoint (Broadcaster + SSEStream)
- HTMX loaded on the page (via `HTMXScriptHandler()` or CDN)

## Architecture

```
Browser Tab ── HTMX mutation (POST/PUT/DELETE)
     │              │
     │      ┌───────┴──────────┐
     │      │ htmx:beforeRequest│ stamps X-Command-Id, sets data-sync-state="pending"
     └──────│──────────────────┘
            │
     ┌──────┴──────┐
     │  ONLINE?    │
     ├── YES ───┤── NO ───────────────────┐
     │         │                          │
  HTMX sends   │               htmx:sendError fires
  request      │                          │
     │         │               admin.js: enqueueCommand(id)
     │         │               SharedWorker: persist to IndexedDB
     │         │               UI: data-sync-queued (amber, "offline")
     │         │                          │
     │         │               ... network returns ...
     │         │                          │
     │         │               SharedWorker: online event
     │         │               → postMessage {type:"retry", id, envelope}
     │         │               admin.js: htmx.trigger(element, "click")
     │         │                          │
     └─────┬───┘──────────────────────────┘
           │
     Server processes
     command (idempotency
     check via X-Command-Id)
           │
     ┌─────┴─────┐
     │ ACK       │
     │ confirmed │──→ SSE broadcast → admin.js: data-sync-state="confirmed"
     │ rejected  │──→ SSE broadcast → admin.js: data-sync-state="rejected"
     └───────────┘
```

## Step 1: SSE Endpoint

```go
broadcaster := cqrshtmx.NewBroadcaster()

mux.Handle("/events", sseHandler(broadcaster))

func sseHandler(bc *cqrshtmx.Broadcaster) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        stream := cqrshtmx.NewSSEStream(w, r)
        defer stream.Close()
        ch := bc.Subscribe()
        defer bc.Unsubscribe(ch)
        for {
            select {
            case <-stream.Context().Done():
                return
            case evt, ok := <-ch:
                if !ok { return }
                stream.Send(evt)
            }
        }
    })
}
```

## Step 2: ACK Middleware (idempotency + broadcast)

```go
idemStore := cqrshtmx.NewMemoryIdempotencyStore(5 * time.Minute)
defer idemStore.Close()

ackHook := broadcaster.BroadcastOnAck()

ackMW := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodPost {
            if cmdID := cqrshtmx.CommandIDFromRequest(r); cmdID != "" {
                if err := idemStore.CheckAndRecord(r.Context(), cmdID, 10*time.Minute); err != nil {
                    w.WriteHeader(http.StatusConflict)
                    return
                }
            }
        }
        next.ServeHTTP(w, r)
        // Broadcast ACK after processing
        if r.Method == http.MethodPost && cqrshtmx.CommandIDFromRequest(r) != "" {
            ackHook(r.Context(), r, nil)
        }
    })
}
```

## Step 3: Serve the sync assets (root module)

The offline sync SharedWorker and tab-side client are embedded in the root
`cqrs-htmx` module and served just like `HTMXScriptHandler()`:

```go
mux.Handle("GET /sync-worker.js", cqrshtmx.SyncWorkerHandler())
mux.Handle("GET /sync-client.js", cqrshtmx.SyncClientHandler())
```

Include the client script in your HTML after the HTMX script tag. Use the
`SyncClientScriptTag` helper or write the tag directly:

```go
// In a Go template/templ handler:
fmt.Fprint(w, cqrshtmx.SyncClientScriptTag("/sync-client.js"))
// => <script src="/sync-client.js"></script>
```

```html
<!-- Or write it directly: -->
<script src="/sync-client.js"></script>
```

The client auto-initializes on DOMContentLoaded if `<body data-sse-url>` is
present. No data-sse-url = no sync (graceful no-op).

### Serving custom sync assets

To serve a modified sync-worker.js or sync-client.js (e.g. with different
retry configuration), use the `With` variants:

```go
customWorker := []byte(/* your modified sync-worker.js */)
mux.Handle("GET /sync-worker.js",
    cqrshtmx.SyncWorkerHandlerWith(customWorker, "2.0.0"))
```

### How the worker URL is derived

The sync client automatically derives the SharedWorker URL by replacing
`sync-client.js` with `sync-worker.js` in its own `<script src>` path. If you
mount them at different paths, add a `data-sync-worker-url` attribute to the
`<script>` tag:

```html
<script src="/assets/client.js" data-sync-worker-url="/workers/sync.js"></script>
```

## Step 3a: Wire the admin panel (if using adminui)

```go
panel, _ := adminui.New(adminui.Config{
    Service: svc,
    SSEURL:  "/events",  // enables SSE + sync indicator + SharedWorker
})
```

Setting `SSEURL` activates:

- `data-sse-url` attribute on `<body>` → sync-client.js connects EventSource
- `.sync-bar` indicator in the header
- `<script src="sync-client.js">` included conditionally (adminui delegates to root handlers)
- SharedWorker registration (offline command queue)

## Step 4: Content Security Policy

The sync system requires three CSP directives when using a restrictive CSP.
`RecommendedCSP` already covers all three via `default-src 'self'`:

| Directive | Why |
|-----------|-----|
| `worker-src 'self'` | SharedWorker loaded from same origin |
| `script-src 'self'` | sync-client.js loaded via `<script>` tag |
| `connect-src 'self'` | SSE EventSource connects to same origin |

```go
cqrshtmx.SecurityHeadersConfig{
    ContentSecurityPolicy: cqrshtmx.RecommendedCSP,
    // default-src 'self' covers worker-src, script-src, connect-src
}
```

If you use a stricter CSP with explicit per-directive sources, ensure all
three directives include `'self'` (or the specific origin you serve the sync
assets from).

## How Offline Detection Works

| Event                | Fired when                                         | admin.js action                                         |
| -------------------- | -------------------------------------------------- | ------------------------------------------------------- |
| `htmx:beforeRequest` | Any HTMX request                                   | Stamps `X-Command-Id`, sets `data-sync-state="pending"` |
| `htmx:sendError`     | Network failure (offline, DNS, server unreachable) | Enqueues to SharedWorker, shows "queued — offline"      |
| `htmx:responseError` | HTTP error response (4xx, 5xx)                     | Shows "rejected" (server rejected)                      |
| SSE `sync:ack`       | Server confirms/rejects command                    | Flips `data-sync-state` to confirmed/rejected           |
| SharedWorker `retry` | Network restored, queued command retried           | `htmx.trigger(element)` re-sends request                |
| SharedWorker `dead`  | Command exceeded max retries (10) or TTL (24h)     | Shows "rejected" + "Sync failed after retries"          |

**Key distinction:** `htmx:sendError` (offline) enqueues for retry. `htmx:responseError` (server error) shows rejected. Offline ≠ rejected.

## Browser Support

| Feature           | Chrome | Firefox | Safari | Edge |
| ----------------- | ------ | ------- | ------ | ---- |
| SharedWorker      | Yes    | Yes     | 16+    | Yes  |
| EventSource (SSE) | Yes    | Yes     | Yes    | Yes  |
| Offline queue     | Yes    | Yes     | 16+    | Yes  |

Browsers without SharedWorker support gracefully degrade: the online path works
normally, offline commands fail with `htmx:sendError` and show as rejected
(honest UI — never silently dropped).

## Limitations

- **IndexedDB persistence (Phase 2b, ADR-0040):** Queued commands survive closed tabs and browser restarts via IndexedDB. Commands are evicted after 10 retries or 24 hours (TTL). Degrades to in-memory-only when IndexedDB is unavailable (private browsing, quota).
- **Element-bound retry**: If the user navigates away from the page, the queued command's DOM element is gone. The worker sends the envelope (persisted in IDB) so `rebuildAndRetry` can synthesize a new host element and re-issue the request via `htmx.ajax`.
- **Not E2E browser-tested**: The `rebuildAndRetry` cross-session path and IndexedDB persistence are unit-test-verified at the protocol level but have not been verified in a real browser (Playwright/Selenium).
