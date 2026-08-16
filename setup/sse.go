package setup

import (
	"context"
	"log/slog"
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/transport"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-sse"
)

// attachSSE builds the shared SSE broadcaster, its event-bus bridge, and a
// journal-backed replay store when the event store supports it. Called at the
// end of [New] so Close/cleanup own it.
func (b *Bundle) attachSSE() {
	if b.config.SSEPath == "" {
		return
	}

	b.Broadcaster = cqrshtmx.NewBroadcaster()
	b.sseDone = make(chan struct{})

	if journal, ok := b.Stores.EventStore.(event.Journal); ok {
		b.sseStore = transport.NewJournalSSEStore(journal, transport.DomainEventToSSE)
	}

	handler := func(_ context.Context, evt event.Event) error {
		select {
		case <-b.sseDone:
			return nil
		default:
		}

		b.Broadcaster.Broadcast(transport.DomainEventToSSE(evt))

		return nil
	}

	//cqrs-lint:ignore(C027,A005) SSE fan-out bridge for the shared endpoint, not a read-model projection
	if err := b.Stores.EventBus.SubscribeAll(handler); err != nil {
		slog.Error("setup: subscribe to event bus for SSE bridge", "error", err)
	}
}

// sseHandler serves the shared SSE endpoint: session-gated feed of every event
// committed to the event bus, with journal replay for reconnects and initial
// backfill when the event store supports it.
func (b *Bundle) sseHandler() http.Handler {
	return b.SessionMiddleware()(requireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if b.Broadcaster == nil {
			http.Error(w, "SSE not available", http.StatusServiceUnavailable)

			return
		}

		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		ch := b.Broadcaster.Subscribe()
		defer b.Broadcaster.Unsubscribe(ch)

		_ = stream.Send(sse.Event{Event: sse.EventConnected, Data: "connected"})

		if b.sseStore != nil {
			lastID := stream.LastEventID()

			if _, err := sse.Replay(stream, b.sseStore, lastID); err != nil {
				slog.Warn("setup: SSE replay failed", "error", err, "lastEventID", lastID.Get())
			}
		}

		// Heartbeat runs alongside the event loop. Derive a cancellable
		// context and join the goroutine before this handler returns: a
		// heartbeat write racing handler teardown is a data race, and
		// net/http forbids touching the ResponseWriter after return.
		hbCtx, hbCancel := context.WithCancel(r.Context())
		hbDone := make(chan struct{})

		if b.config.SSEHeartbeatInterval > 0 {
			go func() {
				defer close(hbDone)

				stream.Heartbeat(hbCtx, b.config.SSEHeartbeatInterval)
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
	})))
}
