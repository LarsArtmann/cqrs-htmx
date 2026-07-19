package cqrshtmx

import (
	"context"
	"net/http"
	"reflect"
	"sync"
)

// defaultSubscriberBuffer is the per-subscriber channel capacity. Broadcasts
// are non-blocking: a subscriber whose buffer is full is dropped (see
// Broadcaster docs). 64 is large enough to absorb short bursts without dropping
// under normal fan-out, while bounding memory per subscriber.
const defaultSubscriberBuffer = 64

// fanOut is the transport-agnostic subscriber hub shared by SSE [Broadcaster]
// and [WSBroadcaster]. It provides thread-safe fan-out with O(1) unsubscribe
// via channel pointer identity and non-blocking broadcast (drops to slow
// consumers).
//
// Both broadcasters embed fanOut, gaining Subscribe/Unsubscribe/Broadcast/
// SubscriberCount via method promotion. Transport-specific hook constructors
// (BroadcastOnSuccess, BroadcastOnSuccessWS, etc.) live on the outer types.
type fanOut[T any] struct {
	mu            sync.RWMutex
	subscribers   map[uintptr]chan T
	onSubscribe   func()
	onUnsubscribe func()
}

// newFanOut creates a fan-out hub with no subscribers.
func newFanOut[T any]() *fanOut[T] {
	return &fanOut[T]{
		mu:            sync.RWMutex{},
		subscribers:   make(map[uintptr]chan T),
		onSubscribe:   nil,
		onUnsubscribe: nil,
	}
}

// Subscribe creates a new subscriber channel that receives all broadcast
// messages. The channel has a buffer of 64; slower consumers may miss messages
// when the buffer is full.
//
// Call Unsubscribe when the client disconnects to prevent memory leaks.
// After Close, Subscribe returns a closed channel (no-op).
func (f *fanOut[T]) Subscribe() <-chan T {
	ch := make(chan T, defaultSubscriberBuffer)

	f.mu.Lock()
	if f.subscribers == nil {
		f.mu.Unlock()
		close(ch) // already closed — return a closed channel

		return ch
	}

	f.subscribers[channelPtr(ch)] = ch
	onSub := f.onSubscribe
	f.mu.Unlock()

	if onSub != nil {
		onSub()
	}

	return ch
}

// Unsubscribe removes a subscriber channel and closes it.
// Call this when a client disconnects to prevent memory leaks.
func (f *fanOut[T]) Unsubscribe(ch <-chan T) {
	f.mu.Lock()
	key := channelPtr(ch)

	sender, ok := f.subscribers[key]
	if ok {
		delete(f.subscribers, key)
		close(sender)
	}

	onUnsub := f.onUnsubscribe
	f.mu.Unlock()

	if ok && onUnsub != nil {
		onUnsub()
	}
}

// Broadcast sends a message to all active subscribers.
// Slow subscribers with full buffers have the message dropped to prevent
// blocking the broadcaster.
//
// The iteration runs under the read lock so that a concurrent Unsubscribe
// cannot close a channel that this loop is about to send on — sending to a
// closed channel would panic. Because sends use a non-blocking select, no
// goroutine blocks here, and the lock is held only for the brief fan-out.
func (f *fanOut[T]) Broadcast(msg T) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, ch := range f.subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
}

// SubscriberCount returns the number of active subscribers.
func (f *fanOut[T]) SubscriberCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.subscribers)
}

// Close shuts down the fan-out hub: it closes all subscriber channels and
// marks the hub as closed so that future Subscribe calls return a closed
// channel (no-op). Broadcasts after Close are silently dropped.
//
// This is the graceful-shutdown primitive for SSE/WS broadcasters. Call it
// when your server is shutting down so connected clients receive a channel-close
// signal and their read loops exit cleanly:
//
//	broadcaster := cqrshtmx.NewBroadcaster()
//	defer broadcaster.Close() // or call in your shutdown handler
func (f *fanOut[T]) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()

	for key, ch := range f.subscribers {
		delete(f.subscribers, key)
		close(ch)
	}

	f.subscribers = nil // marks as closed
}

// broadcastOnSuccessHook builds an AfterDispatchHook that broadcasts the result
// of mapper(r) when dispatch succeeds (err == nil). Shared by SSE and WS
// broadcasters to keep their public-facing hook constructors symmetric.
func (f *fanOut[T]) broadcastOnSuccessHook(mapper func(r *http.Request) T) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		if err != nil {
			return
		}

		f.Broadcast(mapper(r))
	}
}

// broadcastOnErrorHook builds an AfterDispatchHook that broadcasts the result
// of mapper(r, err) when dispatch fails (err != nil). Shared by SSE and WS
// broadcasters to keep their public-facing hook constructors symmetric.
func (f *fanOut[T]) broadcastOnErrorHook(mapper func(r *http.Request, err error) T) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		if err == nil {
			return
		}

		f.Broadcast(mapper(r, err))
	}
}

// setOnSubscribe sets a callback fired after each successful Subscribe.
// Used by Broadcaster.OnSubscribe and WSBroadcaster.OnSubscribe.
func (f *fanOut[T]) setOnSubscribe(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.onSubscribe = fn
}

// setOnUnsubscribe sets a callback fired after each successful Unsubscribe.
// Used by Broadcaster.OnUnsubscribe and WSBroadcaster.OnUnsubscribe.
func (f *fanOut[T]) setOnUnsubscribe(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.onUnsubscribe = fn
}

// channelPtr returns the pointer identity of a channel, regardless of direction.
func channelPtr(ch any) uintptr {
	return reflect.ValueOf(ch).Pointer()
}
