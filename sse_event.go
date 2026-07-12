package cqrshtmx

import (
	"io"

	"github.com/larsartmann/go-cqrs-lite/transport/http/v4"
)

// Delegated to go-cqrs-lite/transport/http/v4.
// These aliases preserve backward compatibility with existing cqrs-htmx consumers.
// New code should import the upstream module directly.

// Common SSE event names. Consumers are free to use custom event names —
// these constants just reduce magic strings for the most common patterns.
const (
	// SSEEventConnected is the conventional event name for the initial
	// connection-acknowledgement event sent when an SSE stream opens.
	SSEEventConnected = "connected"
	// SSEEventHeartbeat is the conventional event name for heartbeat/ping
	// events. Note: stream.Heartbeat() sends comment-frame pings (not named
	// events); this constant is for consumers who prefer named heartbeats.
	SSEEventHeartbeat = "heartbeat"
)

// SSEEventID is a branded identifier for SSE event identifiers.
type SSEEventID = http.SSEEventID

// SSEEvent represents a single Server-Sent Event.
type SSEEvent = http.SSEEvent

// NewSSEEventID constructs an SSEEventID from a string without validation.
func NewSSEEventID(s string) SSEEventID { return http.NewSSEEventID(s) }

// ParseSSEEventID converts a string to an SSEEventID, rejecting newlines.
func ParseSSEEventID(s string) (SSEEventID, error) {
	return http.ParseSSEEventID(s) //nolint:wrapcheck // pure delegation to upstream
}

// MustParseSSEEventID is the panicking variant for tests and constants.
func MustParseSSEEventID(s string) SSEEventID { return http.MustParseSSEEventID(s) }

// WriteSSEEvent writes a single SSE event in the standard wire format.
func WriteSSEEvent(w io.Writer, evt SSEEvent) error {
	return http.WriteSSEEvent(w, evt) //nolint:wrapcheck // pure delegation to upstream
}
