package cqrshtmx

import (
	"context"
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"hash/fnv" //nolint:gosec // FNV-1a is used as a change-detector ETag, not for security
	"net/http"
	"strconv"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	etag "github.com/larsartmann/go-etag/server"
)

// DefaultSyncPullLimit is the batch size used when the client does not request
// one (and the ceiling enforced when it requests more).
const (
	DefaultSyncPullLimit = 500
	MaxSyncPullLimit     = 1000
)

// SyncPayloadEncodingOpaque is the PayloadEncoding reported when an event's
// stamp promises JSON but its bytes are not valid JSON (the metadata-only
// event.WithEncoding trap: event.New still CBOR-encodes map payloads). The
// wire never promises an encoding the delivered bytes cannot keep.
const SyncPayloadEncodingOpaque = "opaque"

// SyncEvent is the JSON shape of one pulled domain event: the SSE metadata
// envelope plus the payload. Unlike the SSE envelope (metadata-only by
// default), the pull endpoint is the explicit opt-in surface for payload
// exposure — gate it with [WithSyncPullFilter].
type SyncEvent struct {
	EventID       string `json:"eventId"`
	Type          string `json:"type"`
	StreamType    string `json:"streamType"`
	StreamID      string `json:"streamId"`
	Version       uint64 `json:"version"`
	SchemaVersion string `json:"schemaVersion,omitempty"`
	OccurredAt    string `json:"occurredAt"`

	// PayloadEncoding names the delivered bytes' framing. Exactly one of
	// Payload/PayloadB64 is set: "json" → Payload holds the verbatim JSON;
	// a codec name ("cbor") → PayloadB64 holds that codec's bytes;
	// "opaque" ([SyncPayloadEncodingOpaque]) → the event's stamp promised
	// JSON but the bytes are not valid JSON, so PayloadB64 carries them
	// without a decodable-framing promise.
	PayloadEncoding string         `json:"payloadEncoding"`
	Payload         jsontext.Value `json:"payload,omitempty"`
	PayloadB64      string         `json:"payloadB64,omitempty"`
}

// SyncPullResponse is the body returned by [SyncPullHandler].
type SyncPullResponse struct {
	// BackendID is the stable identity of the backing journal (LiveStore's
	// backendId). Clients persist it and reset their local cache when it
	// changes — the backend-rebuilt detection mechanism. Empty when no
	// [WithSyncPullBackendID] was configured.
	BackendID string `json:"backendId,omitempty"`

	// Events are the visible events after the cursor, in journal order.
	Events []SyncEvent `json:"events"`

	// NextCursor is the event ID to pass as ?after= on the next pull. It is
	// the last READ event (visible or filtered-out) so invisible events are
	// never re-read. Empty when no events were read — keep the previous cursor.
	NextCursor string `json:"nextCursor,omitempty"`

	// HasMore is true when the journal holds more events beyond this batch.
	HasMore bool `json:"hasMore"`
}

// SyncPullFilter decides, per request, whether an event is visible to the
// caller. Return false to make the event INVISIBLE — not merely omitted from
// this batch: the pull cursor still advances past it. This is the permission
// seam (wire Casbin against the request context); see [WithSyncPullFilter].
type SyncPullFilter func(r *http.Request, evt event.Event) bool

// SyncPullOption configures a [SyncPullHandler].
type SyncPullOption func(*syncPullConfig)

type syncPullConfig struct {
	backendID string
	filter    SyncPullFilter
	limit     int
}

// WithSyncPullBackendID sets the stable backend identity echoed in every pull
// response. Clients reset their local cache when it changes (backend rebuilt,
// journal restored from a different source). Empty (default) omits the field.
func WithSyncPullBackendID(id string) SyncPullOption {
	return func(c *syncPullConfig) { c.backendID = id }
}

// WithSyncPullFilter restricts the pull to events the caller may see. The
// predicate runs per request with the live *http.Request, so session-derived
// authorization (Casbin, tenant scoping) composes naturally. Filtering is
// fail-closed: returning false hides the event entirely. nil (default) exposes
// EVERY event with payloads — only acceptable for non-sensitive feeds; the
// [transport.EventPayload] SSE envelope stays the metadata-only default.
func WithSyncPullFilter(pred SyncPullFilter) SyncPullOption {
	return func(c *syncPullConfig) { c.filter = pred }
}

