package cqrshtmx

import (
	"io"
	"net/http"

	"github.com/larsartmann/go-sse"
)

// SSE types are delegated to [github.com/larsartmann/go-sse].
// These aliases preserve backward compatibility with existing cqrs-htmx consumers.
// New code should import go-sse directly.

// Common SSE event names. Consumers are free to use custom event names.
const (
	// SSEEventConnected is the conventional event name for the initial
	// connection-acknowledgement event sent when an SSE stream opens.
	SSEEventConnected = sse.EventConnected
	// SSEEventHeartbeat is the conventional event name for heartbeat/ping
	// events. Note: stream.Heartbeat() sends comment-frame pings (not named
	// events); this constant is for consumers who prefer named heartbeats.
	SSEEventHeartbeat = sse.EventHeartbeat
)

// ContentTypeSSE is the HTTP content type for Server-Sent Events.
const ContentTypeSSE = sse.ContentType

// SSEEventID is a branded identifier for SSE event identifiers.
type SSEEventID = sse.EventID

// SSEEvent represents a single Server-Sent Event.
type SSEEvent = sse.Event

// NewSSEEventID constructs an SSEEventID from a string without validation.
func NewSSEEventID(s string) SSEEventID { return sse.NewEventID(s) }

// ParseSSEEventID converts a string to an SSEEventID, rejecting newlines.
func ParseSSEEventID(s string) (SSEEventID, error) {
	return sse.ParseEventID(s) //nolint:wrapcheck // pure delegation
}

// MustParseSSEEventID is the panicking variant for tests and constants.
func MustParseSSEEventID(s string) SSEEventID { return sse.MustParseEventID(s) }

// WriteSSEEvent writes a single SSE event in the standard wire format.
func WriteSSEEvent(w io.Writer, evt SSEEvent) error {
	return sse.WriteEvent(w, evt) //nolint:wrapcheck // pure delegation
}

// SetSSEHeaders sets the response headers required for Server-Sent Events.
func SetSSEHeaders(w http.ResponseWriter) {
	sse.SetHeaders(w)
}

// SSEStream manages a single Server-Sent Events connection.
type SSEStream = sse.Stream

// NewSSEStream creates an SSE stream from an HTTP response writer and request.
func NewSSEStream(w http.ResponseWriter, r *http.Request) *SSEStream {
	return sse.NewStream(w, r)
}

// LastEventIDFromRequest extracts the Last-Event-ID header from an HTTP request.
func LastEventIDFromRequest(r *http.Request) SSEEventID {
	return sse.LastEventIDFromRequest(r)
}
