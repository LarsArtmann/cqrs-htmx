# Realtime reference (SSE + WebSocket)

Import: `cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"`. Realtime is **building blocks, not a server** — you own the HTTP/WS handler, the library gives you the stream, fan-out, and the CQRS bridge.

There is **no WebSocket library dependency** — the library provides protocol helpers only. You choose your WS library (e.g. `nhooyr.io/websocket`, `gorilla/websocket`) and do the upgrade yourself.

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

`SSEEvent` is a type alias for `go-cqrs-lite/transport/http/v3.SSEEvent`:

```go
type SSEEvent struct {
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
b.SubscriberCount() int
```

Thread-safe. Non-blocking broadcast means a slow client won't block the publisher — events are dropped if the channel is full. Subscribe/Unsubscribe are safe for concurrent use. **Unsubscribe closes the channel** — callers must not send on it.

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

## WebSocket

WebSockets mirror the SSE pattern but with a string-message fan-out and OOB HTML helpers.

### Inbound parsing (HTMX WebSocket JSON format)

```go
// Generic typed parse — separates HEADERS from body fields:
msg, headers, err := cqrshtmx.ParseWSMessageInto[CreateTaskInput](data)

// Or the untyped form:
wsMsg, err := cqrshtmx.ParseWSMessage(data)  // WSMessage{Headers, Body}
wsMsg.StringBody()                            // typed helper on WSMessage
```

### Outbound encoding

```go
cqrshtmx.WriteWSMessage(w, wsMsg)                       // WSMessage → JSON
cqrshtmx.WriteWSMessageInto[T](w, body, headers)        // typed body → JSON
// Round-trips perfectly through ParseWSMessage.
```

### OOB HTML (HTMX out-of-band swaps)

```go
html := cqrshtmx.WSOOBHTML("tasks", "<ul>...</ul>")                       // default swap
html := cqrshtmx.WSOOBHTML("tasks", "<ul>...</ul>", cqrshtmx.SwapOuterHTML) // explicit strategy
// Passthrough when html already contains hx-swap-oob.
```

### `WSBroadcaster` (WS fan-out)

```go
wb := cqrshtmx.NewWSBroadcaster()
wb.Subscribe()       // chan string (buffered)
wb.Unsubscribe(ch)   // closes channel
wb.Broadcast(msg)    // non-blocking
wb.BroadcastHTML(id, html, swapStrategy...) // convenience: wraps in WSOOBHTML
wb.SubscriberCount() int
```

### Hook factories (bridge CQRS → WS)

```go
wb.BroadcastOnSuccessWS(msg) AfterDispatchHook
wb.BroadcastOnSuccessWSFunc(func(r) string) AfterDispatchHook
wb.BroadcastOnErrorWS() AfterDispatchHook                  // StructuredError JSON
wb.BroadcastOnErrorWSFunc(func(r, err) string) AfterDispatchHook
wb.BroadcastOnAckWS() AfterDispatchHook                    // ACK over WS
wb.BroadcastOnAckWSFunc(func(r, err, cmdID) string) AfterDispatchHook
```

### CQRS bridge (WS → dispatch)

`App.DispatchWSCommand` / `DispatchWSQuery` decode a WS message and dispatch it — running hooks + timeout but **no auth/CSRF/response-writing** (WS is authenticated at upgrade time; you serialize the result back to the connection yourself).

```go
decoder := cqrshtmx.DecodeWSJSON(func(t CreateTaskInput) (command.Command, error) {
    return command.New("CreateTask", t)
})
// in your WS read loop:
err := app.DispatchWSCommand(r, "CreateTask", decoder, data)
if err != nil {
    payload := cqrshtmx.NewStructuredError(err, r)
    _ = conn.WriteMessage(websocket.TextMessage, []byte(payload.JSON()))
    continue
}
// dispatch succeeded — broadcast the new state to all WS clients:
wb.Broadcast(cqrshtmx.WSOOBHTML("tasks", renderTasks()))
```

For queries, `result, err := app.DispatchWSQuery(r, "GetTasks", queryDecoder, data)` returns the result for you to marshal. `cqrshtmx.DecodeWSJSONQuery[T]` is the typed query decoder.

## `StructuredError` (RFC 7807 shape)

Used by `BroadcastOnError`/`BroadcastOnErrorWS` and the WS bridge to carry a typed error payload to realtime clients.

```go
payload := cqrshtmx.NewStructuredError(err, r)   // maps via MapError + extracts request ID
payload := cqrshtmx.NewStructuredErrorWithContext(err, ctx)
payload.JSON()                                   // RFC 7807-shaped JSON string
```

## When to pick SSE vs WebSocket

- **SSE** — one-way server→client (the common case for live updates). Simpler, plain HTTP, auto-reconnects, proxies-friendly. Use `Broadcaster` + `SSEStream`.
- **WebSocket** — bidirectional. Use when the client pushes frequent messages back (collaborative editing, chat). Use `WSBroadcaster` + `DispatchWSCommand`/`DispatchWSQuery`. Authenticated at upgrade time.
- The library provides **transport parity** — both have broadcasters, ACK, hooks, and OOB. See `docs/adr/0010-transport-parity.md`.

## Gotchas

- `Broadcaster.Unsubscribe` **closes the channel**. Never send on a channel after unsubscribing. The read loop pattern above (`defer Unsubscribe(ch)`) is the safe shape.
- Broadcasts are **non-blocking** — slow consumers silently drop events. If you need guaranteed delivery, use the idempotency store + ACK so the client can retry on reconnect.
- The ACK `X-Command-Id` header is **opt-in**. If the client doesn't send it, no ACK is broadcast and idempotency checking is a no-op. The frontend must generate and attach the ID for every mutation.
- For replay to work, your events must have **monotonic IDs** the `SSEEventStore` can order by. `JournalSSEStore` uses ULID event IDs from the journal.
