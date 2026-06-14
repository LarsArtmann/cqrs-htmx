package cqrshtmx

import (
	"io"
	"net/http"
)

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
