package cqrshtmx

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
func ReplayEvents(stream *SSEStream, store SSEEventStore, lastEventID string) (int, error) {
	events := store.EventsAfter(lastEventID)
	for i, evt := range events {
		if err := stream.Send(evt); err != nil {
			return i, err
		}
	}
	return len(events), nil
}

// Broadcaster distributes SSE events to all subscribed clients.
// It is safe for concurrent use.
//
// Create one at application startup and share it across handlers:
//
//	broadcaster := cqrshtmx.NewBroadcaster()
//
//	// In your SSE endpoint handler:
//	ch := broadcaster.Subscribe()
//	defer broadcaster.Unsubscribe(ch)
//
//	// In your CQRS event handler or AfterDispatch hook:
//	broadcaster.Broadcast(cqrshtmx.SSEEvent{
//	    Event: "itemCreated",
//	    Data:  renderTemplate(),
