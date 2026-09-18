# cqrs-htmx/datastar — Datastar Frontend Adapter

Optional Datastar (https://data-star.dev/) adapter module for [cqrs-htmx](https://github.com/larsartmann/cqrs-htmx).

## Why?

cqrs-htmx uses HTMX for frontend reactivity. [Datastar](https://data-star.dev/) offers an alternative with built-in **signals** (reactive state management), **DOM morphing** (no idiomorph extension), and a **structured SSE protocol**. Its philosophy — backend as source of truth, SSE as default transport, no optimistic updates — is identical to cqrs-htmx's CQRS/event-sourcing architecture.

This module lets you use Datastar **instead of or alongside** HTMX, with zero changes to existing HTMX code.

## Install

```bash
go get github.com/larsartmann/cqrs-htmx/datastar/v4
```

## Quick Start

```go
import (
    ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
    "github.com/larsartmann/go-datastar/broadcast"
)

// 1. Serve datastar.js (self-hosted, no CDN)
mux.Handle("GET /datastar.js", ds.ScriptHandler())

// 2. Real-time SSE endpoint (go-datastar/broadcast)
broadcaster := broadcast.NewBroadcaster()
mux.Handle("GET /events", broadcaster)

// 3. Map domain events to Datastar patches
bridge := ds.NewEventBridge(broadcaster)
bridge.Map("TodoCreated", func(e event.Event) (ds.Patch, error) {
    return ds.ElementsPatch(renderTodo(e), ds.WithSelectorID("todo-list"), ds.WithModeAppend()), nil
})

// 4. Command endpoint with signal decoding
mux.HandleFunc("POST /todos", func(w http.ResponseWriter, r *http.Request) {
    var s struct{ Title string `json:"title"` }
    if err := ds.ReadSignals(r, &s); err != nil {
        ds.ErrorResponse(sse.NewStream(w, r), err.Error(), "ERR_400")
        return
    }
    // ... dispatch command ...
    ds.NewResponse(w, r).PatchSignals(map[string]any{"title": ""})
})
```

## API

| Function                         | Description                                     |
| -------------------------------- | ----------------------------------------------- |
| `ScriptHandler()`                | Serve embedded datastar.js with ETag caching    |
| `ScriptTag(path)`                | HTML `<script type="module">` tag               |
| `ReadSignals(r, &target)`        | Decode Datastar signals from request            |
| `NewResponse(w, r)`              | Fluent Datastar SSE response builder            |
| `ElementsPatch(html, opts...)`   | Create a patch-elements instruction             |
| `SignalsPatch(signals, opts...)` | Create a patch-signals instruction              |
| `RemovePatch(selector)`          | Create a remove-element instruction             |
| `NewEventBridge(broadcaster)`    | Declarative event-to-patch mapping              |
| `EventBridge.OnError(fn)`        | Callback for handler errors (logging/metrics)   |
| `ErrorResponse(stream, msg, code)` | Send an error as a Datastar notification signal |

Fan-out, reconnection replay, and hub sharing live in
[`go-datastar/broadcast`](https://github.com/LarsArtmann/go-datastar/tree/main/broadcast)
(`NewBroadcaster`, `NewBroadcasterWithReplay`, `NewBroadcasterFromHub`, `Hub`).
The same-named constructors here are deprecated aliases (v5 removal);
`NewEventBridge` accepts a `*broadcast.Broadcaster` directly.

For SSE keep-alive (proxy idle timeouts), every `broadcast.Broadcaster`
connection already sends an SSE comment-frame heartbeat every 15 seconds.
For a custom interval, subscribe to `Hub()` and run your own
`sse.Stream.Heartbeat(ctx, d)`.

The `Response` builder also exposes `ConsoleLog`, `ConsoleError`,
`DispatchCustomEvent`, `ReplaceURL`, `RemoveElementByID`, `Prefetch`, and
`ExecuteScript` — see the godoc for the full surface.

## With setup/v4

The one-call SDK can mount the feed for you: set `setup.Config.DataStarPath`
(e.g. `/ds/events`) and the bundle serves the session-gated DataStar feed
from the SAME hub as `/sse` plus the SDK script at `/datastar.js`, exposing
`bundle.DataStarBroadcaster` for your own patches. See
`docs/guides/datastar-integration.md` ("Using with setup") and ADR-0050.

## License

MIT
