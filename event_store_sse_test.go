package cqrshtmx

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/oklog/ulid/v2"
)

// testMapper is a minimal EventToSSEMapper for testing.
func testMapper(evt event.Event) SSEEvent {
	return SSEEvent{
		Event: string(evt.Type()),
		Data:  string(evt.Payload()),
		ID:    NewSSEEventID(evt.ID().String()),
	}
}

func TestJournalSSEStore_EventsAfterEmpty(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	sse := NewJournalSSEStore(store, testMapper)

	result := sse.EventsAfter(SSEEventID{})
	if len(result) != 0 {
		t.Fatalf("expected 0 events from empty store, got %d", len(result))
	}
}

func TestJournalSSEStore_EventsAfterAll(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedEventList(t, 5)
	appendEvents(t, store, events)

	sse := NewJournalSSEStore(store, testMapper)

	result := sse.EventsAfter(SSEEventID{})
	if len(result) != 5 {
		t.Fatalf("expected 5 events, got %d", len(result))
	}

	// Verify ascending order (events are stored in order)
	for i, evt := range result {
		expected := events[i].ID().String()
		if evt.ID.Get() != expected {
			t.Errorf("event[%d]: expected ID %q, got %q", i, expected, evt.ID.Get())
		}
	}
}

func TestJournalSSEStore_EventsAfterCursor(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedEventList(t, 5)
	appendEvents(t, store, events)

	sse := NewJournalSSEStore(store, testMapper)

	// Replay after event 3 — should get events 4 and 5
	cursor := events[2].ID().String()

	result := sse.EventsAfter(NewSSEEventID(cursor))
	if len(result) != 2 {
		t.Fatalf("expected 2 events after cursor, got %d", len(result))
	}

	if result[0].ID.Get() != events[3].ID().String() {
		t.Errorf("expected first event ID %q, got %q", events[3].ID().String(), result[0].ID.Get())
	}
}

func TestJournalSSEStore_EventsAfterLastEvent(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedEventList(t, 3)
	appendEvents(t, store, events)

	sse := NewJournalSSEStore(store, testMapper)

	// Cursor at the last event — should get nothing
	cursor := events[2].ID().String()

	result := sse.EventsAfter(NewSSEEventID(cursor))
	if len(result) != 0 {
		t.Fatalf("expected 0 events after last, got %d", len(result))
	}
}

func TestJournalSSEStore_EventsAfterNotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedEventList(t, 3)
	appendEvents(t, store, events)

	sse := NewJournalSSEStore(store, testMapper)

	// Non-existent cursor — upstream ReadFrom returns all events from beginning.
	// This matches the upstream behavior (unknown cursor = no filter).
	result := sse.EventsAfter(NewSSEEventID(ulid.Make().String()))
	if len(result) != 3 {
		t.Fatalf("expected 3 events for unknown cursor (upstream returns all), got %d", len(result))
	}
}

func TestJournalSSEStore_EventsAfterInvalidCursor(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedEventList(t, 3)
	appendEvents(t, store, events)

	sse := NewJournalSSEStore(store, testMapper)

	// Invalid ULID — should return empty
	result := sse.EventsAfter(NewSSEEventID("not-a-valid-ulid"))
	if len(result) != 0 {
		t.Fatalf("expected 0 events for invalid cursor, got %d", len(result))
	}
}

func TestJournalSSEStore_MaxReplay(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedEventList(t, 10)
	appendEvents(t, store, events)

	sse := NewJournalSSEStore(store, testMapper, WithMaxReplay(3))

	// No cursor — should return last 3 events only
	result := sse.EventsAfter(SSEEventID{})
	if len(result) != 3 {
		t.Fatalf("expected 3 events with maxReplay=3, got %d", len(result))
	}

	// Should be events 8, 9, 10
	if result[0].ID.Get() != events[7].ID().String() {
		t.Errorf("expected first event ID %q, got %q", events[7].ID().String(), result[0].ID.Get())
	}
}

func TestJournalSSEStore_SeekableUsed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	sse := NewJournalSSEStore(store, testMapper)

	if sse.seekable == nil {
		t.Fatal("expected SeekableJournal to be detected from memory store")
	}
}

func TestJournalSSEStore_FullScanFallback(t *testing.T) {
	t.Parallel()
	events := seedEventList(t, 5)
	journal := &journalOnlyStore{events: events}

	sse := NewJournalSSEStore(journal, testMapper)

	if sse.seekable != nil {
		t.Fatal("expected SeekableJournal to NOT be detected from journal-only store")
	}

	// Test replay after cursor 3 — should get 4, 5
	cursor := events[2].ID().String()

	result := sse.EventsAfter(NewSSEEventID(cursor))
	if len(result) != 2 {
		t.Fatalf("expected 2 events from full-scan fallback, got %d", len(result))
	}

	if result[0].ID.Get() != events[3].ID().String() {
		t.Errorf("expected first event ID %q, got %q", events[3].ID().String(), result[0].ID.Get())
	}
}

func TestJournalSSEStore_PanicsOnNil(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil journal")
		}
	}()

	NewJournalSSEStore(nil, testMapper)
}

func TestJournalSSEStore_PanicsOnNilMapper(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil mapper")
		}
	}()

	NewJournalSSEStore(store, nil)
}

func TestJournalSSEStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedEventList(t, 100)
	appendEvents(t, store, events)

	sse := NewJournalSSEStore(store, testMapper)

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			for range 50 {
				_ = sse.EventsAfter("")
				if idx > 0 {
					_ = sse.EventsAfter(events[idx%50].ID().String())
				}
			}
		}(i)
	}

	wg.Wait()
}

// --- Helpers ---

func seedEventList(t *testing.T, count int) []event.Event {
	t.Helper()

	aggID, err := id.ParseAggregateID(ulid.Make().String())
	if err != nil {
		t.Fatalf("parse aggregate ID: %v", err)
	}

	events := make([]event.Event, 0, count)
	for i := 1; i <= count; i++ {
		evt, err := event.New(
			event.Type(fmt.Sprintf("test.event.%d", i)),
			aggID,
			"test",
			event.Version(i),
			fmt.Sprintf(`{"id":%d}`, i),
		)
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}

		events = append(events, evt)
	}

	return events
}

func appendEvents(t *testing.T, store *memory.MemoryStore, events []event.Event) {
	t.Helper()

	ref := id.StreamRef{ID: events[0].AggregateID(), Type: "test"}
	if err := store.AppendBatch(context.Background(), ref, events); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}
}

// journalOnlyStore implements event.Journal but NOT event.SeekableJournal,
// for testing the fallback path.
type journalOnlyStore struct {
	events []event.Event
}

func (j *journalOnlyStore) ReadAll(ctx context.Context) ([]event.Event, error) {
	result := make([]event.Event, len(j.events))
	copy(result, j.events)

	return result, nil
}
