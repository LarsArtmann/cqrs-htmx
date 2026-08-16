package transport

import (
	"encoding/json/v2"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-sse"
	"github.com/oklog/ulid/v2"
)

func TestDomainEventToSSE(t *testing.T) {
	t.Parallel()

	aggID, err := id.ParseStreamID(ulid.Make().String())
	if err != nil {
		t.Fatalf("parse stream ID: %v", err)
	}

	wantTime := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)

	evt, err := event.New(
		event.Type("user.registered"),
		aggID,
		"user",
		event.Version(3),
		`{"email":"a@example.com"}`,
		event.WithOccurredAt(wantTime),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	sseEvt := DomainEventToSSE(evt)

	if sseEvt.Event != "event" {
		t.Errorf("sse event name: got %q, want %q", sseEvt.Event, "event")
	}

	if sseEvt.ID.Get() != evt.ID().String() {
		t.Errorf("sse event ID: got %q, want %q", sseEvt.ID.Get(), evt.ID().String())
	}

	if sseEvt.Data == "" {
		t.Fatal("sse event Data must not be empty")
	}

	var payload EventPayload
	if err := json.Unmarshal([]byte(sseEvt.Data), &payload); err != nil {
		t.Fatalf("unmarshal SSE payload: %v", err)
	}

	if payload.Type != "user.registered" {
		t.Errorf("payload.Type: got %q, want %q", payload.Type, "user.registered")
	}

	if payload.StreamType != "user" {
		t.Errorf("payload.StreamType: got %q, want %q", payload.StreamType, "user")
	}

	if payload.StreamID != aggID.String() {
		t.Errorf("payload.StreamID: got %q, want %q", payload.StreamID, aggID.String())
	}

	if payload.Version != 3 {
		t.Errorf("payload.Version: got %d, want %d", payload.Version, 3)
	}

	if payload.OccurredAt != wantTime.Format(time.RFC3339) {
		t.Errorf("payload.OccurredAt: got %q, want %q", payload.OccurredAt, wantTime.Format(time.RFC3339))
	}

	if payload.EventID != evt.ID().String() {
		t.Errorf("payload.EventID: got %q, want %q", payload.EventID, evt.ID().String())
	}
}

func TestDomainEventToSSE_JSONKeys(t *testing.T) {
	t.Parallel()

	aggID, err := id.ParseStreamID(ulid.Make().String())
	if err != nil {
		t.Fatalf("parse stream ID: %v", err)
	}

	evt, err := event.New(
		event.Type("test.event"),
		aggID,
		"test",
		event.Version(1),
		`{}`,
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	sseEvt := DomainEventToSSE(evt)

	for _, key := range []string{`"type"`, `"streamType"`, `"streamId"`, `"version"`, `"occurredAt"`, `"eventId"`} {
		if !strings.Contains(sseEvt.Data, key) {
			t.Errorf("payload missing expected key %q: %s", key, sseEvt.Data)
		}
	}
}

func TestDomainEventToSSE_UsedByJournalSSEStore(t *testing.T) {
	t.Parallel()

	store := memorystorage.NewMemoryStore()
	events := seedEventList(t, 2)
	appendEvents(t, store, events)

	sseStore := NewJournalSSEStore(store, DomainEventToSSE)

	result, err := sseStore.EventsAfter(sse.EventID{})
	if err != nil {
		t.Fatalf("EventsAfter: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result))
	}

	for _, sseEvt := range result {
		if sseEvt.Event != "event" {
			t.Errorf("expected event name %q, got %q", "event", sseEvt.Event)
		}

		if !strings.Contains(sseEvt.Data, `"type"`) {
			t.Errorf("expected metadata envelope in data, got %q", sseEvt.Data)
		}
	}
}
