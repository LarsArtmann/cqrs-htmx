package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestDashboard_SSEReconnectReplay(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	bus := eventtest.NewFakeBus()

	// Seed 3 events to the journal.
	aggID := id.NewAggregateID()
	ref := id.NewStreamRef("Order", aggID)
	events := make([]event.Event, 3)
	for i := range events {
		evt, err := event.New(
			"order.updated",
			aggID,
			"Order",
			event.Version(i+1),
			struct{ Step int }{Step: i + 1},
		)
		if err != nil {
			t.Fatalf("create event %d: %v", i+1, err)
		}

		events[i] = evt
	}

	if err := store.Save(nil, ref, events, event.Version(0)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
		EventBus:    bus,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer d.Close()

	if d.sseStore == nil {
		t.Fatal("sseStore should be built when EventBus + Journal are configured")
	}

	// Reconnect with Last-Event-ID = first event's ID.
	// Should replay events 2 and 3 (strictly after event 1).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/events/stream", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Last-Event-ID", events[0].ID().String())

	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		d.sseHandler(rec, req)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	// Body should contain events 2 and 3, but NOT event 1 (it's the cursor).
	if !strings.Contains(body, events[1].ID().String()) {
		t.Errorf("body should contain replayed event 2 ID %q\nbody:\n%s", events[1].ID().String(), body)
	}

	if !strings.Contains(body, events[2].ID().String()) {
		t.Errorf("body should contain replayed event 3 ID %q\nbody:\n%s", events[2].ID().String(), body)
	}
}

func TestDashboard_SSEInitialBackfill(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	bus := eventtest.NewFakeBus()

	// Seed 2 events.
	aggID := id.NewAggregateID()
	ref := id.NewStreamRef("User", aggID)

	evt1, _ := event.New("user.created", aggID, "User", event.Version(1), struct{}{})
	evt2, _ := event.New("user.updated", aggID, "User", event.Version(2), struct{}{})

	if err := store.Save(nil, ref, []event.Event{evt1, evt2}, event.Version(0)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	d, _ := New(Config{
		EventSource: store,
		Journal:     store,
		EventBus:    bus,
	})
	defer d.Close()

	// First connect — no Last-Event-ID header.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/events/stream", nil)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		d.sseHandler(rec, req)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	// On first connect with no cursor, the store backfills recent events.
	if !strings.Contains(body, evt1.ID().String()) {
		t.Errorf("body should contain backfilled event 1\nbody:\n%s", body)
	}

	if !strings.Contains(body, evt2.ID().String()) {
		t.Errorf("body should contain backfilled event 2\nbody:\n%s", body)
	}
}

func TestDashboard_SSEHeartbeatEmission(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	bus := eventtest.NewFakeBus()

	d, err := New(Config{
		EventSource:          store,
		Journal:              store,
		EventBus:             bus,
		SSEHeartbeatInterval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer d.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/events/stream", nil)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		d.sseHandler(rec, req)
		close(done)
	}()

	// Wait for multiple heartbeat intervals.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	// Heartbeat is a comment frame (line starting with ":").
	heartbeatCount := 0
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, ":") {
			heartbeatCount++
		}
	}

	if heartbeatCount == 0 {
		t.Errorf("expected at least 1 heartbeat comment frame, got 0\nbody:\n%s", body)
	}
}

func TestDashboard_Close(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	bus := eventtest.NewFakeBus()

	d, _ := New(Config{
		EventSource: store,
		Journal:     store,
		EventBus:    bus,
	})

	// Close should not panic.
	d.Close()

	// Double close should be safe (idempotent).
	d.Close()
}
