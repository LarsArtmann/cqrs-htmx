package transport

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

// fakeSSEStore is a minimal sse.EventStore for testing ServeDomainEvents.
type fakeSSEStore struct {
	events []sse.Event
}

func (s *fakeSSEStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	if lastID.Get() == "" {
		return s.events, nil
	}

	for i, evt := range s.events {
		if evt.ID.Get() == lastID.Get() {
			return s.events[i+1:], nil
		}
	}

	return nil, nil
}

func TestServeDomainEvents_NilBroadcaster_503(t *testing.T) {
	t.Parallel()

	h := ServeDomainEvents(nil, nil, 0)
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "SSE not available") {
		t.Fatalf("expected default unavailable message, got %q", rec.Body.String())
	}
}

func TestServeDomainEvents_NilBroadcaster_CustomMessage(t *testing.T) {
	t.Parallel()

	h := ServeDomainEvents(nil, nil, 0, WithSSEUnavailableMessage("no bus"))
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "no bus") {
		t.Fatalf("expected custom unavailable message, got %q", rec.Body.String())
	}
}

func TestServeDomainEvents_RetryHintFirstOnWire(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	h := ServeDomainEvents(b, nil, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.HasPrefix(body, "retry: 5000\n\n") {
		t.Errorf("stream should start with the retry hint, got prefix %q", body[:min(len(body), 30)])
	}

	if !strings.Contains(body, "connected") {
		t.Error("body should still contain the connected event")
	}
}

func TestServeDomainEvents_ConnectedAndLivePump(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	h := ServeDomainEvents(b, nil, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	// Give the handler time to subscribe and send "connected".
	time.Sleep(50 * time.Millisecond)

	b.Broadcast(sse.Event{Event: "ping", Data: "hello"})

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, "connected") {
		t.Errorf("body should contain connected event\nbody:\n%s", body)
	}

	if !strings.Contains(body, "hello") {
		t.Errorf("body should contain live-pumped event data\nbody:\n%s", body)
	}
}

func TestServeDomainEvents_ReplayFromStore(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	store := &fakeSSEStore{events: []sse.Event{
		{Event: "replayed", Data: "old-event", ID: sse.NewEventID("evt-1")},
	}}

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

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, "old-event") {
		t.Errorf("body should contain replayed event from store\nbody:\n%s", body)
	}
}

func TestServeDomainEvents_HeartbeatEmission(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	h := ServeDomainEvents(b, nil, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	heartbeatCount := 0

	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, ":") {
			heartbeatCount++
		}
	}

	if heartbeatCount == 0 {
		t.Errorf("expected at least 1 heartbeat comment frame, got 0\nbody:\n%s", body)
	}
}

func TestServeDomainEvents_ReplayCursored(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	store := &fakeSSEStore{events: []sse.Event{
		{Event: "a", Data: "one", ID: sse.NewEventID("1")},
		{Event: "b", Data: "two", ID: sse.NewEventID("2")},
		{Event: "c", Data: "three", ID: sse.NewEventID("3")},
	}}

	h := ServeDomainEvents(b, store, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	req.Header.Set("Last-Event-ID", "1")

	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, "two") {
		t.Errorf("body should contain replayed event after cursor\nbody:\n%s", body)
	}

	if !strings.Contains(body, "three") {
		t.Errorf("body should contain replayed event after cursor\nbody:\n%s", body)
	}

	if strings.Contains(body, "one") {
		t.Errorf("body should NOT contain the cursor event\nbody:\n%s", body)
	}
}

func TestServeDomainEvents_FilteredLive(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	match := func(evt sse.Event) bool { return strings.Contains(evt.Data, `"streamType":"user"`) }
	h := ServeDomainEvents(b, nil, 0, WithSSEFilter(match))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	b.Broadcast(sse.Event{Event: "domain", Data: `{"streamType":"tenant","version":1}`})
	b.Broadcast(sse.Event{Event: "domain", Data: `{"streamType":"user","version":2}`})

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, `"streamType":"user"`) {
		t.Errorf("filtered subscriber must receive the matching event\nbody:\n%s", body)
	}

	if strings.Contains(body, `"streamType":"tenant"`) {
		t.Errorf("filtered subscriber must NOT receive the non-matching event\nbody:\n%s", body)
	}
}

