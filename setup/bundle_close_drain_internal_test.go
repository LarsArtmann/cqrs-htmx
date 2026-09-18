package setup

import (
	"log/slog"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-sse"
)

func newDrainTestBundle(t *testing.T) *Bundle {
	t.Helper()

	bundle, err := New(Config{
		Title:      "drain-test",
		SSEPath:    "/sse",
		EventStore: memorystorage.NewMemoryStore(),
		EventBus:   eventtest.NewFakeBus(),
		Logger:     slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if bundle.Broadcaster == nil {
		t.Fatal("Broadcaster must be built when SSEPath is set")
	}

	return bundle
}

func TestBundleClose_DrainsQueuedEvents(t *testing.T) {
	t.Parallel()

	bundle := newDrainTestBundle(t)
	ch := bundle.Broadcaster.Subscribe()

	bundle.Broadcaster.Broadcast(sse.Event{Event: "drain-me", Data: "payload"})

	// The event sits unread in the subscriber buffer; Close must deliver it
	// before closing the channel, not drop it.
	if err := bundle.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case got, ok := <-ch:
		if !ok {
			t.Fatal("subscriber channel closed without delivering the queued event")
		}

		if got.Event != "drain-me" {
			t.Errorf("drained event = %q, want %q", got.Event, "drain-me")
		}
	default:
		t.Fatal("queued event lost across Close — drain did not run")
	}

	if _, ok := <-ch; ok {
		t.Error("subscriber channel should be closed after the buffer drains")
	}
}

func TestBundleClose_DrainTimeoutProceeds(t *testing.T) {
	t.Parallel()

	bundle := newDrainTestBundle(t)
	bundle.sseDrainTimeout = 50 * time.Millisecond
	_ = bundle.Broadcaster.Subscribe()

	// Overflow the subscriber buffer (capacity 64) with an unread backlog so
	// the drain cannot complete inside the shrunk timeout window.
	for range 200 {
		bundle.Broadcaster.Broadcast(sse.Event{Event: "flood", Data: "x"})
	}

	start := time.Now()
	if err := bundle.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("Close took %v — drain deadline not honored", elapsed)
	}
}
