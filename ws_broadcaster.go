package cqrshtmx

import (
	"context"
	"net/http"

	"github.com/larsartmann/go-sse"
)

// WSBroadcaster distributes WebSocket messages to all subscribed clients.
// It mirrors the SSE [Broadcaster] API but for WebSocket connections.
//
// Unlike SSE where [SSEStream] manages the connection, WS connections are managed
// by the consumer's WebSocket library. WSBroadcaster only handles message
// fan-out — the consumer reads from the Subscribe channel and writes to their
// WS connection.
//
// Create one at application startup and share it across handlers:
//
//	wsBroadcaster := cqrshtmx.NewWSBroadcaster()
//
//	// In your WS handler (using your WS library of choice):
//	ch := wsBroadcaster.Subscribe()
//	defer wsBroadcaster.Unsubscribe(ch)
//	for msg := range ch {
//	    conn.WriteMessage(websocket.TextMessage, []byte(msg))
//	}
//
//	// Push updates from anywhere:
//	wsBroadcaster.Broadcast("<div hx-swap-oob='true'>Updated</div>")
type WSBroadcaster struct {
	*sse.Broadcaster[string]
}

// NewWSBroadcaster creates a new WebSocket message broadcaster with no subscribers.
func NewWSBroadcaster() *WSBroadcaster {
	return &WSBroadcaster{Broadcaster: sse.NewBroadcaster[string]()}
}

// BroadcastHTML is a convenience method that wraps HTML in an OOB swap
// before broadcasting.
func (b *WSBroadcaster) BroadcastHTML(id, html string, swapStrategy ...SwapStrategy) {
	b.Broadcast(WSOOBHTML(id, html, swapStrategy...))
}

// BroadcastOnSuccessWS creates an AfterDispatchHook that broadcasts a WS message
// when a command dispatch succeeds (err == nil).
func (b *WSBroadcaster) BroadcastOnSuccessWS(msg string) AfterDispatchHook {
	return b.broadcastOnSuccessHook(func(_ *http.Request) string { return msg })
}

// BroadcastOnSuccessWSFunc creates an AfterDispatchHook that generates a WS
// message dynamically from the request when dispatch succeeds.
func (b *WSBroadcaster) BroadcastOnSuccessWSFunc(msgFunc func(r *http.Request) string) AfterDispatchHook {
	return b.broadcastOnSuccessHook(msgFunc)
}

// BroadcastOnErrorWS creates an AfterDispatchHook that broadcasts a WS error
// message when a command dispatch fails (err != nil).
func (b *WSBroadcaster) BroadcastOnErrorWS() AfterDispatchHook {
	return b.broadcastOnErrorHook(func(r *http.Request, err error) string {
		payload := NewStructuredError(err, r)

		return payload.JSON()
	})
}

// BroadcastOnErrorWSFunc creates an AfterDispatchHook that generates a WS error
// message dynamically from the request and error when dispatch fails.
func (b *WSBroadcaster) BroadcastOnErrorWSFunc(errFunc func(r *http.Request, err error) string) AfterDispatchHook {
	return b.broadcastOnErrorHook(errFunc)
}

// broadcastOnSuccessHook builds an AfterDispatchHook that broadcasts the result
// of mapper(r) when dispatch succeeds (err == nil).
func (b *WSBroadcaster) broadcastOnSuccessHook(mapper func(r *http.Request) string) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		if err != nil {
			return
		}

		b.Broadcast(mapper(r))
	}
}

// broadcastOnErrorHook builds an AfterDispatchHook that broadcasts the result
// of mapper(r, err) when dispatch fails (err != nil).
func (b *WSBroadcaster) broadcastOnErrorHook(mapper func(r *http.Request, err error) string) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		if err == nil {
			return
		}

		b.Broadcast(mapper(r, err))
	}
}
