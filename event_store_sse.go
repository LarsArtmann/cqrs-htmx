package cqrshtmx

import (
	"github.com/larsartmann/cqrs-htmx/v4/transport"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// EventToSSEMapper converts a domain event from the event store into an SSE
// event suitable for browser replay.
//
// Deprecated: use [transport.EventToSSEMapper]. The type is identical; only
// the canonical home moved to the lean transport sub-package so consumers can
// import journal-backed SSE replay without compiling the framework root.
type EventToSSEMapper = transport.EventToSSEMapper

// JournalSSEStore implements go-sse's sse.EventStore by reading from a
// go-cqrs-lite [event.Journal] or [event.SeekableJournal].
//
// Deprecated: use [transport.JournalSSEStore]. The implementation moved to
// the transport sub-package (dependency-lean: event/id, go-sse, error-family
// only). This alias keeps existing imports compiling.
type JournalSSEStore = transport.JournalSSEStore

// JournalSSEStoreOption configures a JournalSSEStore.
//
// Deprecated: use [transport.JournalSSEStoreOption].
type JournalSSEStoreOption = transport.JournalSSEStoreOption

// WithMaxReplay limits the number of events returned by EventsAfter.
//
// Deprecated: use [transport.WithMaxReplay].
func WithMaxReplay(n int) JournalSSEStoreOption {
	return transport.WithMaxReplay(n)
}

// DefaultMaxReplay is the default replay limit on first connection.
//
// Deprecated: use [transport.DefaultMaxReplay].
const DefaultMaxReplay = transport.DefaultMaxReplay

// NewJournalSSEStore creates an sse.EventStore backed by the given event
// journal.
//
// Deprecated: use [transport.NewJournalSSEStore] — identical behavior,
// leaner import.
func NewJournalSSEStore(
	journal event.Journal,
	mapper EventToSSEMapper,
	opts ...JournalSSEStoreOption,
) *JournalSSEStore {
	return transport.NewJournalSSEStore(journal, mapper, opts...)
}
