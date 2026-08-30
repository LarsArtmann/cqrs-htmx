package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
