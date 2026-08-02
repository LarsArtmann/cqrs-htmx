package dashboardui

import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// sseEventPayload is the JSON shape sent to the browser on each SSE push.
type sseEventPayload struct {
	Type       string `json:"type"`
	StreamType string `json:"streamType"`
	StreamID   string `json:"streamId"`
	Version    uint64 `json:"version"`
	OccurredAt string `json:"occurredAt"`
	EventID    string `json:"eventId"`
}

func newSSEEvent(evt event.Event) cqrshtmx.SSEEvent {
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
		slog.Error("dashboardui: marshal SSE event", "error", err, "eventType", payload.Type)

		return cqrshtmx.SSEEvent{
			Event: "event",
			ID:    cqrshtmx.NewSSEEventID(payload.EventID),
		}
	}

	return cqrshtmx.SSEEvent{
		Event: "event",
		Data:  string(data),
		ID:    cqrshtmx.NewSSEEventID(payload.EventID),
	}
}

// startEventBridge subscribes to the event bus and forwards every event
// to the internal SSE broadcaster. Called once during [New] when
// Config.EventBus is configured.
func (d *Dashboard) startEventBridge() {
	if d.cfg.EventBus == nil || d.broadcaster == nil {
		return
	}

	handler := func(_ context.Context, evt event.Event) error {
		select {
		case <-d.done:
			return nil
		default:
		}

		d.broadcaster.Broadcast(newSSEEvent(evt))

		return nil
	}

	//cqrs-lint:ignore(A005) SSE fan-out bridge for live dashboard updates, not a read-model projection
	if err := d.cfg.EventBus.SubscribeAll(
		handler,
	); err != nil {
		slog.Error("dashboardui: subscribe to event bus", "error", err)
	}
}

// sseHandler serves the SSE stream endpoint. Each connected client
// receives a live feed of events as they are published to the event bus.
//
// On reconnection (Last-Event-ID header present), missed events are replayed
// from the journal before the live feed begins. On first connect, recent
// history is backfilled so the dashboard shows immediate activity.
func (d *Dashboard) sseHandler(w http.ResponseWriter, r *http.Request) {
	if d.broadcaster == nil {
		http.Error(w, "SSE not available (no event bus configured)", http.StatusServiceUnavailable)

		return
	}

	stream := cqrshtmx.NewSSEStream(w, r)
	defer func() { _ = stream.Close() }()

	// Subscribe BEFORE replay to avoid missing events during the replay window.
	// Live events buffer in the channel while replay writes to the stream.
	ch := d.broadcaster.Subscribe()
	defer d.broadcaster.Unsubscribe(ch)

	_ = stream.Send(cqrshtmx.SSEEvent{Event: cqrshtmx.SSEEventConnected, Data: "connected"})

	// Replay missed events (reconnect with Last-Event-ID) or backfill recent
	// history (first connect with empty Last-Event-ID). Events arrive after
	// "connected" and before the live loop.
	if d.sseStore != nil {
		lastID := stream.LastEventID()

		if _, err := cqrshtmx.ReplayEvents(stream, d.sseStore, lastID); err != nil {
			slog.Warn("dashboardui: SSE replay failed", "error", err, "lastEventID", lastID.Get())
		}
	}

	if d.cfg.SSEHeartbeatInterval > 0 {
		go stream.Heartbeat(stream.Context(), d.cfg.SSEHeartbeatInterval) //nolint:contextcheck
	}

	for {
		select {
		case <-stream.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok || stream.Send(evt) != nil {
				return
			}
		}
	}
}
