package usermgmt

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	_ "modernc.org/sqlite"
)

func newTestSQLiteStore(t *testing.T) *SQLEventStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewSQLEventStore(context.Background(), db, "sqlite")
	if err != nil {
		t.Fatalf("NewSQLEventStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestSQLEventStore_SaveAndLoad(t *testing.T) {
	store := newTestSQLiteStore(t)
	ctx := context.Background()
	aggID := id.NewStreamID()

	payload, _ := marshalPayload(UserRegisteredPayload{
		Email: "sql@test.com",
		Roles: []Role{RoleUser},
	})

	evt, err := event.NewEvent(eventUserRegistered, aggID, aggregateTypeUser, 1, payload)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	ref := id.StreamRef{ID: aggID, Type: aggregateTypeUser}
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
	if loaded[0].Type() != eventUserRegistered {
		t.Errorf("type = %q", loaded[0].Type())
	}
	if loaded[0].AggregateID() != aggID {
		t.Errorf("aggregate ID mismatch")
	}
}

func TestSQLEventStore_OptimisticConcurrency(t *testing.T) {
	store := newTestSQLiteStore(t)
	ctx := context.Background()
	aggID := id.NewStreamID()
	ref := id.StreamRef{ID: aggID, Type: aggregateTypeUser}

	payload, _ := marshalPayload(UserRegisteredPayload{
		Email: "concur@test.com",
	})
	evt, _ := event.NewEvent(eventUserRegistered, aggID, aggregateTypeUser, 1, payload)

	// Save with correct expected version (0)
	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save v0: %v", err)
	}

	// Try to save with wrong expected version (0 again — should fail)
	payload2, _ := marshalPayload(EmailChangedPayload{
		Email: "new@test.com",
	})
	evt2, _ := event.NewEvent(eventEmailChanged, aggID, aggregateTypeUser, 2, payload2)

	err := store.Save(ctx, ref, []event.Event{evt2}, 0)
	if err == nil {
		t.Fatal("expected version conflict error")
	}
}

func TestSQLEventStore_AppendBatch(t *testing.T) {
	store := newTestSQLiteStore(t)
	ctx := context.Background()
	aggID := id.NewStreamID()
	ref := id.StreamRef{ID: aggID, Type: aggregateTypeUser}

	p1, _ := marshalPayload(UserRegisteredPayload{
		Email: "batch@test.com",
	})
	p2, _ := marshalPayload(EmailChangedPayload{
		Email: "new@test.com",
	})
	evt1, _ := event.NewEvent(eventUserRegistered, aggID, aggregateTypeUser, 1, p1)
	evt2, _ := event.NewEvent(eventEmailChanged, aggID, aggregateTypeUser, 2, p2)

	if err := store.AppendBatch(ctx, ref, []event.Event{evt1, evt2}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}
}

func TestSQLEventStore_ReadAll(t *testing.T) {
	store := newTestSQLiteStore(t)
	ctx := context.Background()

	agg1 := id.NewStreamID()
	agg2 := id.NewStreamID()

	p1, _ := marshalPayload(UserRegisteredPayload{
		Email: "ra1@test.com",
	})
	p2, _ := marshalPayload(UserRegisteredPayload{
		Email: "ra2@test.com",
	})
	evt1, _ := event.NewEvent(eventUserRegistered, agg1, aggregateTypeUser, 1, p1)
	evt2, _ := event.NewEvent(eventUserRegistered, agg2, aggregateTypeUser, 1, p2)

	_ = store.AppendBatch(ctx, id.StreamRef{ID: agg1, Type: aggregateTypeUser}, []event.Event{evt1})
	_ = store.AppendBatch(ctx, id.StreamRef{ID: agg2, Type: aggregateTypeUser}, []event.Event{evt2})

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 events, got %d", len(all))
	}
}

func TestSQLEventStore_LoadFromVersion(t *testing.T) {
	store := newTestSQLiteStore(t)
	ctx := context.Background()
	ref := appendThreeTestEvents(t, store, ctx)

	fromV1, err := store.LoadFromVersion(ctx, ref, 1)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}
	if len(fromV1) != 2 {
		t.Fatalf("expected 2 events from version 1, got %d", len(fromV1))
	}
}

func TestSQLEventStore_LoadToVersion(t *testing.T) {
	store := newTestSQLiteStore(t)
	ctx := context.Background()
	ref := appendThreeTestEvents(t, store, ctx)

	toV2, err := store.LoadToVersion(ctx, ref, 2)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}
	if len(toV2) != 2 {
		t.Fatalf("expected 2 events to version 2, got %d", len(toV2))
	}
}

// appendThreeTestEvents inserts a registered → email-changed → display-name-changed
// sequence and returns the aggregate ref. Shared by the LoadFromVersion and
// LoadToVersion tests to avoid duplication.
func appendThreeTestEvents(
	t *testing.T,
	store *SQLEventStore,
	ctx context.Context,
) id.StreamRef {
	t.Helper()
	aggID := id.NewStreamID()
	ref := id.StreamRef{ID: aggID, Type: aggregateTypeUser}

	p1, _ := marshalPayload(UserRegisteredPayload{Email: "v@test.com"})
	p2, _ := marshalPayload(EmailChangedPayload{Email: "new@test.com"})
	p3, _ := marshalPayload(DisplayNameChangedPayload{DisplayName: "New"})
	evt1, _ := event.NewEvent(eventUserRegistered, aggID, aggregateTypeUser, 1, p1)
	evt2, _ := event.NewEvent(eventEmailChanged, aggID, aggregateTypeUser, 2, p2)
	evt3, _ := event.NewEvent(eventDisplayNameChanged, aggID, aggregateTypeUser, 3, p3)

	if err := store.AppendBatch(ctx, ref, []event.Event{evt1, evt2, evt3}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}
	return ref
}

func TestSQLEventStore_EmptyAggregate(t *testing.T) {
	store := newTestSQLiteStore(t)
	ctx := context.Background()
	aggID := id.NewStreamID()
	ref := id.StreamRef{ID: aggID, Type: aggregateTypeUser}

	// Upstream storage.SQLEventStore returns ErrAggregateNotFound when Load
	// finds no events (RequireHit: true). The decider's Repository handles
	// this by returning its Initial state + version 0, so the CQRS flow
	// treats it as a new aggregate correctly.
	_, err := store.Load(ctx, ref)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound for empty aggregate, got %v", err)
	}
}

func TestSQLEventStore_UnsupportedDialect(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer func() { _ = db.Close() }()
	_, err := NewSQLEventStore(context.Background(), db, "oracle")
	if err == nil {
		t.Fatal("expected error for unsupported dialect")
	}
}

func TestSQLEventStore_WithService(t *testing.T) {
	store := newTestSQLiteStore(t)
	svc := newTestServiceWithConfig(t, ServiceConfig{
		EventStore: store,
	})

	reg := registerTestUser(t, svc, "sqlsvc1", "sqlsvc1@test.com")
	if reg.User.Email != "sqlsvc1@test.com" {
		t.Errorf("email = %q", reg.User.Email)
	}

	// Verify events are persisted in SQL
	ref := id.StreamRef{ID: aggIDFromUserID(t, reg.User.ID), Type: aggregateTypeUser}
	loaded, err := store.Load(context.Background(), ref)
	if err != nil {
		t.Fatalf("Load from SQL: %v", err)
	}
	if len(loaded) == 0 {
		t.Fatal("expected events in SQL store")
	}
}

func aggIDFromUserID(t *testing.T, userID UserID) (aggID id.StreamID) {
	t.Helper()
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}
	return aggID
}
