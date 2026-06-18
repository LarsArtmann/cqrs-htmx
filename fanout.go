package cqrshtmx

import (
	"context"
	"net/http"
	"reflect"
	"sync"
)

// fanOut is the transport-agnostic subscriber hub shared by SSE [Broadcaster]
// and [WSBroadcaster]. It provides thread-safe fan-out with O(1) unsubscribe
// via channel pointer identity and non-blocking broadcast (drops to slow
// consumers).
//
// Both broadcasters embed fanOut, gaining Subscribe/Unsubscribe/Broadcast/
// SubscriberCount via method promotion. Transport-specific hook constructors
// (BroadcastOnSuccess, BroadcastOnSuccessWS, etc.) live on the outer types.
type fanOut[T any] struct {
	mu          sync.RWMutex
	subscribers map[uintptr]chan T
}

// newFanOut creates a fan-out hub with no subscribers.
func newFanOut[T any]() *fanOut[T] {
	return &fanOut[T]{
		mu:          sync.RWMutex{},
		subscribers: make(map[uintptr]chan T),
	}
}

// Subscribe creates a new subscriber channel that receives all broadcast
// messages. The channel has a buffer of 64; slower consumers may miss messages
// when the buffer is full.
//
// Call Unsubscribe when the client disconnects to prevent memory leaks.
func (f *fanOut[T]) Subscribe() <-chan T {
	ch := make(chan T, 64)
	f.mu.Lock()
	f.subscribers[channelPtr(ch)] = ch
	f.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel and closes it.
// Call this when a client disconnects to prevent memory leaks.
func (f *fanOut[T]) Unsubscribe(ch <-chan T) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := channelPtr(ch)
	if sender, ok := f.subscribers[key]; ok {
		delete(f.subscribers, key)
		close(sender)
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

// channelPtr returns the pointer identity of a channel, regardless of direction.
func channelPtr(ch any) uintptr {
	return reflect.ValueOf(ch).Pointer()
}
