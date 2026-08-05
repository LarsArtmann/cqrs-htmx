# Realtime reference (SSE)

Import: `cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"` (and `sse "github.com/larsartmann/go-sse"` for the stream/event types). Realtime is **building blocks, not a server** — you own the HTTP handler, the library gives you the stream, fan-out, and the CQRS bridge.

> WebSocket transport was removed in v5 (see ADR-0046); the library is SSE-only.
> SSE covers every realtime use case with strictly better operational properties:
> plain HTTP, native auth/CSRF/cookies, proxy/CDN-friendly, and built-in
> auto-reconnect via `Last-Event-ID`.

## SSE — the happy path

```go
broadcaster := cqrshtmx.NewBroadcaster()

// Wire an AfterDispatchHook so successful commands broadcast:
app := cqrshtmx.MustNew(cqrshtmx.Config{
    Commands:     cmdDisp,
    AfterDispatch: broadcaster.BroadcastOnSuccess("itemCreated", data),
    // or dynamic: broadcaster.BroadcastOnSuccessFunc(func(r) cqrshtmx.SSEEvent { ... })
})

// Your own SSE endpoint:
mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
    stream := cqrshtmx.NewSSEStream(w, r)
    defer stream.Close()
    ch := broadcaster.Subscribe()
    defer broadcaster.Unsubscribe(ch)
    for {
        select {
        case <-stream.Context().Done():
            return
        case evt, ok := <-ch:
            if !ok || stream.Send(evt) != nil { return }
        }
    }
})
```

`SSEStream` sets the correct headers (`Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`) and flushes after each send.

## `SSEStream` API

```go
stream := cqrshtmx.NewSSEStream(w, r)
stream.Send(cqrshtmx.SSEEvent{Event: "x", Data: "...", ID: "...", Retry: "3000"})
stream.SendHTML("itemUpdated", "<div>...</div>") // convenience
stream.LastEventID()    // SSEEventID — from the Last-Event-ID header (reconnection cursor)
stream.Context()        // cancelled when client disconnects
stream.Heartbeat(ctx, 30*time.Second) // comment-frame pings; prevents proxy/LB idle disconnects
stream.OnDisconnect(fn func())        // register cleanup callbacks (fired on Close)
stream.Close()
```

`sse.Event` (re-exported as the deprecated alias `cqrshtmx.SSEEvent`) has this shape:

```go
type Event struct {
    Event string // event name; empty = default message event
    Data  string // multi-line data is auto-split per SSE spec; CRLF normalized to LF
    ID    string // event id; must not contain newlines (branded SSEEventID rejects them)
    Retry string // reconnection hint in ms
}
```

## `Broadcaster` API (SSE fan-out)

```go
b := cqrshtmx.NewBroadcaster()
b.Subscribe()             // → chan SSEEvent (buffered, 64 capacity)
b.Unsubscribe(ch)         // O(1) via channel pointer identity; closes the channel
b.Broadcast(evt)          // non-blocking; slow consumers have events dropped
b.SubscriberCount() int   // current subscriber count (useful for metrics)
b.Close()                 // graceful shutdown: closes all channels, blocks new subscriptions
```

Thread-safe. Non-blocking broadcast means a slow client won't block the publisher — events are dropped if the channel is full. Subscribe/Unsubscribe are safe for concurrent use. **Unsubscribe closes the channel** — callers must not send on it. **Close** closes all subscriber channels at once; after Close, Subscribe returns an already-closed channel. Call Close on server shutdown for graceful SSE drain.

### Standard event names

```go
cqrshtmx.SSEEventConnected   // "connected"
cqrshtmx.SSEEventHeartbeat   // "heartbeat"
```

### Event filtering (consumer concern)

The library intentionally provides no filtering primitives — filtering is domain-specific. The recommended pattern:

```go
// 1. Parse filter from query params
type sseFilter struct {
    EventType string
    ChannelID string
}
func parseSSEFilter(r *http.Request) sseFilter { /* parse ?event_type= etc. */ }

// 2. Apply filter during both replay and live phases
func (f sseFilter) matches(evt cqrshtmx.SSEEvent) bool {
    if f.EventType != "" && evt.Event != f.EventType { return false }
    // ... domain-specific matching
    return true
}

mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
    filter := parseSSEFilter(r)
    stream := cqrshtmx.NewSSEStream(w, r)
    defer stream.Close()
    // Replay filtered:
    cqrshtmx.ReplayEvents(stream, sseStore, lastID) // filter in your mapper
    // Live filtered:
    ch := broadcaster.Subscribe()
    defer broadcaster.Unsubscribe(ch)
    for {
        select {
        case <-stream.Context().Done(): return
        case evt, ok := <-ch:
            if !ok || !filter.matches(evt) || stream.Send(evt) != nil { return }
        }
    }
})
```

### Hook factories (bridge CQRS → SSE)

```go
broadcaster.BroadcastOnSuccess(eventName, data) AfterDispatchHook         // fixed event
broadcaster.BroadcastOnSuccessFunc(func(r) SSEEvent) AfterDispatchHook     // dynamic
broadcaster.BroadcastOnError(eventName) AfterDispatchHook                  // StructuredError on failure
broadcaster.BroadcastOnErrorFunc(func(r, err) SSEEvent) AfterDispatchHook
```

