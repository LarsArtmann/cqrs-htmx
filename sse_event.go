package cqrshtmx

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// SSEEvent represents a single Server-Sent Events message.
// Use WriteSSEEvent to write it in the SSE wire format.
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
//
// Uses io.WriteString and direct byte writes instead of fmt.Fprintf to
// minimize allocations on the SSE hot path.
func WriteSSEEvent(w io.Writer, event SSEEvent) error {
	var buf []byte

	if event.Event != "" {
		buf = append(buf, 'e', 'v', 'e', 'n', 't', ':', ' ')
		buf = append(buf, event.Event...)
		buf = append(buf, '\n')
	}

	for _, line := range splitSSELines(event.Data) {
		buf = append(buf, 'd', 'a', 't', 'a', ':', ' ')
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}

	if event.ID != "" {
		buf = append(buf, 'i', 'd', ':', ' ')
		buf = append(buf, event.ID...)
		buf = append(buf, '\n')
	}

	if event.Retry > 0 {
		buf = append(buf, 'r', 'e', 't', 'r', 'y', ':', ' ')
		buf = strconv.AppendInt(buf, int64(event.Retry), 10)
		buf = append(buf, '\n')
	}

	buf = append(buf, '\n')

	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("write sse event: %w", err)
	}

	return nil
}

// splitSSELines splits a string into lines for SSE data field formatting.
// Each line in the SSE spec must be prefixed with "data: ".
// Fast path: if the data contains no newline, returns a single-element
// slice without allocating a backing array.
func splitSSELines(s string) []string {
	if s == "" {
		return []string{""}
	}

	// Fast path: no newline → single line, no allocation.
	if !strings.Contains(s, "\n") {
		return []string{s}
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
