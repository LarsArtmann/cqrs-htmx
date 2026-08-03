// Package datastar provides an optional Datastar (https://data-star.dev/)
// frontend adapter for cqrs-htmx applications.
//
// Datastar is a hypermedia framework that uses Server-Sent Events for
// real-time updates, reactive signals for frontend state management, and
// DOM morphing by default. Its philosophy — backend as source of truth,
// SSE as default transport, no optimistic updates — aligns perfectly with
// the CQRS/event-sourcing architecture of cqrs-htmx.
//
// This module is fully isolated: the root cqrs-htmx module has zero
// knowledge of Datastar. Consumers who prefer HTMX are completely
// unaffected. Import this module only when you want Datastar features.
//
// # Quick Start
//
// Serve the embedded Datastar JavaScript and create a real-time SSE endpoint:
//
//	mux.Handle("GET /datastar.js", ds.ScriptHandler())
//
//	broadcaster := ds.NewBroadcaster()
//	mux.HandleFunc("GET /events", broadcaster.ServeHTTP)
//
//	bridge := ds.NewEventBridge(broadcaster)
//	bridge.Map("TodoCreated", func(e event.Event) (ds.Patch, error) {
//	    return ds.ElementsPatch(renderTodo(e)), nil
//	})
//
//	// Wire to your event bus — bridge.Handle processes each domain event:
//	eventBus.SubscribeAll(bridge.Handle)
//
// For command endpoints, decode Datastar signals and respond with patches:
//
//	mux.HandleFunc("POST /todos", func(w http.ResponseWriter, r *http.Request) {
//	    var s struct{ Title string `json:"title"` }
//	    if err := ds.ReadSignals(r, &s); err != nil {
//	        ds.ErrorResponse(w, r, err)
//	        return
//	    }
//	    // ... dispatch command ...
//	    ds.NewResponse(w, r).
//	        PatchSignals(map[string]any{"title": ""}).
//	        PatchElements(renderTodoList(todos), ds.WithSelectorID("todo-list"), ds.WithModeInner())
//	})
//
// # API Surface
//
//   - ScriptHandler / ScriptHandlerWith / ScriptTag — embed and serve datastar.js
//   - ReadSignals — decode client signals from requests
//   - NewResponse — fluent Datastar SSE response builder
//   - ElementsPatch / SignalsPatch / RemovePatch / ScriptPatch / RedirectPatch — patch constructors
//   - Broadcaster — fan-out SSE patches to all connected clients
//   - EventBridge — declarative domain-event-to-Datastar-patch mapping
package datastar
