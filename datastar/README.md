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
import ds "github.com/larsartmann/cqrs-htmx/datastar/v4"

// 1. Serve datastar.js (self-hosted, no CDN)
mux.Handle("GET /datastar.js", ds.ScriptHandler())

// 2. Real-time SSE endpoint
broadcaster := ds.NewBroadcaster()
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
        ds.ErrorResponse(w, r, err)
        return
    }
    // ... dispatch command ...
    ds.NewResponse(w, r).PatchSignals(map[string]any{"title": ""})
})
```

## API

| Function                                  | Description                                             |
| ----------------------------------------- | ------------------------------------------------------- |
| `ScriptHandler()`                         | Serve embedded datastar.js with ETag caching            |
| `ScriptTag(path)`                         | HTML `<script type="module">` tag                       |
| `ReadSignals(r, &target)`                 | Decode Datastar signals from request                    |
| `NewResponse(w, r)`                       | Fluent Datastar SSE response builder                    |
| `ElementsPatch(html, opts...)`            | Create a patch-elements instruction                     |
| `SignalsPatch(signals, opts...)`          | Create a patch-signals instruction                      |
| `RemovePatch(selector)`                   | Create a remove-element instruction                     |
| `NewBroadcaster()`                        | Fan-out SSE patches to all clients                      |
| `NewBroadcasterWithReplay(n)`             | Broadcaster with a custom replay buffer                 |
| `NewBroadcasterWithHeartbeat(d)`          | Broadcaster with periodic SSE keep-alive comments       |
| `NewEventBridge(broadcaster)`             | Declarative event-to-patch mapping                      |
| `EventBridge.OnError(fn)`                 | Callback for handler errors (logging/metrics)           |
| `ErrorResponse(w, r, err)`                | Send an error as a Datastar notification signal         |

The `Response` builder also exposes `ConsoleLog`, `ConsoleError`,
`DispatchCustomEvent`, `ReplaceURL`, `RemoveElementByID`, `Prefetch`, and
`ExecuteScript` — see the godoc for the full surface.

## License

MIT
