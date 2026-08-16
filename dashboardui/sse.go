package dashboardui

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/larsartmann/cqrs-htmx/v4/transport"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-sse"
)

// startEventBridge subscribes to the event bus and forwards every event
// to the internal SSE broadcaster. Called once during [New] when
// Config.EventBus is configured.
func (d *Dashboard) startEventBridge() {
	if d.config.EventBus == nil || d.broadcaster == nil {
		return
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
	if err := d.config.EventBus.SubscribeAll(
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

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	// Subscribe BEFORE replay to avoid missing events during the replay window.
	// Live events buffer in the channel while replay writes to the stream.
	ch := d.broadcaster.Subscribe()
	defer d.broadcaster.Unsubscribe(ch)

	_ = stream.Send(sse.Event{Event: sse.EventConnected, Data: "connected"})

	// Replay missed events (reconnect with Last-Event-ID) or backfill recent
	// history (first connect with empty Last-Event-ID). Events arrive after
	// "connected" and before the live loop.
	if d.sseStore != nil {
		lastID := stream.LastEventID()

		if _, err := sse.Replay(stream, d.sseStore, lastID); err != nil {
			slog.Warn("dashboardui: SSE replay failed", "error", err, "lastEventID", lastID.Get())
		}
	}

	// Heartbeat runs alongside the event loop. Derive a cancellable context
	// and join the goroutine before this handler returns: a heartbeat write
	// racing handler teardown is a data race, and net/http forbids touching
	// the ResponseWriter after the handler has returned.
	hbCtx, hbCancel := context.WithCancel(r.Context())
	hbDone := make(chan struct{})

	if d.config.SSEHeartbeatInterval > 0 {
		go func() {
			defer close(hbDone)

			stream.Heartbeat(hbCtx, d.config.SSEHeartbeatInterval)
		}()
	} else {
		close(hbDone)
	}

	defer func() {
		hbCancel()
		<-hbDone
	}()

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
