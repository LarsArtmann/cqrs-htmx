package transport

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/larsartmann/go-sse"
)

// ServeDomainEventsOption configures a [ServeDomainEvents] handler.
type ServeDomainEventsOption func(*serveDomainEventsConfig)

type serveDomainEventsConfig struct {
	logPrefix          string
	unavailableMessage string
}

// WithSSELogPrefix sets the prefix used in slog warnings for replay failures.
// Default: "transport".
func WithSSELogPrefix(prefix string) ServeDomainEventsOption {
	return func(c *serveDomainEventsConfig) { c.logPrefix = prefix }
}

// WithSSEUnavailableMessage sets the body of the 503 response returned when
// the broadcaster is nil. Default: "SSE not available".
func WithSSEUnavailableMessage(msg string) ServeDomainEventsOption {
	return func(c *serveDomainEventsConfig) { c.unavailableMessage = msg }
}

// ServeDomainEvents returns an [http.HandlerFunc] that streams domain events
// over SSE with the full connection lifecycle used by the setup and
// dashboardui SSE endpoints:
//
//  1. "connected" event,
//  2. journal replay/backfill (when store is non-nil),
//  3. heartbeat comment frames (when interval > 0),
//  4. the live event pump until the client disconnects.
//
// The caller owns authentication and authorization — wrap the returned handler
// with session/authz middleware before mounting. This helper only owns the SSE
// mechanics; it does not import the cqrs-htmx root package.
//
// If broadcaster is nil, the handler responds 503 (Service Unavailable) per
// request. If store is nil, replay/backfill is skipped (live-only feed). If
// heartbeat is non-positive, heartbeats are disabled.
//
// Subscribe is called BEFORE replay so live events buffer in the channel while
// replay writes to the stream — no event is lost during the replay window.
func ServeDomainEvents(
	broadcaster *sse.Broadcaster[sse.Event],
	store sse.EventStore,
	heartbeat time.Duration,
	opts ...ServeDomainEventsOption,
) http.HandlerFunc {
	cfg := serveDomainEventsConfig{
		logPrefix:          "transport",
		unavailableMessage: "SSE not available",
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if broadcaster == nil {
			http.Error(w, cfg.unavailableMessage, http.StatusServiceUnavailable)

			return
		}

		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		// Subscribe BEFORE replay to avoid missing events during the replay
		// window — live events buffer in the channel while replay writes.
		ch := broadcaster.Subscribe()
		defer broadcaster.Unsubscribe(ch)

		_ = stream.Send(sse.Event{Event: sse.EventConnected, Data: "connected"})

		if store != nil {
			lastID := stream.LastEventID()

			if _, err := sse.Replay(stream, store, lastID); err != nil {
				slog.Warn(cfg.logPrefix+": SSE replay failed", "error", err, "lastEventID", lastID.Get())
			}
		}

		// Heartbeat runs alongside the event loop. Derive a cancellable
		// context and join the goroutine before this handler returns: a
		// heartbeat write racing handler teardown is a data race, and
		// net/http forbids touching the ResponseWriter after return.
		hbCtx, hbCancel := context.WithCancel(r.Context())
		hbDone := make(chan struct{})

		if heartbeat > 0 {
			go func() {
				defer close(hbDone)

				stream.Heartbeat(hbCtx, heartbeat)
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
}
