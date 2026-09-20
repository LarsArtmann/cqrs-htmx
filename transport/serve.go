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
	maxReplay          int
}

// SSEOptions collects the [ServeDomainEvents] configuration in one struct for
// consumers that prefer structure over functional options. The zero value
// means "all defaults" — empty strings keep the built-in defaults, nil Filter
// disables filtering, and MaxReplay <= 0 keeps the store's own replay limit.
//
// It exists so setup-style composition roots can embed one SSE config block
// instead of threading four option closures through their own option chain.
//
//	handler := transport.ServeDomainEvents(hub, store, 15*time.Second,
//	    transport.SSEOptions{LogPrefix: "myapp"}.Options()...)
//
// [SSEOptions.Options] converts the struct to the equivalent functional
// options; the two forms are interchangeable.
type SSEOptions struct {
	// LogPrefix is the prefix for slog warnings on replay failures.
	// Empty keeps the default ("transport").
	LogPrefix string

	// UnavailableMessage is the 503 body served when the broadcaster is nil.
	// Empty keeps the default ("SSE not available").
	UnavailableMessage string

	// Filter restricts both the live stream and the replay to matching
	// events (see [WithSSEFilter]). Nil disables filtering.
	Filter func(sse.Event) bool

	// MaxReplay caps the number of replayed/backfilled events per
	// connection (see [WithSSEMaxReplay]). Values <= 0 keep the store's own
	// replay limit.
	MaxReplay int
}

// Options converts the struct into the equivalent slice of functional
// options for [ServeDomainEvents]. Zero-valued fields contribute no option,
// so the defaults apply.
func (o SSEOptions) Options() []ServeDomainEventsOption {
	var opts []ServeDomainEventsOption

	if o.LogPrefix != "" {
		opts = append(opts, WithSSELogPrefix(o.LogPrefix))
	}

	if o.UnavailableMessage != "" {
		opts = append(opts, WithSSEUnavailableMessage(o.UnavailableMessage))
	}

	if o.Filter != nil {
		opts = append(opts, WithSSEFilter(o.Filter))
	}

	if o.MaxReplay > 0 {
		opts = append(opts, WithSSEMaxReplay(o.MaxReplay))
	}

	return opts
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

// DefaultRetryHintMillis is the reconnect back-off (in milliseconds) served
// SSE streams advertise to browsers via the retry: field. Sent once before
// the first event; per the SSE spec the value persists across reconnects of
// the same EventSource, so a server restart produces a gentle reconnect
// cadence instead of a stampede.
const DefaultRetryHintMillis uint = 5000

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

// WithSSEMaxReplay caps the number of events replayed/backfilled per
// connection at the HANDLER level, truncating to the MOST RECENT n events —
// the same tail semantics [NewJournalSSEStore] applies via [WithMaxReplay].
// It works with any [sse.EventStore], including test fakes and custom
// stores, so consumers do not need the journal-backed store just for the cap.
//
// A value <= 0 (the default) keeps the store's own replay limit untouched.
// When both this option and a store-level limit are set, the tighter cap wins
// (the handler truncates after the store has already limited).
func WithSSEMaxReplay(n int) ServeDomainEventsOption {
	return func(c *serveDomainEventsConfig) { c.maxReplay = n }
}

// replayAdjustedStore wraps an [sse.EventStore] with the handler-level
// replay adjustments: tail-truncation to maxReplay (> 0) and the reconnect
// retry hint stamped on replayed events whose Retry is zero. Both apply to
// the reconnect/backfill path only — live events are untouched.
type replayAdjustedStore struct {
	inner    sse.EventStore
	maxN     int
	retryHit uint
}

func (s *replayAdjustedStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	events, err := s.inner.EventsAfter(lastID)
	if err != nil {
		//nolint:wrapcheck // adapter propagates the store error verbatim
		return nil, err
	}

	if s.maxN > 0 && len(events) > s.maxN {
		events = events[len(events)-s.maxN:]
	}

	for i := range events {
		if events[i].Retry == 0 && s.retryHit > 0 {
			events[i].Retry = s.retryHit
		}
	}

	return events, nil
}

// filteredEventStore adapts any [sse.EventStore] to [sse.FilteredEventStore]
// by filtering in-memory. It is the fail-closed replay path for WithSSEFilter.
type filteredEventStore struct {
	inner sse.EventStore
	match func(sse.Event) bool
}

func (f *filteredEventStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	return f.inner.EventsAfter(lastID) //nolint:wrapcheck // adapter propagates the store error verbatim
}

func (f *filteredEventStore) EventsAfterFiltered(
	lastID sse.EventID,
	pred func(sse.Event) bool,
) ([]sse.Event, error) {
	events, err := f.inner.EventsAfter(lastID)
	if err != nil {
		return nil, err //nolint:wrapcheck // adapter propagates the store error verbatim
	}

	out := make([]sse.Event, 0, len(events))

	for _, evt := range events {
		if pred(evt) {
			out = append(out, evt)
		}
	}

	return out, nil
}

// subscribe applies the configured filter to the live subscription (nil
// filter = unfiltered Subscribe).
func (c serveDomainEventsConfig) subscribe(b *sse.Broadcaster[sse.Event]) <-chan sse.Event {
	if c.filter != nil {
		return b.SubscribeFilter(c.filter)
	}

	return b.Subscribe()
}

// replayEvents writes the journal backfill to the stream, honoring the
// configured filter. A nil store means live-only (no backfill).
//
// Replayed events whose Retry is zero carry the handler's reconnect hint
// ([DefaultRetryHintMillis]) — a client whose original retry frame was lost
// (proxy buffering, page reload mid-stream) re-learns the cadence from the
// backfill itself, and per the SSE spec the field simply persists for the
// connection.
func (c serveDomainEventsConfig) replayEvents(stream *sse.Stream, store sse.EventStore) {
	if store == nil {
		return
	}

	store = &replayAdjustedStore{inner: store, maxN: c.maxReplay, retryHit: DefaultRetryHintMillis}

	lastID := stream.LastEventID()

	if c.filter != nil {
		filtered := &filteredEventStore{inner: store, match: c.filter}

		if _, err := sse.ReplayFiltered(stream, filtered, lastID, c.filter); err != nil {
			slog.Warn(c.logPrefix+": SSE replay failed", "error", err, "lastEventID", lastID.Get())
		}

		return
	}

	if _, err := sse.Replay(stream, store, lastID); err != nil {
		slog.Warn(c.logPrefix+": SSE replay failed", "error", err, "lastEventID", lastID.Get())
	}
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
		filter:             nil,
		maxReplay:          0,
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

		// Sent before the first event so even a connection dropped during
		// startup carries the back-off hint to the browser.
		if err := sse.WriteRetry(w, DefaultRetryHintMillis); err != nil {
			return
		}

		// Subscribe BEFORE replay to avoid missing events during the replay
		// window — live events buffer in the channel while replay writes.
		ch := cfg.subscribe(broadcaster)
		defer broadcaster.Unsubscribe(ch)

		_ = stream.Send(sse.Event{Event: sse.EventConnected, Data: "connected"})

		cfg.replayEvents(stream, store)

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