// WithSyncPullLimit sets the default batch size (applied when the request does
// not carry ?limit=, and the ceiling when it does). Values are clamped to
// [MaxSyncPullLimit]. Default: [DefaultSyncPullLimit].
func WithSyncPullLimit(n int) SyncPullOption {
	return func(c *syncPullConfig) { c.limit = n }
}

// SyncPullHandler returns an http.HandlerFunc implementing the read half of
// the frontend sync protocol (ADR-0056): a cursor-based, permission-filtered,
// cache-aware batch pull of domain events WITH payloads.
//
// GET ?after=<eventID>&limit=<n>
//
// The cursor is the last event ID the client has seen (ULIDs order globally,
// so the cursor need not still exist — compacted journals stay navigable).
// An empty cursor bootstraps from the beginning of the journal; clients page
// with nextCursor/hasMore. Responses carry an FNV-1a ETag over
// (backendId, nextCursor, hasMore) so conditional re-polls cost one hash.
//
// The handler is safe for concurrent use and works with any event.Journal:
// event.SeekableJournal journals get efficient position-based reads, others
// fall back to ReadAll with in-memory seeking (same shape as
// transport.JournalSSEStore).
//
// Panics if journal is nil (programming error).
//
//	mux.Handle("GET /sync/pull", cqrshtmx.SyncPullHandler(journal,
//	    cqrshtmx.WithSyncPullBackendID("prod-2026-10"),
//	    cqrshtmx.WithSyncPullFilter(casbinFilter),
//	))
func SyncPullHandler(journal event.Journal, opts ...SyncPullOption) http.HandlerFunc {
	if journal == nil {
		panic("cqrs-htmx: SyncPullHandler: journal must not be nil")
	}

	config := syncPullConfig{limit: DefaultSyncPullLimit}
	for _, opt := range opts {
		opt(&config)
	}

	seekable, _ := journal.(event.SeekableJournal)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeSyncPullError(w, http.StatusMethodNotAllowed, "cqrshtmx.sync.pull.method_not_allowed",
				"sync pull is a GET endpoint")

			return
		}

		limit := config.limit
		if raw := r.URL.Query().Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				writeSyncPullError(w, http.StatusBadRequest, "cqrshtmx.sync.pull.limit_invalid",
					"limit must be a positive integer")

				return
			}

			limit = parsed
		}

		if limit > MaxSyncPullLimit {
			limit = MaxSyncPullLimit
		}

		var afterID id.EventID

		if raw := r.URL.Query().Get("after"); raw != "" {
			parsed, err := id.ParseEventID(raw)
			if err != nil {
				writeSyncPullError(w, http.StatusBadRequest, "cqrshtmx.sync.pull.cursor_invalid",
					"after must be a valid event ID (ULID)")

				return
			}

			afterID = parsed
		}

		events, hasMore, err := readSyncPullWindow(r.Context(), journal, seekable, afterID, limit)
		if err != nil {
			writeSyncPullError(w, http.StatusInternalServerError, "cqrshtmx.sync.pull.journal_read",
				"journal read failed")

			return
		}

		visible := make([]SyncEvent, 0, len(events))
		nextCursor := ""

		for _, evt := range events {
			nextCursor = evt.ID().String()

			if config.filter != nil && !config.filter(r, evt) {
				continue
			}

			visible = append(visible, newSyncEvent(evt))
		}

		response := SyncPullResponse{
			BackendID:  config.backendID,
			Events:     visible,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		}

		writeSyncPullResponse(w, r, &response)
	}
}

// readSyncPullWindow reads at most limit events strictly after afterID and
// reports whether more remain (it reads limit+1 and peeks).
func readSyncPullWindow(
	ctx context.Context,
	journal event.Journal,
	seekable event.SeekableJournal,
	afterID id.EventID,
	limit int,
) ([]event.Event, bool, error) {
	var (
		events  []event.Event
		readErr error
	)

	if seekable != nil {
		events, readErr = seekable.ReadFrom(ctx, afterID, limit+1)
	} else {
		events, readErr = readAllAfter(ctx, journal, afterID, limit+1)
	}

	if readErr != nil {
		return nil, false, readErr
	}

	if len(events) > limit {
		return events[:limit], true, nil
	}

	return events, false, nil
}

