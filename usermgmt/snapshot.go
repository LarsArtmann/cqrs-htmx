package usermgmt

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// SnapshotConfig wires opt-in aggregate snapshotting into the event-sourced
// repositories. Snapshotting accelerates loading of high-event-volume
// aggregates: instead of replaying the full event journal on every Load, the
// repository restores the most recent snapshot and replays only the events
// appended since.
//
// All three fields are optional, but Store and Codec must be set together, and
// a Strategy is required for snapshots to actually be written. A zero-value
// SnapshotConfig (the default) leaves repositories in full-replay mode — zero
// behavior change for existing consumers.
//
// Recommended starter configuration:
//
//	strategy, _ := snapshot.EveryNEvents(500)
//	cfg := usermgmt.SnapshotConfig{
//	    Store:    usermgmt.NewMemorySnapshotStore(), // dev/test; use SQL/pebble in production
//	    Codec:    codec.JSONCodec{},
//	    Strategy: strategy,
//	}
//
// For "hot read" aggregates (read often, written rarely) prefer
// snapshot.NewReadPressure, which snapshots based on read count rather than
// write count. See docs/adr/0041-snapshot-integration.md.
type SnapshotConfig struct {
	// Store persists snapshots. When nil (the default), snapshotting is
	// disabled and repositories replay the full journal. Use
	// NewMemorySnapshotStore for dev/test, or any snapshot.SnapshotStore
	// backed by persistent storage (SQL, Pebble) for production.
	Store snapshot.SnapshotStore

	// Codec encodes/decodes aggregate State to bytes for snapshot storage.
	// codec.JSONCodec{} is a safe default; codec.CBORCodec{} is more compact.
	// Required when Store is set; ignored when Store is nil. go-cqrs-lite
	// rejects a non-nil Store with a nil Codec as incomplete configuration.
	Codec codec.Codec

	// Strategy decides when to write a snapshot after persisting events.
	// snapshot.EveryNEvents(n) snapshots on a fixed write cadence;
	// snapshot.NewReadPressure(threshold) snapshots hot-read aggregates.
	// When nil with a non-nil Store, snapshots are never written (the
	// repository still reads existing snapshots if any are present).
	Strategy snapshot.SnapshotStrategy
}

// snapshotOptions translates a SnapshotConfig into the typed
// decider.RepositoryOption list expected by NewRepository / stack.Repository.
// Generic so the same config applies uniformly to every aggregate State type.
// Returns nil when snapshotting is disabled (Store is nil) so existing
// repositories are untouched.
func snapshotOptions[State any](cfg SnapshotConfig) []decider.RepositoryOption[State] {
	if cfg.Store == nil {
		return nil
	}

	opts := []decider.RepositoryOption[State]{decider.WithSnapshotStore[State](cfg.Store)}

	if cfg.Codec != nil {
		opts = append(opts, decider.WithCodec[State](cfg.Codec))
	}

	if cfg.Strategy != nil {
		opts = append(opts, decider.WithSnapshotStrategy[State](cfg.Strategy))
	}

	return opts
}

// MemorySnapshotStore is an in-process snapshot.SnapshotStore for development
// and testing. Snapshots are kept in a map keyed by aggregate ref and are lost
// when the process exits — for production use a persistent store backed by SQL
// or Pebble. Safe for concurrent use.
//
// The store keeps only the latest snapshot per aggregate (matching the
// read pattern of the decider Repository, which always loads the newest).
type MemorySnapshotStore struct {
	mu        sync.RWMutex
	snapshots map[id.AggregateRef]snapshot.Snapshot
}

// NewMemorySnapshotStore creates an empty in-memory snapshot store.
func NewMemorySnapshotStore() *MemorySnapshotStore {
	return &MemorySnapshotStore{ //nolint:exhaustruct // mutex zero-value, snapshots populated lazily
		snapshots: make(map[id.AggregateRef]snapshot.Snapshot),
	}
}

// Save stores a snapshot, overwriting any previous snapshot for the same
// aggregate. Sets CreatedAt to now when the caller left it zero.
func (m *MemorySnapshotStore) Save(_ context.Context, s snapshot.Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}

	ref := id.AggregateRef{Type: s.StreamType, ID: s.StreamID}
	m.snapshots[ref] = s

	return nil
}

// Delete removes the snapshot for the given aggregate, if any. It is not an
// error to delete a snapshot that does not exist.
func (m *MemorySnapshotStore) Delete(_ context.Context, ref id.AggregateRef) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.snapshots, ref)

	return nil
}

// Load returns the latest snapshot for the aggregate, or (nil, nil) if none
// exists. The returned state bytes are copied so callers may mutate them
// freely.
func (m *MemorySnapshotStore) Load(_ context.Context, ref id.AggregateRef) (*snapshot.Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.snapshots[ref]
	if !ok {
		return nil, nil //nolint:nilnil // (nil,nil) is the SnapshotStore "no snapshot" convention
	}

	return cloneSnapshot(s), nil
}

// LoadAtVersion returns the snapshot for the aggregate only when its recorded
// version matches the requested version exactly, otherwise (nil, nil). This
// supports the decider Repository's versioned reload path.
func (m *MemorySnapshotStore) LoadAtVersion(
	_ context.Context, ref id.AggregateRef, version event.Version,
) (*snapshot.Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.snapshots[ref]
	if !ok || s.Version != version {
		return nil, nil //nolint:nilnil // (nil,nil) is the SnapshotStore "no snapshot" convention
	}

	return cloneSnapshot(s), nil
}

// cloneSnapshot returns a deep copy of s so callers cannot mutate the stored
// snapshot's state bytes through aliasing.
func cloneSnapshot(s snapshot.Snapshot) *snapshot.Snapshot {
	stateCopy := make([]byte, len(s.State))
	copy(stateCopy, s.State)

	return &snapshot.Snapshot{
		StreamID:   s.StreamID,
		StreamType: s.StreamType,
		Version:       s.Version,
		State:         stateCopy,
		CreatedAt:     s.CreatedAt,
	}
}
