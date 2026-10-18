// Package transport provides lean transport-layer adapters that bridge
// go-cqrs-lite event storage onto wire protocols — without importing the
// cqrs-htmx root package. Consumers that only need, say, journal-backed SSE
// replay can import this sub-package alone: it depends on nothing beyond
// go-cqrs-lite event/id, go-sse, and go-error-family, so the framework's
// root package (authz, CSRF, dispatch machinery) never enters their build.
package transport

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

// JournalSSEStore implements [sse.EventStore] by reading from a go-cqrs-lite
// [event.Journal] or [event.SeekableJournal]. It is the production
// implementation for SSE replay — replacing test-only in-memory stores.
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
	maxReplay int // <= 0 = DefaultMaxReplay
}

// JournalSSEStoreOption configures a JournalSSEStore.
type JournalSSEStoreOption func(*JournalSSEStore)

// WithMaxReplay limits the number of events returned by EventsAfter. Use this
// to prevent unbounded replay on first connection (when lastID is empty).
// A value <= 0 keeps the default ([DefaultMaxReplay]).
func WithMaxReplay(n int) JournalSSEStoreOption {
	return func(s *JournalSSEStore) {
		s.maxReplay = n
	}
}

// DefaultMaxReplay is the default limit for EventsAfter when lastID is empty
// (first connection). Prevents replaying the entire event history on connect.
const DefaultMaxReplay = 1000

// NewJournalSSEStore creates a production [sse.EventStore] backed by the given
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
		panic(
			"transport: NewJournalSSEStore: journal must not be nil",
		)
	}

	if mapper == nil {
		panic(
			"transport: NewJournalSSEStore: mapper must not be nil",
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
// ascending by event ID. This implements [sse.EventStore].
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

// replayLimit returns the configured replay cap, defaulting to
// DefaultMaxReplay when unset or non-positive.
func (s *JournalSSEStore) replayLimit() int {
	if s.maxReplay <= 0 {
		return DefaultMaxReplay
	}

	return s.maxReplay
}

// eventsAfterSeekable uses ReadFrom for efficient position-based replay.
func (s *JournalSSEStore) eventsAfterSeekable(
	ctx context.Context,
	lastID sse.EventID,
) ([]sse.Event, error) {
	afterID := id.EventID{}
	if lastID.Get() != "" {
		parsed, err := id.ParseEventID(lastID.Get())
		if err != nil {
			// Invalid cursor — treat as "not found" rather than a store error.
			return nil, nil //nolint:nilerr // intentional: invalid cursor is recoverable, not a store failure
		}
		afterID = parsed
	}

	// Tail semantics: the replay cap must deliver the MOST RECENT n events
	// after the cursor, not the first n. ReadFrom is forward-only and returns
	// the OLDEST events first, so passing the cap as the ReadFrom limit would
	// deliver [cursor+1, cursor+limit] and then jump to live — silently and
	// permanently dropping everything in between. Paged reads with a rolling
	// tail keep the delivery window correct at bounded memory (≤ cap + one
	// batch) and also replace the per-connection full ReadAll the no-cursor
	// path previously paid.
	limit := s.replayLimit()

	var (
		tail   []event.Event
		cursor = afterID
	)

	for {
		batch, err := s.seekable.ReadFrom(ctx, cursor, limit)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"transport.sse.journal_read_failed", "read events from journal").
				WithContext("last_id", lastID.Get())
		}

		if len(batch) == 0 {
			break
		}

		tail = append(tail, batch...)
		if len(tail) > limit {
			tail = tail[len(tail)-limit:]
		}

		cursor = batch[len(batch)-1].ID()

		if len(batch) < limit {
			break
		}
	}

	return s.mapEvents(tail), nil
}

// eventsAfterFullScan falls back to ReadAll + in-memory filter when the
// journal does not support SeekableJournal.
func (s *JournalSSEStore) eventsAfterFullScan(
	ctx context.Context,
	lastID sse.EventID,
) ([]sse.Event, error) {
	events, err := s.journal.ReadAll(ctx)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"transport.sse.journal_readall_failed", "read all events from journal").
			WithContext("last_id", lastID.Get())
	}

	if lastID.Get() == "" {
		// No cursor — return the last maxReplay events.
		limit := s.replayLimit()
		start := 0
		if limit > 0 && len(events) > limit {
			start = len(events) - limit
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

	// Tail semantics: a capped reconnect delivers the MOST RECENT n events
	// after the cursor — head-truncating here would silently drop the gap
	// between the delivered window and the live subscription point.
	limit := s.replayLimit()
	result := events[startIdx+1:]
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
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
