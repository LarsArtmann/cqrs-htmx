package cqrshtmx

import (
	"context"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-sse"
)

// EventToSSEMapper converts a domain event from the event store into an SSE
// event suitable for browser replay. The consumer provides this function
// because only the consumer knows how to render event payloads (e.g., as HTML
// fragments for HTMX swaps).
//
// The mapper MUST set evt.ID to the event's ID (typically evt.ID().String())
// so that reconnection cursors work correctly.
type EventToSSEMapper func(evt event.Event) sse.Event

// JournalSSEStore implements [SSEEventStore] by reading from a go-cqrs-lite
// [event.Journal] or [event.SeekableJournal]. It is the production
// implementation of the SSEEventStore interface — replacing the test-only
// memoryEventStore.
//
// When the underlying journal supports [event.SeekableJournal] (position-based
// ReadFrom), EventsAfter uses it for efficient cursor-based replay. Otherwise
// it falls back to [event.Journal.ReadAll] and filters in-memory.
//
// All methods are safe for concurrent access (the underlying journal must also
// be safe for concurrent access, which all go-cqrs-lite stores are).
type JournalSSEStore struct {
	journal   event.Journal
	seekable  event.SeekableJournal // non-nil if the journal supports ReadFrom
	mapper    EventToSSEMapper
	maxReplay int // 0 = unlimited
}

// JournalSSEStoreOption configures a JournalSSEStore.
type JournalSSEStoreOption func(*JournalSSEStore)

// WithMaxReplay limits the number of events returned by EventsAfter. Use this
// to prevent unbounded replay on first connection (when lastID is empty). A
// value of 0 means unlimited (default: 1000).
func WithMaxReplay(n int) JournalSSEStoreOption {
	return func(s *JournalSSEStore) {
		s.maxReplay = n
	}
}

// DefaultMaxReplay is the default limit for EventsAfter when lastID is empty
// (first connection). Prevents replaying the entire event history on reconnect.
const DefaultMaxReplay = 1000

// NewJournalSSEStore creates a production [SSEEventStore] backed by the given
// event journal. The mapper function converts domain events to SSE events.
//
// If the journal also implements [event.SeekableJournal], position-based
// ReadFrom is used for efficient cursor-based replay. Otherwise, ReadAll is
// used with in-memory filtering.
//
// Panics if journal or mapper is nil (programming error).
func NewJournalSSEStore(
	journal event.Journal,
	mapper EventToSSEMapper,
	opts ...JournalSSEStoreOption,
) *JournalSSEStore {
	if journal == nil {
		//cqrs-lint:ignore(C009) constructor guard: nil journal is a programmer error
		panic(
			"cqrshtmx: NewJournalSSEStore: journal must not be nil",
		)
	}

	if mapper == nil {
		//cqrs-lint:ignore(C009) constructor guard: nil mapper is a programmer error
		panic(
			"cqrshtmx: NewJournalSSEStore: mapper must not be nil",
		)
	}

	s := &JournalSSEStore{
		journal:   journal,
		mapper:    mapper,
		maxReplay: DefaultMaxReplay,
	}

	if seekable, ok := journal.(event.SeekableJournal); ok {
		s.seekable = seekable
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// EventsAfter returns events with IDs strictly after the given lastID, ordered
// ascending by event ID. This implements [SSEEventStore].
//
// If lastID is empty, returns the most recent events (up to maxReplay).
// If lastID is not found, returns an empty slice.
//
// Returns an error if the underlying journal read fails. Invalid cursor IDs
// are treated as "not found" (empty slice, no error).
func (s *JournalSSEStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	ctx := context.Background()

	if s.seekable != nil {
		return s.eventsAfterSeekable(ctx, lastID)
	}

	return s.eventsAfterFullScan(ctx, lastID)
}

// eventsAfterSeekable uses ReadFrom for efficient position-based replay.
func (s *JournalSSEStore) eventsAfterSeekable(ctx context.Context, lastID sse.EventID) ([]sse.Event, error) {
	if lastID.Get() == "" {
		// No cursor — return the most recent events (consistent with fullScan path).
		// We can't use ReadFrom(zero, limit) because that returns the FIRST N,
		// not the last N. So we ReadAll and slice the tail.
		events, err := s.journal.ReadAll(ctx)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"cqrshtmx.sse.journal_readall_failed", "read all events from journal")
		}

		limit := s.maxReplay
		if limit == 0 {
			limit = DefaultMaxReplay
		}

		if limit > 0 && len(events) > limit {
			events = events[len(events)-limit:]
		}

		return s.mapEvents(events), nil
	}

	afterID, err := id.ParseEventID(lastID.Get())
	if err != nil {
		// Invalid cursor — treat as "not found" rather than a store error.
		return nil, nil //nolint:nilerr // intentional: invalid cursor is recoverable, not a store failure
	}

	limit := s.maxReplay
	if limit == 0 {
		limit = DefaultMaxReplay
	}

	events, err := s.seekable.ReadFrom(ctx, afterID, limit)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"cqrshtmx.sse.journal_read_failed", "read events from journal")
	}

	return s.mapEvents(events), nil
}

// eventsAfterFullScan falls back to ReadAll + in-memory filter when the
// journal does not support SeekableJournal.
func (s *JournalSSEStore) eventsAfterFullScan(ctx context.Context, lastID sse.EventID) ([]sse.Event, error) {
	events, err := s.journal.ReadAll(ctx)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"cqrshtmx.sse.journal_readall_failed", "read all events from journal")
	}

	if lastID.Get() == "" {
		// No cursor — return the last maxReplay events
		start := 0
		if s.maxReplay > 0 && len(events) > s.maxReplay {
			start = len(events) - s.maxReplay
		}

		return s.mapEvents(events[start:]), nil
	}

	// Find the position after lastID
	startIdx := slices.IndexFunc(events, func(evt event.Event) bool {
		return evt.ID().String() == lastID.Get()
	})

	if startIdx == -1 {
		// lastID not found — return empty (matches memoryEventStore behavior)
		return nil, nil
	}

	result := events[startIdx+1:]
	if s.maxReplay > 0 && len(result) > s.maxReplay {
		result = result[:s.maxReplay]
	}

	return s.mapEvents(result), nil
}

// mapEvents converts domain events to SSE events using the consumer-provided mapper.
func (s *JournalSSEStore) mapEvents(events []event.Event) []sse.Event {
	if len(events) == 0 {
		return nil
	}

	result := make([]sse.Event, 0, len(events))
	for _, evt := range events {
		result = append(result, s.mapper(evt))
	}

	return result
}
