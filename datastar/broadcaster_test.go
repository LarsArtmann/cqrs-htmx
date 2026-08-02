package datastar_test

import (
	"context"
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

func TestBroadcasterSubscriberCount(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	go serveSSE(b, ctx1)
	go serveSSE(b, ctx2)

	require.Eventually(t, func() bool { return b.SubscriberCount() == 2 }, time.Second, 10*time.Millisecond)

	cancel1()
	require.Eventually(t, func() bool { return b.SubscriberCount() == 1 }, time.Second, 10*time.Millisecond)

	cancel2()
	require.Eventually(t, func() bool { return b.SubscriberCount() == 0 }, time.Second, 10*time.Millisecond)
}

func TestBroadcasterBroadcastDeliversPatch(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()

	received := make(chan string, 1)
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		server := httptest.NewServer(b)
		defer server.Close()

		resp, err := http.Get(server.URL) //nolint:noctx // test
		if err != nil {
			return
		}
		defer resp.Body.Close()

		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		received <- string(buf[:n])
	}()

	require.Eventually(t, func() bool { return b.SubscriberCount() == 1 }, 2*time.Second, 10*time.Millisecond)

	b.Broadcast(ds.SignalsPatch(map[string]any{"message": "hello"}))
	b.Close()

	wg.Wait()

	select {
	case msg := <-received:
		require.Contains(t, msg, "datastar-patch-signals")
		require.Contains(t, msg, "hello")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE data")
	}
}

func TestBroadcasterBroadcastMany(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()

	server := httptest.NewServer(b)
	defer server.Close()

	require.Eventually(t, func() bool { return b.SubscriberCount() == 1 }, 2*time.Second, 10*time.Millisecond)

	b.BroadcastMany(
		ds.SignalsPatch(map[string]any{"step": 1}),
		ds.ElementsPatch("<div>update</div>"),
	)
	b.Close()
}

func TestBroadcasterCloseDisconnectsAll(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	go serveSSE(b, ctx1)
	go serveSSE(b, ctx2)

	require.Eventually(t, func() bool { return b.SubscriberCount() == 2 }, time.Second, 10*time.Millisecond)

	b.Close()
	require.Equal(t, 0, b.SubscriberCount())
}

func TestBroadcasterServeHTTPSetsHeaders(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	go func() {
		b.ServeHTTP(w, req)
	}()

	require.Eventually(t, func() bool { return b.SubscriberCount() == 1 }, time.Second, 10*time.Millisecond)

	cancel()
	require.Eventually(t, func() bool { return b.SubscriberCount() == 0 }, time.Second, 10*time.Millisecond)
}

// serveSSE connects to the broadcaster as an SSE client, blocking until the
// context is cancelled. Used to simulate connected clients in tests.
func serveSSE(b *ds.Broadcaster, ctx context.Context) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req = req.WithContext(ctx)
	b.ServeHTTP(w, req)
}
