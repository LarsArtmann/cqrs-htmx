package datastar

import (
	"context"
	"net/http"
	"time"

	godatastar "github.com/larsartmann/go-datastar"
	"github.com/larsartmann/go-sse"
)

// Broadcaster fans out DataStar patches to all connected SSE clients. It wraps
// [sse.Broadcaster[sse.Event]] and implements [http.Handler].
//
// Mount it at your SSE endpoint:
//
//	broadcaster := ds.NewBroadcaster()
//	mux.Handle("GET /events", broadcaster)
//
// To push updates to all clients, call Broadcast:
//
//	broadcaster.Broadcast(ds.ElementsPatch(renderTodo(todo), ds.WithSelectorID("list")))
//
// The underlying sse.Broadcaster provides SubscribeFilter, Shutdown, Health,
// and configurable buffer size — all available for free via go-sse.
type Broadcaster struct {
	inner *sse.Broadcaster[sse.Event]
}

// NewBroadcaster creates a DataStar patch broadcaster with default settings.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{inner: sse.NewBroadcaster[sse.Event]()}
}

// NewBroadcasterWithBufferSize creates a broadcaster with a custom subscriber
// buffer size.
func NewBroadcasterWithBufferSize(size int) *Broadcaster {
	return &Broadcaster{inner: sse.NewBroadcaster[sse.Event](sse.WithBufferSize[sse.Event](size))}
}

// Broadcast sends a patch to all connected clients. The patch's Event() is
// computed once and the resulting [sse.Event] is fan-out to all subscribers.
// Slow clients whose channel buffer is full silently miss the event.
func (b *Broadcaster) Broadcast(patch Patch) {
	b.inner.Broadcast(patch.Event())
}

// BroadcastMany sends multiple patches to all connected clients.
func (b *Broadcaster) BroadcastMany(patches ...Patch) {
	for _, p := range patches {
		b.Broadcast(p)
	}
}

// BroadcastEvent sends a raw [sse.Event] to all connected clients.
func (b *Broadcaster) BroadcastEvent(evt sse.Event) {
	b.inner.Broadcast(evt)
}

// SubscriberCount returns the number of currently connected SSE clients.
func (b *Broadcaster) SubscriberCount() int {
	return b.inner.Health().SubscriberCount
}

// Health returns a health snapshot of the underlying broadcaster.
func (b *Broadcaster) Health() sse.BroadcasterHealth {
	return b.inner.Health()
}

// Shutdown gracefully drains all subscribers within the context deadline.
func (b *Broadcaster) Shutdown(ctx context.Context) error {
	return b.inner.Shutdown(ctx)
}

// Close immediately disconnects all subscribers.
func (b *Broadcaster) Close() {
	b.inner.Close()
}

// ServeHTTP handles a DataStar SSE connection. It creates an [sse.Stream],
// subscribes to the broadcaster, and forwards events to the client until
// the request is cancelled or the connection breaks.
func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	ch := b.inner.Subscribe()
	defer b.inner.Unsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			if err := stream.Send(evt); err != nil {
				return
			}
		}
	}
}

// OnSubscribe registers a callback invoked when a client connects.
func (b *Broadcaster) OnSubscribe(fn func()) {
	b.inner.OnSubscribe(fn)
}

// OnUnsubscribe registers a callback invoked when a client disconnects.
func (b *Broadcaster) OnUnsubscribe(fn func()) {
	b.inner.OnUnsubscribe(fn)
}

// LastEventID extracts the last event ID from an HTTP request, checking
// the Last-Event-ID header and the lastEventId query parameter.
// This is a re-export of [godatastar.LastEventID].
func LastEventID(r *http.Request) sse.EventID {
	return godatastar.LastEventID(r)
}

// HeartbeatInterval is a helper that starts a heartbeat loop on the given
// stream at the given interval. The loop runs until the context is cancelled.
func HeartbeatInterval(ctx context.Context, stream *sse.Stream, interval time.Duration) {
	stream.Heartbeat(ctx, interval)
}
