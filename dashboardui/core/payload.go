package core

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"sync"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// PayloadRenderer formats event payloads for display without decoding
// into consumer domain types. Returns pretty-printed bytes.
type PayloadRenderer interface {
	Render(payload []byte, encoding codec.Encoding) ([]byte, error)
}

// TypedPayloadRenderer formats a decoded domain payload. P is the
// consumer's payload struct for one event type; the registry decodes the
// raw bytes into P (JSON or CBOR, matching the event encoding) before
// calling RenderTyped, so implementations never touch raw bytes.
type TypedPayloadRenderer[P any] interface {
	RenderTyped(payload P, encoding codec.Encoding) ([]byte, error)
}

// payloadRenderFunc is the type-erased form stored per event type.
type payloadRenderFunc func(payload []byte, encoding codec.Encoding) ([]byte, error)

// PayloadTypeRegistry maps event types to typed renderers. Reads are
// safe from any goroutine; registration may happen at any time (last
// registration for a type wins).
type PayloadTypeRegistry struct {
	mu      sync.RWMutex
	entries map[event.Type]payloadRenderFunc
}

// NewPayloadTypeRegistry returns an empty registry.
func NewPayloadTypeRegistry() *PayloadTypeRegistry {
	return &PayloadTypeRegistry{mu: sync.RWMutex{}, entries: make(map[event.Type]payloadRenderFunc)}
}

// Render renders payload with the typed renderer registered for evtType.
// ok is false when no renderer is registered OR the typed path failed
// (decode mismatch, renderer error) — the caller falls back to its
// untyped PayloadRenderer. Display must survive schema drift, so the
// registry degrades instead of failing the render.
func (reg *PayloadTypeRegistry) Render(evtType event.Type, payload []byte, encoding codec.Encoding) ([]byte, bool) {
	if reg == nil {
		return nil, false
	}

	reg.mu.RLock()
	fn := reg.entries[evtType]
	reg.mu.RUnlock()

	if fn == nil {
		return nil, false
	}

	out, err := fn(payload, encoding)
	if err != nil || out == nil {
		return nil, false
	}

	return out, true
}

// defaultPayloadTypes backs the package-level RegisterPayloadType.
//
//nolint:gochecknoglobals // package registry is the documented extension point (database/sql-driver pattern)
var defaultPayloadTypes = NewPayloadTypeRegistry()

// DefaultPayloadTypes returns the package registry behind
// RegisterPayloadType. Configs with a nil PayloadTypes resolve to it, so
// startup-time registrations work without any config.
func DefaultPayloadTypes() *PayloadTypeRegistry {
	return defaultPayloadTypes
}

// RegisterPayloadType registers a typed renderer for evtType in the
// package default registry, which RenderPayload consults BEFORE the
// configured PayloadRenderer. Registration is process-global: the
// domain owner registers its payload types once at startup, the same way
// database/sql drivers or image formats register.
func RegisterPayloadType[P any](evtType event.Type, r TypedPayloadRenderer[P]) {
	RegisterPayloadTypeIn(defaultPayloadTypes, evtType, r)
}

// RegisterPayloadTypeIn is RegisterPayloadType for an explicit registry
// (test isolation, multi-tenant hosts with per-tenant rendering).
func RegisterPayloadTypeIn[P any](reg *PayloadTypeRegistry, evtType event.Type, r TypedPayloadRenderer[P]) {
	reg.mu.Lock()
	reg.entries[evtType] = typedRender(r)
	reg.mu.Unlock()
}

// typedRender erases P behind a closure that decodes raw bytes into P
// according to the event encoding, then delegates to r.
func typedRender[P any](r TypedPayloadRenderer[P]) payloadRenderFunc {
	return func(payload []byte, encoding codec.Encoding) ([]byte, error) {
		var p P

		switch encoding {
		case codec.EncodingJSON, "":
			if err := json.Unmarshal(payload, &p); err != nil {
				return nil, errorfamily.WrapCorruption(err,
					"dashboardui.payload.typed_decode_failed", "decode JSON payload into registered type")
			}

		case codec.EncodingCBOR:
			c := codec.CBORCodec{}
			if err := c.Decode(payload, &p); err != nil {
				return nil, errorfamily.WrapCorruption(err,
					"dashboardui.payload.typed_decode_failed", "decode CBOR payload into registered type")
			}

		case codec.EncodingRaw:
			return nil, errorfamily.NewCorruption(
				"dashboardui.payload.typed_encoding_unsupported", "no typed decode for raw payload encoding")

		default:
			return nil, errorfamily.NewCorruption(
				"dashboardui.payload.typed_encoding_unsupported", "no typed decode for payload encoding")
		}

		return r.RenderTyped(p, encoding)
	}
}

// DefaultPayloadRenderer handles JSON and CBOR encodings without
// needing consumer domain types. It decodes to map[string]any and
// re-encodes as pretty-printed JSON.
type DefaultPayloadRenderer struct{}

// Render pretty-prints an event payload for display. JSON and CBOR
// payloads are decoded then re-encoded with 2-space indent; raw
// payloads are returned verbatim.
func (DefaultPayloadRenderer) Render(payload []byte, encoding codec.Encoding) ([]byte, error) {
	if len(payload) == 0 {
		return []byte("{}"), nil
	}

	var raw any

	switch encoding {
	case codec.EncodingJSON, "":
		if err := json.Unmarshal(payload, &raw); err != nil {
			return nil, errorfamily.WrapCorruption(err,
				"dashboardui.payload.json_decode_failed", "decode JSON payload")
		}

	case codec.EncodingCBOR:
		c := codec.CBORCodec{}
		if err := c.Decode(payload, &raw); err != nil {
			return nil, errorfamily.WrapCorruption(err,
				"dashboardui.payload.cbor_decode_failed", "decode CBOR payload")
		}

	case codec.EncodingRaw:
		return payload, nil

	default:
		return payload, nil
	}

	out, err := json.Marshal(raw, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"dashboardui.payload.pretty_print_failed", "pretty-print payload")
	}

	return out, nil
}

// RenderPayload renders an event payload through the typed registry
// first (types registered via RegisterPayloadType win), then the given
// renderer, and finally falls back to the raw payload bytes — it never
// fails. The typed-first order turns silent field-sniffing renderers
// into compile-time registrations without unseating untyped customs for
// unregistered types.
func RenderPayload(r PayloadRenderer, evt event.Event) []byte {
	if out, ok := defaultPayloadTypes.Render(evt.Type(), evt.Payload(), evt.Encoding()); ok {
		return out
	}

	out, err := r.Render(evt.Payload(), evt.Encoding())
	if err != nil || out == nil {
		return evt.Payload()
	}

	return out
}

// PrettyJSON attempts to pretty-print a JSON payload. Falls back to the raw
// bytes if the payload is not valid JSON.
func PrettyJSON(raw []byte) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}

	out, err := json.Marshal(v, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		return string(raw)
	}

	return string(out)
}
