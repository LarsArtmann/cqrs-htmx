package cqrshtmx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/larsartmann/go-sse"
)

// Broadcaster distributes SSE events to all subscribed clients.
// It embeds [sse.Broadcaster] for the core fan-out mechanics and adds
// CQRS dispatch-hook constructors ([BroadcastOnSuccess], [BroadcastOnError]).
// The embedded hub is the canonical shareable object — see [Broadcaster.Hub].
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
// underlying [*sse.Broadcaster].
//
// Deprecated: use [Broadcaster.Hub] and pass [*sse.Broadcaster] (the hub
// itself) instead — the hub is the canonical shareable object, and both
// broadcasters now embed it, so unwrapping via an interface is unnecessary.
// Removal is bundled with v5.
type RawBroadcaster interface {
	Raw() *sse.Broadcaster[sse.Event]
}

// NewBroadcaster creates a new event broadcaster with no subscribers.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{Broadcaster: sse.NewBroadcaster[sse.Event]()}
}

// NewBroadcasterFromHub wraps an existing [*sse.Broadcaster] in a
// [*Broadcaster], enabling cross-transport fan-out hub sharing. Use this when
// you want HTMX SSE and DataStar SSE to distribute from the same hub:
//
//	hub := sse.NewBroadcaster[sse.Event]()
//	htmx := cqrshtmx.NewBroadcasterFromHub(hub)
//	dsBC := ds.NewBroadcasterFromHub(hub)
func NewBroadcasterFromHub(hub *sse.Broadcaster[sse.Event]) *Broadcaster {
	return &Broadcaster{Broadcaster: hub}
}

// NewBroadcasterFromRaw wraps an existing [*sse.Broadcaster] in a [*Broadcaster].
//
// Deprecated: use [NewBroadcasterFromHub] — the hub is the canonical shareable
// object, not a "raw" escape hatch. Removal is bundled with v5.
func NewBroadcasterFromRaw(raw *sse.Broadcaster[sse.Event]) *Broadcaster {
	return NewBroadcasterFromHub(raw)
}

// Hub returns the embedded [*sse.Broadcaster] — the canonical fan-out hub.
// Use it to share one hub across transport adapters (via
// [NewBroadcasterFromHub] or datastar's equivalent) or to access go-sse
// features directly (SubscribeFilter, Health, Shutdown, custom buffer size).
func (b *Broadcaster) Hub() *sse.Broadcaster[sse.Event] {
	return b.Broadcaster
}

// Raw returns the underlying [*sse.Broadcaster].
//
// Deprecated: use [Broadcaster.Hub] — the hub is the canonical shareable
// object, not a "raw" escape hatch. Removal is bundled with v5.
func (b *Broadcaster) Raw() *sse.Broadcaster[sse.Event] {
	return b.Hub()
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

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	if err := stream.Send(sse.Event{Event: sse.EventConnected, Data: "connected"}); err != nil {
		return
	}

	go stream.Heartbeat(r.Context(), 15*time.Second)

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
