package cqrshtmx

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

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
	w            io.Writer
	r            *http.Request
	fw           flusher
	ctx          context.Context
	onDisconnect []func()

	// mu guards every write to w and every flush against concurrent access.
	// Send runs from the event-loop goroutine while Heartbeat runs from a
	// separate goroutine (see Heartbeat docs). http.ResponseWriter is not
	// safe for concurrent use, so both paths must hold mu.
	mu sync.Mutex
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
	return &SSEStream{
		w:            w,
		r:            r,
		fw:           fw,
		ctx:          r.Context(),
		onDisconnect: nil,
		mu:           sync.Mutex{},
	}
}

// Send writes an SSE event to the stream and flushes the response.
// Returns an error if the write fails (e.g., client disconnected).
//
// Send is safe to call concurrently with Heartbeat: both serialize on the
// stream's mutex so the underlying ResponseWriter is never written by two
// goroutines at once.
func (s *SSEStream) Send(event SSEEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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

// Context returns the stream's context.Context. It is cancelled when the
// client disconnects. Use ctx.Done() in select statements to detect
// disconnection, or ctx.Err() to check the cancellation reason.
func (s *SSEStream) Context() context.Context {
	return s.ctx
}

// Close flushes any buffered data and fires any registered OnDisconnect callbacks.
// Call this (typically via defer) when done with the stream.
func (s *SSEStream) Close() {
	s.mu.Lock()
	if s.fw != nil {
		s.fw.Flush()
	}
	callbacks := s.onDisconnect
	s.onDisconnect = nil
	s.mu.Unlock()
	for _, fn := range callbacks {
		fn()
	}
}

// LastEventID returns the Last-Event-ID header from the connection request.
// The browser sends this on reconnection to indicate the last event it received.
// Returns empty SSEEventID if not present.
func (s *SSEStream) LastEventID() SSEEventID {
	return LastEventIDFromRequest(s.r)
}

// Heartbeat sends SSE comment-frame pings at the given interval until ctx
// is cancelled. This prevents reverse proxies (Nginx, Cloudflare, AWS ALB)
// and corporate firewalls from killing idle SSE connections after 30–60s
// of silence.
//
// Run it in a goroutine alongside your event loop:
//
//	go stream.Heartbeat(stream.Context(), 15*time.Second)
//
// The ping is a standard SSE comment frame (": keepalive\n\n") which browsers
// ignore but proxies use to reset their idle timers.
func (s *SSEStream) Heartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			_, err := s.w.Write([]byte(": keepalive\n\n"))
			if err == nil && s.fw != nil {
				s.fw.Flush()
			}
			s.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// OnDisconnect registers a callback that fires when Close is called.
// Use this for cleanup, metrics, logging, or session deregistration.
// Multiple callbacks can be registered and fire in registration order.
func (s *SSEStream) OnDisconnect(fn func()) {
	s.mu.Lock()
	s.onDisconnect = append(s.onDisconnect, fn)
	s.mu.Unlock()
}

// LastEventIDFromRequest extracts the Last-Event-ID header from an HTTP request,
// returning the branded SSEEventID. This is the SSE reconnection mechanism: when a
// client reconnects after a connection drop, the browser sends the ID of the last
// event it received.
//
// Use this to replay missed events:
//
//	lastID := cqrshtmx.LastEventIDFromRequest(r)
//	if !lastID.IsZero() {
//	    events := store.EventsAfter(lastID.String())
//	    for _, evt := range events {
//	        stream.Send(evt)
//	    }
//	}
func LastEventIDFromRequest(r *http.Request) SSEEventID {
	return NewSSEEventID(r.Header.Get("Last-Event-ID"))
}