Pass these to `cqrshtmx.Config.AfterDispatch`.

## ACK protocol (honest UI sync)

For "optimistic UI" that shows _pending_ state and then reconciles with the server. The client sends `X-Command-Id: <client-id>` with a mutation; the server broadcasts an ACK when the command completes.

```go
type CommandAck struct {
    CommandID string    `json:"commandId"`
    Status    AckStatus `json:"status"`            // "confirmed" | "rejected"
    Error     string    `json:"error,omitempty"`   // populated when rejected
}
```

Wire it as an `AfterDispatchHook` (opt-in — only fires when the request carries `X-Command-Id`):

```go
app := cqrshtmx.MustNew(cqrshtmx.Config{
    Commands:     cmdDisp,
    AfterDispatch: broadcaster.BroadcastOnAck(),
})
```

The client listens for the `sync:ack` SSE event and matches on `commandId`. See `docs/adr/0024-honest-ui.md`. The frontend side is demonstrated in `adminui/assets/admin.js`.

## Idempotency (pair with ACK for offline safety)

ACK tells the client a command finished; idempotency prevents the _same_ command from executing twice on reconnect/retry. **Not auto-wired** — opt in via a `BeforeDispatchHook` or middleware.

```go
store := cqrshtmx.NewMemoryIdempotencyStore(5 * time.Minute) // sweep interval
defer store.Close()

// On mutation requests, check + record the command ID:
if cmdID := cqrshtmx.CommandIDFromRequest(r); cmdID != "" {
    if err := store.CheckAndRecord(r.Context(), cmdID, 10*time.Minute); err != nil {
        // cqrshtmx.ErrDuplicateCommand — maps to HTTP 409 Conflict
        http.Error(w, err.Error(), http.StatusConflict)
        return
    }
}
```

`cqrshtmx.IdempotencyStore` / `MemoryIdempotencyStore` / `ErrDuplicateCommand` are thin aliases over `go-cqrs-lite/idempotency/v3`. For multi-instance deployments implement the same interface against Redis (`SET NX`) or SQL (`INSERT ON CONFLICT`). See `docs/adr/0026-command-idempotency-store.md`.

## Reconnection / replay

SSE clients send `Last-Event-ID` when reconnecting. Replay the events they missed:

```go
mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
    stream := cqrshtmx.NewSSEStream(w, r)
    defer stream.Close()
    lastID := cqrshtmx.LastEventIDFromRequest(r)         // branded SSEEventID

    // Replay missed events first, then live:
    if n, err := cqrshtmx.ReplayEvents(stream, sseStore, lastID); err != nil { return }
    _ = n

    ch := broadcaster.Subscribe()
    defer broadcaster.Unsubscribe(ch)
    for {
        select {
        case <-stream.Context().Done(): return
        case evt, ok := <-ch:
            if !ok || stream.Send(evt) != nil { return }
        }
    }
})
```

`SSEEventStore` interface:

```go
type SSEEventStore interface {
    EventsAfter(lastID string) []SSEEvent // ordered ascending by ID; empty if none/unknown
}
```

### Production store: `JournalSSEStore`

Backs replay with a `go-cqrs-lite/event.Journal`. Uses `ReadFrom(afterEventID, limit)` when the journal is `event.SeekableJournal` (efficient cursor replay); falls back to `ReadAll` + in-memory filter otherwise.

```go
type EventToSSEMapper func(evt event.Event) SSEEvent

store := cqrshtmx.NewJournalSSEStore(
    journal,            // event.Journal (and ideally event.SeekableJournal)
    mapper,             // convert domain events → SSEEvent
    cqrshtmx.WithMaxReplay(1000), // cap initial sync; 0 = unlimited, default 1000
)
```

Consumer provides the mapper (the library can't render consumer domain payloads). See `docs/adr/0023-command-sync.md`.

## `StructuredError` (RFC 7807 shape)

Used by `BroadcastOnError` to carry a typed error payload to SSE clients.

```go
payload := cqrshtmx.NewStructuredError(err, r)   // maps via MapError + extracts request ID
payload := cqrshtmx.NewStructuredErrorWithContext(err, ctx)
payload.JSON()                                   // RFC 7807-shaped JSON string
```

## When to use SSE

SSE is the only realtime transport (since v5). It covers every server→client update pattern: live lists, notifications, ACKs, projection health. Client→server mutations go through the normal HTTP command/query pipeline (`app.Command`/`app.Query` + `DecodeJSON`), which already handles auth, CSRF, and content negotiation. See `docs/adr/0046-drop-websocket-sse-only.md`.

## Gotchas

- `Broadcaster.Unsubscribe` **closes the channel**. Never send on a channel after unsubscribing. The read loop pattern above (`defer Unsubscribe(ch)`) is the safe shape.
- Broadcasts are **non-blocking** — slow consumers silently drop events. If you need guaranteed delivery, use the idempotency store + ACK so the client can retry on reconnect.
- The ACK `X-Command-Id` header is **opt-in**. If the client doesn't send it, no ACK is broadcast and idempotency checking is a no-op. The frontend must generate and attach the ID for every mutation.
- For replay to work, your events must have **monotonic IDs** the `SSEEventStore` can order by. `JournalSSEStore` uses ULID event IDs from the journal.
