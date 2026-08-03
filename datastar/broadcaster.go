package datastar

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	sdk "github.com/starfederation/datastar-go/datastar"
)

const (
	subscriberBufferSize = 64
	defaultReplaySize    = 256
	// heartbeatEventType is the SSE event type used for keep-alive heartbeats.
	// The Datastar client ignores unknown event types, making this a safe
	// lightweight signal that resets proxy idle timers without side effects.
	heartbeatEventType = EventType("ping")
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
// configure the buffer size, or pass 0 to disable replay entirely. Use
// NewBroadcasterWithHeartbeat to enable periodic keep-alive events that
// prevent proxy idle-timeout disconnects.
type Broadcaster struct {
	mu          sync.Mutex
	subscribers map[chan patchEntry]struct{}
	entries     []patchEntry
	nextID      uint64
	maxReplay   int
	heartbeat   time.Duration
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

// NewBroadcasterWithHeartbeat creates a broadcaster that sends periodic SSE
// heartbeat events to each connected client. This keeps the connection
// alive through proxies (nginx, Cloudflare, etc.) that close idle
// connections after a timeout. Pass a reasonable interval (e.g. 30s) — too
// frequent and you waste bandwidth, too rare and proxies may drop the
// connection before the first heartbeat arrives.
//
//	broadcaster := ds.NewBroadcasterWithHeartbeat(30 * time.Second)
func NewBroadcasterWithHeartbeat(interval time.Duration) *Broadcaster {
	b := NewBroadcaster()
	b.heartbeat = interval
	return b
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
	replayEntries := b.collectReplayEntries(r)
	b.mu.Unlock()

	defer b.removeSubscriber(ch)

	sse := sdk.NewSSE(w, r)

	b.replayPatches(w, sse, replayEntries)
	b.pumpPatches(w, sse, ch, r)
}

// collectReplayEntries returns patches the reconnecting client has missed.
// Must be called while holding b.mu so that the snapshot is atomic with
// subscriber registration (no gaps, no duplicates). New clients (no
// Last-Event-ID) get an empty slice — only reconnecting clients receive replay.
func (b *Broadcaster) collectReplayEntries(r *http.Request) []patchEntry {
	if b.maxReplay <= 0 || !hasLastEventID(r) {
		return nil
	}
	lastID := parseLastEventID(r)
	var missed []patchEntry
	for _, e := range b.entries {
		if e.id > lastID {
			missed = append(missed, e)
		}
	}
	return missed
}

// removeSubscriber safely removes a subscriber channel from the broadcaster.
func (b *Broadcaster) removeSubscriber(ch chan patchEntry) {
	b.mu.Lock()
	delete(b.subscribers, ch)
	b.mu.Unlock()
}

// replayPatches sends missed patches to a reconnecting client before live
// updates resume. Stops on the first write error.
func (b *Broadcaster) replayPatches(
	w http.ResponseWriter,
	sse *sdk.ServerSentEventGenerator,
	entries []patchEntry,
) {
	for _, entry := range entries {
		writeEventID(w, entry.id)
		if err := entry.patch.apply(sse); err != nil {
			return
		}
	}
}

// pumpPatches forwards live patches from the subscriber channel to the SSE
// client until the request context is cancelled or the connection breaks.
func (b *Broadcaster) pumpPatches(
	w http.ResponseWriter,
	sse *sdk.ServerSentEventGenerator,
	ch chan patchEntry,
	r *http.Request,
) {
	var heartbeatC <-chan time.Time
	if b.heartbeat > 0 {
		ticker := time.NewTicker(b.heartbeat)
		defer ticker.Stop()
		heartbeatC = ticker.C
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
		case <-heartbeatC:
			if sse.IsClosed() {
				return
			}
			if err := writeHeartbeat(sse); err != nil {
				return
			}
		}
	}
}

// writeEventID writes the SSE id: field so browsers track the last event for
// automatic reconnection via the Last-Event-ID header. Written directly to the
// ResponseWriter before the SDK writes the event body — both are flushed
// together by the SDK's Send method.
//
// Note: this writes to the raw ResponseWriter, not through the SDK's internal
// writer (sse.w). This is correct when SSE compression is not enabled (the
// default — the Broadcaster creates the SSE generator without compression
// options). If compression support is added to the Broadcaster in the future,
// this function must be updated to write through the SDK's writer path.
func writeEventID(w http.ResponseWriter, id uint64) {
	if id == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "id: %d\n", id)
}

// writeHeartbeat sends a lightweight SSE event to keep the connection alive.
// It uses the SDK's Send method so the heartbeat respects the SSE generator's
// mutex, compression writer (if configured), and ResponseController-based
// flush. The Datastar client ignores unknown event types, making this a safe
// keep-alive signal that resets proxy idle timers without side effects.
func writeHeartbeat(sse *sdk.ServerSentEventGenerator) error {
	return sse.Send(heartbeatEventType, nil)
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
