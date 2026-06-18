package cqrshtmx

import (
	"context"
	"net/http"
	"sync"
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
	mu          sync.RWMutex
	subscribers map[uintptr]chan string
}

// NewWSBroadcaster creates a new WebSocket message broadcaster with no subscribers.
func NewWSBroadcaster() *WSBroadcaster {
	return &WSBroadcaster{
		mu:          sync.RWMutex{},
		subscribers: make(map[uintptr]chan string),
	}
}

// Subscribe creates a new subscriber channel that receives all broadcast messages.
// The channel has a buffer of 64 messages; slower consumers may miss messages
// when the buffer is full.
//
// Call Unsubscribe when the client disconnects to prevent memory leaks.
func (b *WSBroadcaster) Subscribe() <-chan string {
	ch := make(chan string, 64)
	b.mu.Lock()
	b.subscribers[channelPtr(ch)] = ch
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel and closes it.
// Call this when a client disconnects to prevent memory leaks.
func (b *WSBroadcaster) Unsubscribe(ch <-chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := channelPtr(ch)
	if sender, ok := b.subscribers[key]; ok {
		delete(b.subscribers, key)
		close(sender)
	}
}

// Broadcast sends a message to all active subscribers.
// Slow subscribers with full buffers have the message dropped to prevent
// blocking the broadcaster.
func (b *WSBroadcaster) Broadcast(msg string) {
	b.mu.RLock()
	if len(b.subscribers) == 0 {
		b.mu.RUnlock()
		return
	}
	snapshot := make([]chan string, 0, len(b.subscribers))
	for _, ch := range b.subscribers {
		snapshot = append(snapshot, ch)
	}
	b.mu.RUnlock()

	for _, ch := range snapshot {
		select {
		case ch <- msg:
		default:
		}
	}
}

// BroadcastHTML is a convenience method that wraps HTML in an OOB swap
// before broadcasting. The target element must have a matching hx-swap-oob
// attribute or the wrapping applies it automatically.
func (b *WSBroadcaster) BroadcastHTML(id, html string, swapStrategy ...SwapStrategy) {
	b.Broadcast(WSOOBHTML(id, html, swapStrategy...))
}

// SubscriberCount returns the number of active subscribers.
func (b *WSBroadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
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
