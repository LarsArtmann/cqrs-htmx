package eventtest

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// StoreTestConfig configures a store test suite with event type/payload details.
type StoreTestConfig struct {
	AggType event.AggregateType
	EvtType event.Type
	Payload func(version event.Version) []byte
}

// NewTestEvent creates a test event using the config's type and payload.
func (c StoreTestConfig) NewTestEvent(
	t *testing.T,
	aggID id.AggregateID,
	version event.Version,
	extraOpts ...event.Option,
) *event.ImmutableEvent {
	t.Helper()

	evt, err := event.NewEvent(
		c.EvtType,
		aggID,
		c.AggType,
		version,
		c.Payload(version),
		extraOpts...,
	)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	return evt
}

// NewStoreTestConfig creates a StoreTestConfig with a JSON payload template.
func NewStoreTestConfig(
	aggType event.AggregateType,
	evtType event.Type,
	jsonField, valuePrefix string,
) StoreTestConfig {
	return StoreTestConfig{
		AggType: aggType,
		EvtType: evtType,
		Payload: func(v event.Version) []byte {
			return fmt.Appendf(nil, `{"%s":"%s-%d"}`, jsonField, valuePrefix, v)
		},
	}
}

// IssueStoreConfig returns a StoreTestConfig for "IssueCreated" events.
func IssueStoreConfig() StoreTestConfig {
	return NewStoreTestConfig("Issue", "IssueCreated", "title", "test")
}

// OrderStoreConfig returns a StoreTestConfig for "OrderPlaced" events.
func OrderStoreConfig() StoreTestConfig {
	return NewStoreTestConfig("Order", "OrderPlaced", "item", "widget")
}

// SaveEvent saves a single event at version 0 using the given config.
func SaveEvent(
	t *testing.T,
	store event.Store,
	cfg StoreTestConfig,
	aggID id.AggregateID,
	evt *event.ImmutableEvent,
) {
	t.Helper()

	err := store.Save(
		context.Background(),
		event.NewAggregateRef(cfg.AggType, aggID),
		[]event.Event{evt},
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
}

// TestStoreSaveAndLoad tests basic save and load round-trip.
func TestStoreSaveAndLoad(t *testing.T, store event.Store, cfg StoreTestConfig) {
	t.Helper()

	aggID := id.NewAggregateID()
	evt := cfg.NewTestEvent(t, aggID, 1)

	err := store.Save(
		context.Background(),
		event.NewAggregateRef(cfg.AggType, aggID),
		[]event.Event{evt},
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), event.NewAggregateRef(cfg.AggType, aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if loaded[0].Type() != cfg.EvtType {
		t.Errorf("Type = %q, want %q", loaded[0].Type(), cfg.EvtType)
	}

	if loaded[0].ID() != evt.ID() {
		t.Errorf("ID = %v, want %v", loaded[0].ID(), evt.ID())
	}
}

// TestStoreConcurrencyConflict tests that saving with wrong version returns ErrVersionConflict.
func TestStoreConcurrencyConflict(t *testing.T, store event.Store, cfg StoreTestConfig) {
	t.Helper()

	aggID := id.NewAggregateID()
	evt := cfg.NewTestEvent(t, aggID, 1)

	SaveEvent(t, store, cfg, aggID, evt)

	evt2 := cfg.NewTestEvent(t, aggID, 2)

	err := store.Save(
		context.Background(),
		event.NewAggregateRef(cfg.AggType, aggID),
		[]event.Event{evt2},
		event.Version(0),
	)
	if !errors.Is(err, event.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

// TestStoreAppendBatch tests batch appending events.
func TestStoreAppendBatch(t *testing.T, store event.Store, cfg StoreTestConfig) {
	t.Helper()

	aggID := id.NewAggregateID()
	evt1 := cfg.NewTestEvent(t, aggID, 1)
	evt2 := cfg.NewTestEvent(t, aggID, 2)

	err := store.AppendBatch(
		context.Background(),
		event.NewAggregateRef(cfg.AggType, aggID),
		[]event.Event{evt1, evt2},
	)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.Load(context.Background(), event.NewAggregateRef(cfg.AggType, aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}

	AssertEventVersion(t, loaded, 0, 1)
	AssertEventVersion(t, loaded, 1, 2)
}

// TestStoreLoadFromVersion tests loading events from a specific version.
func TestStoreLoadFromVersion(t *testing.T, store event.Store, cfg StoreTestConfig) {
	t.Helper()

	aggID := id.NewAggregateID()
	evt1 := cfg.NewTestEvent(t, aggID, 1)
	evt2 := cfg.NewTestEvent(t, aggID, 2)
	evt3 := cfg.NewTestEvent(t, aggID, 3)

	err := store.AppendBatch(
		context.Background(),
		event.NewAggregateRef(cfg.AggType, aggID),
		[]event.Event{evt1, evt2, evt3},
	)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.LoadFromVersion(
		context.Background(),
		event.NewAggregateRef(cfg.AggType, aggID),
		event.Version(1),
	)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events after version 1, got %d", len(loaded))
	}

	AssertEventVersion(t, loaded, 0, 2)
}

// TestStoreMetadataRoundtrip tests that metadata survives save/load.
func TestStoreMetadataRoundtrip(
	t *testing.T,
	store event.Store,
	cfg StoreTestConfig,
	customEnv string,
) {
	t.Helper()

	aggID := id.NewAggregateID()
	cid := id.NewCorrelationID()
	uid := id.NewUserID()

	evt := cfg.NewTestEvent(
		t, aggID, 1,
		event.WithCorrelationID(cid),
		event.WithUserID(uid),
		event.WithCustom("env", customEnv),
	)

	SaveEvent(t, store, cfg, aggID, evt)

	loaded, err := store.Load(context.Background(), event.NewAggregateRef(cfg.AggType, aggID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	meta := loaded[0].Metadata()
	if meta.CorrelationID != cid {
		t.Errorf("CorrelationID = %v, want %v", meta.CorrelationID, cid)
	}

	if meta.UserID != uid {
		t.Errorf("UserID = %v, want %v", meta.UserID, uid)
	}

	if meta.Custom["env"] != customEnv {
		t.Errorf("Custom[env] = %q, want %q", meta.Custom["env"], customEnv)
	}
}
