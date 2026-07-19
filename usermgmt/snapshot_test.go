package usermgmt

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

func TestMemorySnapshotStore_SaveLoadDelete(t *testing.T) {
	t.Parallel()

	store := NewMemorySnapshotStore()
	ctx := context.Background()
	aggID, _ := id.ParseAggregateID("user-1")
	ref := id.AggregateRef{Type: "User", ID: aggID}

	// Load on empty store returns (nil, nil) — the "no snapshot" convention.
	if snap, err := store.Load(ctx, ref); err != nil {
		t.Fatalf("Load empty: %v", err)
	} else if snap != nil {
		t.Fatalf("Load empty: expected nil snapshot, got %+v", snap)
	}

	// Save a snapshot, then Load returns it.
	original := snapshot.Snapshot{
		AggregateID:   ref.ID,
		AggregateType: ref.Type,
		Version:       event.Version(3),
		State:         []byte(`{"email":"a@b.com"}`),
	}
	if err := store.Save(ctx, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if loaded == nil {
		t.Fatal("Load after Save: expected non-nil snapshot")
	}
	if loaded.Version != original.Version {
		t.Errorf("Version = %d, want %d", loaded.Version, original.Version)
	}
	if string(loaded.State) != string(original.State) {
		t.Errorf("State = %q, want %q", loaded.State, original.State)
	}

	// Mutating the returned state must not corrupt the stored snapshot.
	loaded.State[0] = 'X'
	again, _ := store.Load(ctx, ref)
	if string(again.State) == "X"+string(original.State[1:]) {
		t.Fatal("Load did not copy state bytes; caller mutated stored snapshot")
	}

	// LoadAtVersion matches only when the version is exact.
	if s, _ := store.LoadAtVersion(ctx, ref, event.Version(3)); s == nil {
		t.Error("LoadAtVersion(3): expected snapshot, got nil")
	}
	if s, _ := store.LoadAtVersion(ctx, ref, event.Version(2)); s != nil {
		t.Error("LoadAtVersion(2): expected nil for version mismatch, got snapshot")
	}

	// Delete removes the snapshot; subsequent Load returns nil.
	if err := store.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if s, _ := store.Load(ctx, ref); s != nil {
		t.Error("Load after Delete: expected nil, got snapshot")
	}

	// Delete on a missing snapshot is not an error.
	if err := store.Delete(ctx, ref); err != nil {
		t.Errorf("Delete missing: expected no error, got %v", err)
	}
}

func TestSnapshotConfig_ZeroValueDisablesSnapshotting(t *testing.T) {
	t.Parallel()

	// A zero-value SnapshotConfig must yield no repository options (Store nil).
	opts := snapshotOptions[UserState](SnapshotConfig{})
	if len(opts) != 0 {
		t.Fatalf("zero-value SnapshotConfig produced %d options, want 0", len(opts))
	}
}

func TestSnapshotConfig_PartialConfigStillArmsStore(t *testing.T) {
	t.Parallel()

	// Store set without Codec/Strategy still arms the store option; go-cqrs-lite
	// rejects a non-nil Store with nil Codec at Execute time as incomplete
	// configuration. We only assert the option is wired here.
	store := NewMemorySnapshotStore()
	opts := snapshotOptions[UserState](SnapshotConfig{Store: store, Codec: codec.JSONCodec{}})
	if len(opts) < 2 {
		t.Fatalf("expected at least 2 options (store+codec), got %d", len(opts))
	}
}

// TestService_SnapshotIntegration verifies that wiring SnapshotConfig through
// NewService causes the repository to persist snapshots after writes, and that
// the read path still returns correct state (snapshot restore + replay).
func TestService_SnapshotIntegration(t *testing.T) {
	t.Parallel()

	store := NewMemorySnapshotStore()
	strategy, err := snapshot.EveryNEvents(1) // snapshot after every persisted event
	if err != nil {
		t.Fatalf("EveryNEvents: %v", err)
	}

	svc, err := NewService(ServiceConfig{
		SnapshotConfig: SnapshotConfig{
			Store:    store,
			Codec:    codec.JSONCodec{},
			Strategy: strategy,
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close() //nolint:errcheck // test cleanup

	email := "snap@example.com"
	resp, err := svc.Register(context.Background(), RegisterRequest{
		ID:    NewUserID("snap-user"),
		Email: email,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.User == nil {
		t.Fatal("Register returned nil user")
	}

	// EveryNEvents(1) guarantees a snapshot after the first persisted event.
	store.mu.RLock()
	snapshotCount := len(store.snapshots)
	store.mu.RUnlock()
	if snapshotCount == 0 {
		t.Fatal("expected at least one snapshot after Register, got none — snapshot wiring did not take effect")
	}

	// The read path must still resolve the user correctly (snapshot + replay).
	loaded, err := svc.GetUser(context.Background(), resp.User.ID)
	if err != nil {
		t.Fatalf("GetUser after snapshot: %v", err)
	}
	if loaded.Email != email {
		t.Errorf("Email = %q, want %q (snapshot restore corrupted state)", loaded.Email, email)
	}
}
