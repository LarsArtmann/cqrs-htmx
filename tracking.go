package cqrshtmx

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// The tracking endpoint receives the four signals only the browser can see
// (which logical page was viewed, how long it stayed visible, which tracked
// element was clicked, which one became visible) and hands them to a sink
// together with the identity the SERVER observed. The request body never
// carries identity or a timestamp: the client is untrusted, so who and when
// come from the request context and the server clock, and every string the
// client sends is capped or dropped before it reaches a sink.
//
// Two rules shape the handler. A misbehaving page must never learn anything
// or break the app: every well-formed envelope answers 202, an invalid signal
// is dropped and counted, and a sink failure is logged and never returned.
// And the cheap checks (method, cross-site evidence, media type) run before a
// byte of the body is read, so a forged request costs nothing but a header
// parse.

// Default limits of [TrackHandler]. The body bound is the fetchLater quota
// per origin, so one honest beacon can never exceed it; the batch bound is
// what one 64 KiB body of capped signals can hold.
const (
	MaxTrackBodyBytes int64 = 64 << 10
	MaxTrackBatch           = 50
)

// Per-field caps. A path and a referrer are URLs; a name is what the page
// wrote into data-track; a target names an element. Dwell is clamped to one
// day because a longer page view is a clock or tab that was never closed.
const (
	maxTrackedPathBytes     = 2048
	maxTrackedReferrerBytes = 2048
	maxTrackedNameBytes     = 200
	maxTrackedTargetBytes   = 512
	maxTrackedVisibleMS     = 24 * 60 * 60 * 1000
)

// trackWriteTimeout bounds one sink write. The context handed to the sink
// outlives the request (the browser may be gone), so it needs its own bound.
const trackWriteTimeout = 5 * time.Second

// trackMediaType is the one media type the endpoint decodes, compared after
// mime.ParseMediaType so a charset parameter is accepted.
const trackMediaType = "application/json"

// TrackedEventKind names what the browser saw.
type TrackedEventKind string

// The four client signals.
const (
	// TrackedPageview is one logical page view: on load and on every in-app navigation.
	TrackedPageview TrackedEventKind = "pageview"
	// TrackedEngagement is the visible time of one finished page view (VisibleMS).
	TrackedEngagement TrackedEventKind = "engagement"
	// TrackedClick is a click on an element the page marked with data-track.
	TrackedClick TrackedEventKind = "click"
	// TrackedImpression is an element marked with data-track-impression becoming at least half visible.
	TrackedImpression TrackedEventKind = "impression"
)

// TrackedEvent is one validated client signal with the identity the server
// saw. It is produced once and appended, never updated. UserID is zero for an
// anonymous visitor, who is then known only by the device key ClientID when
// the browser sent one. At is the server's receive time: a client clock is
// not trusted and not stored. No IP address or user agent is kept.
type TrackedEvent struct {
	Kind      TrackedEventKind `json:"kind"`
	Path      string           `json:"path"`
	Referrer  string           `json:"referrer,omitempty"`
	Name      string           `json:"name,omitempty"`
	Target    string           `json:"target,omitempty"`
	VisibleMS int64            `json:"visibleMs,omitzero"`
	At        time.Time        `json:"at"`
	UserID    id.UserID        `json:"userId,omitzero"`
	ActorID   id.ActorID       `json:"actorId,omitzero"`
	ClientID  id.ClientID      `json:"clientId,omitzero"`
	RequestID id.RequestID     `json:"requestId,omitzero"`
}

// TrackReceipt is the tracking endpoint's response: how many signals of the
// envelope were handed to the sink and how many were dropped as invalid. A
// dropped signal is never reported per signal: the endpoint does not echo
// client input back.
type TrackReceipt struct {
	Accepted int `json:"accepted"`
	Dropped  int `json:"dropped"`
}

// TrackingSink stores validated client signals. Record receives a whole batch
// and a context that outlives the request but carries a deadline. An error
// means the batch was not stored; the handler logs it and still answers 202.
type TrackingSink interface {
	Record(ctx context.Context, events []TrackedEvent) error
}

