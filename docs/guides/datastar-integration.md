# Datastar Integration Guide

> How to use [Datastar](https://data-star.dev/) instead of (or alongside) HTMX in your cqrs-htmx application.

## What is Datastar?

Datastar is a hypermedia framework that uses Server-Sent Events for real-time updates, reactive **signals** for frontend state management, and DOM **morphing** by default. Its philosophy — backend as source of truth, SSE as default transport, no optimistic updates — is identical to cqrs-htmx's CQRS/event-sourcing architecture.

Datastar replaces Alpine.js, HTMX extensions (idiomorph, SSE, WS), and hand-rolled JavaScript with one 11.76 KiB file.

## Installation

```bash
go get github.com/larsartmann/cqrs-htmx/datastar/v4
```

Import as:

```go
import ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
```

## Quick Start

### 1. Serve the Datastar JavaScript

Self-host datastar.js with ETag caching (no CDN dependency):

```go
mux.Handle("GET /datastar.js", ds.ScriptHandler())
```

Add the script tag to your HTML:

```html
<script type="module" src="/datastar.js"></script>
```

### 2. Set up real-time SSE

The `Broadcaster` fans out Datastar patches to all connected clients with built-in reconnection replay:

```go
broadcaster := ds.NewBroadcaster()
mux.Handle("GET /events", broadcaster)
```

When a client reconnects (sending the `Last-Event-ID` header), missed patches are replayed automatically. Configure the replay buffer size:

```go
broadcaster := ds.NewBroadcasterWithReplay(1024) // 1024-patch ring buffer
```

Pass `0` to disable replay entirely (clients that disconnect miss patches until reconnect).

#### Heartbeat (keep-alive)

Proxies (nginx, Cloudflare, load balancers) close idle SSE connections after a timeout. Enable periodic heartbeats to keep the connection alive:

```go
broadcaster := ds.NewBroadcasterWithHeartbeat(30 * time.Second)
```

Heartbeats are sent as lightweight SSE events (`event: ping`) that the Datastar client silently ignores. They reset proxy idle timers without producing any visible UI update.

### 3. Decode client signals

Datastar automatically sends all signals with every request. Decode them into Go structs:

```go
mux.HandleFunc("POST /todos", func(w http.ResponseWriter, r *http.Request) {
    var s struct {
        Title string `json:"title"`
    }
    if err := ds.ReadSignals(r, &s); err != nil {
        ds.ErrorResponse(w, r, err)
        return
    }
    // ... dispatch command ...
})
```

### 4. Send Datastar responses

For command endpoints, build a Datastar SSE response:

```go
ds.NewResponse(w, r).
    PatchSignals(map[string]any{"title": ""}).              // clear input
    PatchElements(renderTodoList(todos),                     // update DOM
        ds.WithSelectorID("todo-list"),
        ds.WithModeInner())
```

The Response builder also supports:

| Method                                  | Description                                                                       |
| --------------------------------------- | --------------------------------------------------------------------------------- |
| `ConsoleLog(msg)` / `ConsoleError(err)` | Write to the browser developer console                                            |
| `DispatchCustomEvent(name, detail)`     | Fire a custom browser event with a JSON payload                                   |
| `ReplaceURL(rawURL)`                    | Update the browser address bar without navigation (invalid URLs silently ignored) |
| `RemoveElementByID(id)`                 | Remove a DOM element by its ID (shorthand for `RemoveElement("#" + id)`)          |
| `Prefetch(urls...)`                     | Hint the browser to preload URLs via the Speculation Rules API                    |

### 5. Map domain events to patches

Use `EventBridge` to declaratively map go-cqrs-lite domain events to Datastar patches:

```go
bridge := ds.NewEventBridge(broadcaster)
bridge.Map("TodoCreated", func(e event.Event) (ds.Patch, error) {
    return ds.ElementsPatch(renderTodo(e), ds.WithSelectorID("todo-list"), ds.WithModeAppend()), nil
})
bridge.Map("TodoDeleted", func(e event.Event) (ds.Patch, error) {
    return ds.RemovePatch("#todo-" + e.StreamID().String()), nil
})

// Wire to your event bus:
eventBus.SubscribeAll(bridge.Handle)
```

### Error observability

Set an `OnError` callback on the EventBridge to capture errors from PatchFuncs without changing control flow:

```go
bridge.OnError(func(err error) {
    slog.Error("datastar patch error", "err", err)
})
```

Without `OnError`, errors are silently dropped (the original behavior). The callback fires only for registered event types whose handler returns an error — unmapped events are always skipped silently.

## Patch Types

| Constructor                               | SSE Event                 | Description                       |
| ----------------------------------------- | ------------------------- | --------------------------------- |
| `ElementsPatch(html, opts...)`            | `datastar-patch-elements` | Morph HTML into the DOM           |
| `ElementsTemplPatch(component, opts...)`  | `datastar-patch-elements` | Render a templ component          |
| `SignalsPatch(signals, opts...)`          | `datastar-patch-signals`  | Update reactive signals           |
| `SignalsIfMissingPatch(signals, opts...)` | `datastar-patch-signals`  | Set signals only if absent        |
| `RemovePatch(selector)`                   | `datastar-patch-elements` | Remove elements matching selector |
| `ScriptPatch(script, opts...)`            | `datastar-execute-script` | Execute JavaScript on client      |
| `RedirectPatch(url)`                      | `datastar-execute-script` | Navigate client to URL            |

## Reconnection and Replay

The Broadcaster maintains a bounded ring buffer of recent patches. Each patch receives a monotonically increasing ID (sent as the SSE `id:` field).

When a client disconnects and reconnects:

1. The browser's `EventSource` automatically sends `Last-Event-ID` with the last received ID
2. The Broadcaster replays all buffered patches after that ID
3. Live updates resume seamlessly

New clients (no `Last-Event-ID`) start fresh — they receive only live patches from connection time forward.

For non-EventSource clients, use the `?lastEventId=N` query parameter as a fallback.

## Using Datastar Alongside HTMX

The datastar module is fully isolated. You can serve both HTMX and Datastar endpoints in the same application:

```go
// HTMX endpoints (unchanged)
mux.Handle("GET /htmx/users", app.Query("ListUsers", cqrshtmx.DecodeJSONQuery(...)))

// Datastar endpoints (new)
mux.Handle("GET /datastar.js", ds.ScriptHandler())
mux.Handle("GET /ds/events", broadcaster)
mux.HandleFunc("POST /ds/todos", createTodoHandler)
```

## SDK Option Re-exports

The adapter re-exports all Datastar SDK options for single-import convenience:

```go
// Element patch options
ds.WithSelectorID("todo-list")
ds.WithModeInner()
ds.WithModeAppend()
ds.WithNamespaceMathML()
ds.WithViewTransitions()

// Signal patch options
ds.WithOnlyIfMissing(true)

// Script options
ds.WithExecuteScriptAutoRemove(false)
ds.WithExecuteScriptAttributes("type", "module")
```

## Using with setup

The one-call SDK (`setup/v4`) can mount the DataStar feed for you — two
config fields, no manual wiring (ADR-0050):

```go
bundle, err := setup.New(setup.Config{
    Title:        "My App",
    SSEPath:      "/sse",          // HTMX feed (session-gated)
    SSEURL:       "/sse",          // admin sync indicator
    DataStarPath: "/ds/events",    // DataStar feed — same hub as /sse
    // DataStarScriptPath defaults to "/datastar.js"; set "-" to skip the
    // script mount and load the SDK yourself.
})
```

What you get:

- `/ds/events` — DataStar SSE feed of every committed domain event,
  session-gated 401 exactly like `/sse` (event metadata is not public data).
- `/datastar.js` — the SDK script, ETag-cached.
- `bundle.DataStarBroadcaster` — broadcast your own patches to the feed
  (`ds.SignalsPatch`, `ds.ElementsPatch`, ...). It shares the fan-out hub
  with `bundle.Broadcaster`, so one broadcast reaches BOTH transports.

The feed is live fan-out; `/sse` remains the replay-capable endpoint
(Last-Event-ID backfill from the journal). See `fullstack-wiring.md` for the
manual-wiring version of this recipe and `sse-and-datastar.md` for the hub
architecture underneath.

## Demo Application

See `examples/datastar-demo/` for a complete working example:

- Event-sourced CQRS todo app
- Real-time SSE with patch replay on reconnection
- Multi-user simulation (10 bot goroutines)
- Self-hosted datastar.js (no CDN)
- Uses the adapter module APIs throughout
