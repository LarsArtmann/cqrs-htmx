package cqrshtmx

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	brandid "github.com/larsartmann/go-branded-id"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// sseEventBrand is the phantom brand type for SSEEventID.
type sseEventBrand struct{}

func (sseEventBrand) Name() string { return "SSEEvent" }

// SSEEventID is a branded identifier for SSE event identifiers (the `id:` field
// and the `Last-Event-ID` request header). It prevents accidental cross-assignment
// with other string-typed IDs (CorrelationID, RequestID, UserID, ActorID, etc.).
//
// SSE event IDs are arbitrary server-defined strings — they are NOT ULIDs.
// Use ParseSSEEventID to construct from a string (rejects control characters
// and newlines, which would corrupt the SSE wire format).
type SSEEventID = brandid.ID[sseEventBrand, string]

// NewSSEEventID constructs an SSEEventID from a string. Performs no validation —
// use ParseSSEEventID for untrusted input (e.g., from request headers).
func NewSSEEventID(s string) SSEEventID { return brandid.NewID[sseEventBrand](s) }

// errSSEEventIDInvalid is returned by ParseSSEEventID for malformed values.
var errSSEEventIDInvalid = event.NewRejection(
	"cqrshtmx.sse.event_id_invalid",
	"sse event id: contains forbidden character (newline or carriage return)",
)

// ParseSSEEventID converts a string to an SSEEventID, rejecting values that
// would corrupt the SSE wire format (newlines, carriage returns). Empty strings
// are allowed (representing "no ID" / initial connection).
func ParseSSEEventID(s string) (SSEEventID, error) {
	if strings.ContainsAny(s, "\n\r") {
		return SSEEventID{}, event.Wrapf(errSSEEventIDInvalid, event.Rejection,
			"cqrshtmx.sse.event_id_invalid", "%q", s)
	}
	return NewSSEEventID(s), nil
}

// MustParseSSEEventID is the panicking variant of ParseSSEEventID for tests
// and constants. Panics if the input contains newlines.
func MustParseSSEEventID(s string) SSEEventID {
	id, err := ParseSSEEventID(s)
	if err != nil {
		panic(fmt.Sprintf("MustParseSSEEventID: %v", err))
	}
	return id
}

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
	ID SSEEventID

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
func WriteSSEEvent(w io.Writer, evt SSEEvent) error {
	var buf []byte

	if evt.Event != "" {
		buf = append(buf, 'e', 'v', 'e', 'n', 't', ':', ' ')
		buf = append(buf, evt.Event...)
		buf = append(buf, '\n')
	}

	for _, line := range splitSSELines(evt.Data) {
		buf = append(buf, 'd', 'a', 't', 'a', ':', ' ')
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}

	if !evt.ID.IsZero() {
		buf = append(buf, 'i', 'd', ':', ' ')
		buf = append(buf, evt.ID.Get()...)
		buf = append(buf, '\n')
	}

	if evt.Retry > 0 {
		buf = append(buf, 'r', 'e', 't', 'r', 'y', ':', ' ')
		buf = strconv.AppendInt(buf, int64(evt.Retry), 10)
		buf = append(buf, '\n')
	}

	buf = append(buf, '\n')

	if _, err := w.Write(buf); err != nil {
		return event.Wrapf(err, event.Transient, "cqrshtmx.sse.write_failed", "write sse event")
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
