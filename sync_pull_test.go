package cqrshtmx

import (
	"context"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/oklog/ulid/v2"
)

// seedSyncEvents creates count events on one stream, alternating JSON and CBOR
// encodings so both payload paths are exercised.
func seedSyncEvents(t *testing.T, count int) []event.Event {
	t.Helper()

	aggID, err := id.ParseStreamID(ulid.Make().String())
	if err != nil {
		t.Fatalf("parse aggregate ID: %v", err)
	}

	events := make([]event.Event, 0, count)
	for i := 1; i <= count; i++ {
		opts := []event.Option{}
		if i%2 == 1 {
			opts = append(opts, event.WithEncoding(codec.EncodingJSON))
		}

		evt, err := event.New(
			event.Type(fmt.Sprintf("sync.event.%d", i)),
			aggID,
			"test",
			event.Version(i),
			fmt.Sprintf(`{"seq":%d}`, i),
			opts...,
		)
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}

		events = append(events, evt)
	}

	return events
}

func appendSyncEvents(t *testing.T, store *memory.MemoryStore, events []event.Event) {
	t.Helper()

	if len(events) == 0 {
		return
	}

	ref := id.StreamRef{ID: events[0].StreamID(), Type: "test"}
	if err := store.AppendBatch(context.Background(), ref, events); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}
}

func pullSyncBody(t *testing.T, rec *httptest.ResponseRecorder) SyncPullResponse {
	t.Helper()

	var resp SyncPullResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal pull response: %v\nbody: %s", err, rec.Body.String())
	}

	return resp
}

func doPull(handler http.HandlerFunc, url string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}

	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec
}

func TestSyncPullHandler_BootstrapReturnsAllEventsWithPayloads(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedSyncEvents(t, 4)
	appendSyncEvents(t, store, events)

	handler := SyncPullHandler(store, WithSyncPullBackendID("backend-A"))
	rec := doPull(handler, "/sync/pull")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}

	resp := pullSyncBody(t, rec)
	if resp.BackendID != "backend-A" {
		t.Errorf("backendId = %q, want %q", resp.BackendID, "backend-A")
	}

	if len(resp.Events) != 4 {
		t.Fatalf("events = %d, want 4", len(resp.Events))
	}

	if resp.NextCursor != events[3].ID().String() {
		t.Errorf("nextCursor = %q, want last event ID", resp.NextCursor)
	}

	if resp.HasMore {
		t.Error("hasMore = true, want false")
	}

	// Odd events are JSON-encoded: payload embedded verbatim.
	if got := string(resp.Events[0].Payload); got != `{"seq":1}` {
		t.Errorf("event[0] payload = %q, want raw JSON", got)
	}

	if resp.Events[0].PayloadB64 != "" {
		t.Errorf("event[0] payloadB64 = %q, want empty for JSON encoding", resp.Events[0].PayloadB64)
	}

	// Even events are CBOR-stamped: payload arrives base64-encoded.
	if resp.Events[1].Payload != nil {
		t.Errorf("event[1] payload = %q, want nil for CBOR encoding", string(resp.Events[1].Payload))
	}

	want, err := base64.StdEncoding.DecodeString(resp.Events[1].PayloadB64)
	if err != nil || len(want) == 0 {
		t.Errorf("event[1] payloadB64 not valid base64 payload: %q (err: %v)", resp.Events[1].PayloadB64, err)
	}

	if resp.Events[0].PayloadEncoding != "json" || resp.Events[1].PayloadEncoding != "cbor" {
		t.Errorf("payloadEncoding = %q/%q, want json/cbor",
			resp.Events[0].PayloadEncoding, resp.Events[1].PayloadEncoding)
	}
}

func TestSyncPullHandler_CursorReturnsOnlyEventsAfter(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedSyncEvents(t, 5)
	appendSyncEvents(t, store, events)

	handler := SyncPullHandler(store)
	rec := doPull(handler, "/sync/pull?after="+events[2].ID().String())

	resp := pullSyncBody(t, rec)
	if len(resp.Events) != 2 {
		t.Fatalf("events = %d, want 2 (after event 3 of 5)", len(resp.Events))
	}

	if resp.Events[0].EventID != events[3].ID().String() {
		t.Errorf("first event = %s, want event 4", resp.Events[0].EventID)
	}
}

