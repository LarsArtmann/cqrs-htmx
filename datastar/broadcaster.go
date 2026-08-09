package datastar

import (
	"context"
	"net/http"

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
// For reconnection replay, use NewBroadcasterWithReplay. When a client
// reconnects with a Last-Event-ID header, missed events are replayed from an
// in-memory ring buffer before the live event stream resumes.
//
// The underlying sse.Broadcaster provides SubscribeFilter, Shutdown, Health,
// and configurable buffer size — all available for free via go-sse.
type Broadcaster struct {
	inner *sse.Broadcaster[sse.Event]
	store *godatastar.MemoryStore
}

// NewBroadcaster creates a DataStar patch broadcaster with default settings
// and no replay support.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{inner: sse.NewBroadcaster[sse.Event]()}
}

// NewBroadcasterWithBufferSize creates a broadcaster with a custom subscriber
// buffer size and no replay support.
func NewBroadcasterWithBufferSize(size int) *Broadcaster {
	return &Broadcaster{inner: sse.NewBroadcaster[sse.Event](sse.WithBufferSize[sse.Event](size))}
}

// NewBroadcasterWithReplay creates a broadcaster that retains the last capacity
// events in an in-memory ring buffer for reconnection replay. When a client
// reconnects with a Last-Event-ID header, missed events are replayed before
// the live stream resumes.
func NewBroadcasterWithReplay(capacity int) *Broadcaster {
	return &Broadcaster{
		inner: sse.NewBroadcaster[sse.Event](),
		store: godatastar.NewMemoryStore(capacity),
	}
}

// NewBroadcasterFromRaw wraps an existing [*sse.Broadcaster] in a [*Broadcaster],
// enabling cross-transport fan-out hub sharing without replay support. Use this
// when you want HTMX SSE and DataStar SSE to share the same underlying event
// distribution.
func NewBroadcasterFromRaw(raw *sse.Broadcaster[sse.Event]) *Broadcaster {
	return &Broadcaster{inner: raw}
}

// Raw returns the underlying [*sse.Broadcaster] so consumers can access
// advanced features (SubscribeFilter, direct Subscribe) or share the fan-out
// hub with another Broadcaster via NewBroadcasterFromRaw.
func (b *Broadcaster) Raw() *sse.Broadcaster[sse.Event] {
	return b.inner
}

// Broadcast sends a patch to all connected clients. The patch's Event() is
// computed once and the resulting [sse.Event] is fan-out to all subscribers.
// If replay is enabled, the event is also appended to the store. Slow clients
// whose channel buffer is full silently miss the event.
func (b *Broadcaster) Broadcast(patch Patch) {
	evt := patch.Event()
	b.inner.Broadcast(evt)
	if b.store != nil {
		b.store.Append(evt)
	}
}

// BroadcastMany sends multiple patches to all connected clients.
func (b *Broadcaster) BroadcastMany(patches ...Patch) {
	for _, p := range patches {
		b.Broadcast(p)
	}
}

// BroadcastEvent sends a raw [sse.Event] to all connected clients. If replay
// is enabled, the event is also appended to the store.
func (b *Broadcaster) BroadcastEvent(evt sse.Event) {
	b.inner.Broadcast(evt)
	if b.store != nil {
		b.store.Append(evt)
	}
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
// replays missed events from the store (if replay is enabled and the client
// sends a Last-Event-ID), subscribes to the broadcaster, and forwards events
// to the client until the request is cancelled or the connection breaks.
func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	if b.store != nil {
		if lastID := stream.LastEventID(); !lastID.IsZero() {
			if _, err := sse.Replay(stream, b.store, lastID); err != nil {
				return
			}
		}
	}

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
