package cqrshtmx

import (
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// DecodePayloadAuto decodes an event's payload bytes into a typed value,
// dispatching on the event's recorded encoding (JSON, CBOR) so the caller
// does not need to know or pass the codec.
//
// Generic functions cannot be aliased in Go, so this is a thin typed
// re-export of event.DecodePayloadAuto — it exists so SSE subscribers, event
// bus listeners, and AfterDispatchHook consumers of this library can decode
// payloads without importing the event package's codec machinery.
//
// Returns a corruption error if the event's encoding has no built-in codec
// (e.g. "raw", "encrypted") — such payloads must be decoded manually.
//
//nolint:wrapcheck // thin generic re-export of event.DecodePayloadAuto
func DecodePayloadAuto[T any](evt event.Event) (T, error) {
	return event.DecodePayloadAuto[T](evt)
}

// DecodePayload decodes an event's payload bytes into a typed value using
// the provided codec. This is the standard way to deserialize event data in
// event handlers and projectors.
//
// Returns a rejection error if the codec's encoding does not match the
// event's declared encoding. For mixed JSON+CBOR streams, prefer
// [DecodePayloadAuto].
//
//nolint:wrapcheck // thin generic re-export of event.DecodePayload
func DecodePayload[T any](evt event.Event, c codec.Codec) (T, error) {
	return event.DecodePayload[T](evt, c)
}

// DecodePayloads decodes the payloads of a batch of events into a typed
// slice, using the provided codec for every element. It fails on the first
// event whose payload does not decode.
//
//nolint:wrapcheck // thin generic re-export of event.DecodePayloads
func DecodePayloads[T any](events []event.Event, c codec.Codec) ([]T, error) {
	return event.DecodePayloads[T](events, c)
}
