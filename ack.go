package cqrshtmx

import (
	"context"
	"encoding/json"
	"net/http"
)

// CommandIDHeader is the request header that carries a client-generated command
// identifier. When present on a dispatch request, the server echoes it back in
// ACK broadcasts so the client can match the ACK to its pending UI item.
//
// ACK is opt-in: if this header is absent, no ACK is broadcast.
const CommandIDHeader = "X-Command-Id"

// AckStatus indicates the outcome of a command dispatch.
type AckStatus string

const (
	AckConfirmed AckStatus = "confirmed" // server accepted and processed the command
	AckRejected  AckStatus = "rejected"  // server rejected (validation, auth, etc.)
)

// CommandAck is the structured acknowledgment payload broadcast over SSE/WS
// after a command dispatch completes. The JSON shape is designed to be consumed
// directly by client-side JavaScript:
//
//	{"commandId":"abc123","status":"confirmed"}
//	{"commandId":"abc123","status":"rejected","error":"email already exists"}
type CommandAck struct {
	CommandID string    `json:"commandId"`
	Status    AckStatus `json:"status"`
	Error     string    `json:"error,omitempty"` // populated when Status == AckRejected
}

// MarshalJSON serializes the ack. AckStatus is marshaled as its string value.
func (a CommandAck) MarshalJSON() ([]byte, error) {
	type wire struct {
		CommandID string `json:"commandId"`
		Status    string `json:"status"`
		Error     string `json:"error,omitempty"`
	}
	return json.Marshal(wire{
		CommandID: a.CommandID,
		Status:    string(a.Status),
		Error:     a.Error,
	})
}

// ackJSON serializes to a string for SSEEvent.Data, with a safe fallback.
func (a CommandAck) ackJSON() string {
	data, err := json.Marshal(a)
	if err != nil {
		return `{"commandId":"","status":"rejected","error":"ack marshal failed"}`
	}
	return string(data)
}

// CommandIDFromRequest extracts the client-generated command ID from the
// X-Command-Id header. Returns empty string if absent (ACK is opt-in).
func CommandIDFromRequest(r *http.Request) string {
	return r.Header.Get(CommandIDHeader)
}

// defaultAckEventName is the SSE/WS event name used for ACK broadcasts.
const defaultAckEventName = "sync:ack"

// newAck builds a CommandAck from a dispatch result.
func newAck(commandID string, err error) CommandAck {
	ack := CommandAck{CommandID: commandID}
	if err != nil {
		ack.Status = AckRejected
		ack.Error = err.Error()
	} else {
		ack.Status = AckConfirmed
	}
	return ack
}

// ackToSSEEvent converts a CommandAck to an SSEEvent with the default event name.
func ackToSSEEvent(ack CommandAck) SSEEvent {
	return SSEEvent{Event: defaultAckEventName, Data: ack.ackJSON()}
}

// ackToWSMessage converts a CommandAck to a WS message string.
func ackToWSMessage(ack CommandAck) string {
	return ack.ackJSON()
}

// BroadcastOnAck returns an [AfterDispatchHook] that broadcasts a [CommandAck]
// over SSE when the request carries an [X-Command-Id] header. Opt-in: if the
// header is absent, no ACK is broadcast.
//
// On success: broadcasts {commandId, status: "confirmed"}.
// On failure: broadcasts {commandId, status: "rejected", error: msg}.
func (b *Broadcaster) BroadcastOnAck() AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		cmdID := CommandIDFromRequest(r)
		if cmdID == "" {
			return
		}
		b.Broadcast(ackToSSEEvent(newAck(cmdID, err)))
	}
}

// BroadcastOnAckFunc returns an [AfterDispatchHook] with a custom ACK mapper.
// The mapper receives the request, the dispatch error (nil on success), and the
// extracted command ID, and returns the SSE event to broadcast. If the mapper
// returns an SSEEvent with empty Data, no broadcast is sent.
func (b *Broadcaster) BroadcastOnAckFunc(
	fn func(r *http.Request, err error, commandID string) SSEEvent,
) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		cmdID := CommandIDFromRequest(r)
		if cmdID == "" {
			return
		}
		evt := fn(r, err, cmdID)
		if evt.Data == "" {
			return
		}
		b.Broadcast(evt)
	}
}

// BroadcastOnAckWS returns an [AfterDispatchHook] that broadcasts a [CommandAck]
// over WebSocket (via [WSBroadcaster]) when the request carries an
// [X-Command-Id] header. Opt-in, same semantics as [Broadcaster.BroadcastOnAck].
func (b *WSBroadcaster) BroadcastOnAckWS() AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		cmdID := CommandIDFromRequest(r)
		if cmdID == "" {
			return
		}
		b.Broadcast(ackToWSMessage(newAck(cmdID, err)))
	}
}

// BroadcastOnAckWSFunc returns an [AfterDispatchHook] with a custom ACK mapper
// for WebSocket transport.
func (b *WSBroadcaster) BroadcastOnAckWSFunc(
	fn func(r *http.Request, err error, commandID string) string,
) AfterDispatchHook {
	return func(_ context.Context, r *http.Request, err error) {
		cmdID := CommandIDFromRequest(r)
		if cmdID == "" {
			return
		}
		msg := fn(r, err, cmdID)
		if msg == "" {
			return
		}
		b.Broadcast(msg)
	}
}
