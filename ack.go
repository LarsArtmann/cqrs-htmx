package cqrshtmx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// CommandIDHeader is the request/response header that carries a client-generated
// command identifier. When present on a dispatch request, the server echoes it
// back in ACK broadcasts so the client can match the ACK to its pending item.
//
// Clients generate this ID (typically a ULID or UUID) before sending the
// request and use it to track the optimistic state in the UI.
const CommandIDHeader = "X-Command-Id"

// AckStatus indicates the outcome of a command dispatch, for client-side
// sync-state transitions.
type AckStatus string

const (
	// AckConfirmed means the server accepted and processed the command.
	AckConfirmed AckStatus = "confirmed"

	// AckRejected means the server rejected the command (validation, auth, etc.).
	AckRejected AckStatus = "rejected"
)

// CommandAck is the structured acknowledgment payload broadcast over SSE/WS
// after a command dispatch completes. It carries enough information for the
// client to transition its optimistic UI state.
//
// The JSON shape is designed to be consumed directly by client-side JavaScript
// via EventSource or WebSocket onmessage handlers.
type CommandAck struct {
	// CommandID matches the X-Command-Id header sent by the client. Empty if
	// the client did not send one (in which case no ACK is broadcast).
	CommandID string `json:"commandId"`

	// Status is "confirmed" or "rejected".
	Status AckStatus `json:"status"`

	// Error contains the error message when Status is "rejected". Empty otherwise.
	Error string `json:"error,omitempty"`

	// EventType is the SSE event name the client listens for (e.g., "sync:ack").
	// Defaults to "sync:ack" in BroadcastOnAck.
	EventType string `json:"-"`
}

// JSON serializes the ack to a JSON string suitable for SSEEvent.Data.
// Falls back to a minimal JSON object on marshal failure.
func (a CommandAck) JSON() string {
	data, err := json.Marshal(a)
	if err != nil {
		return `{"commandId":"","status":"rejected","error":"ack marshal failed"}`
	}

	return string(data)
}

// CommandIDFromRequest extracts the client-generated command ID from the
// X-Command-Id header. Returns empty string if not present (ACK is opt-in).
func CommandIDFromRequest(r *http.Request) string {
	return r.Header.Get(CommandIDHeader)
}

// defaultAckEventName is the SSE event name used for ACK broadcasts.
const defaultAckEventName = "sync:ack"

// BroadcastOnAck returns an [AfterDispatchHook] that broadcasts a [CommandAck]
// over SSE when the request carries an [X-Command-Id] header. If the header is
// absent, no ACK is broadcast (opt-in).
//
// On success (err == nil): broadcasts {commandId, status: "confirmed"}.
// On failure (err != nil): broadcasts {commandId, status: "rejected", error: msg}.
//
// Use this with [App] config:
//
//	broadcaster := NewBroadcaster()
//	app := cqrshtmx.New(cfg, cqrshtmx.WithAfterDispatch(
//	    broadcaster.BroadcastOnAck(),
//	))
func (b *Broadcaster) BroadcastOnAck() AfterDispatchHook {
	return b.broadcastOnAckFunc(defaultAckEventName, defaultAckMapper)
}

// BroadcastOnAckFunc returns an [AfterDispatchHook] with a custom ACK mapper.
// The mapper receives the request, the dispatch error (nil on success), and the
// extracted command ID, and returns the SSE event to broadcast. If the mapper
// returns an SSEEvent with an empty Data field, no broadcast is sent.
//
// This allows consumers to customize the ACK payload or event name.
func (b *Broadcaster) BroadcastOnAckFunc(
	eventFunc func(r *http.Request, err error, commandID string) SSEEvent,
) AfterDispatchHook {
	return b.broadcastOnAckFunc("", func(r *http.Request, err error) SSEEvent {
		cmdID := CommandIDFromRequest(r)
		return eventFunc(r, err, cmdID)
	})
}

// broadcastOnAckFunc is the shared builder for ACK hooks. It wraps the standard
// broadcastOnSuccessHook/broadcastOnErrorHook pattern but adds command-ID
// extraction and opt-in behavior.
func (b *Broadcaster) broadcastOnAckFunc(
	eventName string,
	mapper func(r *http.Request, err error) SSEEvent,
) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		// Opt-in: only broadcast ACK if the client sent a command ID
		if CommandIDFromRequest(r) == "" {
			return
		}

		evt := mapper(r, err)
		if evt.Data == "" {
			return
		}

		if eventName != "" && evt.Event == "" {
			evt.Event = eventName
		}

		b.Broadcast(evt)
	}
}

// defaultAckMapper builds the standard ACK SSEEvent from a request and dispatch error.
func defaultAckMapper(r *http.Request, err error) SSEEvent {
	cmdID := CommandIDFromRequest(r)

	ack := CommandAck{
		CommandID: cmdID,
		EventType: defaultAckEventName,
	}

	if err != nil {
		ack.Status = AckRejected
		ack.Error = err.Error()
	} else {
		ack.Status = AckConfirmed
	}

	return SSEEvent{
		Event: defaultAckEventName,
		Data:  ack.JSON(),
	}
}

// Compile-time interface check.
var _ json.Marshaler = (*CommandAck)(nil)

// MarshalJSON implements json.Marshaler for CommandAck.
// Excludes the EventType field (it's transport metadata, not payload).
func (a CommandAck) MarshalJSON() ([]byte, error) {
	type alias struct {
		CommandID string `json:"commandId"`
		Status    string `json:"status"`
		Error     string `json:"error,omitempty"`
	}

	return json.Marshal(alias{
		CommandID: a.CommandID,
		Status:    string(a.Status),
		Error:     a.Error,
	})
}

// WrapAckError wraps a dispatch error for ACK purposes, preserving the
// error family. This is a convenience for consumers who want to classify
// ACK errors using the existing error-family system.
func WrapAckError(err error, code, msg string) error {
	return event.Wrapf(err, event.Classify(err), code, "%s", msg)
}
