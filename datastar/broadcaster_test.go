package datastar_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/go-datastar/broadcast"
	"github.com/larsartmann/go-sse"
	"github.com/stretchr/testify/require"
)

// The Broadcaster implementation lives in go-datastar/broadcast since
// 2026-09-17; the full behavior suite (fan-out, replay, hub sharing,
// subscribe-before-replay ordering) is maintained upstream. The tests here pin
// only the deprecated facade: alias transparency and constructor delegation.

// acceptBroadcaster takes the deprecated facade type. The call in the test
// below compiles only because ds.Broadcaster is a transparent alias of
// broadcast.Broadcaster — EventBridge and every existing consumer signature
// keep compiling unchanged.
func acceptBroadcaster(_ *ds.Broadcaster) {}

func TestBroadcasterFacadeIsAlias(t *testing.T) {
	t.Parallel()

	b := broadcast.NewBroadcaster()
	acceptBroadcaster(b)
	require.NotNil(t, b)
	require.Equal(t, 0, b.SubscriberCount())
}

// Deprecated constructor pins — each must delegate to the upstream module and
// stay functional until the v5 removal.
func TestBroadcasterDeprecatedConstructors(t *testing.T) {
	t.Parallel()

	require.NotNil(t, ds.NewBroadcaster())
	require.NotNil(t, ds.NewBroadcasterWithBufferSize(8))
	require.NotNil(t, ds.NewBroadcasterWithReplay(4))
	require.NotNil(t, ds.NewBroadcasterFromHub(sse.NewBroadcaster[sse.Event]()))
}

// Deprecated API pin — NewBroadcasterFromRaw stays functional until v5 removal.
func TestBroadcasterDeprecatedFromRaw(t *testing.T) {
	t.Parallel()

	hub := sse.NewBroadcaster[sse.Event]()
	b := ds.NewBroadcasterFromRaw(hub)
	require.Equal(t, hub, b.Hub())
}

// connectSubscriber starts one SSE connection against b in a goroutine and
// returns a disconnect func. It blocks until the subscriber is registered.
func connectSubscriber(t *testing.T, b *broadcast.Broadcaster) func() {
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
