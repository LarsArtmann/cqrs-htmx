package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
	"github.com/larsartmann/go-sse/ssetest"
)

// TestSSE_RealServer_ReconnectionWithLastEventID verifies the SSE
// reconnection path end-to-end: a client reconnects with a
// Last-Event-ID header, the server reads it, and replays missed events
// from the SSEEventStore.
func TestSSE_RealServer_ReconnectionWithLastEventID(t *testing.T) {
	t.Parallel()

	store := newReconnectStore()
	mux := newReconnectMux(store, true)

	server := httptest.NewServer(mux)
	defer server.Close()

	resp := doReconnectRequest(t, server.URL+"/events", "2")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Errorf("expected text/event-stream content type, got %q", got)
	}

	out := readReconnectBody(t, resp)

	assertReplayedAfterID2(t, out)
}

const reconnectEventKind = "itemCreated"

func newReconnectStore() *memoryEventStore {
	return &memoryEventStore{
		events: []sse.Event{
			{Event: reconnectEventKind, Data: "<li>first</li>", ID: sse.NewEventID("1")},
			{Event: reconnectEventKind, Data: "<li>second</li>", ID: sse.NewEventID("2")},
			{Event: reconnectEventKind, Data: "<li>third</li>", ID: sse.NewEventID("3")},
			{Event: reconnectEventKind, Data: "<li>fourth</li>", ID: sse.NewEventID("4")},
		},
	}
}

func newReconnectMux(store *memoryEventStore, includeReplayOnError bool) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		if lastID := sse.LastEventIDFromRequest(r); !lastID.IsZero() {
			if _, err := sse.Replay(stream, store, lastID); err != nil {
				if includeReplayOnError {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}

				return
			}
		}
	})

	return mux
}

func doReconnectRequest(t *testing.T, url, lastID string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if lastID != "" {
		req.Header.Set("Last-Event-ID", lastID)
	}

	req.Header.Set("Accept", "text/event-stream")

	// Per-test client with a short timeout. We intentionally do NOT bind
	// the request to a context whose cancel is deferred here: doing so
	// cancels the context before the caller reads resp.Body, which causes
	// intermittent "context canceled" failures under -race with t.Parallel().
	client := &http.Client{Timeout: 3 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}

	return resp
}

func readReconnectBody(t *testing.T, resp *http.Response) []ssetest.Event {
	t.Helper()

	events, err := ssetest.ReadEvents(resp.Body)
	if err != nil {
		t.Fatalf("read SSE events: %v", err)
	}

	return events
}

func assertReplayedAfterID2(t *testing.T, events []ssetest.Event) {
	t.Helper()

	ids := make(map[string]bool)

	for _, evt := range events {
		ids[evt.ID] = true
	}

	if !ids["3"] {
		t.Errorf("expected replayed id 3, got IDs: %v", ids)
	}

	if !ids["4"] {
		t.Errorf("expected replayed id 4, got IDs: %v", ids)
	}

	if ids["1"] || ids["2"] {
		t.Errorf("did not expect ids 1 or 2 to be replayed, got IDs: %v", ids)
	}

	dataLines := make([]string, 0, len(events))
	for _, evt := range events {
		dataLines = append(dataLines, evt.Data())
	}

	if !strings.Contains(strings.Join(dataLines, "\n"), "<li>third</li>") {
		t.Errorf("expected third event data, got: %v", dataLines)
	}

	if !strings.Contains(strings.Join(dataLines, "\n"), "<li>fourth</li>") {
		t.Errorf("expected fourth event data, got: %v", dataLines)
	}
}

// TestSSE_RealServer_ReconnectionNoLastID verifies that a fresh client
// (no Last-Event-ID) does not trigger replay; only the initial stream
// handshake occurs.
func TestSSE_RealServer_ReconnectionNoLastID(t *testing.T) {
	t.Parallel()

	store := &memoryEventStore{
		events: []sse.Event{
			{Event: "x", Data: "y", ID: sse.NewEventID("1")},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		if lastID := sse.LastEventIDFromRequest(r); !lastID.IsZero() {
			_, _ = sse.Replay(stream, store, lastID)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resp := doReconnectRequest(t, server.URL+"/events", "")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
