package eventtest

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

type FakeStore struct {
	mu                sync.RWMutex
	events            map[string][]event.Event
	saveFn            event.SaveFunc
	loadFn            func(ref event.AggregateRef) ([]event.Event, error)
	loadFromVersionFn func(ref event.AggregateRef, version event.Version) ([]event.Event, error)
	loadToVersionFn   func(ref event.AggregateRef, maxVersion event.Version) ([]event.Event, error)
	loadToTimestampFn func(ref event.AggregateRef, maxTime time.Time) ([]event.Event, error)
	appendBatchFn     func(ref event.AggregateRef, events []event.Event) error
	closeFn           func() error
	readAllFn         func() ([]event.Event, error)
	readFromFn        func(afterEventID id.EventID, limit int) ([]event.Event, error)
}

func NewFakeStore() *FakeStore {
	return &FakeStore{events: make(map[string][]event.Event)}
}

func getOverride[T any](s *FakeStore, fn *T) T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return *fn
}

func (s *FakeStore) Save(
	ctx context.Context,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if fn := getOverride(s, &s.saveFn); fn != nil {
		return fn(ctx, ref, events, expectedVersion)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := ref.StreamKey()
	s.events[key] = append(s.events[key], events...)

	return nil
}

func (s *FakeStore) AppendBatch(
	_ context.Context,
	ref event.AggregateRef,
	events []event.Event,
) error {
	if fn := getOverride(s, &s.appendBatchFn); fn != nil {
		return fn(ref, events)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := ref.StreamKey()
	s.events[key] = append(s.events[key], events...)

	return nil
}

func (s *FakeStore) Load(
	_ context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	if fn := getOverride(s, &s.loadFn); fn != nil {
		return fn(ref)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := ref.StreamKey()

	return append([]event.Event{}, s.events[key]...), nil
}

func (s *FakeStore) loadEventsHelper(
	ref event.AggregateRef,
) []event.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := ref.StreamKey()

	return s.events[key]
}

func (s *FakeStore) LoadFromVersion(
	_ context.Context,
	ref event.AggregateRef,
	version event.Version,
) ([]event.Event, error) {
	if fn := getOverride(s, &s.loadFromVersionFn); fn != nil {
		return fn(ref, version)
	}

	result := event.SliceFromVersion(s.loadEventsHelper(ref), version)
	if len(result) == 0 {
		return nil, nil
	}

	return append([]event.Event{}, result...), nil
}

func (s *FakeStore) LoadToVersion(
	_ context.Context,
	ref event.AggregateRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	if fn := getOverride(s, &s.loadToVersionFn); fn != nil {
		return fn(ref, maxVersion)
	}

	return event.SliceToVersion(s.loadEventsHelper(ref), maxVersion), nil
}

func (s *FakeStore) LoadToTimestamp(
	_ context.Context,
	ref event.AggregateRef,
	maxTime time.Time,
) ([]event.Event, error) {
	if fn := getOverride(s, &s.loadToTimestampFn); fn != nil {
		return fn(ref, maxTime)
	}

	return event.FilterByTimestamp(s.loadEventsHelper(ref), maxTime), nil
}

func (s *FakeStore) ReadAll(_ context.Context) ([]event.Event, error) {
	if fn := getOverride(s, &s.readAllFn); fn != nil {
		return fn()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []event.Event

	for _, evts := range s.events {
		all = append(all, evts...)
	}

	return all, nil
}

func (s *FakeStore) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	if fn := getOverride(s, &s.readFromFn); fn != nil {
		return fn(afterEventID, limit)
	}

	all, err := s.ReadAll(ctx)
	if err != nil {
		return nil, errorfamily.Wrapf(err, event.Infrastructure, "eventtest.read_from",
			"read all for ReadFrom (limit=%d, after=%s)", limit, afterEventID)
	}

	startIdx := 0

	if !afterEventID.IsZero() {
		for i, e := range all {
			if e.ID() == afterEventID {
				startIdx = i + 1

				break
			}
		}
	}

	filtered := all[startIdx:]
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

func (s *FakeStore) Close() error {
	if fn := getOverride(s, &s.closeFn); fn != nil {
		return fn()
	}

	return nil
}

func (s *FakeStore) SaveFn(fn event.SaveFunc) *FakeStore {
	s.saveFn = fn

	return s
}

func (s *FakeStore) LoadFn(
	fn func(ref event.AggregateRef) ([]event.Event, error),
) *FakeStore {
	s.loadFn = fn

	return s
}

func (s *FakeStore) LoadFromVersionFn(
	fn func(ref event.AggregateRef, version event.Version) ([]event.Event, error),
) *FakeStore {
	s.loadFromVersionFn = fn

	return s
}

func (s *FakeStore) LoadToVersionFn(
	fn func(ref event.AggregateRef, maxVersion event.Version) ([]event.Event, error),
) *FakeStore {
	s.loadToVersionFn = fn

	return s
}

func (s *FakeStore) LoadToTimestampFn(
	fn func(ref event.AggregateRef, maxTime time.Time) ([]event.Event, error),
) *FakeStore {
	s.loadToTimestampFn = fn

	return s
}

func (s *FakeStore) CloseFn(fn func() error) *FakeStore {
	s.closeFn = fn

	return s
}

func (s *FakeStore) AppendBatchFn(
	fn func(ref event.AggregateRef, events []event.Event) error,
) *FakeStore {
	s.appendBatchFn = fn

	return s
}

func VersionQueryFn(
	called *bool,
) func(event.AggregateRef, event.Version) ([]event.Event, error) {
	return func(_ event.AggregateRef, _ event.Version) ([]event.Event, error) {
		*called = true

		return nil, nil
	}
}

var (
	_ event.Store           = (*FakeStore)(nil)
	_ event.Journal         = (*FakeStore)(nil)
	_ event.SeekableJournal = (*FakeStore)(nil)
)
