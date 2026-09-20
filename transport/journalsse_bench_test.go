package transport

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-sse"
	"github.com/oklog/ulid/v2"
)

// benchSeedJournal appends count events to a fresh memory journal and returns
// it together with the event ID at the midpoint (a realistic reconnect cursor).
func benchSeedJournal(b *testing.B, count int) (*memory.MemoryStore, sse.EventID) {
	b.Helper()

	store := memory.NewMemoryStore()

	streamID, err := id.ParseStreamID(ulid.Make().String())
	if err != nil {
		b.Fatalf("parse stream ID: %v", err)
	}

	events := make([]event.Event, 0, count)
	for i := 1; i <= count; i++ {
		evt, err := event.New(
			event.Type(fmt.Sprintf("bench.event.%d", i)),
			streamID,
			"bench",
			event.Version(i),
			fmt.Sprintf(`{"seq":%d,"padding":"0123456789abcdef"}`, i),
		)
		if err != nil {
			b.Fatalf("create event %d: %v", i, err)
		}

		events = append(events, evt)
	}

	ref := id.StreamRef{ID: streamID, Type: "bench"}
	if err := store.AppendBatch(context.Background(), ref, events); err != nil {
		b.Fatalf("AppendBatch: %v", err)
	}

	mid := events[count/2]

	return store, sse.NewEventID(mid.ID().String())
}

// BenchmarkJournalSSEStore_EventsAfter measures the replay-read cost on large
// journals — the operation every SSE reconnect performs. Two sub-benches pin
// both access patterns: the first-connect backfill (no cursor, tail-capped)
// and the reconnect replay from a midpoint cursor.
func BenchmarkJournalSSEStore_EventsAfter(b *testing.B) {
	b.Run("first-connect", func(b *testing.B) {
		store, _ := benchSeedJournal(b, 10_000)
		sseStore := NewJournalSSEStore(store, DomainEventToSSE)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			events, err := sseStore.EventsAfter(sse.EventID{})
			if err != nil {
				b.Fatalf("EventsAfter: %v", err)
			}

			if len(events) > DefaultMaxReplay {
				b.Fatalf("replay exceeded DefaultMaxReplay: %d", len(events))
			}
		}
	})

	b.Run("reconnect-midpoint", func(b *testing.B) {
		store, cursor := benchSeedJournal(b, 10_000)
		sseStore := NewJournalSSEStore(store, DomainEventToSSE)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			events, err := sseStore.EventsAfter(cursor)
			if err != nil {
				b.Fatalf("EventsAfter: %v", err)
			}

			if len(events) == 0 {
				b.Fatal("replay from a live cursor must return events")
			}
		}
	})
}
