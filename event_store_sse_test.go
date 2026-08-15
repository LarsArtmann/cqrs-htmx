package cqrshtmx

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

// JournalSSEStore behavior tests moved to transport/journalsse_test.go when
// the implementation moved to the transport sub-package. This file keeps the
// shared test helpers used by the root integration tests.

// testMapper is a minimal EventToSSEMapper for testing.
func testMapper(evt event.Event) sse.Event {
	return sse.Event{
		Event: string(evt.Type()),
		Data:  string(evt.Payload()),
		ID:    sse.NewEventID(evt.ID().String()),
	}
}

// seedEventList builds count uniquely-versioned events on one stream.
func seedEventList(t *testing.T, count int) []event.Event {
	t.Helper()

	aggID, err := id.ParseStreamID(ulid.Make().String())
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

	ref := id.StreamRef{ID: events[0].StreamID(), Type: "test"}
	if err := store.AppendBatch(context.Background(), ref, events); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}
}
