package usermgmt

import (
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// eventHandler dispatches a single event type to a read-model method.
type eventHandler[T any] func(m T, aggID id.StreamID, evt event.Event) error

// readModelCore provides thread-safe event dispatch for in-memory read models.
// Embed this struct to get mutex-protected handler-map dispatch: register
// handlers in the constructor, then delegate Handle to handleEvent.
type readModelCore[T any] struct {
	mu       sync.RWMutex
	handlers map[event.Type]eventHandler[T]
	// drop optionally reports, under the write lock, that an event must not be
	// applied. A read model whose streams can reach a terminal state uses it to
	// keep a retired stream retired when an event that precedes the retirement
	// is delivered after it.
	drop func(m T, aggID id.StreamID, evt event.Event) bool
}

// handleEvent dispatches evt to the registered handler under a write lock.
// Unknown event types are silently ignored (return nil).
func (c *readModelCore[T]) handleEvent(m T, evt event.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	aggID := evt.StreamID()
	if c.drop != nil && c.drop(m, aggID, evt) {
		return nil
	}
	h, ok := c.handlers[evt.Type()]
	if !ok {
		return nil
	}
	return h(m, evt.StreamID(), evt)
}
