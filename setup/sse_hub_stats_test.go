package setup

import (
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

// TestSSEHubStatsAdapter pins the setup-side adapter that feeds
// adminui.SSEHubStatsSource: live subscriber counts off the shared hub, the
// lifetime drop count off the wired DropStats, and zero-safety on a bundle
// without a hub.
func TestSSEHubStatsAdapter(t *testing.T) {
	bundle, err := New(Config{Title: "hubstats", SSEPath: "/sse"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	if bundle.Broadcaster == nil || bundle.SSEDrops == nil {
		t.Fatal("SSEPath must wire Broadcaster and SSEDrops")
	}

	// The flooded, unread subscriber cannot drain; shrink Close's drain
	// budget so the test spends milliseconds, not the 5s default, waiting
	// for it to time out (same precedent as the Close-drain internal test).
	bundle.sseDrainTimeout = 50 * time.Millisecond

	stats := sseHubStats{bundle: bundle}

	if got := stats.SubscriberCount(); got != 0 {
		t.Errorf("SubscriberCount with no subscribers = %d, want 0", got)
	}

	// Overflow the subscriber buffer (capacity 64) with an unread backlog so
	// real OnDrop callbacks fire through the hub into the wired counter.
	_ = bundle.Broadcaster.Subscribe()
	for range 70 {
		bundle.Broadcaster.Broadcast(sse.Event{Event: "flood", Data: "x"})
	}

	if got := stats.SubscriberCount(); got != 1 {
		t.Errorf("SubscriberCount with one subscriber = %d, want 1", got)
	}

	if got := stats.DroppedEvents(); got < 6 {
		t.Errorf("DroppedEvents after 70 unread broadcasts into a 64 buffer = %d, want >= 6", got)
	}
}

// TestSSEHubStatsAdapterZeroSafeOnHublessBundle: the adapter must read as
// zeros on a bundle without a hub (defense in depth — the wiring is gated on
// the SSE/DataStar paths, but the adapter itself must never panic).
func TestSSEHubStatsAdapterZeroSafeOnHublessBundle(t *testing.T) {
	bundle, err := New(Config{Title: "hubstats-bare"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	stats := sseHubStats{bundle: bundle}

	if got := stats.SubscriberCount(); got != 0 {
		t.Errorf("SubscriberCount on hubless bundle = %d, want 0", got)
	}

	if got := stats.DroppedEvents(); got != 0 {
		t.Errorf("DroppedEvents on hubless bundle = %d, want 0", got)
	}
}
