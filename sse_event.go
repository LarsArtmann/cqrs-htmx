package cqrshtmx

import (
	"io"

	"github.com/larsartmann/go-cqrs-lite/transport/http/v3"
)

// Delegated to go-cqrs-lite/transport/http/v3 (tagged v3.3.1).
// These aliases preserve backward compatibility with existing cqrs-htmx consumers.
// New code should import the upstream module directly.

// SSEEventID is a branded identifier for SSE event identifiers.
type SSEEventID = http.SSEEventID

// SSEEvent represents a single Server-Sent Event.
type SSEEvent = http.SSEEvent

// NewSSEEventID constructs an SSEEventID from a string without validation.
var NewSSEEventID = http.NewSSEEventID

// ParseSSEEventID converts a string to an SSEEventID, rejecting newlines.
var ParseSSEEventID = http.ParseSSEEventID

// MustParseSSEEventID is the panicking variant for tests and constants.
var MustParseSSEEventID = http.MustParseSSEEventID

// WriteSSEEvent writes a single SSE event in the standard wire format.
func WriteSSEEvent(w io.Writer, evt SSEEvent) error {
	return http.WriteSSEEvent(w, evt)
}
