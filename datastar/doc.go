// Package datastar provides a DataStar adapter for cqrs-htmx applications.
//
// This module wraps [go-datastar] and [go-sse] to provide a DataStar protocol
// layer for CQRS event-sourced applications. Its own contribution is the
// domain coupling:
//
//   - EventBridge: declarative domain-event → Patch mapping
//   - Re-exports of common go-datastar types for single-import convenience
//
// The Broadcaster (fan-out SSE serving, reconnection replay, hub sharing)
// moved upstream to [go-datastar/broadcast] on 2026-09-17; this package keeps
// a deprecated type alias plus constructor shims (removal bundled with v5).
// Pass a broadcast.NewBroadcaster result straight into NewEventBridge.
//
// # Quick Start
//
// Serve the embedded DataStar JavaScript and create a real-time SSE endpoint:
//
//	mux.Handle("GET /datastar.js", ds.ScriptHandler())
//
//	broadcaster := broadcast.NewBroadcaster()
//	mux.Handle("GET /events", broadcaster)
//
//	bridge := ds.NewEventBridge(broadcaster)
//	bridge.Map("TodoCreated", func(e event.Event) (ds.Patch, error) {
//	    return ds.ElementsPatch(renderTodo(e), ds.WithSelectorID("todo-list")), nil
//	})
//
//	// Wire to your event bus — bridge.Handle processes each domain event:
//	eventBus.SubscribeAll(bridge.Handle)
//
// For command endpoints, decode DataStar signals and respond with patches:
//
//	mux.HandleFunc("POST /todos", func(w http.ResponseWriter, r *http.Request) {
//	    var s struct{ Title string `json:"title"` }
//	    if err := ds.ReadSignals(r, &s); err != nil {
//	        // handle error
//	        return
//	    }
//	    resp := ds.NewResponse(w, r)
//	    resp.PatchSignals(map[string]any{"title": ""})
//	    resp.PatchElements(renderTodoList(todos), ds.WithSelectorID("todo-list"), ds.WithModeInner())
//	})
//
// # Architecture
//
// This module is a thin adapter. The layers are:
//
//   - [go-sse] — SSE transport (Stream, Broadcaster, Replay, Heartbeat)
//   - [go-datastar] — DataStar protocol vocabulary (patches as values)
//   - [go-datastar/broadcast] — connection lifecycle (fan-out, replay, hubs)
//   - this package — CQRS domain coupling (EventBridge) + re-exports
//
// [go-datastar]: https://github.com/LarsArtmann/go-datastar
// [go-datastar/broadcast]: https://github.com/LarsArtmann/go-datastar/tree/main/broadcast
// [go-sse]: https://github.com/LarsArtmann/go-sse
package datastar
