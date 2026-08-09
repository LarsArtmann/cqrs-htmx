package cqrshtmx

import (
	"context"
	"log/slog"
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
//	broadcaster.Broadcast(sse.Event{Event: "itemCreated", Data: html})
type Broadcaster struct {
	*sse.Broadcaster[sse.Event]
}

// RawBroadcaster is implemented by Broadcaster types that expose their
// underlying [*sse.Broadcaster]. This enables sharing a single fan-out hub
// across transports (e.g., HTMX SSE and DataStar SSE from the same event
// source). Both the root [*Broadcaster] and datastar's *Broadcaster satisfy
// this interface structurally (duck typing — no import required).
type RawBroadcaster interface {
	Raw() *sse.Broadcaster[sse.Event]
}

// NewBroadcaster creates a new event broadcaster with no subscribers.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{Broadcaster: sse.NewBroadcaster[sse.Event]()}
}

// NewBroadcasterFromRaw wraps an existing [*sse.Broadcaster] in a [*Broadcaster],
// enabling cross-transport fan-out hub sharing. Use this when you want HTMX
// SSE and DataStar SSE to share the same underlying event distribution.
func NewBroadcasterFromRaw(raw *sse.Broadcaster[sse.Event]) *Broadcaster {
	return &Broadcaster{Broadcaster: raw}
}

// Raw returns the underlying [*sse.Broadcaster] so consumers can access
// advanced features (SubscribeFilter, custom health checks) or share the
// fan-out hub with another Broadcaster via NewBroadcasterFromRaw.
func (b *Broadcaster) Raw() *sse.Broadcaster[sse.Event] {
	return b.Broadcaster
}

// BroadcastOnSuccess creates an AfterDispatchHook that broadcasts an SSE event
// when a command dispatch succeeds (err == nil).
func (b *Broadcaster) BroadcastOnSuccess(eventName, data string) AfterDispatchHook {
	return b.broadcastOnSuccessHook(func(_ *http.Request) sse.Event {
		return sse.Event{Event: eventName, Data: data}
	})
}

// BroadcastOnSuccessFunc creates an AfterDispatchHook that generates an SSE event
// dynamically from the request when dispatch succeeds.
func (b *Broadcaster) BroadcastOnSuccessFunc(eventFunc func(r *http.Request) sse.Event) AfterDispatchHook {
	return b.broadcastOnSuccessHook(eventFunc)
}

// BroadcastOnError creates an AfterDispatchHook that broadcasts an SSE event
// when a command dispatch fails (err != nil).
func (b *Broadcaster) BroadcastOnError(eventName string) AfterDispatchHook {
	return b.broadcastOnErrorHook(func(r *http.Request, err error) sse.Event {
		payload := NewStructuredError(err, r)

		return sse.Event{Event: eventName, Data: payload.JSON()}
	})
}

// BroadcastOnErrorFunc creates an AfterDispatchHook that generates an SSE error
// event dynamically from the request and error when dispatch fails.
func (b *Broadcaster) BroadcastOnErrorFunc(errFunc func(r *http.Request, err error) sse.Event) AfterDispatchHook {
	return b.broadcastOnErrorHook(errFunc)
}

// broadcastOnSuccessHook builds an AfterDispatchHook that broadcasts the result
// of mapper(r) when dispatch succeeds (err == nil).
func (b *Broadcaster) broadcastOnSuccessHook(mapper func(r *http.Request) sse.Event) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		if err != nil {
			return
		}

		b.Broadcast(mapper(r))
	}
}

// broadcastOnErrorHook builds an AfterDispatchHook that broadcasts the result
// of mapper(r, err) when dispatch fails (err != nil).
func (b *Broadcaster) broadcastOnErrorHook(mapper func(r *http.Request, err error) sse.Event) AfterDispatchHook {
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
	stream := sse.NewStream(w, r)
	defer func() {
		if err := stream.Close(); err != nil {
			slog.Debug("cqrshtmx: sse stream close failed", "error", err)
		}
	}()

	//cqrs-lint:ignore(C027) SSE fan-out channel for real-time delivery, not a read-model projection
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	if err := stream.Send(sse.Event{Event: sse.EventConnected, Data: "connected"}); err != nil {
		return
	}

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
