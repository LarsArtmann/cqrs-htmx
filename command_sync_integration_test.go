package cqrshtmx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-sse"
)

// --- Integration Test Helpers ---

// sseReadResult captures the output read from an SSE stream.
type sseReadResult struct {
	body string
	err  error
}

// readSSEUntil reads lines from body until the output contains the pattern
// or the context is cancelled. Returns the accumulated body.
func readSSEUntil(ctx context.Context, body io.Reader, patterns ...string) (string, error) {
	resultCh := make(chan sseReadResult, 1)

	go func() {
		scanner := bufio.NewScanner(body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		var sb strings.Builder

		for scanner.Scan() {
			line := scanner.Text()
			sb.WriteString(line)
			sb.WriteByte('\n')

			combined := sb.String()
			allFound := true

			for _, p := range patterns {
				if !strings.Contains(combined, p) {
					allFound = false

					break
				}
			}

			if allFound && len(patterns) > 0 {
				resultCh <- sseReadResult{body: combined, err: nil}

				return
			}
		}

		if err := scanner.Err(); err != nil {
			resultCh <- sseReadResult{body: sb.String(), err: err}

			return
		}

		resultCh <- sseReadResult{body: sb.String(), err: nil}
	}()

	select {
	case r := <-resultCh:
		return r.body, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// newSSEHandler returns an HTTP handler that replays missed events from the
// JournalSSEStore then streams live events from the Broadcaster.
func newSSEHandler(store *JournalSSEStore, bc *Broadcaster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		// Flush response headers immediately so the client receives 200 OK
		// before any event data is written (Go buffers until first Write/Flush).
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		if store != nil {
			if lastID := sse.LastEventIDFromRequest(r); !lastID.IsZero() {
				if _, err := sse.Replay(stream, store, lastID); err != nil {
					return
				}
			}
		}

		ch := bc.Subscribe()
		defer bc.Unsubscribe(ch)

		for {
			select {
			case <-stream.Context().Done():
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}

				if err := stream.Send(evt); err != nil {
					return
				}
			}
		}
	}
}

// makeAckRequest builds a fake *http.Request with the given X-Command-Id header.
func makeAckRequest(cmdID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/commands", nil)
	req.Header.Set(CommandIDHeader, cmdID)

	return req
}

// dialSSE connects to the given URL with optional Last-Event-ID and returns
// the response. The caller must close resp.Body. Uses a transport-level
// ResponseHeaderTimeout (not Client.Timeout) so the streaming body is never
// prematurely cut off.
func dialSSE(t *testing.T, url, lastEventID string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	req.Header.Set("Accept", "text/event-stream")

	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}

	client := &http.Client{ //nolint:exhaustruct // Timeout intentionally omitted for streaming
		Transport: &http.Transport{ //nolint:exhaustruct // defaults are fine
			ResponseHeaderTimeout: 5 * time.Second,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}

	return resp
}

// --- T01: JournalSSEStore replay in a real HTTP handler ---

func TestIntegration_JournalSSEStore_Replay(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedEventList(t, 5)
	appendEvents(t, store, events)

	sseStore := NewJournalSSEStore(store, testMapper)

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		lastID := sse.LastEventIDFromRequest(r)
		_, _ = sse.Replay(stream, sseStore, lastID)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Replay events after event 2 (should get events 3, 4, 5)
	cursor := events[1].ID().String()

	resp := dialSSE(t, server.URL+"/events", cursor)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	body, err := readSSEUntil(
		ctx, resp.Body,
		"id: "+events[2].ID().String(),
		"id: "+events[4].ID().String(),
	)
	if err != nil {
		t.Fatalf("read SSE: %v\nbody:\n%s", err, body)
	}

	// Should NOT contain events 1 or 2 (they were before the cursor)
	if strings.Contains(body, "id: "+events[0].ID().String()) {
		t.Errorf("body should not contain event before cursor:\n%s", body)
	}

	if strings.Contains(body, "id: "+events[1].ID().String()) {
		t.Errorf("body should not contain cursor event:\n%s", body)
	}
}

// --- T02: ACK confirmed over SSE ---

func TestIntegration_ACK_ConfirmedOverSSE(t *testing.T) {
	t.Parallel()

	bc := NewBroadcaster()

	server := httptest.NewServer(newSSEHandler(nil, bc))
	defer server.Close()

	resp := dialSSE(t, server.URL+"/events", "")
	defer func() { _ = resp.Body.Close() }()

	// Give the handler time to subscribe before we broadcast
	time.Sleep(100 * time.Millisecond)

	// Simulate a successful dispatch
	hook := bc.BroadcastOnAck()
	hook(t.Context(), makeAckRequest("cmd-confirmed-001"), nil)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	body, err := readSSEUntil(ctx, resp.Body, `"commandId":"cmd-confirmed-001"`, `"status":"confirmed"`)
	if err != nil {
		t.Fatalf("read SSE: %v\nbody:\n%s", err, body)
	}

	if !strings.Contains(body, "event: sync:ack") {
		t.Errorf("expected sync:ack event name in body:\n%s", body)
	}
}

// --- T03: ACK rejected over SSE ---

func TestIntegration_ACK_RejectedOverSSE(t *testing.T) {
	t.Parallel()

	bc := NewBroadcaster()

	server := httptest.NewServer(newSSEHandler(nil, bc))
	defer server.Close()

	resp := dialSSE(t, server.URL+"/events", "")
	defer func() { _ = resp.Body.Close() }()

	time.Sleep(100 * time.Millisecond)

	// Simulate a failed dispatch
	dispatchErr := errors.New("email already exists")
	hook := bc.BroadcastOnAck()
	hook(t.Context(), makeAckRequest("cmd-rejected-002"), dispatchErr)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	body, err := readSSEUntil(
		ctx, resp.Body,
		`"commandId":"cmd-rejected-002"`,
		`"status":"rejected"`,
		`"error":"email already exists"`,
	)
	if err != nil {
		t.Fatalf("read SSE: %v\nbody:\n%s", err, body)
	}
}

// --- T04: Reconnect replay + live ACK in one stream ---

func TestIntegration_ReconnectAndLiveACK(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedEventList(t, 4)
	appendEvents(t, store, events)

	sseStore := NewJournalSSEStore(store, testMapper)
	bc := NewBroadcaster()

	server := httptest.NewServer(newSSEHandler(sseStore, bc))
	defer server.Close()

	// Connect with Last-Event-ID = event 2 → should replay events 3, 4
	cursor := events[1].ID().String()

	resp := dialSSE(t, server.URL+"/events", cursor)
	defer func() { _ = resp.Body.Close() }()

	// Phase 1: read the replayed events
	ctx1, cancel1 := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel1()

	replayBody, err := readSSEUntil(
		ctx1, resp.Body,
		"id: "+events[2].ID().String(),
		"id: "+events[3].ID().String(),
	)
	if err != nil {
		t.Fatalf("replay phase: %v\nbody:\n%s", err, replayBody)
	}

	// Phase 2: trigger a live ACK broadcast — must arrive in the SAME stream
	time.Sleep(100 * time.Millisecond) // ensure handler is in the live loop

	hook := bc.BroadcastOnAck()
	hook(t.Context(), makeAckRequest("cmd-live-003"), nil)

	ctx2, cancel2 := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel2()

	liveBody, err := readSSEUntil(ctx2, resp.Body, `"commandId":"cmd-live-003"`, `"status":"confirmed"`)
	if err != nil {
		t.Fatalf("live ACK phase: %v\nbody:\n%s", err, liveBody)
	}

	// The combined stream had both replayed events and a live ACK
	if !strings.Contains(replayBody+liveBody, "sync:ack") {
		t.Error("expected sync:ack event in the combined stream")
	}
}

// --- No X-Command-Id → no ACK (opt-in guard, integration level) ---

func TestIntegration_ACK_NoCommandID_NoBroadcast(t *testing.T) {
	t.Parallel()

	bc := NewBroadcaster()

	server := httptest.NewServer(newSSEHandler(nil, bc))
	defer server.Close()

	resp := dialSSE(t, server.URL+"/events", "")
	defer func() { _ = resp.Body.Close() }()

	time.Sleep(100 * time.Millisecond)

	// Request WITHOUT X-Command-Id → hook should be a no-op
	req := httptest.NewRequest(http.MethodPost, "/commands", nil)
	hook := bc.BroadcastOnAck()
	hook(t.Context(), req, nil)

	// Wait a bit and verify nothing was broadcast
	time.Sleep(200 * time.Millisecond)

	// Subscriber count should still be 1 (no events sent, channel empty)
	// We verify by checking that reading yields nothing immediately
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()

	body, _ := readSSEUntil(ctx, resp.Body, "this-pattern-does-not-exist")
	if strings.Contains(body, "sync:ack") {
		t.Errorf("no ACK should be broadcast without X-Command-Id header:\n%s", body)
	}
}

// --- Concurrent replay + broadcast (race detector) ---

func TestIntegration_ConcurrentReplayAndBroadcast(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedEventList(t, 20)
	appendEvents(t, store, events)

	sseStore := NewJournalSSEStore(store, testMapper)
	bc := NewBroadcaster()

	server := httptest.NewServer(newSSEHandler(sseStore, bc))
	defer server.Close()

	var wg sync.WaitGroup

	// 5 concurrent SSE clients, each reconnecting with different cursors
	for i := range 5 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			cursor := ""
			if idx > 0 {
				cursor = events[idx*2-1].ID().String()
			}

			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()

			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/events", nil)
			req.Header.Set("Last-Event-ID", cursor)

			client := &http.Client{} //nolint:exhaustruct // intentional

			resp, err := client.Do(req)
			if err != nil {
				return
			}

			defer func() { _ = resp.Body.Close() }()

			_, _ = io.ReadAll(resp.Body)
		}(i)
	}

	// Concurrently broadcast ACKs while replays happen
	for i := range 5 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			hook := bc.BroadcastOnAck()
			hook(t.Context(), makeAckRequest(fmt.Sprintf("concurrent-%d", idx)), nil)
		}(i)
	}

	wg.Wait()
}
