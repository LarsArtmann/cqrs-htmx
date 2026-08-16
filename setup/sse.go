package setup

import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-sse"
)

// sseEventPayload is the JSON envelope streamed on the [Config.SSEPath]
// endpoint. It mirrors the dashboardui SSE shape so consumers can reuse the
// same client-side parsing for both feeds.
type sseEventPayload struct {
	Type       string `json:"type"`
	StreamType string `json:"streamType"`
	StreamID   string `json:"streamId"`
	Version    uint64 `json:"version"`
	OccurredAt string `json:"occurredAt"`
	EventID    string `json:"eventId"`
}

func newSSEEvent(evt event.Event) sse.Event {
	payload := sseEventPayload{
		Type:       string(evt.Type()),
		StreamType: string(evt.StreamType()),
		StreamID:   evt.StreamID().String(),
		Version:    evt.Version().UInt64(),
		OccurredAt: evt.OccurredAt().Format(time.RFC3339),
		EventID:    evt.ID().String(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("setup: marshal SSE event", "error", err, "eventType", payload.Type)

		return sse.Event{ //nolint:exhaustruct // Data intentionally empty when marshalling fails; Retry unused
			Event: "event",
			ID:    sse.NewEventID(payload.EventID),
		}
	}

	return sse.Event{ //nolint:exhaustruct // Retry is an optional SSE reconnection hint, unset by design
		Event: "event",
		Data:  string(data),
		ID:    sse.NewEventID(payload.EventID),
	}
}

// attachSSE builds the shared SSE broadcaster and its event-bus bridge when
// [Config.SSEPath] is set. Called at the end of [New] so Close/cleanup own it.
func (b *Bundle) attachSSE() {
	if b.config.SSEPath == "" {
		return
	}

	b.Broadcaster = cqrshtmx.NewBroadcaster()
	b.sseDone = make(chan struct{})

	handler := func(_ context.Context, evt event.Event) error {
		select {
		case <-b.sseDone:
			return nil
		default:
		}

		b.Broadcaster.Broadcast(newSSEEvent(evt))

		return nil
	}

	//cqrs-lint:ignore(C027,A005) SSE fan-out bridge for the shared endpoint, not a read-model projection
	if err := b.Stores.EventBus.SubscribeAll(handler); err != nil {
		slog.Error("setup: subscribe to event bus for SSE bridge", "error", err)
	}
}

// sseHandler serves the shared SSE endpoint: session-gated live feed of every
// event committed to the event bus.
func (b *Bundle) sseHandler() http.Handler {
	return b.SessionMiddleware()(requireSession(http.HandlerFunc(b.Broadcaster.ServeSSE)))
}