// trackRequest is the wire envelope. It deliberately has no identity and no
// time field: whatever a client sends under those names is not decoded.
type trackRequest struct {
	Events []trackSignal `json:"events"`
}

// trackSignal is one client-supplied signal before validation.
type trackSignal struct {
	Kind      string  `json:"kind"`
	Path      string  `json:"path"`
	Referrer  string  `json:"referrer"`
	Name      string  `json:"name"`
	Target    string  `json:"target"`
	VisibleMS float64 `json:"visibleMs"`
}

// TrackOption tunes a [TrackHandler] at construction time.
type TrackOption func(*trackConfig)

type trackConfig struct {
	sink       TrackingSink
	bodyLimit  int64
	batchLimit int
	onError    func(http.ResponseWriter, *http.Request, error)
}

func defaultTrackConfig(sink TrackingSink) trackConfig {
	return trackConfig{
		sink:       sink,
		bodyLimit:  MaxTrackBodyBytes,
		batchLimit: MaxTrackBatch,
		onError:    httperr.JSONErrorHandler,
	}
}

// WithTrackBodyLimit bounds the request body the handler reads; a larger body
// is answered with 413. A value of zero or less restores [MaxTrackBodyBytes].
func WithTrackBodyLimit(bytes int64) TrackOption {
	return func(cfg *trackConfig) {
		if bytes > 0 {
			cfg.bodyLimit = bytes
		}
	}
}

// WithTrackBatchLimit bounds the signals one request may carry; a larger batch
// is refused whole with 413, before any signal is recorded. A value of zero or
// less restores [MaxTrackBatch].
func WithTrackBatchLimit(signals int) TrackOption {
	return func(cfg *trackConfig) {
		if signals > 0 {
			cfg.batchLimit = signals
		}
	}
}

// WithTrackErrorHandler renders the endpoint's error responses. The default is
// [httperr.JSONErrorHandler]; pass [httperr.ProblemDetailsErrorHandler] for an
// RFC 9457 body.
func WithTrackErrorHandler(handler func(http.ResponseWriter, *http.Request, error)) TrackOption {
	return func(cfg *trackConfig) {
		if handler != nil {
			cfg.onError = handler
		}
	}
}

// TrackHandler returns the endpoint that receives the signals of tracker.js:
//
//	POST <path>  Content-Type: application/json
//	{"events":[{"kind":"pageview","path":"/items","referrer":"https://g.example/search"}]}
//	  -> 202 {"accepted":1,"dropped":0}
//
// Signals are validated, capped, attributed to the request (session user,
// actor, device key, request ID) and handed to the sink with the server's
// time. A nil sink accepts and discards, which keeps the endpoint mountable
// while nothing durable is configured.
//
// Mount it where the session middleware enriches the context and, for
// anonymous visitors, where [ContextEnrichmentMiddleware] reads the
// X-Client-Id header; the device key is also read from that header directly,
// so a bare mount still attributes devices.
func TrackHandler(sink TrackingSink, opts ...TrackOption) http.Handler {
	cfg := defaultTrackConfig(sink)

	for _, opt := range opts {
		opt(&cfg)
	}

	return http.HandlerFunc(cfg.serve)
}

func (cfg trackConfig) serve(w http.ResponseWriter, r *http.Request) {
	if err := cfg.reject(w, r); err != nil {
		cfg.onError(w, r, err)

		return
	}

	request, err := cfg.decode(w, r)
	if err != nil {
		cfg.onError(w, r, err)

		return
	}

	events, dropped := trackedEvents(request.Events, r)
	cfg.record(r, events)

	_ = WriteJSON(w, http.StatusAccepted, TrackReceipt{Accepted: len(events), Dropped: dropped})
}

// reject applies the checks that need no body. A refused request has cost one
// header parse.
func (cfg trackConfig) reject(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)

		return errorfamily.Wrapf(httperr.ErrMethodNotAllowed, errorfamily.Rejection,
			"cqrshtmx.track.method_not_allowed",
			"the tracking endpoint accepts POST only (got %s)", r.Method)
	}

	if err := rejectCrossSite(r); err != nil {
		return err
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != trackMediaType {
		return errorfamily.Wrapf(httperr.ErrUnsupportedMediaType, errorfamily.Rejection,
			"cqrshtmx.track.media_type",
			"the tracking endpoint reads %s bodies only", trackMediaType)
	}

	return nil
}

