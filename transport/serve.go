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
	filter             func(sse.Event) bool
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

// WithSSEFilter restricts both stream paths — live delivery and journal
// replay — to events matching pred. This is the mechanism behind
// stream-type-scoped SSE endpoints: the domain envelope's stream type lives
// in the payload, so a predicate on it is all a scoped endpoint needs.
//
// The live path subscribes via [sse.Broadcaster.SubscribeFilter]. For replay,
// the store is wrapped so only matching events are delivered, even when the
// store itself only implements [sse.EventStore] — a filter that leaked
// excluded events during reconnect backfill would be a security hole, never
// a degradation.
//
// nil (the default) means no filtering: every event reaches every subscriber.
func WithSSEFilter(pred func(sse.Event) bool) ServeDomainEventsOption {
	return func(c *serveDomainEventsConfig) { c.filter = pred }
}

// filteredEventStore adapts any [sse.EventStore] to [sse.FilteredEventStore]
// by filtering in-memory. It is the fail-closed replay path for WithSSEFilter.
type filteredEventStore struct {
	inner sse.EventStore
	match func(sse.Event) bool
}

func (f *filteredEventStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	return f.inner.EventsAfter(lastID)
}

func (f *filteredEventStore) EventsAfterFiltered(
	lastID sse.EventID,
	pred func(sse.Event) bool,
) ([]sse.Event, error) {
	events, err := f.inner.EventsAfter(lastID)
	if err != nil {
		return nil, err
	}

	out := make([]sse.Event, 0, len(events))

	for _, evt := range events {
		if pred(evt) {
			out = append(out, evt)
		}
	}

	return out, nil
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
// With [WithSSEFilter], both the subscription and the replay deliver only
// matching events.
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
		var ch <-chan sse.Event
		if cfg.filter != nil {
			ch = broadcaster.SubscribeFilter(cfg.filter)
		} else {
			ch = broadcaster.Subscribe()
		}
		defer broadcaster.Unsubscribe(ch)

		_ = stream.Send(sse.Event{Event: sse.EventConnected, Data: "connected"})

		if store != nil {
			lastID := stream.LastEventID()

			if cfg.filter != nil {
				if _, err := sse.ReplayFiltered(
					stream,
					&filteredEventStore{inner: store, match: cfg.filter},
					lastID,
					cfg.filter,
				); err != nil {
					slog.Warn(cfg.logPrefix+": SSE replay failed", "error", err, "lastEventID", lastID.Get())
				}
			} else if _, err := sse.Replay(stream, store, lastID); err != nil {
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
