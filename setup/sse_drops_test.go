package setup

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-sse"
)

// The SSE drop-observability wiring (A2): the bundle's shared hub counts
// slow-consumer drops, the counter is exposed for polling, and a bundle
// without SSE surfaces nothing.

func TestSSEDrops_WiredOnSharedHub(t *testing.T) {
	t.Parallel()

	bundle := newDrainTestBundle(t)
	t.Cleanup(func() { _ = bundle.Close() })

	if bundle.SSEDrops == nil {
		t.Fatal("SSEDrops must be built alongside the shared broadcaster")
	}
	if bundle.SSEDrops.Total() != 0 {
		t.Errorf("fresh bundle drop count = %d, want 0", bundle.SSEDrops.Total())
	}

	ch := bundle.Broadcaster.Subscribe()
	defer bundle.Broadcaster.Unsubscribe(ch)

	// Default buffer is 64: fill it, then overflow once.
	for range 65 {
		bundle.Broadcaster.Broadcast(sse.Event{Event: "overflow"})
	}

	if got := bundle.SSEDrops.Total(); got != 1 {
		t.Errorf("drop count after overflow = %d, want 1", got)
	}

	snapshot := bundle.SSEDrops.Snapshot(bundle.Broadcaster)
	if snapshot.Drops != 1 || snapshot.Health.SubscriberCount != 1 {
		t.Errorf("snapshot = %+v, want drops=1 subscribers=1", snapshot)
	}
}

func TestSSEDrops_NilWithoutSSEPath(t *testing.T) {
	t.Parallel()

	bundle, err := New(Config{
		Title:      "no-sse",
		EventStore: memorystorage.NewMemoryStore(),
		EventBus:   eventtest.NewFakeBus(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	if bundle.SSEDrops != nil {
		t.Error("SSEDrops must be nil when neither SSEPath nor DataStarPath is set")
	}
}
