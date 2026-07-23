package usermgmt

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
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
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    event.Version(3),
		State:      []byte(`{"email":"a@b.com"}`),
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

// countingSnapshotStore wraps a snapshot.SnapshotStore and counts Load/Save
// calls so tests can prove the repository consults the snapshot at all.
type countingSnapshotStore struct {
	snapshot.SnapshotStore

	mu        sync.Mutex
	loadCalls int
	saveCalls int
}

func (c *countingSnapshotStore) Load(ctx context.Context, ref id.AggregateRef) (*snapshot.Snapshot, error) {
	c.mu.Lock()
	c.loadCalls++
	c.mu.Unlock()

	return c.SnapshotStore.Load(ctx, ref)
}

func (c *countingSnapshotStore) Save(ctx context.Context, s snapshot.Snapshot) error {
	c.mu.Lock()
	c.saveCalls++
	c.mu.Unlock()

	return c.SnapshotStore.Save(ctx, s)
}

// countingEventStore wraps an event.Store and counts full-Load vs
// LoadFromVersion (tail-only) calls. When a snapshot is in use, the decider
// Repository should call LoadFromVersion for the tail instead of Load for the
// full journal — that is the measurable proof snapshotting earns its keep.
type countingEventStore struct {
	event.Store

	mu               sync.Mutex
	fullLoads        int
	loadsFromVersion int
}

func (c *countingEventStore) Load(ctx context.Context, ref id.AggregateRef) ([]event.Event, error) {
	c.mu.Lock()
	c.fullLoads++
	c.mu.Unlock()

	return c.Store.Load(ctx, ref)
}

func (c *countingEventStore) LoadFromVersion(
	ctx context.Context, ref id.AggregateRef, version event.Version,
) ([]event.Event, error) {
	c.mu.Lock()
	c.loadsFromVersion++
	c.mu.Unlock()

	return c.Store.LoadFromVersion(ctx, ref, version)
}

func (c *countingEventStore) snapshot() (full, fromVersion int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.fullLoads, c.loadsFromVersion
}

// TestSnapshot_WritePathConsultsSnapshot proves that wiring SnapshotConfig
// through NewService makes the decider Repository actually consult the
// snapshot on the WRITE path (command dispatch). Without this test, the
// existing TestService_SnapshotIntegration only proves a snapshot is WRITTEN —
// not that it is ever READ back. A snapshot that is written but never loaded
// is pure overhead.
//
// Proof strategy: wrap both stores with counters, register a user (which
// writes the first snapshot via EveryNEvents(1)), reset the counters, then
// issue a second command (ChangeEmail) that must load the aggregate state. We
// then assert:
//  1. snapshot.Load was called at least once (the snapshot was CONSULTED), and
//  2. event.LoadFromVersion was called (the tail-only path was used). If only
//     event.Load was called, the repository fell back to full replay and the
//     snapshot is dead weight.
func TestSnapshot_WritePathConsultsSnapshot(t *testing.T) {
	t.Parallel()

	snapStore := &countingSnapshotStore{SnapshotStore: NewMemorySnapshotStore()}
	strategy, err := snapshot.EveryNEvents(1) // snapshot after every persisted event
	if err != nil {
		t.Fatalf("EveryNEvents: %v", err)
	}
	eventStore := &countingEventStore{Store: memory.NewMemoryStore()}

	svc, err := NewService(ServiceConfig{
		EventStore: eventStore,
		SnapshotConfig: SnapshotConfig{
			Store:    snapStore,
			Codec:    codec.JSONCodec{},
			Strategy: strategy,
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close() //nolint:errcheck // test cleanup

	ctx := context.Background()
	resp, err := svc.Register(ctx, RegisterRequest{
		ID:    NewUserID("snap-proof"),
		Email: "snap-proof@example.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Reset all counters AFTER register — the proof is about the second write.
	snapStore.mu.Lock()
	snapStore.loadCalls = 0
	snapStore.saveCalls = 0
	snapStore.mu.Unlock()
	eventStore.mu.Lock()
	eventStore.fullLoads = 0
	eventStore.loadsFromVersion = 0
	eventStore.mu.Unlock()

	// ChangeEmail MUST load the User aggregate to apply the command. This is
	// the load the snapshot is supposed to accelerate.
	if err := svc.ChangeEmail(ctx, resp.User.ID, "after-snap@example.com"); err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}

	snapStore.mu.Lock()
	snapLoads := snapStore.loadCalls
	snapStore.mu.Unlock()
	fullLoads, loadsFromVersion := eventStore.snapshot()

	if snapLoads == 0 {
		t.Fatal("snapshot.Load was not called during ChangeEmail — " +
			"snapshot is configured but the write path ignores it (dead weight)")
	}
	if loadsFromVersion == 0 {
		t.Errorf("LoadFromVersion was not called during ChangeEmail — " +
			"expected tail-only load after snapshot; the repository fell back to full replay. " +
			"This means the snapshot is not actually accelerating loads.")
	}
	t.Logf("during ChangeEmail: snapshot.Load=%d, event.Load=%d (full replay), event.LoadFromVersion=%d (tail)",
		snapLoads, fullLoads, loadsFromVersion)
}