// readAllAfter implements cursor seeking over a non-seekable journal via
// ReadAll plus an in-memory strictly-after scan (the JournalSSEStore fallback
// shape). Event IDs are ULIDs, so lexical comparison preserves journal order.
func readAllAfter(
	ctx context.Context,
	journal event.Journal,
	afterID id.EventID,
	limit int,
) ([]event.Event, error) {
	all, err := journal.ReadAll(ctx)
	if err != nil {
		return nil, err
	}

	cursor := afterID.String()

	out := make([]event.Event, 0, len(all))
	for _, evt := range all {
		if afterID != (id.EventID{}) && evt.ID().String() <= cursor {
			continue
		}

		out = append(out, evt)
		if len(out) == limit {
			break
		}
	}

	return out, nil
}

// newSyncEvent converts a domain event into the pull wire shape, embedding the
// payload verbatim when the event encoding is JSON and base64-encoding it
// otherwise (clients cannot decode CBOR in the JSON field without help).
// JSON-stamped payloads are validity-checked defensively: an event whose
// stamp and bytes disagree (the metadata-only WithEncoding trap — event.New
// still CBOR-encodes map payloads) degrades to the base64 path with
// payloadEncoding [SyncPayloadEncodingOpaque] instead of promising JSON the
// bytes cannot keep. Non-JSON stamps are reported as-is: they name the
// bytes' real codec.
func newSyncEvent(evt event.Event) SyncEvent {
	out := SyncEvent{
		EventID:         evt.ID().String(),
		Type:            string(evt.Type()),
		StreamType:      string(evt.StreamType()),
		StreamID:        evt.StreamID().Get(),
		Version:         evt.Version().UInt64(),
		SchemaVersion:   evt.SchemaVersion().String(),
		OccurredAt:      evt.OccurredAt().Format(time.RFC3339),
		PayloadEncoding: string(evt.Encoding()),
	}

	payload := evt.Payload()
	if v := jsontext.Value(payload); string(evt.Encoding()) == "json" && v.IsValid() {
		out.Payload = v

		return out
	}

	out.PayloadB64 = base64.StdEncoding.EncodeToString(payload)
	if string(evt.Encoding()) == "json" {
		out.PayloadEncoding = SyncPayloadEncodingOpaque
	}

	return out
}

// syncPullETag hashes the response's position fields into a weak-change
// detector ETag: two pulls at the same cursor state share a tag, so a
// conditional re-poll costs one FNV-1a hash instead of a body.
func syncPullETag(resp *SyncPullResponse) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(resp.BackendID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(resp.NextCursor))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strconv.Itoa(len(resp.Events))))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strconv.FormatBool(resp.HasMore)))

	return `"` + strconv.FormatUint(h.Sum64(), 16) + `"`
}

// writeSyncPullResponse writes a successful pull with the protocol's caching
// headers: private + no-cache (always revalidate — the ETag makes that cheap).
func writeSyncPullResponse(w http.ResponseWriter, r *http.Request, resp *SyncPullResponse) {
	tag := syncPullETag(resp)

	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("ETag", tag)

	if current, ok := etag.ParseETag(tag); ok && etag.MatchesIfNoneMatch(current, r.Header.Get("If-None-Match")) {
		w.WriteHeader(http.StatusNotModified)

		return
	}

	if err := WriteJSON(w, http.StatusOK, resp); err != nil {
		// WriteJSON only fails before committing (buffered marshal), so a 500
		// with a fresh body is still possible — never a silent empty 200.
		writeSyncPullError(w, http.StatusInternalServerError, "cqrshtmx.sync.pull.encode",
			"pull response encoding failed")
	}
}

// writeSyncPullError emits the standalone JSON error shape used by the sync
// protocol endpoints ({"error", "status", "code"} — the JSONErrorHandler body).
func writeSyncPullError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)

	_ = json.MarshalWrite(w, map[string]any{
		JSONKeyError:  message,
		JSONKeyStatus: status,
		JSONKeyCode:   code,
	}) //nolint:errcheck // best-effort error body on an already-committed status
}
