# Recipe: Offline Command Sync (SSE + ACK + Honest UI + SharedWorker)

This recipe shows how to wire the full offline-first command sync stack in a
real consumer application. It covers SSE reconnection, command ACK, honest UI,
idempotency, and the offline SharedWorker command queue.

## Prerequisites

- A cqrs-htmx `App` with commands registered
- An SSE endpoint (Broadcaster + SSEStream)
- The adminui panel (or your own HTMX frontend with `admin.js`)

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
     │         │               SharedWorker: queue [{id, port}]
     │         │               UI: data-sync-queued (amber, "offline")
     │         │                          │
     │         │               ... network returns ...
     │         │                          │
     │         │               SharedWorker: online event
     │         │               → postMessage {type:"retry", id}
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

## Step 3: Wire the admin panel

```go
panel, _ := adminui.New(adminui.Config{
    Service: svc,
    SSEURL:  "/events",  // enables SSE + sync indicator + SharedWorker
})
```

Setting `SSEURL` activates:
- `data-sse-url` attribute on `<body>` → admin.js connects EventSource
- `.sync-bar` indicator in the header
- `sync-worker.js` registration (offline command queue)

## Step 4: CSP (if using SecurityHeadersMiddleware)

SharedWorker scripts from the same origin are covered by `default-src 'self'`
or `worker-src 'self'`. No additional CSP directives needed.

```go
cqrshtmx.SecurityHeadersConfig{
    ContentSecurityPolicy: cqrshtmx.RecommendedCSP,
    // default-src 'self' covers worker-src
}
```

## How Offline Detection Works

| Event | Fired when | admin.js action |
|-------|-----------|-----------------|
| `htmx:beforeRequest` | Any HTMX request | Stamps `X-Command-Id`, sets `data-sync-state="pending"` |
| `htmx:sendError` | Network failure (offline, DNS, server unreachable) | Enqueues to SharedWorker, shows "queued — offline" |
| `htmx:responseError` | HTTP error response (4xx, 5xx) | Shows "rejected" (server rejected) |
| SSE `sync:ack` | Server confirms/rejects command | Flips `data-sync-state` to confirmed/rejected |
| SharedWorker `retry` | Network restored, queued command retried | `htmx.trigger(element)` re-sends request |

**Key distinction:** `htmx:sendError` (offline) enqueues for retry. `htmx:responseError` (server error) shows rejected. Offline ≠ rejected.

## Browser Support

| Feature | Chrome | Firefox | Safari | Edge |
|---------|--------|---------|--------|------|
| SharedWorker | Yes | Yes | 16+ | Yes |
| EventSource (SSE) | Yes | Yes | Yes | Yes |
| Offline queue | Yes | Yes | 16+ | Yes |

Browsers without SharedWorker support gracefully degrade: the online path works
normally, offline commands fail with `htmx:sendError` and show as rejected
(honest UI — never silently dropped).

## Limitations (Phase 2a)

- **Commands lost on last-tab-close**: The SharedWorker dies when all tabs close. This is the Queue-Only contract (ADR-0027). Phase 2b (Service Worker + Background Sync) can extend this if needed.
- **Element-bound retry**: If the user navigates away from the page, the queued command's DOM element is gone. The retry shows "rejected (element not found)" — honest, not silent.
- **No persistence**: In-memory only. IndexedDB is banned (ADR-0029). OPFS is deferred to Phase 2b.
