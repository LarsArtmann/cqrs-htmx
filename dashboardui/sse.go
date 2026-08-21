package dashboardui

import (
	"context"
	"net/http"

	"github.com/larsartmann/cqrs-htmx/v4/transport"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// startEventBridge subscribes to the event bus and forwards every event
// to the internal SSE broadcaster. Called once during [New] when
// Config.EventBus is configured.
//
// Returns an error if the event bus cannot accept the subscription — a
// dashboard that cannot subscribe to live events is a construction failure,
// not a silently dead SSE endpoint answering 200.
func (d *Dashboard) startEventBridge() error {
	if d.config.EventBus == nil || d.broadcaster == nil {
		return nil
	}

	handler := func(_ context.Context, evt event.Event) error {
		select {
		case <-d.done:
			return nil
		default:
		}

		d.broadcaster.Broadcast(transport.DomainEventToSSE(evt))

		return nil
	}

	//cqrs-lint:ignore(C027,A005) SSE fan-out bridge for live dashboard updates, not a read-model projection
	if err := d.config.EventBus.SubscribeAll(handler); err != nil {
		return errorfamily.WrapInfrastructure(err,
			"dashboardui.sse_subscribe_failed", "subscribe to event bus for SSE bridge")
	}

	return nil
}

// sseHandler returns the SSE stream handler. Each connected client
// receives a live feed of events as they are published to the event bus.
//
// On reconnection (Last-Event-ID header present), missed events are replayed
// from the journal before the live feed begins. On first connect, recent
// history is backfilled so the dashboard shows immediate activity.
//
// If no event bus was configured, the handler responds 503 per request.
func (d *Dashboard) sseHandler() http.HandlerFunc {
	if d.broadcaster == nil {
		return transport.ServeDomainEvents(
			nil, nil, d.config.SSEHeartbeatInterval,
			transport.WithSSEUnavailableMessage("SSE not available (no event bus configured)"),
		)
	}

	return transport.ServeDomainEvents(
		d.broadcaster.Hub(),
		d.sseStore,
		d.config.SSEHeartbeatInterval,
		transport.WithSSELogPrefix("dashboardui"),
		transport.WithSSEUnavailableMessage("SSE not available (no event bus configured)"),
	)
}
