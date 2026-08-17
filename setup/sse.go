package setup

import (
	"context"
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/transport"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// attachSSE builds the shared SSE broadcaster, its event-bus bridge, and a
// journal-backed replay store when the event store supports it. Called at the
// end of [New] so Close/cleanup own it.
//
// Returns an error if the event bus cannot accept the SSE bridge subscription
// — an SSE endpoint that cannot subscribe is a construction failure, not a
// silently dead feed answering 200.
func (b *Bundle) attachSSE() error {
	if b.config.SSEPath == "" {
		return nil
	}

	b.Broadcaster = cqrshtmx.NewBroadcaster()
	b.sseDone = make(chan struct{})

	if journal, ok := b.Stores.EventStore.(event.Journal); ok {
		b.sseStore = transport.NewJournalSSEStore(
			journal, transport.DomainEventToSSE,
			transport.WithMaxReplay(b.config.SSEMaxReplay),
		)
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
		// Roll back the broadcaster/sseDone we just created — New returns nil on error.
		close(b.sseDone)
		b.sseDone = nil
		b.Broadcaster.Close()
		b.Broadcaster = nil

		return errorfamily.WrapInfrastructure(err,
			"setup.sse_subscribe_failed", "subscribe to event bus for SSE bridge")
	}

	return nil
}

// sseHandler serves the shared SSE endpoint: session-gated feed of every event
// committed to the event bus, with journal replay for reconnects and initial
// backfill when the event store supports it.
func (b *Bundle) sseHandler() http.Handler {
	return b.SessionMiddleware()(requireSession(transport.ServeDomainEvents(
		b.Broadcaster.Raw(),
		b.sseStore,
		b.config.SSEHeartbeatInterval,
		transport.WithSSELogPrefix("setup"),
	)))
}
