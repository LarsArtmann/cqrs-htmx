package cqrshtmx

import (
	"testing"

	"github.com/larsartmann/go-sse"
)

// The drop-observability contract (A2): a full subscriber buffer must be
// COUNTED, not silent — once per full subscriber per broadcast, via an
// atomic increment that respects go-sse's in-lock callback contract.

func TestDropStats_OptionCountsDroppedEvents(t *testing.T) {
	drops := NewDropStats()
	b := NewBroadcaster(
		drops.Option(),
		sse.WithBufferSize[sse.Event](1),
	)
	defer b.Close()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// First broadcast fills the buffer slot; the next two find it full.
	for range 3 {
		b.Broadcast(sse.Event{Event: "itemCreated", Data: "x"})
	}

	if got := drops.Total(); got != 2 {
		t.Errorf("drops.Total() = %d, want 2 (first broadcast buffered, two dropped)", got)
	}
}

func TestDropStats_AttachWiresExistingBroadcaster(t *testing.T) {
	drops := NewDropStats()
	b := NewBroadcaster(sse.WithBufferSize[sse.Event](1))
	defer b.Close()

	drops.Attach(b)

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Broadcast(sse.Event{Event: "e"})
	b.Broadcast(sse.Event{Event: "e"})

	if got := drops.Total(); got != 1 {
		t.Errorf("drops.Total() = %d, want 1 (second broadcast dropped)", got)
	}
}

func TestDropStats_SnapshotPairsHealthWithDrops(t *testing.T) {
	drops := NewDropStats()
	b := NewBroadcaster(drops.Option(), sse.WithBufferSize[sse.Event](1))
	defer b.Close()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Broadcast(sse.Event{Event: "a"})
	b.Broadcast(sse.Event{Event: "b"})

	snapshot := drops.Snapshot(b)
	if snapshot.Health.SubscriberCount != 1 {
		t.Errorf("snapshot.SubscriberCount = %d, want 1", snapshot.Health.SubscriberCount)
	}
	if snapshot.Health.BufferSize != 1 {
		t.Errorf("snapshot.BufferSize = %d, want 1", snapshot.Health.BufferSize)
	}
	if snapshot.Drops != 1 {
		t.Errorf("snapshot.Drops = %d, want 1", snapshot.Drops)
	}
}

func TestDropStats_MultipleFullSubscribersCountPerSubscriber(t *testing.T) {
	drops := NewDropStats()
	b := NewBroadcaster(drops.Option(), sse.WithBufferSize[sse.Event](1))
	defer b.Close()

	ch1 := b.Subscribe()
	defer b.Unsubscribe(ch1)
	ch2 := b.Subscribe()
	defer b.Unsubscribe(ch2)

	b.Broadcast(sse.Event{Event: "a"}) // both buffer slots fill
	b.Broadcast(sse.Event{Event: "b"}) // both full: 2 drops, same message

	if got := drops.Total(); got != 2 {
		t.Errorf("drops.Total() = %d, want 2 (once per full subscriber per broadcast)", got)
	}
}
