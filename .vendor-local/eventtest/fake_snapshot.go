package eventtest

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
)

type FakeSnapshotStore struct {
	mu       sync.RWMutex
	snapshot *snapshot.Snapshot
	saved    []snapshot.Snapshot
	loadErr  error
	saveErr  error
}

func NewFakeSnapshotStore() *FakeSnapshotStore {
	return &FakeSnapshotStore{}
}

func (s *FakeSnapshotStore) SetSnapshot(snap *snapshot.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot = snap
}

func (s *FakeSnapshotStore) SetLoadError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.loadErr = err
}

func (s *FakeSnapshotStore) SetSaveError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.saveErr = err
}

func (s *FakeSnapshotStore) Saved() []snapshot.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]snapshot.Snapshot{}, s.saved...)
}

func (s *FakeSnapshotStore) Save(_ context.Context, snap snapshot.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.saveErr != nil {
		return s.saveErr
	}

	s.saved = append(s.saved, snap)

	return nil
}

func (s *FakeSnapshotStore) Load(
	_ context.Context,
	_ event.AggregateRef,
) (*snapshot.Snapshot, error) {
	return s.loadSnapshot()
}

func (s *FakeSnapshotStore) LoadAtVersion(
	_ context.Context,
	_ event.AggregateRef,
	_ event.Version,
) (*snapshot.Snapshot, error) {
	return s.loadSnapshot()
}

func (s *FakeSnapshotStore) loadSnapshot() (*snapshot.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot, s.loadErr
}

func (s *FakeSnapshotStore) Delete(
	_ context.Context,
	_ event.AggregateRef,
) error {
	return nil
}

func (s *FakeSnapshotStore) Close() error { return nil }

var (
	_ snapshot.SnapshotSink   = (*FakeSnapshotStore)(nil)
	_ snapshot.SnapshotSource = (*FakeSnapshotStore)(nil)
	_ snapshot.SnapshotStore  = (*FakeSnapshotStore)(nil)
)
