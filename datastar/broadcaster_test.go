package datastar_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/stretchr/testify/require"
)

func TestNewBroadcaster(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()
	require.NotNil(t, b)
	require.Equal(t, 0, b.SubscriberCount())
}

// connectSubscriber starts b.ServeHTTP in a goroutine with a cancellable context.
// Returns a disconnect function. Waits until the subscriber is registered before returning.
func connectSubscriber(t *testing.T, b *ds.Broadcaster) func() {
	t.Helper()

	expected := b.SubscriberCount() + 1

	ctx, ctxCancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/events", nil)
		req = req.WithContext(ctx)
		b.ServeHTTP(w, req)
		close(done)
	}()

	require.Eventually(t, func() bool { return b.SubscriberCount() >= expected }, 2*time.Second, 5*time.Millisecond)

	return func() {
		ctxCancel()
		<-done
	}
}

func TestBroadcasterSubscriberCount(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()

	disconnect1 := connectSubscriber(t, b)
	disconnect2 := connectSubscriber(t, b)
	require.Equal(t, 2, b.SubscriberCount())

	disconnect1()
	require.Eventually(t, func() bool { return b.SubscriberCount() == 1 }, 2*time.Second, 5*time.Millisecond)

	disconnect2()
	require.Eventually(t, func() bool { return b.SubscriberCount() == 0 }, 2*time.Second, 5*time.Millisecond)
}

func TestBroadcasterBroadcastDeliversPatch(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()

	server := httptest.NewServer(b)
	defer server.Close()

	var (
		bodyData string
		bodyMu   sync.Mutex
		wg       sync.WaitGroup
	)

	wg.Add(1)

	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		if err != nil {
			return
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		defer func() { _ = resp.Body.Close() }()

		buf := make([]byte, 8192)
		n, _ := resp.Body.Read(buf)

		bodyMu.Lock()
		bodyData = string(buf[:n])
		bodyMu.Unlock()
	}()

	require.Eventually(t, func() bool { return b.SubscriberCount() == 1 }, 2*time.Second, 5*time.Millisecond)

	b.Broadcast(ds.SignalsPatch(map[string]any{"message": "hello"}))
	b.Close()

	wg.Wait()

	bodyMu.Lock()
	defer bodyMu.Unlock()

	require.Contains(t, bodyData, "datastar-patch-signals")
	require.Contains(t, bodyData, "hello")
}

func TestBroadcasterBroadcastMany(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()
	disconnect := connectSubscriber(t, b)

	b.BroadcastMany(
		ds.SignalsPatch(map[string]any{"step": 1}),
		ds.ElementsPatch("<div>update</div>"),
	)

	disconnect()
	require.Equal(t, 0, b.SubscriberCount())
}

func TestBroadcasterCloseDisconnectsAll(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()

	_ = connectSubscriber(t, b)
	_ = connectSubscriber(t, b)

	require.Equal(t, 2, b.SubscriberCount())

	b.Close()
	require.Equal(t, 0, b.SubscriberCount())
}

func TestBroadcasterServeHTTPLifecycle(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()
	disconnect := connectSubscriber(t, b)

	require.Equal(t, 1, b.SubscriberCount())
	disconnect()
	require.Equal(t, 0, b.SubscriberCount())
}

// readSSEBody connects to serverURL with an optional Last-Event-ID header,
// waits for the client to connect, gives the server time to flush replay data,
// then cancels the context and returns whatever was read.
func readSSEBody(t *testing.T, b *ds.Broadcaster, serverURL, lastEventID string) string {
	t.Helper()

	var (
		body string
		mu   sync.Mutex
		wg   sync.WaitGroup
	)

	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)

	go func() {
		defer wg.Done()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, nil)
		if err != nil {
			cancel()
			return
		}

		if lastEventID != "" {
			req.Header.Set("Last-Event-ID", lastEventID)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			return
		}
		defer func() { _ = resp.Body.Close() }()

		data, _ := io.ReadAll(resp.Body)
		mu.Lock()
		body = string(data)
		mu.Unlock()
	}()

	require.Eventually(t, func() bool { return b.SubscriberCount() >= 1 }, 2*time.Second, 5*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	cancel()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	return body
}

func TestBroadcasterReplayOnReconnect(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcasterWithReplay(10)

	// Broadcast 3 patches before any client connects
	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "first"}))
	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "second"}))
	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "third"}))

	server := httptest.NewServer(b)
	defer server.Close()

	body := readSSEBody(t, b, server.URL, "1")

	// Should replay patches 2 and 3 (not 1)
	require.Contains(t, body, "second")
	require.Contains(t, body, "third")
	require.NotContains(t, body, "first")
	// Should have SSE id: fields for reconnection tracking
	require.Contains(t, body, "id: 2")
	require.Contains(t, body, "id: 3")
}

func TestBroadcasterNoReplayForNewClient(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcasterWithReplay(10)

	// Broadcast patches before any client connects
	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "old-data"}))

	server := httptest.NewServer(b)
	defer server.Close()

	// Connect WITHOUT Last-Event-ID — new client gets no replay
	body := readSSEBody(t, b, server.URL, "")

	// A new client should NOT receive old patches
	require.NotContains(t, body, "old-data")
}

func TestBroadcasterReplayDisabledWithZero(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcasterWithReplay(0)

	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "buffered"}))

	server := httptest.NewServer(b)
	defer server.Close()

	// Even with Last-Event-ID, replay is disabled
	body := readSSEBody(t, b, server.URL, "0")

	require.NotContains(t, body, "buffered")
}

func TestBroadcasterReplayWithQueryParam(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcasterWithReplay(10)

	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "alpha"}))
	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "beta"}))

	server := httptest.NewServer(b)
	defer server.Close()

	body := readSSEBody(t, b, server.URL+"?lastEventId=1", "")

	// Should replay patch 2 (beta) but not patch 1 (alpha)
	require.Contains(t, body, "beta")
	require.NotContains(t, body, "alpha")
}

func TestBroadcasterReplayRingBufferEviction(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcasterWithReplay(3) // small buffer

	// Broadcast 5 patches — only last 3 are retained
	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "p1"}))
	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "p2"}))
	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "p3"}))
	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "p4"}))
	b.Broadcast(ds.SignalsPatch(map[string]any{"seq": "p5"}))

	server := httptest.NewServer(b)
	defer server.Close()

	// Connect with Last-Event-ID: 0 (replay everything in buffer)
	body := readSSEBody(t, b, server.URL, "0")

	// p1 and p2 should have been evicted (buffer holds only last 3)
	require.NotContains(t, body, "p1")
	require.NotContains(t, body, "p2")
	// p3, p4, p5 should be in the replay
	require.Contains(t, body, "p3")
	require.Contains(t, body, "p4")
	require.Contains(t, body, "p5")
}