func TestSyncPullHandler_LimitPagesThroughJournal(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedSyncEvents(t, 7)
	appendSyncEvents(t, store, events)

	handler := SyncPullHandler(store)

	resp := pullSyncBody(t, doPull(handler, "/sync/pull?limit=3"))
	if len(resp.Events) != 3 || !resp.HasMore {
		t.Fatalf("page 1: events=%d hasMore=%v, want 3/true", len(resp.Events), resp.HasMore)
	}

	got := len(resp.Events)
	cursor := resp.NextCursor

	for resp.HasMore {
		resp = pullSyncBody(t, doPull(handler, "/sync/pull?limit=3&after="+cursor))
		got += len(resp.Events)
		cursor = resp.NextCursor
	}

	if got != 7 {
		t.Fatalf("paged through %d events total, want 7", got)
	}
}

func TestSyncPullHandler_FilterHidesEventsButAdvancesCursor(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedSyncEvents(t, 3)
	appendSyncEvents(t, store, events)

	handler := SyncPullHandler(store, WithSyncPullFilter(func(_ *http.Request, evt event.Event) bool {
		return evt.ID() == events[0].ID() // only the first event is visible
	}))

	resp := pullSyncBody(t, doPull(handler, "/sync/pull"))

	if len(resp.Events) != 1 {
		t.Fatalf("events = %d, want 1 (filtered)", len(resp.Events))
	}

	// The cursor must advance past ALL read events (visible or not), so the
	// next pull does not re-read the invisible ones.
	if resp.NextCursor != events[2].ID().String() {
		t.Errorf("nextCursor = %q, want last READ event (invisible included)", resp.NextCursor)
	}
}

func TestSyncPullHandler_ETagServesNotModified(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	appendSyncEvents(t, store, seedSyncEvents(t, 2))

	handler := SyncPullHandler(store)

	first := doPull(handler, "/sync/pull")
	tag := first.Header().Get("ETag")
	if tag == "" {
		t.Fatal("first pull: no ETag header")
	}

	if cc := first.Header().Get("Cache-Control"); cc != "private, no-cache" {
		t.Errorf("cache-control = %q, want private, no-cache", cc)
	}

	second := doPull(handler, "/sync/pull", "If-None-Match", tag)
	if second.Code != http.StatusNotModified {
		t.Fatalf("re-poll status = %d, want 304 (body: %s)", second.Code, second.Body.String())
	}
}

func TestSyncPullHandler_RejectsInvalidCursorAndLimit(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	handler := SyncPullHandler(store)

	if rec := doPull(handler, "/sync/pull?after=not-a-ulid"); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid cursor: status = %d, want 400", rec.Code)
	}

	if rec := doPull(handler, "/sync/pull?limit=zero"); rec.Code != http.StatusBadRequest {
		t.Errorf("non-numeric limit: status = %d, want 400", rec.Code)
	}

	if rec := doPull(handler, "/sync/pull?limit=-5"); rec.Code != http.StatusBadRequest {
		t.Errorf("negative limit: status = %d, want 400", rec.Code)
	}
}

func TestSyncPullHandler_RejectsNonGet(t *testing.T) {
	t.Parallel()

	handler := SyncPullHandler(memory.NewMemoryStore())

	req := httptest.NewRequest(http.MethodPost, "/sync/pull", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want 405", rec.Code)
	}
}

// syncPullJournalOnly is an event.Journal WITHOUT SeekableJournal, forcing the
// ReadAll fallback path.
type syncPullJournalOnly struct{ inner *memory.MemoryStore }

func (j *syncPullJournalOnly) ReadAll(ctx context.Context) ([]event.Event, error) {
	return j.inner.ReadAll(ctx)
}

func TestSyncPullHandler_ReadAllFallback(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedSyncEvents(t, 4)
	appendSyncEvents(t, store, events)

	handler := SyncPullHandler(&syncPullJournalOnly{inner: store})

	resp := pullSyncBody(t, doPull(handler, "/sync/pull?after="+events[0].ID().String()))
	if len(resp.Events) != 3 {
		t.Fatalf("fallback events = %d, want 3", len(resp.Events))
	}
}

func TestSyncPullHandler_EmptyJournal(t *testing.T) {
	t.Parallel()

	handler := SyncPullHandler(memory.NewMemoryStore())

	resp := pullSyncBody(t, doPull(handler, "/sync/pull"))
	if len(resp.Events) != 0 || resp.NextCursor != "" || resp.HasMore {
		t.Fatalf("empty journal: events=%d cursor=%q hasMore=%v, want 0/\"\"/false",
			len(resp.Events), resp.NextCursor, resp.HasMore)
	}
}

func TestSyncPullHandler_PanicsOnNilJournal(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil journal")
		}
	}()

	SyncPullHandler(nil)
}