// rejectCrossSite refuses a request that provably comes from another site.
// Browsers stamp Sec-Fetch-Site on every request; only a first-party page
// produces same-origin (or none, a user-initiated navigation). "same-site" is
// refused too: cookies are site-scoped, so a sibling subdomain would ride the
// visitor's session. Where the header is absent (plain HTTP off localhost),
// Origin is compared by host, because a page cannot forge either header and
// the scheme behind a TLS-terminating proxy is not knowable here. A request
// carrying neither is judged by its media type and body.
func rejectCrossSite(r *http.Request) error {
	site := r.Header.Get("Sec-Fetch-Site")

	if site != "" && site != "same-origin" && site != "none" {
		return errorfamily.Wrapf(httperr.ErrForbidden, errorfamily.Rejection,
			"cqrshtmx.track.cross_site",
			"tracking requests from another site are refused (Sec-Fetch-Site: %s)", site)
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return nil
	}

	parsed, err := url.Parse(origin)
	if err != nil || !strings.EqualFold(parsed.Host, r.Host) {
		return errorfamily.Wrapf(httperr.ErrForbidden, errorfamily.Rejection,
			"cqrshtmx.track.cross_site",
			"tracking requests from another site are refused (Origin: %s)", origin)
	}

	return nil
}

// decode reads the body under the byte limit and decodes the envelope. An
// oversized body, an undecodable body, and an oversized batch are refused
// whole; nothing of them reaches the sink.
func (cfg trackConfig) decode(w http.ResponseWriter, r *http.Request) (trackRequest, error) {
	var request trackRequest

	body, err := io.ReadAll(
		http.MaxBytesReader(w, r.Body, cfg.bodyLimit),
	)
	if err != nil {
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			return request, errorfamily.Wrapf(httperr.ErrRequestTooLarge, errorfamily.Rejection,
				"cqrshtmx.track.body_too_large",
				"a tracking request body is at most %d bytes", cfg.bodyLimit)
		}

		return request, errorfamily.Wrapf(err, errorfamily.Rejection,
			"cqrshtmx.track.body_read_failed", "read the tracking request body")
	}

	if err := json.Unmarshal(body, &request); err != nil {
		return request, errorfamily.Wrapf(httperr.ErrDecodeFailed, errorfamily.Rejection,
			"cqrshtmx.track.decode",
			"the tracking request body must be a JSON object with an events array")
	}

	if len(request.Events) > cfg.batchLimit {
		return request, errorfamily.Wrapf(httperr.ErrRequestTooLarge, errorfamily.Rejection,
			"cqrshtmx.track.batch_too_large",
			"a tracking request carries at most %d signals (got %d)", cfg.batchLimit, len(request.Events))
	}

	return request, nil
}

// record hands the batch to the sink. The write outlives the request's
// cancellation, because the browser may already be gone: a keepalive fetch and
// a fetchLater request both end with the page. Its own deadline bounds it.
func (cfg trackConfig) record(r *http.Request, events []TrackedEvent) {
	if cfg.sink == nil || len(events) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), trackWriteTimeout)
	defer cancel()

	if err := cfg.sink.Record(ctx, events); err != nil {
		slog.WarnContext(ctx, "cqrs-htmx: tracking sink failed",
			"error", err, "signals", len(events))
	}
}

// trackAttribution is the identity the server observed for one request. It is
// computed once per batch: every signal of a request shares it.
type trackAttribution struct {
	userID    id.UserID
	actorID   id.ActorID
	clientID  id.ClientID
	requestID id.RequestID
}

func trackAttributionFrom(r *http.Request) trackAttribution {
	ctx := r.Context()
	clientID := ClientIDFromContext(ctx)

	if clientID.IsZero() {
		// A bare mount without ContextEnrichmentMiddleware still attributes devices.
		if parsed, err := id.ParseClientID(r.Header.Get(HeaderClientID)); err == nil {
			clientID = parsed
		}
	}

	return trackAttribution{
		userID:    UserIDFromContext(ctx),
		actorID:   ActorIDFromContext(ctx),
		clientID:  clientID,
		requestID: RequestIDFromContext(ctx),
	}
}

