package core

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// PayloadRenderer formats event payloads for display without decoding
// into consumer domain types. Returns pretty-printed bytes.
type PayloadRenderer interface {
	Render(payload []byte, encoding codec.Encoding) ([]byte, error)
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

// RenderPayload is a convenience that never fails — it falls back to
// the raw payload bytes on error.
func RenderPayload(r PayloadRenderer, evt event.Event) []byte {
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
