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
}

// handleEvent dispatches evt to the registered handler under a write lock.
// Unknown event types are silently ignored (return nil).
func (c *readModelCore[T]) handleEvent(m T, evt event.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	h, ok := c.handlers[evt.Type()]
	if !ok {
		return nil
	}
	return h(m, evt.AggregateID(), evt)
}