// trackedEvents validates every signal of a batch against one identity and
// one server time, and counts what it dropped.
func trackedEvents(signals []trackSignal, r *http.Request) ([]TrackedEvent, int) {
	who := trackAttributionFrom(r)
	now := time.Now().UTC()

	events := make([]TrackedEvent, 0, len(signals))
	for _, signal := range signals {
		if event, ok := newTrackedEvent(signal, who, now); ok {
			events = append(events, event)
		}
	}

	return events, len(signals) - len(events)
}

// newTrackedEvent turns one client signal into a record, or reports that the
// signal is not one. A signal is dropped (not repaired) when its kind is
// unknown or its path is not a same-origin path; everything else is capped.
func newTrackedEvent(signal trackSignal, who trackAttribution, now time.Time) (TrackedEvent, bool) {
	kind := TrackedEventKind(signal.Kind)

	switch kind {
	case TrackedPageview, TrackedEngagement, TrackedClick, TrackedImpression:
	default:
		return noSignal()
	}

	path := trackPath(signal.Path)
	if path == "" {
		return noSignal()
	}

	return TrackedEvent{
		Kind:      kind,
		Path:      path,
		Referrer:  reduceReferrer(signal.Referrer),
		Name:      truncateUTF8(signal.Name, maxTrackedNameBytes),
		Target:    truncateUTF8(signal.Target, maxTrackedTargetBytes),
		VisibleMS: clampVisibleMS(signal.VisibleMS),
		At:        now,
		UserID:    who.userID,
		ActorID:   who.actorID,
		ClientID:  who.clientID,
		RequestID: who.requestID,
	}, true
}

// noSignal is what newTrackedEvent answers for a signal it drops.
func noSignal() (TrackedEvent, bool) {
	var zero TrackedEvent

	return zero, false
}

// trackPath keeps the pathname of a same-origin path. The query and the
// fragment are cut: they routinely hold tokens and typed search terms, and
// the logical page does not need them. "" means the value was not a path.
func trackPath(path string) string {
	path, _, _ = strings.Cut(path, "?")
	path, _, _ = strings.Cut(path, "#")

	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return ""
	}

	return truncateUTF8(path, maxTrackedPathBytes)
}

// reduceReferrer keeps the origin and the path of a referrer and drops its
// userinfo, query and fragment. A referrer that is not an absolute URL is
// dropped whole: a relative one describes the app's own page, which the path
// already names.
func reduceReferrer(referrer string) string {
	if referrer == "" {
		return ""
	}

	parsed, err := url.Parse(referrer)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.RawFragment = ""

	return truncateReferrer(parsed.String())
}

// percentEscapeBytes is the width of one "%XX" in a serialized URL.
const percentEscapeBytes = 3

// truncateReferrer caps a serialized referrer without leaving half of a
// percent escape at its end: a dangling "%C3" would not parse as a URL again.
func truncateReferrer(referrer string) string {
	cut := truncateUTF8(referrer, maxTrackedReferrerBytes)
	if len(cut) == len(referrer) {
		return cut
	}

	tail := cut[max(len(cut)-(percentEscapeBytes-1), 0):]

	if percent := strings.IndexByte(tail, '%'); percent >= 0 {
		return cut[:len(cut)-(len(tail)-percent)]
	}

	return cut
}

// clampVisibleMS floors a dwell to whole milliseconds inside [0, one day].
// The bounds are applied in float space first: converting an out-of-range
// float to int64 is not defined in Go.
func clampVisibleMS(visible float64) int64 {
	switch {
	case visible <= 0:
		return 0
	case visible >= maxTrackedVisibleMS:
		return maxTrackedVisibleMS
	default:
		return int64(visible)
	}
}

// truncateUTF8 cuts s to at most maxBytes without splitting a rune.
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	head := s[:maxBytes]
	for len(head) > 0 && !utf8.ValidString(head) {
		head = head[:len(head)-1]
	}

	return head
}
