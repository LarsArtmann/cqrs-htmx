package cqrshtmx_test

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
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
		events: []cqrshtmx.SSEEvent{
			{Event: reconnectEventKind, Data: "<li>first</li>", ID: cqrshtmx.NewSSEEventID("1")},
			{Event: reconnectEventKind, Data: "<li>second</li>", ID: cqrshtmx.NewSSEEventID("2")},
			{Event: reconnectEventKind, Data: "<li>third</li>", ID: cqrshtmx.NewSSEEventID("3")},
			{Event: reconnectEventKind, Data: "<li>fourth</li>", ID: cqrshtmx.NewSSEEventID("4")},
		},
	}
}

func newReconnectMux(store *memoryEventStore, includeReplayOnError bool) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		stream := cqrshtmx.NewSSEStream(w, r)
		defer stream.Close()
		if lastID := cqrshtmx.LastEventIDFromRequest(r); !lastID.IsZero() {
			if _, err := cqrshtmx.ReplayEvents(stream, store, lastID); err != nil {
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
	client := &http.Client{Timeout: 3 * time.Second} //nolint:exhaustruct // Transport/CheckRedirect/Jar use defaults
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func readReconnectBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	scanner := bufio.NewScanner(resp.Body)
	var body strings.Builder
	for scanner.Scan() {
		body.WriteString(scanner.Text())
		body.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan body: %v", err)
	}
	return body.String()
}

func assertReplayedAfterID2(t *testing.T, out string) {
	t.Helper()
	if !strings.Contains(out, "id: 3") {
		t.Errorf("expected replayed id: 3, body:\n%s", out)
	}
	if !strings.Contains(out, "id: 4") {
		t.Errorf("expected replayed id: 4, body:\n%s", out)
	}
	if strings.Contains(out, "id: 1") || strings.Contains(out, "id: 2") {
		t.Errorf("did not expect ids 1 or 2 to be replayed, body:\n%s", out)
	}
	if !strings.Contains(out, "data: <li>third</li>") {
		t.Errorf("expected third event data, body:\n%s", out)
	}
	if !strings.Contains(out, "data: <li>fourth</li>") {
		t.Errorf("expected fourth event data, body:\n%s", out)
	}
}

// TestSSE_RealServer_ReconnectionNoLastID verifies that a fresh client
// (no Last-Event-ID) does not trigger replay; only the initial stream
// handshake occurs.
func TestSSE_RealServer_ReconnectionNoLastID(t *testing.T) {
	t.Parallel()
	store := &memoryEventStore{
		events: []cqrshtmx.SSEEvent{
			{Event: "x", Data: "y", ID: cqrshtmx.NewSSEEventID("1")},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		stream := cqrshtmx.NewSSEStream(w, r)
		defer stream.Close()
		if lastID := cqrshtmx.LastEventIDFromRequest(r); !lastID.IsZero() {
			_, _ = cqrshtmx.ReplayEvents(stream, store, lastID)
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
