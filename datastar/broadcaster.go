package datastar

import (
	"net/http"
	"sync"

	sdk "github.com/starfederation/datastar-go/datastar"
)

// subscriberBufferSize is the channel buffer size for each SSE client.
// If a client falls behind (slow connection, paused tab), patches are
// dropped to prevent one slow client from blocking the broadcaster.
const subscriberBufferSize = 64

// Broadcaster fans out Datastar patches to all connected SSE clients.
// It implements http.Handler — mount it at your SSE endpoint:
//
//	broadcaster := ds.NewBroadcaster()
//	mux.Handle("GET /events", broadcaster)
//
// To push updates to all clients, call Broadcast:
//
//	broadcaster.Broadcast(ds.ElementsPatch(renderTodo(todo), ds.WithSelectorID("list")))
type Broadcaster struct {
	mu          sync.Mutex
	subscribers map[chan Patch]struct{}
}

// NewBroadcaster creates a Datastar patch broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan Patch]struct{}),
	}
}

// Broadcast sends a patch to all connected clients. Slow clients whose
// channel buffer is full silently miss the patch (non-blocking send).
func (b *Broadcaster) Broadcast(patch Patch) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.subscribers {
		select {
		case ch <- patch:
		default:
			// Client buffer full — drop patch to avoid blocking the broadcaster.
		}
	}
}

// BroadcastMany sends multiple patches to all connected clients.
func (b *Broadcaster) BroadcastMany(patches ...Patch) {
	for _, p := range patches {
		b.Broadcast(p)
	}
}

// SubscriberCount returns the number of currently connected SSE clients.
func (b *Broadcaster) SubscriberCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.subscribers)
}

// Close disconnects all subscribers by closing their channels. The
// Broadcaster cannot be reused after Close.
func (b *Broadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.subscribers {
		close(ch)
		delete(b.subscribers, ch)
	}
}

// ServeHTTP handles a Datastar SSE connection. It subscribes to the
// broadcaster, creates a Datastar SSE stream, and pumps patches to the
// client until the request is cancelled or the connection breaks.
func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ch := make(chan Patch, subscriberBufferSize)

	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.subscribers, ch)
		b.mu.Unlock()
	}()

	sse := sdk.NewSSE(w, r)

	for {
		select {
		case <-r.Context().Done():
			return
		case patch, ok := <-ch:
			if !ok || sse.IsClosed() {
				return
			}

			if err := patch.apply(sse); err != nil {
				return
			}
		}
	}
}
