package datastar_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/go-sse"
	"github.com/stretchr/testify/require"
)

func TestNewBroadcaster(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()
	require.NotNil(t, b)
	require.Equal(t, 0, b.SubscriberCount())
}

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

	patch, err := ds.SignalsPatch(map[string]any{"message": "hello"})
	require.NoError(t, err)

	b.Broadcast(patch)
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

	sigPatch, err := ds.SignalsPatch(map[string]any{"step": 1})
	require.NoError(t, err)

	b.BroadcastMany(
		sigPatch,
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

func TestBroadcasterShutdown(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()

	_ = connectSubscriber(t, b)
	require.Equal(t, 1, b.SubscriberCount())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := b.Shutdown(ctx)
	require.NoError(t, err)
}

func TestBroadcasterHealth(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()
	health := b.Health()

	require.Equal(t, 0, health.SubscriberCount)
	require.False(t, health.Closed)
	require.False(t, health.Draining)
}

func TestBroadcasterBroadcastEvent(t *testing.T) {
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

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
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

	// Broadcast a raw event
	patch := ds.ElementsPatch("<div>raw</div>")
	b.Broadcast(patch)
	b.Close()

	wg.Wait()

	bodyMu.Lock()
	defer bodyMu.Unlock()

	require.Contains(t, bodyData, "datastar-patch-elements")
	require.Contains(t, bodyData, "elements <div>raw</div>")
}

func TestBroadcasterOnSubscribeCallback(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()

	var count int
	var mu sync.Mutex

	b.OnSubscribe(func() {
		mu.Lock()
		count++
		mu.Unlock()
	})

	disconnect := connectSubscriber(t, b)

	mu.Lock()
	require.Equal(t, 1, count)
	mu.Unlock()

	disconnect()
}

func TestBroadcasterReplayOnReconnect(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcasterWithReplay(10)

	for i := range 3 {
		b.BroadcastEvent(sse.Event{
			ID:    sse.NewEventID(strconv.Itoa(i + 1)),
			Event: "feed",
			Data:  fmt.Sprintf("item-%d", i+1),
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req.Header.Set("Last-Event-ID", "1")
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		b.ServeHTTP(w, req)
		close(done)
	}()

	require.Eventually(t, func() bool { return b.SubscriberCount() == 1 }, 2*time.Second, 5*time.Millisecond)

	cancel()
	<-done

	body := w.Body.String()
	require.Contains(t, body, "item-2")
	require.Contains(t, body, "item-3")
	require.NotContains(t, body, "item-1")
}

func TestBroadcasterRaw(t *testing.T) {
	t.Parallel()
	b := ds.NewBroadcaster()
	require.NotNil(t, b.Raw())
}

func TestNewBroadcasterFromRaw(t *testing.T) {
	t.Parallel()
	raw := sse.NewBroadcaster[sse.Event]()
	b := ds.NewBroadcasterFromRaw(raw)
	require.Equal(t, raw, b.Raw())
}

func TestBroadcasterRawSharesFanOut(t *testing.T) {
	t.Parallel()
	raw := sse.NewBroadcaster[sse.Event]()
	b := ds.NewBroadcasterFromRaw(raw)

	ch := raw.Subscribe()
	defer raw.Unsubscribe(ch)

	b.BroadcastEvent(sse.Event{Event: "cross", Data: "transport"})

	require.Eventually(t, func() bool {
		select {
		case evt := <-ch:
			require.Equal(t, "cross", evt.Event)
			return true
		default:
			return false
		}
	}, 2*time.Second, 5*time.Millisecond)
}
