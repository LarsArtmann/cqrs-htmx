package usermgmt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestSQLEventStore_LoadToTimestamp(t *testing.T) {
	store := newTestSQLiteStore(t)
	ctx := context.Background()
	aggID := id.NewStreamID()
	ref := id.StreamRef{ID: aggID, Type: aggregateTypeUser}

	p1, _ := marshalPayload(UserRegisteredPayload{Email: "ts@test.com"})
	p2, _ := marshalPayload(EmailChangedPayload{Email: "new@test.com"})
	evt1, _ := event.NewEvent(eventUserRegistered, aggID, aggregateTypeUser, 1, p1,
		event.WithOccurredAt(time.Now().Add(-2*time.Hour)))
	evt2, _ := event.NewEvent(eventEmailChanged, aggID, aggregateTypeUser, 2, p2,
		event.WithOccurredAt(time.Now().Add(-1*time.Hour)))

	if err := store.Save(ctx, ref, []event.Event{evt1, evt2}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Future cutoff returns both events.
	all, err := store.LoadToTimestamp(ctx, ref, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("LoadToTimestamp future: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 events, got %d", len(all))
	}

	// Cutoff between the two events returns only the older one.
	firstOnly, err := store.LoadToTimestamp(ctx, ref, time.Now().Add(-90*time.Minute))
	if err != nil {
		t.Fatalf("LoadToTimestamp mid: %v", err)
	}
	if len(firstOnly) != 1 || firstOnly[0].Type() != eventUserRegistered {
		t.Fatalf("expected 1 registered event, got %d (%v)", len(firstOnly), firstOnly)
	}

	// Cutoff before any event — upstream returns ErrAggregateNotFound
	// (RequireHit: true on LoadToTimestamp).
	none, err := store.LoadToTimestamp(ctx, ref, time.Now().Add(-3*time.Hour))
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound for past cutoff, got %v (events: %d)", err, len(none))
	}
}

func TestSQLEventStore_EmptyInputsAreNoOps(t *testing.T) {
	store := newTestSQLiteStore(t)
	ctx := context.Background()
	ref := id.StreamRef{ID: id.NewStreamID(), Type: aggregateTypeUser}

	if err := store.Save(ctx, ref, nil, 0); err != nil {
		t.Fatalf("Save(nil) should be a no-op, got: %v", err)
	}
	if err := store.AppendBatch(ctx, ref, nil); err != nil {
		t.Fatalf("AppendBatch(nil) should be a no-op, got: %v", err)
	}
}
