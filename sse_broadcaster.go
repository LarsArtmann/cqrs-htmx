package cqrshtmx

import (
	"context"
	"net/http"

	"github.com/larsartmann/go-sse"
)

// Broadcaster distributes SSE events to all subscribed clients.
// It embeds [sse.Broadcaster] for the core fan-out mechanics and adds
// CQRS dispatch-hook constructors ([BroadcastOnSuccess], [BroadcastOnError]).
//
// Create one at application startup and share it across handlers:
//
//	broadcaster := cqrshtmx.NewBroadcaster()
//
//	// In your SSE endpoint handler:
//	ch := broadcaster.Subscribe()
//	defer broadcaster.Unsubscribe(ch)
//
//	// In your CQRS event handler or AfterDispatch hook:
//	broadcaster.Broadcast(cqrshtmx.SSEEvent{Event: "itemCreated", Data: html})
type Broadcaster struct {
	*sse.Broadcaster[sse.Event]
}

// NewBroadcaster creates a new event broadcaster with no subscribers.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{Broadcaster: sse.NewBroadcaster[sse.Event]()}
}

// BroadcastOnSuccess creates an AfterDispatchHook that broadcasts an SSE event
// when a command dispatch succeeds (err == nil).
func (b *Broadcaster) BroadcastOnSuccess(eventName, data string) AfterDispatchHook {
	return b.broadcastOnSuccessHook(func(_ *http.Request) SSEEvent {
		return SSEEvent{Event: eventName, Data: data}
	})
}

// BroadcastOnSuccessFunc creates an AfterDispatchHook that generates an SSE event
// dynamically from the request when dispatch succeeds.
func (b *Broadcaster) BroadcastOnSuccessFunc(eventFunc func(r *http.Request) SSEEvent) AfterDispatchHook {
	return b.broadcastOnSuccessHook(eventFunc)
}

// BroadcastOnError creates an AfterDispatchHook that broadcasts an SSE event
// when a command dispatch fails (err != nil).
func (b *Broadcaster) BroadcastOnError(eventName string) AfterDispatchHook {
	return b.broadcastOnErrorHook(func(r *http.Request, err error) SSEEvent {
		payload := NewStructuredError(err, r)

		return SSEEvent{Event: eventName, Data: payload.JSON()}
	})
}

// BroadcastOnErrorFunc creates an AfterDispatchHook that generates an SSE error
// event dynamically from the request and error when dispatch fails.
func (b *Broadcaster) BroadcastOnErrorFunc(errFunc func(r *http.Request, err error) SSEEvent) AfterDispatchHook {
	return b.broadcastOnErrorHook(errFunc)
}

// broadcastOnSuccessHook builds an AfterDispatchHook that broadcasts the result
// of mapper(r) when dispatch succeeds (err == nil).
func (b *Broadcaster) broadcastOnSuccessHook(mapper func(r *http.Request) SSEEvent) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		if err != nil {
			return
		}

		b.Broadcast(mapper(r))
	}
}

// broadcastOnErrorHook builds an AfterDispatchHook that broadcasts the result
// of mapper(r, err) when dispatch fails (err != nil).
func (b *Broadcaster) broadcastOnErrorHook(mapper func(r *http.Request, err error) SSEEvent) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		if err == nil {
			return
		}

		b.Broadcast(mapper(r, err))
	}
}

// ServeSSE is a high-level helper that handles the full SSE connection lifecycle:
// creates a stream, subscribes, sends a "connected" event, pumps events until
// the client disconnects, then unsubscribes and closes the stream.
func (b *Broadcaster) ServeSSE(w http.ResponseWriter, r *http.Request) {
	stream := NewSSEStream(w, r)
	defer stream.Close()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	_ = stream.Send(SSEEvent{Event: SSEEventConnected, Data: "connected"})

	for {
		select {
		case <-stream.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok || stream.Send(evt) != nil {
				return
			}
		}
	}
}
