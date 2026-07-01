package eventtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func newTestEvent(t *testing.T, aggID id.AggregateID, v event.Version) event.Event {
	t.Helper()
	evt, err := event.NewEvent("TestEvent", aggID, "Test", v, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func appendTestEvents(
	t *testing.T,
	store *FakeStore,
	ctx context.Context,
	ref event.AggregateRef,
	n int,
) {
	t.Helper()
	for i := range n {
		_ = store.AppendBatch(ctx, ref, []event.Event{newTestEvent(t, ref.ID, event.Version(i+1))})
	}
}

func TestFakeStore_Save_Default(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)
	evt := newTestEvent(t, aggID, 1)

	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
}

func TestFakeStore_Save_Override(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	store.saveFn = func(context.Context, event.AggregateRef, []event.Event, event.Version) error {
		return errors.New("save override")
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)
	evt := newTestEvent(t, aggID, 1)

	if err := store.Save(
		ctx,
		ref,
		[]event.Event{evt},
		0,
	); err == nil ||
		err.Error() != "save override" {
		t.Fatalf("expected save override error, got: %v", err)
	}
}

func TestFakeStore_Load_Default(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	evt1 := newTestEvent(t, aggID, 1)
	evt2 := newTestEvent(t, aggID, 2)
	_ = store.AppendBatch(ctx, ref, []event.Event{evt1, evt2})

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}
}

func TestFakeStore_Load_DefensiveCopy(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	evt := newTestEvent(t, aggID, 1)
	_ = store.AppendBatch(ctx, ref, []event.Event{evt})

	loaded1, _ := store.Load(ctx, ref)
	loaded2, _ := store.Load(ctx, ref)

	loaded1[0] = nil
	if loaded2[0] == nil {
		t.Fatal("Load returned non-defensive copy")
	}
}

func TestFakeStore_Load_Override(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	store.loadFn = func(event.AggregateRef) ([]event.Event, error) {
		return nil, errors.New("load override")
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	if _, err := store.Load(ctx, ref); err == nil || err.Error() != "load override" {
		t.Fatalf("expected load override error, got: %v", err)
	}
}

func TestFakeStore_LoadFromVersion_Default(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	appendTestEvents(t, store, ctx, ref, 5)

	loaded, err := store.LoadFromVersion(ctx, ref, 2)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected 3 events (v3,v4,v5), got %d", len(loaded))
	}
}

func TestFakeStore_LoadFromVersion_Override(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	store.loadFromVersionFn = func(event.AggregateRef, event.Version) ([]event.Event, error) {
		return nil, errors.New("override")
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	if _, err := store.LoadFromVersion(ctx, ref, 0); err == nil || err.Error() != "override" {
		t.Fatalf("expected override error, got: %v", err)
	}
}

func TestFakeStore_LoadToVersion_Default(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	appendTestEvents(t, store, ctx, ref, 5)

	loaded, err := store.LoadToVersion(ctx, ref, 3)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected 3 events (v1,v2,v3), got %d", len(loaded))
	}
}

func TestFakeStore_LoadToTimestamp_Default(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	now := time.Now()
	for i := range 3 {
		evt, _ := event.NewEvent("TestEvent", aggID, "Test", event.Version(i+1), nil,
			event.WithOccurredAt(now.Add(time.Duration(i)*time.Hour)))
		_ = store.AppendBatch(ctx, ref, []event.Event{evt})
	}

	loaded, err := store.LoadToTimestamp(ctx, ref, now.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
}

func TestFakeStore_AppendBatch_Default(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	evt1 := newTestEvent(t, aggID, 1)
	evt2 := newTestEvent(t, aggID, 2)
	_ = store.AppendBatch(ctx, ref, []event.Event{evt1, evt2})

	loaded, _ := store.Load(ctx, ref)
	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}
}

func TestFakeStore_ReadAll_Default(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	ctx := context.Background()

	for range 3 {
		aggID := id.NewAggregateID()
		evt := newTestEvent(t, aggID, 1)
		_ = store.AppendBatch(ctx, event.NewAggregateRef("Test", aggID), []event.Event{evt})
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 events, got %d", len(all))
	}
}

func TestFakeStore_ReadAll_Override(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	store.readAllFn = func() ([]event.Event, error) {
		return nil, errors.New("readall override")
	}

	if _, err := store.ReadAll(
		context.Background(),
	); err == nil ||
		err.Error() != "readall override" {
		t.Fatalf("expected override error, got: %v", err)
	}
}

func TestFakeStore_ReadFrom_Default(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt := newTestEvent(t, aggID, 1)
	_ = store.AppendBatch(ctx, event.NewAggregateRef("Test", aggID), []event.Event{evt})

	// ReadFrom after the only event should return 0
	from, err := store.ReadFrom(ctx, evt.ID(), 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if len(from) != 0 {
		t.Fatalf("expected 0 events after last ID, got %d", len(from))
	}
}

func TestFakeStore_ReadFrom_Override(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	store.readFromFn = func(id.EventID, int) ([]event.Event, error) {
		return nil, errors.New("readfrom override")
	}

	if _, err := store.ReadFrom(
		context.Background(),
		id.EventID{},
		0,
	); err == nil ||
		err.Error() != "readfrom override" {
		t.Fatalf("expected override error, got: %v", err)
	}
}

func TestFakeStore_Close_Default(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestFakeStore_Close_Override(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	store.closeFn = func() error {
		return errors.New("close override")
	}

	if err := store.Close(); err == nil || err.Error() != "close override" {
		t.Fatalf("expected close override error, got: %v", err)
	}
}

func TestFakeStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	store := NewFakeStore()
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Test", aggID)

	for i := range 100 {
		go func(v int) {
			evt := newTestEvent(t, aggID, event.Version(v))
			_ = store.Save(ctx, ref, []event.Event{evt}, event.Version(v-1))
		}(i + 1)
	}

	time.Sleep(50 * time.Millisecond)

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 100 {
		t.Fatalf("expected 100 events, got %d", len(loaded))
	}
}
