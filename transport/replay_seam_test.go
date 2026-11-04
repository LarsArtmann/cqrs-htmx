package transport

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/dedup/v4"
	"github.com/larsartmann/go-sse"
)

// seamEvents builds n ID-bearing replay events (e0..e{n-1}); the IDs are what
// the seam recorder deduplicates on, the data is what the wire shows.
func seamEvents(n int) []sse.Event {
	events := make([]sse.Event, n)
	for i := range n {
		events[i] = sse.Event{
			ID: sse.NewEventID(fmt.Sprintf("e%d", i)), Event: "replay", Data: fmt.Sprintf("d%d", i),
		}
	}

	return events
}

// assertSeamWindow fails when the dedup verdict for IDs e{from}..e{to}
// diverges from wantDup — the one helper both eviction directions assert
// through, so the bounded-memory tradeoff is pinned in a single place.
func assertSeamWindow(t *testing.T, ring *dedup.Ring[string], from, to int, wantDup bool) {
	t.Helper()

	for i := from; i < to; i++ {
		evt := sse.Event{ID: sse.NewEventID(fmt.Sprintf("e%d", i))}
		if got := isReplayDuplicate(evt, ring); got != wantDup {
			t.Errorf("seam dedup for e%d = %v, want %v", i, got, wantDup)
		}
	}
}

// TestReplaySeam_RingBoundedAndTailExact pins the memory-bounded seam
// recorder: the ring keeps only the NEWEST replay IDs — the realistic overlap
// window, since events published during the replay window are always the tail
// of the replay sequence — and evicts the oldest beyond capacity. The tiny
// ring here forces the eviction on purpose; production geometry uses
// dedup.DefaultCapacity (1024), an order of magnitude above the live channel
// buffer, so eviction cannot touch a real overlap.
func TestReplaySeam_RingBoundedAndTailExact(t *testing.T) {
	t.Parallel()

	const (
		total    = 10
		capacity = 4
	)

	t.Run("replayAdjustedStore records what it returns", func(t *testing.T) {
		t.Parallel()

		ring := dedup.NewRing[string](capacity)
		store := &replayAdjustedStore{inner: &fakeSSEStore{events: seamEvents(total)}, replayed: ring}

		got, err := store.EventsAfter(sse.EventID{})
		if err != nil {
			t.Fatalf("EventsAfter: %v", err)
		}

		if len(got) != total {
			t.Fatalf("no cap set: want all %d events, got %d", total, len(got))
		}

		if ring.Len() != capacity {
			t.Fatalf("ring must bound at capacity %d, holds %d", capacity, ring.Len())
		}

		assertSeamWindow(t, ring, total-capacity, total, true)
		assertSeamWindow(t, ring, 0, total-capacity, false)
	})

	t.Run("filteredReplayStore records matches only", func(t *testing.T) {
		t.Parallel()

		all := seamEvents(total)
		all = append(all, sse.Event{Event: "replay", Data: "no-id"}) // unidentified: never dedups

		ring := dedup.NewRing[string](capacity)
		store := &filteredReplayStore{
			inner: &fakeSSEStore{events: all}, match: func(sse.Event) bool { return true },
			replayed: ring,
		}

		got, err := store.EventsAfterFiltered(sse.EventID{}, store.match)
		if err != nil {
			t.Fatalf("EventsAfterFiltered: %v", err)
		}

		if len(got) != total+1 {
			t.Fatalf("match-all pred: want %d events, got %d", total+1, len(got))
		}

		if ring.Len() != capacity {
			t.Fatalf("ring must bound at capacity %d, holds %d", capacity, ring.Len())
		}

		assertSeamWindow(t, ring, total-capacity, total, true)
		assertSeamWindow(t, ring, 0, total-capacity, false)
	})
}

// TestReplaySeam_NilRingMeansNoReplay pins the nil-safety contract the seam
// relies on: a nil ring records nothing and deduplicates nothing. The seam
// code has no explicit nil guard — upstream dedup.Ring.Add/Has are nil-safe —
// so an upstream regression must fail HERE, not on a customer's connection.
func TestReplaySeam_NilRingMeansNoReplay(t *testing.T) {
	t.Parallel()

	var ring *dedup.Ring[string]

	recordReplayedIDs(ring, seamEvents(3))

	if ring.Len() != 0 {
		t.Fatalf("nil ring must stay empty, holds %d", ring.Len())
	}

	for i := range 3 {
		evt := sse.Event{ID: sse.NewEventID(fmt.Sprintf("e%d", i))}
		if isReplayDuplicate(evt, ring) {
			t.Errorf("nil ring means no replay: e%d must not be a duplicate", i)
		}
	}
}

// TestServeDomainEvents_ReplayLiveSeam_NoDuplicateDelivery exercises the full
// seam through the handler: events broadcast while the replay window is open
// carry the same IDs the replay then writes — the live pump must drop each
// duplicate exactly once, and a genuinely new live event must still flow.
func TestServeDomainEvents_ReplayLiveSeam_NoDuplicateDelivery(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	store := &blockingSSEStore{
		events: []sse.Event{
			{ID: sse.NewEventID("r1"), Event: "replay", Data: "seam-1"},
			{ID: sse.NewEventID("r2"), Event: "replay", Data: "seam-2"},
		},
		started: make(chan struct{}),
		release: make(chan struct{}),
	}

	h := ServeDomainEvents(b, store, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	<-store.started

	// Same IDs the blocked replay is about to write: the subscription already
	// buffered them, so without seam dedup the client would see each twice.
	b.Broadcast(sse.Event{ID: sse.NewEventID("r1"), Event: "live", Data: "seam-1"})
	b.Broadcast(sse.Event{ID: sse.NewEventID("r2"), Event: "live", Data: "seam-2"})
	b.Broadcast(sse.Event{ID: sse.NewEventID("l3"), Event: "live", Data: "fresh-3"})

	close(store.release)

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	for _, data := range []string{"seam-1", "seam-2"} {
		if n := strings.Count(body, data); n != 1 {
			t.Fatalf("%q must be delivered exactly once at the seam, got %d\nbody:\n%s", data, n, body)
		}
	}

	if !strings.Contains(body, "fresh-3") {
		t.Fatalf("a new live event must still flow through the seam\nbody:\n%s", body)
	}
}