func TestServeDomainEvents_FilteredReplay(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	store := &fakeSSEStore{events: []sse.Event{
		{ID: sse.NewEventID("1"), Event: "domain", Data: `{"streamType":"user","version":1}`},
		{ID: sse.NewEventID("2"), Event: "domain", Data: `{"streamType":"tenant","version":1}`},
		{ID: sse.NewEventID("3"), Event: "domain", Data: `{"streamType":"user","version":2}`},
	}}

	match := func(evt sse.Event) bool { return strings.Contains(evt.Data, `"streamType":"user"`) }
	h := ServeDomainEvents(b, store, 0, WithSSEFilter(match))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	// Give the handler time to finish the (filtered) replay, then disconnect.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not return")
	}

	body := rec.Body.String()

	if got := strings.Count(body, `"streamType":"user"`); got != 2 {
		t.Errorf("filtered replay must deliver exactly the 2 matching events, got %d\nbody:\n%s", got, body)
	}

	if strings.Contains(body, `"streamType":"tenant"`) {
		t.Errorf("filtered replay must NOT deliver the non-matching event\nbody:\n%s", body)
	}
}

// blockingSSEStore holds the first-connect replay window open so a live event
// can be broadcast mid-replay — the exact race the subscribe-before-replay
// order in ServeDomainEvents exists to win.
type blockingSSEStore struct {
	events  []sse.Event
	started chan struct{}
	release chan struct{}
}

func (s *blockingSSEStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	if lastID.Get() == "" {
		close(s.started)
		<-s.release

		return s.events, nil
	}

	return nil, nil
}

func TestServeDomainEvents_ReplayBeforeSubscribe_Ordering(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	store := &blockingSSEStore{
		events: []sse.Event{
			{Event: "replay", Data: "replay-1"},
			{Event: "replay", Data: "replay-2"},
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

	// A live event committed while the replay window is still open must not
	// be lost (the subscription already buffers it) and must be delivered
	// after the older replayed events.
	b.Broadcast(sse.Event{Event: "live", Data: "live-during-replay"})
	close(store.release)

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	for _, want := range []string{"replay-1", "replay-2", "live-during-replay"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q — live event must survive the replay window\nbody:\n%s", want, body)
		}
	}

	if strings.Index(body, "replay-2") > strings.Index(body, "live-during-replay") {
		t.Fatalf("ordering violated: replayed events must precede the live event\nbody:\n%s", body)
	}
}

func TestServeDomainEvents_HeartbeatJoinOnExit(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	h := ServeDomainEvents(b, nil, 5*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	// The heartbeat goroutine MUST be joined before ServeHTTP returns: a
	// heartbeat write racing handler teardown is a data race, and net/http
	// forbids touching the ResponseWriter after return. A missing join hangs
	// here until the test timeout; the -race detector catches the write.
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return after cancel — heartbeat goroutine not joined")
	}
}

func TestServeDomainEvents_BroadcasterCloseMidStream(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	h := ServeDomainEvents(b, nil, 0)

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(t.Context())
	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	// Server shutdown closes the broadcaster while the client is connected:
	// the subscriber channel closes, the handler must return promptly and
	// cleanly (no panic, no hang).
	b.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return after broadcaster close")
	}

	if !strings.Contains(rec.Body.String(), "connected") {
		t.Errorf("body should still contain the connected event\nbody:\n%s", rec.Body.String())
	}
}

const fanoutBroadcastMarker = "to-everyone"

