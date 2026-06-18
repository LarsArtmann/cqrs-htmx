package cqrshtmx

import (
	"context"
	"net/http"
)

// WSBroadcaster distributes WebSocket messages to all subscribed clients.
// It mirrors the SSE [Broadcaster] API but for WebSocket connections.
//
// Unlike SSE where SSEStream manages the connection, WS connections are managed
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
	*fanOut[string]
}

// NewWSBroadcaster creates a new WebSocket message broadcaster with no subscribers.
func NewWSBroadcaster() *WSBroadcaster {
	return &WSBroadcaster{fanOut: newFanOut[string]()}
}

// BroadcastHTML is a convenience method that wraps HTML in an OOB swap
// before broadcasting. The target element must have a matching hx-swap-oob
// attribute or the wrapping applies it automatically.
func (b *WSBroadcaster) BroadcastHTML(id, html string, swapStrategy ...SwapStrategy) {
	b.Broadcast(WSOOBHTML(id, html, swapStrategy...))
}

// BroadcastOnSuccessWS creates an AfterDispatchHook that broadcasts a WS message
// when a command dispatch succeeds (err == nil). This is the WebSocket equivalent
// of [Broadcaster.BroadcastOnSuccess].
func (b *WSBroadcaster) BroadcastOnSuccessWS(msg string) AfterDispatchHook {
	return func(_ context.Context, _ *http.Request, err error) {
		if err != nil {
			return
		}
		b.Broadcast(msg)
	}
}

// BroadcastOnSuccessWSFunc creates an AfterDispatchHook that generates a WS
// message dynamically from the request when dispatch succeeds. The msgFunc
// receives the request and returns the message string to broadcast.
//
// This is the WebSocket equivalent of [Broadcaster.BroadcastOnSuccessFunc].
func (b *WSBroadcaster) BroadcastOnSuccessWSFunc(msgFunc func(r *http.Request) string) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		if err != nil {
			return
		}
		b.Broadcast(msgFunc(r))
	}
}

// BroadcastOnErrorWS creates an AfterDispatchHook that broadcasts a WS error
// message when a command dispatch fails (err != nil). The error is serialized
// as a StructuredError JSON string. This is the WebSocket equivalent of
// [Broadcaster.BroadcastOnError].
func (b *WSBroadcaster) BroadcastOnErrorWS() AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		if err == nil {
			return
		}
		payload := NewStructuredError(err, r)
		b.Broadcast(payload.JSON())
	}
}

// BroadcastOnErrorWSFunc creates an AfterDispatchHook that generates a WS error
// message dynamically from the request and error when dispatch fails. The
// errFunc receives both the request and the error, allowing callers to customize
// the message based on the error type.
//
// This is the WebSocket equivalent of [Broadcaster.BroadcastOnErrorFunc].
func (b *WSBroadcaster) BroadcastOnErrorWSFunc(errFunc func(r *http.Request, err error) string) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		if err == nil {
			return
		}
		b.Broadcast(errFunc(r, err))
	}
}
