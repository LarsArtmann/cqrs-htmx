package datastar

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	sdk "github.com/starfederation/datastar-go/datastar"
)

const (
	subscriberBufferSize = 64
	defaultReplaySize    = 256
)

// patchEntry pairs a patch with its monotonically increasing replay ID.
type patchEntry struct {
	id    uint64
	patch Patch
}

// Broadcaster fans out Datastar patches to all connected SSE clients. It
// implements http.Handler — mount it at your SSE endpoint:
//
//	broadcaster := ds.NewBroadcaster()
//	mux.Handle("GET /events", broadcaster)
//
// To push updates to all clients, call Broadcast:
//
//	broadcaster.Broadcast(ds.ElementsPatch(renderTodo(todo), ds.WithSelectorID("list")))
//
// The Broadcaster maintains a bounded ring buffer of recent patches. When a
// client reconnects (sending the Last-Event-ID header), missed patches are
// replayed before live updates resume. Use NewBroadcasterWithReplay to
// configure the buffer size, or pass 0 to disable replay entirely.
type Broadcaster struct {
	mu          sync.Mutex
	subscribers map[chan patchEntry]struct{}
	entries     []patchEntry
	nextID      uint64
	maxReplay   int
}

// NewBroadcaster creates a Datastar patch broadcaster with a default replay
// buffer of 256 patches.
func NewBroadcaster() *Broadcaster {
	return NewBroadcasterWithReplay(defaultReplaySize)
}

// NewBroadcasterWithReplay creates a broadcaster with a custom replay buffer.
// maxReplay is the maximum number of patches stored for reconnection replay.
// Pass 0 to disable replay (clients that disconnect miss patches until
// reconnect).
func NewBroadcasterWithReplay(maxReplay int) *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan patchEntry]struct{}),
		maxReplay:   maxReplay,
	}
}

// Broadcast sends a patch to all connected clients and stores it in the replay
// buffer. Slow clients whose channel buffer is full silently miss the patch
// (non-blocking send) — they can recover via reconnection replay.
func (b *Broadcaster) Broadcast(patch Patch) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry := patchEntry{patch: patch}
	if b.maxReplay > 0 {
		b.nextID++
		entry.id = b.nextID
		b.entries = append(b.entries, entry)
		if len(b.entries) > b.maxReplay {
			b.entries = b.entries[len(b.entries)-b.maxReplay:]
		}
	}

	for ch := range b.subscribers {
		select {
		case ch <- entry:
		default:
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
// broadcaster, replays missed patches if the client is reconnecting (via
// Last-Event-ID header or lastEventId query param), creates a Datastar SSE
// stream, and pumps patches to the client until the request is cancelled.
func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ch := make(chan patchEntry, subscriberBufferSize)

	b.mu.Lock()
	b.subscribers[ch] = struct{}{}

	// Snapshot replay entries atomically with subscriber registration.
	// This prevents gaps (patches broadcast between snapshot and subscribe)
	// and duplicates (patches in both the snapshot and the live channel).
	// Only reconnecting clients (with Last-Event-ID) get replay — new clients
	// start fresh and receive only live patches from this point forward.
	var replayEntries []patchEntry
	if b.maxReplay > 0 && hasLastEventID(r) {
		lastID := parseLastEventID(r)
		for _, e := range b.entries {
			if e.id > lastID {
				replayEntries = append(replayEntries, e)
			}
		}
	}
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.subscribers, ch)
		b.mu.Unlock()
	}()

	sse := sdk.NewSSE(w, r)

	// Replay missed patches (if any) before entering the live pump loop.
	for _, entry := range replayEntries {
		writeEventID(w, entry.id)
		if err := entry.patch.apply(sse); err != nil {
			return
		}
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case entry, ok := <-ch:
			if !ok || sse.IsClosed() {
				return
			}

			writeEventID(w, entry.id)
			if err := entry.patch.apply(sse); err != nil {
				return
			}
		}
	}
}

// writeEventID writes the SSE id: field so browsers track the last event for
// automatic reconnection via the Last-Event-ID header. Written directly to the
// ResponseWriter before the SDK writes the event body — both are flushed
// together by the SDK's Send method.
func writeEventID(w http.ResponseWriter, id uint64) {
	if id == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "id: %d\n", id)
}

// parseLastEventID extracts the last event ID from the standard SSE
// Last-Event-ID header or the lastEventId query parameter fallback.
func parseLastEventID(r *http.Request) uint64 {
	raw := lastEventIDValue(r)
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// hasLastEventID reports whether the client sent a Last-Event-ID header or
// lastEventId query parameter, indicating this is a reconnection.
func hasLastEventID(r *http.Request) bool {
	return lastEventIDValue(r) != ""
}

func lastEventIDValue(r *http.Request) string {
	if id := r.Header.Get("Last-Event-ID"); id != "" {
		return id
	}
	return r.URL.Query().Get("lastEventId")
}
