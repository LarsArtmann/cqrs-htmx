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
	requestFilter      RequestFilterFunc
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

	// RequestFilter derives the filter per request (see
	// [WithRequestFilter]). When both are set, a request whose derivation
	// returns a predicate uses it; the construction-time Filter applies
	// only where the derivation returns nil.
	RequestFilter RequestFilterFunc

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

	if o.RequestFilter != nil {
		opts = append(opts, WithRequestFilter(o.RequestFilter))
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

// RequestFilterFunc derives a per-request event filter for
// [WithRequestFilter]. Returning an error aborts the request with 400 Bad
// Request before any stream is opened — use it to reject malformed filter
// input (unknown query params, unparsable values). Returning a nil predicate
// with a nil error means "no per-request filter"; the construction-time
// [WithSSEFilter] (if any) applies to such requests.
type RequestFilterFunc func(r *http.Request) (pred func(sse.Event) bool, err error)

// WithRequestFilter derives the SSE filter PER REQUEST instead of at
// construction time, so filter dimensions that live in the request (query
// params like ?types=&channels=&guilds=, headers, path values) can scope
// both stream paths:
//
//	h := transport.ServeDomainEvents(hub, store, 15*time.Second,
//	    transport.WithRequestFilter(func(r *http.Request) (func(sse.Event) bool, error) {
//	        typ := r.URL.Query().Get("type")
//	        if typ == "" {
//	            return nil, nil // unfiltered (or falls back to WithSSEFilter)
//	        }
//	        return func(evt sse.Event) bool { return strings.Contains(evt.Data, typ) }, nil
//	    }))
//
// The live path subscribes via [sse.Broadcaster.SubscribeFilter] with the
// derived predicate; the replay path wraps the store so only matching
// events are delivered, with the [WithSSEMaxReplay] cap counting MATCHING
// events (filter runs before truncation, so a capped reconnect delivers
// the newest N matching events — the same semantics as [WithSSEFilter]).
//
// A derivation error aborts the request with 400 Bad Request before any
// stream is opened. When both this option and [WithSSEFilter] are set, a
// non-nil derived predicate wins for that request; (nil, nil) falls back
// to the construction-time filter.
func WithRequestFilter(derive RequestFilterFunc) ServeDomainEventsOption {
	return func(c *serveDomainEventsConfig) { c.requestFilter = derive }
}

// WithSSEMaxReplay caps the number of events replayed/backfilled per
// connection at the HANDLER level, truncating to the MOST RECENT n events —
// the same tail semantics [NewJournalSSEStore] applies via [WithMaxReplay].
// It works with any [sse.EventStore], including test fakes and custom
// stores, so consumers do not need the journal-backed store just for the cap.
// When a filter is active ([WithSSEFilter]) the cap counts MATCHING events
// only, so a burst of non-matching events can never consume a scoped feed's
// backfill budget.
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
//
// When replayed is non-nil, the IDs of the events actually returned
// (post-truncation — exactly what the stream writes) are recorded in it, so
// the live pump can drop the same IDs at the replay/live seam.
type replayAdjustedStore struct {
	inner    sse.EventStore
	maxN     int
	retryHit uint
	replayed map[string]struct{}
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

	recordReplayedIDs(s.replayed, events)

	for i := range events {
		if events[i].Retry == 0 && s.retryHit > 0 {
			events[i].Retry = s.retryHit
		}
	}

	return events, nil
}

// filteredReplayStore adapts any [sse.EventStore] to
// [sse.FilteredEventStore] for the replay path. It is the fail-closed replay
// path for WithSSEFilter, with two handler-level adjustments applied AFTER
// filtering so they count MATCHING events, not raw ones:
//
//   - the replay cap tail-truncates to the most recent maxN matches (capping
//     before filtering would let a burst of non-matching events consume the
//     entire budget and starve a scoped feed's backfill to zero),
//   - the reconnect retry hint is stamped on replayed matches whose Retry
//     is zero.
//
// A panicking consumer predicate is recovered and treated as a non-match,
// mirroring go-sse's live-path guard (safePredCall) — the replay path must
// not be the one place a bad filter kills the HTTP handler goroutine.
type filteredReplayStore struct {
	inner    sse.EventStore
	match    func(sse.Event) bool
	maxN     int
	retryHit uint
	replayed map[string]struct{}
}

func (f *filteredReplayStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	return f.inner.EventsAfter(lastID) //nolint:wrapcheck // adapter propagates the store error verbatim
}

func (f *filteredReplayStore) EventsAfterFiltered(
	lastID sse.EventID,
	pred func(sse.Event) bool,
) ([]sse.Event, error) {
	events, err := f.inner.EventsAfter(lastID)
	if err != nil {
		return nil, err //nolint:wrapcheck,erraudit // HELPER: inner EventStore wraps attach last_id context (JournalSSEStore)
	}

	out := make([]sse.Event, 0, len(events))

	for _, evt := range events {
		if safePredCall(pred, evt) {
			out = append(out, evt)
		}
	}

	if f.maxN > 0 && len(out) > f.maxN {
		out = out[len(out)-f.maxN:]
	}

	recordReplayedIDs(f.replayed, out)

	for i := range out {
		if out[i].Retry == 0 && f.retryHit > 0 {
			out[i].Retry = f.retryHit
		}
	}

	return out, nil
}

// recordReplayedIDs collects the IDs of events the replay pass actually
// writes (nil map = not recording). Unidentified events (empty ID) are
// skipped — they cannot participate in seam dedup.
func recordReplayedIDs(replayed map[string]struct{}, events []sse.Event) {
	if replayed == nil {
		return
	}

	for _, evt := range events {
		if id := evt.ID.Get(); id != "" {
			replayed[id] = struct{}{}
		}
	}
}

// safePredCall mirrors go-sse's fanout predicate guard: a panicking consumer
// predicate is recovered and treated as a non-match instead of propagating
// out of the replay path into the HTTP handler goroutine.
func safePredCall(
	pred func(sse.Event) bool,
	evt sse.Event,
) (match bool) { //nolint:nonamedreturns // recover needs named return
	defer func() {
		if recover() != nil {
			match = false
		}
	}()

	return pred(evt)
}

// resolveFilter derives the filter for ONE request: a non-nil derived
// predicate wins; (nil, nil) falls back to the construction-time filter. A
// derivation error aborts the request before any stream is opened.
func (c serveDomainEventsConfig) resolveFilter(r *http.Request) (func(sse.Event) bool, error) {
	if c.requestFilter == nil {
		return c.filter, nil
	}

	pred, err := c.requestFilter(r)
	if err != nil {
		return nil, err
	}

	if pred == nil {
		return c.filter, nil
	}

	return pred, nil
}

// subscribe applies the configured filter to the live subscription (nil
// filter = unfiltered Subscribe).
func subscribe(b *sse.Broadcaster[sse.Event], filter func(sse.Event) bool) <-chan sse.Event {
	if filter != nil {
		return b.SubscribeFilter(filter)
	}

	return b.Subscribe()
}

// replayEvents writes the journal backfill to the stream, honoring the
// per-request filter. A nil store means live-only (no backfill). The IDs of
// the events written are recorded into replayed (when non-nil) so the live
// pump can drop the same IDs at the replay/live seam — subscribe runs BEFORE
// replay, so anything published during the replay window is delivered by
// BOTH paths.
//
// Replayed events whose Retry is zero carry the handler's reconnect hint
// ([DefaultRetryHintMillis]) — a client whose original retry frame was lost
// (proxy buffering, page reload mid-stream) re-learns the cadence from the
// backfill itself, and per the SSE spec the field simply persists for the
// connection.
func replayEvents(
	stream *sse.Stream,
	store sse.EventStore,
	filter func(sse.Event) bool,
	maxReplay int,
	logPrefix string,
	replayed map[string]struct{},
) {
	if store == nil {
		return
	}

	lastID := stream.LastEventID()

	if filter != nil {
		// Filter FIRST (raw store), cap only the matches: the replay budget
		// belongs to events the subscriber can actually receive.
		filtered := &filteredReplayStore{
			inner: store, match: filter, maxN: maxReplay, retryHit: DefaultRetryHintMillis,
			replayed: replayed,
		}

		if _, err := sse.ReplayFiltered(stream, filtered, lastID, filter); err != nil {
			slog.Warn(logPrefix+": SSE replay failed", "error", err, "lastEventID", lastID.Get())
		}

		return
	}

	store = &replayAdjustedStore{
		inner: store, maxN: maxReplay, retryHit: DefaultRetryHintMillis,
		replayed: replayed,
	}

	if _, err := sse.Replay(stream, store, lastID); err != nil {
		slog.Warn(logPrefix+": SSE replay failed", "error", err, "lastEventID", lastID.Get())
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
// heartbeat is non-positive, heartbeats are disabled. A [WithRequestFilter]
// derivation error responds 400 (Bad Request) before any stream is opened.
//
// Subscribe is called BEFORE replay so live events buffer in the channel while
// replay writes to the stream — no event is lost during the replay window,
// and the IDs the replay writes are deduplicated against the live pump, so
// an event published during that window is delivered exactly once.
// With [WithSSEFilter] or a per-request [WithRequestFilter] predicate, both
// the subscription and the replay deliver only matching events.
//
// # Cursor semantics
//
// Last-Event-ID handling is delegated to the store and is therefore
// store-defined. The bundled [NewJournalSSEStore] treats an unknown-but-
// valid cursor as "from the beginning" (upstream ReadFrom semantics — a
// full, capped backfill), while hand-rolled stores commonly return nothing.
// Clients must only send IDs the SAME store issued; a cursor from another
// deployment or after a store rotation produces a full backfill, not an
// error.
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
		requestFilter:      nil,
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

		// Resolve the filter for THIS request; a derivation error aborts
		// with 400 before any stream is opened.
		filter, err := cfg.resolveFilter(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

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
		ch := subscribe(broadcaster, filter)
		defer broadcaster.Unsubscribe(ch)

		_ = stream.Send(sse.Event{Event: sse.EventConnected, Data: "connected"})

		// replayed holds the IDs the backfill writes; the live pump skips them
		// (subscribe ran BEFORE replay, so the overlap window double-delivers).
		replayed := make(map[string]struct{})
		replayEvents(stream, store, filter, cfg.maxReplay, cfg.logPrefix, replayed)

		joinHeartbeat := startHeartbeat(stream, r.Context(), heartbeat)
		defer joinHeartbeat()

		pumpLiveEvents(stream, ch, replayed)
	}
}

// startHeartbeat launches the stream heartbeat (when enabled) and returns a
// join function that stops it and waits for the goroutine to exit. Joining
// before the handler returns matters twice over: a heartbeat write racing
// handler teardown is a data race, and net/http forbids touching the
// ResponseWriter after return.
func startHeartbeat(stream *sse.Stream, parent context.Context, heartbeat time.Duration) (join func()) {
	hbCtx, hbCancel := context.WithCancel(parent)
	hbDone := make(chan struct{})

	if heartbeat > 0 {
		go func() {
			defer close(hbDone)

			stream.Heartbeat(hbCtx, heartbeat)
		}()
	} else {
		close(hbDone)
	}

	return func() {
		hbCancel()

		<-hbDone
	}
}

// pumpLiveEvents forwards live events to the stream until it closes, skipping
// IDs the replay pass already wrote — subscribe ran BEFORE replay, so events
// published during the replay window sit in both delivery paths.
func pumpLiveEvents(stream *sse.Stream, ch <-chan sse.Event, replayed map[string]struct{}) {
	for {
		select {
		case <-stream.Context().Done():
			return

		case evt, ok := <-ch:
			if !ok {
				return
			}

			if isReplayDuplicate(evt, replayed) {
				continue // already written by the replay pass
			}

			if stream.Send(evt) != nil {
				return
			}
		}
	}
}

// isReplayDuplicate reports whether the event's ID was already written by
// the replay pass. Events without an ID never deduplicate.
func isReplayDuplicate(evt sse.Event, replayed map[string]struct{}) bool {
	id := evt.ID.Get()

	if id == "" {
		return false
	}

	_, dup := replayed[id]

	return dup
}