func TestServeDomainEvents_ConcurrentClients(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	h := ServeDomainEvents(b, nil, 0)

	const clients = 16

	recorders := make([]*httptest.ResponseRecorder, clients)
	delivered := make([]chan struct{}, clients)
	dones := make([]chan struct{}, clients)

	for i := range clients {
		req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(t.Context())
		rec := httptest.NewRecorder()
		recorders[i] = rec

		delivered[i] = make(chan struct{})
		signal := &deliverySignalWriter{ResponseRecorder: rec, delivered: delivered[i]}

		done := make(chan struct{})
		dones[i] = done

		go func() {
			h.ServeHTTP(signal, req)
			close(done)
		}()
	}

	// The handler subscribes before sending "connected", so a full
	// SubscriberCount proves every client receives the fan-out. The former
	// fixed sleep raced slow goroutines under load: they subscribed after
	// the broadcast and legitimately missed it.
	waitForSubscribers(t, b, clients)

	b.Broadcast(sse.Event{Event: "fanout", Data: fanoutBroadcastMarker})

	for i := range clients {
		select {
		case <-delivered[i]:
		case <-time.After(5 * time.Second):
			t.Fatalf("client %d never received the broadcast", i)
		}
	}

	for i := range clients {
		select {
		case <-dones[i]:
			t.Fatalf("client %d disconnected early", i)
		default:
		}
	}

	// Cancel all clients by closing the broadcaster — every handler returns.
	b.Close()

	for i := range clients {
		select {
		case <-dones[i]:
		case <-time.After(5 * time.Second):
			t.Fatalf("client %d did not return after broadcaster close", i)
		}

		body := recorders[i].Body.String()
		if !strings.Contains(body, fanoutBroadcastMarker) {
			t.Errorf("client %d missed the broadcast\nbody:\n%s", i, body)
		}
	}
}

// deliverySignalWriter closes delivered once the broadcast marker has
// streamed through. The sniff buffer is written and read only by the handler
// goroutine; the channel close is the only cross-goroutine handoff, so the
// test can wait on it race-free.
type deliverySignalWriter struct {
	*httptest.ResponseRecorder

	delivered chan struct{}
	seen      []byte
	once      sync.Once
}

func (w *deliverySignalWriter) Write(p []byte) (int, error) {
	w.seen = append(w.seen, p...)
	if bytes.Contains(w.seen, []byte(fanoutBroadcastMarker)) {
		w.once.Do(func() { close(w.delivered) })
	}

	return w.ResponseRecorder.Write(p)
}

func waitForSubscribers(t *testing.T, b *sse.Broadcaster[sse.Event], want int) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for b.SubscriberCount() < want {
		if time.Now().After(deadline) {
			t.Fatalf("only %d/%d clients subscribed within 5s", b.SubscriberCount(), want)
		}

		time.Sleep(2 * time.Millisecond)
	}
}

func TestServeDomainEvents_MaxReplayCapsBackfill(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	events := make([]sse.Event, 0, 10)
	for i := range 10 {
		events = append(events, sse.Event{
			Event: "replayed",
			Data:  fmt.Sprintf("evt-%d", i),
			ID:    sse.NewEventID(fmt.Sprintf("id-%d", i)),
		})
	}

	h := ServeDomainEvents(b, &fakeSSEStore{events: events}, 0, WithSSEMaxReplay(3))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	// The cap keeps the MOST RECENT 3 events (tail semantics).
	for _, want := range []string{"evt-7", "evt-8", "evt-9"} {
		if !strings.Contains(body, want) {
			t.Errorf("capped replay missing the recent event %q\nbody:\n%s", want, body)
		}
	}

	for _, dropped := range []string{"evt-0", "evt-6"} {
		if strings.Contains(body, dropped) {
			t.Errorf("capped replay must drop the old event %q\nbody:\n%s", dropped, body)
		}
	}
}

func TestServeDomainEvents_ReplayedEventsCarryRetry(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	store := &fakeSSEStore{events: []sse.Event{
		{Event: "replayed", Data: "backfill", ID: sse.NewEventID("r-1")},
		{Event: "replayed", Data: "backfill", ID: sse.NewEventID("r-2"), Retry: 9999},
	}}

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

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	// Unset retry gets the handler default; an explicit retry survives.
	if got := strings.Count(body, "retry: 5000\n"); got < 1 {
		t.Errorf("replayed events without a retry hint must carry the default\nbody:\n%s", body)
	}

	if !strings.Contains(body, "retry: 9999\n") {
		t.Errorf("an explicitly set retry must survive the replay wrapper\nbody:\n%s", body)
	}
}

