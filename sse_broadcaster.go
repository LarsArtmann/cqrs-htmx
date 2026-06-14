package cqrshtmx

import (
	"context"
	"net/http"
	"reflect"
	"sync"
)

type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[uintptr]chan SSEEvent
}

// NewBroadcaster creates a new event broadcaster with no subscribers.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[uintptr]chan SSEEvent),
	}
}

// channelPtr returns the pointer identity of a channel, regardless of direction.
func channelPtr(ch any) uintptr {
	return reflect.ValueOf(ch).Pointer()
}

// Subscribe creates a new subscriber channel that will receive all broadcast events.
// The channel has a buffer of 64 events; slower consumers may miss events
// when the buffer is full.
//
// Call Unsubscribe when the client disconnects to prevent memory leaks.
func (b *Broadcaster) Subscribe() <-chan SSEEvent {
	ch := make(chan SSEEvent, 64)
	b.mu.Lock()
	b.subscribers[channelPtr(ch)] = ch
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel and closes it.
// Call this when a client disconnects to prevent memory leaks.
// The channel must be one previously returned by Subscribe.
func (b *Broadcaster) Unsubscribe(ch <-chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := channelPtr(ch)
	if sender, ok := b.subscribers[key]; ok {
		delete(b.subscribers, key)
		close(sender)
	}
}

// Broadcast sends an event to all active subscribers.
// Slow subscribers with full buffers have the event dropped to prevent
// blocking the broadcaster.
func (b *Broadcaster) Broadcast(event SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

// SubscriberCount returns the number of active subscribers.
func (b *Broadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// splitSSELines splits a string into lines for SSE data field formatting.
// Each line in the SSE spec must be prefixed with "data: ".
func splitSSELines(s string) []string {
	if s == "" {
		return []string{""}
	}

	var lines []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}

	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// BroadcastOnSuccess creates an AfterDispatchHook that broadcasts an SSE event
// when a command dispatch succeeds (err == nil). This bridges the CQRS dispatch
// lifecycle with SSE real-time updates.
//
// Use it in Config.AfterDispatch to automatically notify SSE clients after
// successful command dispatch:
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    Commands: cmdDispatcher,
//	    AfterDispatch: broadcaster.BroadcastOnSuccess("itemUpdated", ""),
//	})
//
// For dynamic event data based on the request, use BroadcastOnSuccessFunc.
func (b *Broadcaster) BroadcastOnSuccess(eventName, data string) AfterDispatchHook {
	return func(_ context.Context, _ *http.Request, err error) {
		if err != nil {
			return
		}
		b.Broadcast(SSEEvent{Event: eventName, Data: data})
	}
}

// BroadcastOnSuccessFunc creates an AfterDispatchHook that generates an SSE event
// dynamically from the request when dispatch succeeds. The eventFunc receives the
// request and returns the SSE event to broadcast.
//
// Use this when the SSE event data depends on the dispatched command:
//
//	app, _ := cqrshtmx.New(cqrshtmx.Config{
//	    Commands: cmdDispatcher,
//	    AfterDispatch: broadcaster.BroadcastOnSuccessFunc(func(r *http.Request) cqrshtmx.SSEEvent {
//	        return cqrshtmx.SSEEvent{
//	            Event: "itemUpdated",
//	            Data:  renderItemsHTML(),
//	        }
//	    }),
//	})
func (b *Broadcaster) BroadcastOnSuccessFunc(eventFunc func(r *http.Request) SSEEvent) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		if err != nil {
			return
		}
		b.Broadcast(eventFunc(r))
	}
}
