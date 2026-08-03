package datastar

import (
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// PatchFunc converts a domain event into a Datastar patch. Register PatchFuncs
// on an EventBridge to declaratively map domain events to real-time UI updates.
//
//	bridge.Map("TodoCreated", func(e event.Event) (ds.Patch, error) {
//	    return ds.ElementsPatch(renderTodo(e), ds.WithSelectorID("todo-list"), ds.WithModeAppend()), nil
//	})
type PatchFunc func(event.Event) (Patch, error)

// EventBridge maps domain events to Datastar patches and broadcasts them to
// all connected SSE clients. It replaces manual switch-statement event
// handling with a declarative per-event-type mapping.
//
//	bridge := ds.NewEventBridge(broadcaster)
//	bridge.Map("TodoCreated", renderCreatedPatch)
//	bridge.Map("TodoDeleted", func(e event.Event) (ds.Patch, error) {
//	    return ds.RemovePatch("#todo-" + e.StreamID().String()), nil
//	})
//
//	// Wire to your event bus:
//	eventBus.SubscribeAll(bridge.Handle)
type EventBridge struct {
	broadcaster *Broadcaster
	mu          sync.RWMutex
	handlers    map[string]PatchFunc
	onError     func(error)
}

// NewEventBridge creates an event bridge that broadcasts patches via the
// given broadcaster.
func NewEventBridge(broadcaster *Broadcaster) *EventBridge {
	return &EventBridge{
		broadcaster: broadcaster,
		handlers:    make(map[string]PatchFunc),
	}
}

// Map registers a handler that converts events of the given type into
// Datastar patches. If a handler is already registered for the event type,
// it is replaced.
func (b *EventBridge) Map(eventType string, fn PatchFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = fn
}

// Unmap removes the handler for the given event type. Unmapped events are
// silently skipped by Handle.
func (b *EventBridge) Unmap(eventType string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.handlers, eventType)
}

// OnError sets a callback invoked when a registered PatchFunc returns an
// error. If nil (the default), errors are silently dropped, preserving the
// original behavior. Use this for observability (logging, metrics) without
// changing control flow — Handle never blocks on the callback.
//
//	bridge.OnError(func(err error) {
//	    slog.Error("datastar event bridge error", "err", err)
//	})
func (b *EventBridge) OnError(fn func(error)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.onError = fn
}

// Handle processes a single domain event. It looks up the registered handler
// for the event type, generates a patch, and broadcasts it to all connected
// clients. Unmapped events are silently skipped. If the handler returns an
// error and an OnError callback is set, the callback is invoked.
func (b *EventBridge) Handle(e event.Event) {
	b.mu.RLock()
	fn, ok := b.handlers[string(e.Type())]
	onError := b.onError
	b.mu.RUnlock()

	if !ok {
		return
	}

	patch, err := fn(e)
	if err != nil {
		if onError != nil {
			onError(err)
		}
		return
	}

	if patch != nil {
		b.broadcaster.Broadcast(patch)
	}
}

// MappedEventTypes returns the event types that have registered handlers.
// Useful for diagnostics and testing.
func (b *EventBridge) MappedEventTypes() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	types := make([]string, 0, len(b.handlers))
	for t := range b.handlers {
		types = append(types, t)
	}

	return types
}
