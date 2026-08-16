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

	if sseEvt.Event != sseEventType {
		t.Errorf("sse event name: got %q, want %q", sseEvt.Event, sseEventType)
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

	assertPayloadFields(t, payload, aggID.String(), evt.ID().String(), wantTime)
}

// assertPayloadFields checks every field of the EventPayload envelope so the
// main test body stays under the cyclop limit.
func assertPayloadFields(t *testing.T, p EventPayload, streamID, eventID string, wantTime time.Time) {
	t.Helper()

	if p.Type != "user.registered" {
		t.Errorf("payload.Type: got %q, want %q", p.Type, "user.registered")
	}

	if p.StreamType != "user" {
		t.Errorf("payload.StreamType: got %q, want %q", p.StreamType, "user")
	}

	if p.StreamID != streamID {
		t.Errorf("payload.StreamID: got %q, want %q", p.StreamID, streamID)
	}

	if p.Version != 3 {
		t.Errorf("payload.Version: got %d, want %d", p.Version, 3)
	}

	if p.OccurredAt != wantTime.Format(time.RFC3339) {
		t.Errorf("payload.OccurredAt: got %q, want %q", p.OccurredAt, wantTime.Format(time.RFC3339))
	}

	if p.EventID != eventID {
		t.Errorf("payload.EventID: got %q, want %q", p.EventID, eventID)
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

// TestDomainEventToSSE_WireFormatGolden pins the exact serialized bytes of the
// envelope. EventPayload is published language shared by every SSE endpoint in
// the family, so field order and key spelling must never drift silently.
func TestDomainEventToSSE_WireFormatGolden(t *testing.T) {
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

	want := `{"type":"user.registered","streamType":"user","streamId":"` + aggID.String() +
		`","version":3,"occurredAt":"2026-08-16T10:00:00Z","eventId":"` + evt.ID().String() + `"}`

	if sseEvt.Data != want {
		t.Errorf("envelope wire format drift:\n got: %s\nwant: %s", sseEvt.Data, want)
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
		if sseEvt.Event != sseEventType {
			t.Errorf("expected event name %q, got %q", sseEventType, sseEvt.Event)
		}

		if !strings.Contains(sseEvt.Data, `"type"`) {
			t.Errorf("expected metadata envelope in data, got %q", sseEvt.Data)
		}
	}
}
