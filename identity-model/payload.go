package identitymodel

import (
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// UnmarshalPayload decodes an event's payload into a typed value using the
// codec that matches the event's declared encoding (JSON or CBOR).
//
// Upcasters run first: they may transform legacy payload shapes before decoding.
// The codec is resolved per-event via codec.ForEncoding(evt.Encoding()), so
// mixed JSON+CBOR event streams decode correctly.
func UnmarshalPayload[T any](evt event.Event) (T, error) {
	raw, err := applyUpcasters(evt.Type(), evt.Payload())
	if err != nil {
		return *new(T), errorfamily.WrapCorruption(err, "identitymodel.payload.upcast_failed", "upcast payload")
	}

	c, err := codec.ForEncoding(evt.Encoding())
	if err != nil {
		return *new(T), errorfamily.WrapCorruption(err,
			"identitymodel.payload_decode_failed",
			"resolve codec for encoding "+string(evt.Encoding())+
				" (event "+string(evt.Type())+")")
	}

	var target T
	if err := c.Decode(raw, &target); err != nil {
		return target, errorfamily.WrapCorruption(err,
			"identitymodel.payload_decode_failed",
			"decode payload for event "+string(evt.Type()))
	}

	return target, nil
}
