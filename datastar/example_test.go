package datastar_test

import (
	"net/http/httptest"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
)

// ExampleNewResponse demonstrates the fluent Datastar SSE response builder.
// In a real handler, w and r come from your net/http handler.
func ExampleNewResponse() {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/todos", nil)

	ds.NewResponse(w, r).
		PatchSignals(map[string]any{"title": ""}).
		PatchElements("<div id='todo-1'>Buy milk</div>", ds.WithSelectorID("todo-list"), ds.WithModeAppend()).
		ConsoleLog("todo created")
}

// ExampleNewBroadcaster demonstrates mounting a real-time SSE endpoint and
// pushing a patch to all connected clients.
func ExampleNewBroadcaster() {
	broadcaster := ds.NewBroadcaster()

	// Push an update to every connected client:
	broadcaster.Broadcast(ds.ElementsPatch("<div>Updated</div>", ds.WithSelectorID("content")))
}
