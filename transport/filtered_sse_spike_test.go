package transport_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/v4/transport"
	"github.com/larsartmann/go-sse"
)

// filteredSSEStore adapts an sse.EventStore to sse.FilteredEventStore with a
// stream-type predicate applied at replay time.
type filteredSSEStore struct {
	inner sse.EventStore
	match func(sse.Event) bool
}

func (f *filteredSSEStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	return f.inner.EventsAfter(lastID)
}

func (f *filteredSSEStore) EventsAfterFiltered(
	lastID sse.EventID,
	pred func(sse.Event) bool,
) ([]sse.Event, error) {
	events, err := f.inner.EventsAfter(lastID)
	if err != nil {
		return nil, err
	}

	out := make([]sse.Event, 0, len(events))
	for _, evt := range events {
		if pred(evt) {
			out = append(out, evt)
		}
	}

	return out, nil
}

// TestSpikeFilteredSSE_SessionGatingBasis is the SPIKE behind the open /sse
// authz-posture decision: it proves the mechanism a stream-type-scoped SSE
// endpoint would use — go-sse's SubscribeFilter for the live path and
// ReplayFiltered for the reconnect/backfill path — so the decision can reason
// about a concrete mechanism instead of an abstraction.
//
// Findings recorded in TODO_LIST P2 (the /sse posture decision):
//   - live filtering: Broadcast events carry the stream type in the payload;
//     a predicate on the envelope's streamType field is all SubscribeFilter needs.
//   - replay filtering: wrapping the journal-backed store in
//     EventsAfterFiltered is mechanical (this file does exactly that).
//   - no go-sse changes are required; a filtered variant of
//     transport.ServeDomainEvents is an additive option (e.g.
//     WithSSEFilter(pred)) away.
func TestSpikeFilteredSSE_SessionGatingBasis(t *testing.T) {
	hub := sse.NewBroadcaster[sse.Event]()

	// Backfill journal: two matching events and one non-matching (tenant).
	// IDs are chronological.
	store := &filteredSSEStore{
		inner: &sliceStore{events: []sse.Event{
			{ID: mustEventID("1"), Event: "domain", Data: `{"streamType":"user","version":1}`},
			{ID: mustEventID("2"), Event: "domain", Data: `{"streamType":"tenant","version":1}`},
			{ID: mustEventID("3"), Event: "domain", Data: `{"streamType":"user","version":2}`},
		}},
		match: func(evt sse.Event) bool {
			return strings.Contains(evt.Data, `"streamType":"user"`)
		},
	}

	stream := sse.NewStream(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/sse", nil))
	// The recorder client never reads; bound the replay write window.
	go func() {
		time.Sleep(2 * time.Second)

		_ = stream.Close()
	}()

	n, err := sse.ReplayFiltered(stream, store, sse.EventID{}, store.match)
	if err != nil {
		t.Fatalf("ReplayFiltered: %v", err)
	}

	if n != 2 {
		t.Fatalf("ReplayFiltered delivered %d events, want 2 (the tenant event must be excluded)", n)
	}

	// Live path: a filtered subscriber receives only matching broadcasts.
	ch := hub.SubscribeFilter(store.match)
	defer hub.Unsubscribe(ch)

	hub.Broadcast(sse.Event{Event: "domain", Data: `{"streamType":"tenant","version":2}`})
	hub.Broadcast(sse.Event{Event: "domain", Data: `{"streamType":"user","version":3}`})

	select {
	case evt := <-ch:
		if !strings.Contains(evt.Data, `"streamType":"user"`) {
			t.Fatalf("filtered subscriber received non-matching event: %s", evt.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("filtered subscriber never received the matching live event")
	}
	// The tenant broadcast must NOT have been delivered: the channel should
	// be empty (only one event was consumed above).
	select {
	case evt := <-ch:
		t.Fatalf("filtered subscriber received a second, non-matching event: %s", evt.Data)
	case <-time.After(100 * time.Millisecond):
	}
}

// --- minimal in-memory stores for the spike ---

type sliceStore struct{ events []sse.Event }

func (s *sliceStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	out := []sse.Event{}

	for _, evt := range s.events {
		if lastID.IsZero() || evt.ID.String() > lastID.String() {
			out = append(out, evt)
		}
	}

	return out, nil
}

func mustEventID(s string) sse.EventID {
	id, err := sse.ParseEventID(s)
	if err != nil {
		panic(err)
	}

	return id
}

// compile-time proof the spike store satisfies the go-sse filtered interface.
var _ sse.FilteredEventStore = (*filteredSSEStore)(nil)

// keep the transport import honest if the spike evolves away from it.
var _ = transport.ServeDomainEvents
