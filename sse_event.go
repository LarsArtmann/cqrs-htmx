package cqrshtmx

import (
	"fmt"
	"io"
)

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