func TestSSEOptions_MatchesFunctionalOptions(t *testing.T) {
	t.Parallel()

	match := func(sse.Event) bool { return true }

	structCfg := serveDomainEventsConfig{}
	structOpts := SSEOptions{
		LogPrefix:          "app",
		UnavailableMessage: "gone",
		Filter:             match,
		MaxReplay:          7,
	}.Options()

	for _, opt := range structOpts {
		opt(&structCfg)
	}

	funcCfg := serveDomainEventsConfig{}
	for _, opt := range []ServeDomainEventsOption{
		WithSSELogPrefix("app"),
		WithSSEUnavailableMessage("gone"),
		WithSSEFilter(match),
		WithSSEMaxReplay(7),
	} {
		opt(&funcCfg)
	}

	if structCfg.logPrefix != funcCfg.logPrefix ||
		structCfg.unavailableMessage != funcCfg.unavailableMessage ||
		structCfg.maxReplay != funcCfg.maxReplay {
		t.Fatalf("struct and functional options diverged: %+v vs %+v", structCfg, funcCfg)
	}

	// Function values are not comparable; both must be non-nil.
	if structCfg.filter == nil || funcCfg.filter == nil {
		t.Fatal("both configurations must carry the filter predicate")
	}

	// The zero struct contributes no options at all.
	if got := len(SSEOptions{}.Options()); got != 0 {
		t.Fatalf("zero SSEOptions must produce no options, got %d", got)
	}
}

func TestServeDomainEvents_FilterCapCountsMatchesOnly(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	// A burst of 8 non-matching events, then 3 matching ones. With a cap of 3
	// applied BEFORE the filter, the entire budget would be consumed by the
	// non-matching burst and the scoped feed would receive zero backfill.
	events := make([]sse.Event, 0, 11)
	for i := range 8 {
		events = append(events, sse.Event{
			Event: "other",
			Data:  fmt.Sprintf("other-%d", i),
			ID:    sse.NewEventID(fmt.Sprintf("id-o-%d", i)),
		})
	}

	for i := range 3 {
		events = append(events, sse.Event{
			Event: "wanted",
			Data:  fmt.Sprintf("want-%d", i),
			ID:    sse.NewEventID(fmt.Sprintf("id-w-%d", i)),
		})
	}

	h := ServeDomainEvents(b, &fakeSSEStore{events: events}, 0,
		WithSSEFilter(func(evt sse.Event) bool { return evt.Event == "wanted" }),
		WithSSEMaxReplay(3),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	for _, want := range []string{"want-0", "want-1", "want-2"} {
		if !strings.Contains(body, want) {
			t.Errorf("scoped backfill missing %q — cap was consumed by non-matching events\nbody:\n%s", want, body)
		}
	}

	if strings.Contains(body, "other-") {
		t.Errorf("non-matching event leaked into filtered backfill\nbody:\n%s", body)
	}
}

func TestServeDomainEvents_PanickingFilterPredicateDoesNotKillReplay(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	events := []sse.Event{
		{Event: "boom", Data: "panics-the-predicate", ID: sse.NewEventID("id-1")},
		{Event: "safe", Data: "fine", ID: sse.NewEventID("id-2")},
	}

	pred := func(evt sse.Event) bool {
		if evt.Event == "boom" {
			panic("consumer predicate bug")
		}

		return evt.Event == "safe"
	}

	h := ServeDomainEvents(b, &fakeSSEStore{events: events}, 0, WithSSEFilter(pred))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return — panicking replay predicate killed the goroutine")
	}

	if !strings.Contains(rec.Body.String(), "fine") {
		t.Errorf("non-panicking event missing from backfill\nbody:\n%s", rec.Body.String())
	}
}
