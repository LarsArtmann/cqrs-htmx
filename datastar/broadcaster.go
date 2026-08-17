package datastar

import (
	"net/http"

	godatastar "github.com/larsartmann/go-datastar"
	"github.com/larsartmann/go-sse"
)

// Broadcaster fans out DataStar patches to all connected SSE clients. It embeds
// [sse.Broadcaster[sse.Event]] and implements [http.Handler].
//
// The embedded hub is the canonical shareable object: use [Broadcaster.Hub] to
// access it directly (Subscribe, SubscribeFilter, Health, Shutdown, Close,
// OnSubscribe, OnUnsubscribe are promoted) or to share one fan-out hub with
// another transport adapter via [NewBroadcasterFromHub].
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
type Broadcaster struct {
	*sse.Broadcaster[sse.Event]
	store *godatastar.MemoryStore
}

// NewBroadcaster creates a DataStar patch broadcaster with default settings
// and no replay support.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{Broadcaster: sse.NewBroadcaster[sse.Event]()}
}

// NewBroadcasterWithBufferSize creates a broadcaster with a custom subscriber
// buffer size and no replay support.
func NewBroadcasterWithBufferSize(size int) *Broadcaster {
	return &Broadcaster{
		Broadcaster: sse.NewBroadcaster[sse.Event](sse.WithBufferSize[sse.Event](size)),
	}
}

// NewBroadcasterWithReplay creates a broadcaster that retains the last capacity
// events in an in-memory ring buffer for reconnection replay. When a client
// reconnects with a Last-Event-ID header, missed events are replayed before the
// live stream resumes.
func NewBroadcasterWithReplay(capacity int) *Broadcaster {
	return &Broadcaster{
		Broadcaster: sse.NewBroadcaster[sse.Event](),
		store:       godatastar.NewMemoryStore(capacity),
	}
}

// NewBroadcasterFromHub wraps an existing [*sse.Broadcaster] in a
// [*Broadcaster], enabling cross-transport fan-out hub sharing. Use this when
// you want HTMX SSE and DataStar SSE to distribute from the same hub.
//
// The returned broadcaster has no replay store; [BroadcastEvent] on it reaches
// all hub subscribers, but nothing is retained for reconnection replay.
func NewBroadcasterFromHub(hub *sse.Broadcaster[sse.Event]) *Broadcaster {
	return &Broadcaster{Broadcaster: hub}
}

// NewBroadcasterFromRaw wraps an existing [*sse.Broadcaster] in a
// [*Broadcaster].
//
// Deprecated: use [NewBroadcasterFromHub] — the hub is the canonical shareable
// object, not a "raw" escape hatch. Removal is bundled with v5.
func NewBroadcasterFromRaw(raw *sse.Broadcaster[sse.Event]) *Broadcaster {
	return NewBroadcasterFromHub(raw)
}

// Hub returns the embedded [*sse.Broadcaster] — the canonical fan-out hub.
// Use it to share one hub across transport adapters (via
// [NewBroadcasterFromHub]) or to access go-sse features directly
// (SubscribeFilter, Health, Shutdown, configurable buffer size).
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

// Broadcast sends a patch to all connected clients. The patch's Event() is
// computed once and the resulting [sse.Event] is fan-out to all subscribers.
// If replay is enabled, the event is also appended to the store. Slow clients
// whose channel buffer is full silently miss the event.
//
// Broadcast shadows the embedded hub's Broadcast(sse.Event) — to send a raw
// event, use [Broadcaster.BroadcastEvent].
func (b *Broadcaster) Broadcast(patch Patch) {
	evt := patch.Event()
	b.Broadcaster.Broadcast(evt)
	if b.store != nil {
		b.store.Append(evt)
	}
}

// BroadcastMany sends multiple patches to all connected clients. It shadows the
// embedded hub's BroadcastMany([]sse.Event); to send raw events, broadcast via
// [Broadcaster.Hub].
func (b *Broadcaster) BroadcastMany(patches ...Patch) {
	for _, p := range patches {
		b.Broadcast(p)
	}
}

// BroadcastEvent sends a raw [sse.Event] to all connected clients. If replay
// is enabled, the event is also appended to the store.
func (b *Broadcaster) BroadcastEvent(evt sse.Event) {
	b.Broadcaster.Broadcast(evt)
	if b.store != nil {
		b.store.Append(evt)
	}
}

// SubscriberCount returns the number of currently connected SSE clients.
func (b *Broadcaster) SubscriberCount() int {
	return b.Health().SubscriberCount
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

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

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

// LastEventID extracts the last event ID from an HTTP request, checking
// the Last-Event-ID header and the lastEventId query parameter.
// This is a re-export of [godatastar.LastEventID].
func LastEventID(r *http.Request) sse.EventID {
	return godatastar.LastEventID(r)
}
