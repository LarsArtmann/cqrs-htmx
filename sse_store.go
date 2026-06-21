package cqrshtmx

import "github.com/larsartmann/go-cqrs-lite/event/v2"

// SSEEventStore retrieves events for SSE reconnection replay.
// Implementations must be safe for concurrent access.
type SSEEventStore interface {
	// EventsAfter returns events with IDs strictly after the given lastID.
	// Returns an empty slice if no events are found or lastID is unknown.
	// The returned slice must be ordered by event ID (ascending).
	EventsAfter(lastID string) []SSEEvent
}

// ReplayEvents sends all events from the store after the given lastEventID
// through the stream. This is used for SSE reconnection: when a client
// reconnects with a Last-Event-ID header, replay the events it missed.
//
// Returns the number of events replayed, or an error if writing fails.
func ReplayEvents(stream *SSEStream, store SSEEventStore, lastEventID SSEEventID) (int, error) {
	events := store.EventsAfter(lastEventID.String())
	for i, evt := range events {
		if err := stream.Send(evt); err != nil {
			return i, event.Wrapf(err, event.Transient, "cqrshtmx.sse.replay_failed",
				"replay after %q (sent %d of %d)", lastEventID, i, len(events))
		}
	}
	return len(events), nil
}
