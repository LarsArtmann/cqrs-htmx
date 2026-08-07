// Package datastar provides a DataStar adapter for cqrs-htmx applications.
//
// This module wraps [go-datastar] and [go-sse] to provide a DataStar protocol
// layer for CQRS event-sourced applications. It adds two things beyond what
// go-datastar provides:
//
//   - Broadcaster: fan-out SSE patches via [sse.Broadcaster[sse.Event]]
//   - EventBridge: declarative domain-event → Patch mapping
//
// All DataStar types (patches, options, modes, namespaces) are re-exported
// from go-datastar for single-import convenience.
//
// # Quick Start
//
// Serve the embedded DataStar JavaScript and create a real-time SSE endpoint:
//
//	mux.Handle("GET /datastar.js", ds.ScriptHandler())
//
//	broadcaster := ds.NewBroadcaster()
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
// This module is a thin adapter. The real implementation lives in:
//
//   - [go-datastar] — DataStar protocol vocabulary (patches as values)
//   - [go-sse] — SSE transport (Stream, Broadcaster, Replay, Heartbeat)
//
// The SDK dependency (starfederation/datastar-go) has been fully replaced.
// Patches are now first-class values implementing [Patch] (Event() sse.Event),
// enabling composition with go-sse's Broadcaster, SubscribeFilter, and
// Shutdown infrastructure.
//
// [go-datastar]: https://github.com/LarsArtmann/go-datastar
// [go-sse]: https://github.com/LarsArtmann/go-sse
package datastar
