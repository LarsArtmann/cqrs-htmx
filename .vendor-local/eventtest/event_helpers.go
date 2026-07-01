package eventtest

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
)

const createdEventType = "Created"

type TimelineEvent struct {
	Type    string
	Version event.Version
	Offset  time.Duration
}

func MakeTimelineEvents(
	tb testing.TB,
	aggType event.AggregateType,
	aggID id.AggregateID,
	events []TimelineEvent,
) []event.Event {
	tb.Helper()

	now := time.Now()

	result := make([]event.Event, len(events))
	for i, e := range events {
		evt, err := event.NewEvent(
			event.Type(e.Type),
			aggID,
			aggType,
			e.Version,
			nil,
			event.WithOccurredAt(now.Add(e.Offset)),
		)
		if err != nil {
			tb.Fatalf("MakeTimelineEvents: create event %q: %v", e.Type, err)
		}

		result[i] = evt
	}

	return result
}

func NewTestEvent() (event.Event, error) {
	return MakeEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
}

func NewEventOpts(
	tb testing.TB,
	typ event.Type,
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	payload []byte,
	opts ...event.Option,
) event.Event {
	tb.Helper()

	evt, err := event.NewEvent(typ, aggID, aggType, version, payload, opts...)
	if err != nil {
		tb.Fatalf("create event %q: %v", typ, err)
	}

	return evt
}

func NewEvent(
	t *testing.T,
	eventType event.Type,
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	payload []byte,
) event.Event {
	t.Helper()

	evt, err := event.NewEvent(eventType, aggID, aggType, version, payload)
	if err != nil {
		t.Fatalf("create event %q: %v", eventType, err)
	}

	return evt
}

func MakeEvent(
	eventType event.Type,
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	payload []byte,
) (event.Event, error) {
	evt, err := event.NewEvent(eventType, aggID, aggType, version, payload)

	return evt, err //nolint:wrapcheck // thin wrapper, caller adds context if needed
}

func QuickEvent(
	eventType event.Type,
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	payload []byte,
) event.Event {
	evt, _ := event.NewEvent(eventType, aggID, aggType, version, payload)

	return evt
}

func QuickEventOpts(
	eventType event.Type,
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	payload []byte,
	opts ...event.Option,
) event.Event {
	evt, _ := event.NewEvent(eventType, aggID, aggType, version, payload, opts...)

	return evt
}

func TamperEvent(original event.Event, newPayload []byte) event.Event {
	tampered, _ := event.NewEvent(
		original.Type(),
		original.AggregateID(),
		original.AggregateType(),
		original.Version(),
		newPayload,
		event.WithEventID(original.ID()),
		event.WithOccurredAt(original.OccurredAt()),
		event.WithSchemaVersion(original.SchemaVersion()),
		event.WithMetadata(original.Metadata()),
	)

	return tampered
}

type AppendBatcher interface {
	AppendBatch(
		ctx context.Context,
		ref event.AggregateRef,
		events []event.Event,
	) error
}

func MakeLoadToTimestampFixtures(
	tb testing.TB,
	store AppendBatcher,
	ctx context.Context,
	aggType event.AggregateType,
	aggID id.AggregateID,
	versions [3]event.Version,
) (time.Time, id.AggregateID) {
	tb.Helper()

	now := time.Now()
	eventTypeCreated := createdEventType
	events := MakeTimelineEvents(tb, aggType, aggID, []TimelineEvent{
		{Type: eventTypeCreated, Version: versions[0], Offset: -2 * time.Hour},
		{Type: "Updated", Version: versions[1], Offset: -1 * time.Hour},
		{Type: "Deleted", Version: versions[2], Offset: 0},
	})

	err := store.AppendBatch(ctx, event.NewAggregateRef(aggType, aggID), events)
	if err != nil {
		tb.Fatalf("AppendBatch: %v", err)
	}

	return now, aggID
}

func MakeThreeTimelineEvents(
	tb testing.TB,
	aggType1 event.AggregateType,
	aggID1 id.AggregateID,
	aggType2 event.AggregateType,
	aggID2 id.AggregateID,
) ([]event.Event, []event.Event, []event.Event) {
	tb.Helper()

	evt1 := MakeTimelineEvents(tb, aggType1, aggID1, []TimelineEvent{
		{Type: createdEventType, Version: 1, Offset: -2 * time.Hour},
	})
	evt2 := MakeTimelineEvents(tb, aggType2, aggID2, []TimelineEvent{
		{Type: createdEventType, Version: 1, Offset: -1 * time.Hour},
	})
	evt3 := MakeTimelineEvents(tb, aggType1, aggID1, []TimelineEvent{
		{Type: "Updated", Version: 1, Offset: 0},
	})

	return evt1, evt2, evt3
}

func QuickSnapshot(
	aggID id.AggregateID,
	aggType event.AggregateType,
	version event.Version,
	state []byte,
) snapshot.Snapshot {
	return snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: aggType,
		Version:       version,
		State:         state,
		CreatedAt:     time.Now(),
	}
}
