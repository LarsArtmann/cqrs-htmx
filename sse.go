package cqrshtmx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"
)

// SSEEvent represents a Server-Sent Event for the HTMX SSE extension.
//
// The HTMX SSE extension connects to an EventSource and swaps HTML content
// from SSE data into the DOM. The Event field must match the client's
// sse-swap attribute value.
//
// Server-side:
//
//	stream := cqrshtmx.NewSSEStream(w, r)
//	_ = stream.Send(cqrshtmx.SSEEvent{
//	    Event: "todoUpdated",
//	    Data:  "<div id='todos'><ul><li>Buy milk</li></ul></div>",
//	})
//
// Client-side (HTML):
//
//	<div hx-ext="sse" sse-connect="/events" sse-swap="todoUpdated">
//	  <!-- content swapped here -->
//	</div>
type SSEEvent struct {
	// Event is the SSE event name. Must match the client's sse-swap attribute.
	// For unnamed events, use "message" (the browser default).
	Event string

	// Data is the event payload. For HTMX, this is typically HTML content
	// that will be swapped into the DOM. Multi-line data is supported;
	// each line is prefixed with "data: " per the SSE specification.
	Data string

	// ID is an optional event identifier. The browser sends this as
	// Last-Event-ID on reconnection, enabling replay of missed events.
	ID string

	// Retry is an optional reconnection time in milliseconds.
	// Instructs the browser to wait this long before reconnecting
	// after a connection drop.
	Retry int
}

// WriteSSEEvent writes a single SSE event to the writer in the standard
// Server-Sent Events wire format:
//
//	event: <name>
//	data: <line1>
//	data: <line2>
//	id: <id>        (optional)
//	retry: <ms>     (optional)
//
// Each event is terminated by a blank line ("\n\n").
func WriteSSEEvent(w io.Writer, event SSEEvent) error {
	if event.Event != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", event.Event); err != nil {
			return fmt.Errorf("write sse event name: %w", err)
		}
	}

	// SSE spec: data field can be multi-line; each line gets its own "data: " prefix.
	for _, line := range splitSSELines(event.Data) {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return fmt.Errorf("write sse data: %w", err)
		}
	}

	if event.ID != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
			return fmt.Errorf("write sse id: %w", err)
		}
	}

	if event.Retry > 0 {
		if _, err := fmt.Fprintf(w, "retry: %d\n", event.Retry); err != nil {
			return fmt.Errorf("write sse retry: %w", err)
		}
	}

	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("write sse terminator: %w", err)
	}

	return nil
}

// SSEStream manages a single Server-Sent Events connection.
// It sets the required HTTP headers and provides methods to send events
// to one connected client.
//
// Create one per HTTP handler invocation:
//
//	func handleEvents(w http.ResponseWriter, r *http.Request) {
//	    stream := cqrshtmx.NewSSEStream(w, r)
//	    defer stream.Close()
//
//	    ch := broadcaster.Subscribe()
//	    defer broadcaster.Unsubscribe(ch)
//
//	    for {
//	        select {
//	        case <-stream.Context().Done():
//	            return
//	        case event := <-ch:
//	            if err := stream.Send(event); err != nil {
//	                return
//	            }
//	        }
//	    }
//	}
type SSEStream struct {
	w   io.Writer
	r   *http.Request
	fw  flusher
	ctx interface{ Done() <-chan struct{} }
}

type flusher interface{ Flush() }

// NewSSEStream creates an SSE stream from an HTTP response writer and request.
// Sets the required SSE headers (Content-Type, Cache-Control, Connection).
// Returns an SSEStream that can be used to send events to the client.
//
// The stream is cancelled when the request context is done (client disconnects).
// Callers should defer stream.Close() to ensure cleanup.
func NewSSEStream(w http.ResponseWriter, r *http.Request) *SSEStream {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	fw, _ := w.(flusher)
	return &SSEStream{w: w, r: r, fw: fw, ctx: r.Context()}
}

// Send writes an SSE event to the stream and flushes the response.
// Returns an error if the write fails (e.g., client disconnected).
func (s *SSEStream) Send(event SSEEvent) error {
	if err := WriteSSEEvent(s.w, event); err != nil {
		return err
	}
	if s.fw != nil {
		s.fw.Flush()
	}
	return nil
}

// SendHTML is a convenience method that sends an HTML fragment as a named SSE event.
// The eventName must match the client's sse-swap attribute.
func (s *SSEStream) SendHTML(eventName, html string) error {
	return s.Send(SSEEvent{Event: eventName, Data: html})
}

// Context returns the stream's context. It is cancelled when the client disconnects.
// Use this in select statements to detect disconnection.
func (s *SSEStream) Context() interface{ Done() <-chan struct{} } {
	return s.ctx
}

// Close flushes any buffered data. Call this (typically via defer) when done
// with the stream.
func (s *SSEStream) Close() {
	if s.fw != nil {
		s.fw.Flush()
	}
}

// LastEventID returns the Last-Event-ID header from the connection request.
// The browser sends this on reconnection to indicate the last event it received.
// Returns empty string if not present.
func (s *SSEStream) LastEventID() string {
	return s.r.Header.Get("Last-Event-ID")
}

// LastEventIDFromRequest extracts the Last-Event-ID header from an HTTP request.
// This is the SSE reconnection mechanism: when a client reconnects after a
// connection drop, the browser sends the ID of the last event it received.
//
// Use this to replay missed events:
//
//	lastID := cqrshtmx.LastEventIDFromRequest(r)
//	if lastID != "" {
//	    events := store.EventsAfter(lastID)
//	    for _, evt := range events {
//	        stream.Send(evt)
//	    }
//	}
func LastEventIDFromRequest(r *http.Request) string {
	return r.Header.Get("Last-Event-ID")
}

// SSEEventStore retrieves events for SSE reconnection replay.
// Implementations decide the storage backend and retention policy.
type SSEEventStore interface {
	// EventsAfter returns events with IDs strictly after the given lastID.
	// Returns an empty slice if no events are found or lastID is unknown.
	// The returned slice must be ordered by event ID (ascending).
	EventsAfter(lastID string) []SSEEvent
}

// ReplayEvents sends all events from the store after the given lastEventID
// through the stream. This is used for SSE reconnection: when a client
// reconnects with a Last-Event-ID header, replay the events it missed.
//
// Returns the number of events replayed, or an error if writing fails.
func ReplayEvents(stream *SSEStream, store SSEEventStore, lastEventID string) (int, error) {
	events := store.EventsAfter(lastEventID)
	for i, evt := range events {
		if err := stream.Send(evt); err != nil {
			return i, err
		}
	}
	return len(events), nil
}

// Broadcaster distributes SSE events to all subscribed clients.
// It is safe for concurrent use.
//
// Create one at application startup and share it across handlers:
//
//	broadcaster := cqrshtmx.NewBroadcaster()
//
//	// In your SSE endpoint handler:
//	ch := broadcaster.Subscribe()
//	defer broadcaster.Unsubscribe(ch)
//
//	// In your CQRS event handler or AfterDispatch hook:
//	broadcaster.Broadcast(cqrshtmx.SSEEvent{
//	    Event: "itemCreated",
//	    Data:  renderTemplate(),
//	})
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
