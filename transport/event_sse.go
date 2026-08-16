package transport

import (
	"encoding/json/v2"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-sse"
)

// sseEventType is the SSE event name for all domain events in the envelope.
const sseEventType = "event"

// EventPayload is the small JSON envelope streamed to browsers for each domain
// event. It intentionally contains only metadata (IDs, type, version,
// occurredAt) — not the full payload — so it is safe to expose over
// authenticated SSE feeds without leaking arbitrary event data.
//
// This is the canonical shape used by both the dashboardui live feed and the
// setup shared SSE endpoint; consumers building their own event bridges should
// reuse [DomainEventToSSE] rather than hand-rolling a parallel envelope.
type EventPayload struct {
	Type       string `json:"type"`
	StreamType string `json:"streamType"`
	StreamID   string `json:"streamId"`
	Version    uint64 `json:"version"`
	OccurredAt string `json:"occurredAt"`
	EventID    string `json:"eventId"`
}

// DomainEventToSSE converts a go-cqrs-lite event into a transport-neutral SSE
// event using the standard metadata envelope. It is shared by dashboardui,
// setup, and any consumer building a domain-event SSE feed, so the wire shape
// stays consistent across all endpoints.
func DomainEventToSSE(evt event.Event) sse.Event {
	payload := EventPayload{
		Type:       string(evt.Type()),
		StreamType: string(evt.StreamType()),
		StreamID:   evt.StreamID().String(),
		Version:    evt.Version().UInt64(),
		OccurredAt: evt.OccurredAt().Format(time.RFC3339),
		EventID:    evt.ID().String(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("transport: marshal SSE event payload", "error", err, "eventType", payload.Type)

		return sse.Event{
			Event: sseEventType,
			ID:    sse.NewEventID(payload.EventID),
		}
	}

	return sse.Event{
		Event: sseEventType,
		Data:  string(data),
		ID:    sse.NewEventID(payload.EventID),
	}
}
