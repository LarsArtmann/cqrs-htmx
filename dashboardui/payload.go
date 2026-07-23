package dashboardui

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// PayloadRenderer formats event payloads for display without decoding
// into consumer domain types. Returns pretty-printed bytes.
type PayloadRenderer interface {
	// Render returns a human-readable representation of the payload.
	// For JSON-encoded payloads: pretty-printed JSON.
	// For CBOR-encoded: decoded to JSON then pretty-printed.
	Render(payload []byte, encoding codec.Encoding) ([]byte, error)
}

// DefaultPayloadRenderer handles JSON and CBOR encodings without
// needing consumer domain types. It decodes to map[string]any and
// re-encodes as pretty-printed JSON.
type DefaultPayloadRenderer struct{}

func (DefaultPayloadRenderer) Render(payload []byte, encoding codec.Encoding) ([]byte, error) {
	if len(payload) == 0 {
		return []byte("{}"), nil
	}

	var raw any

	switch encoding {
	case codec.EncodingJSON, "":
		if err := json.Unmarshal(payload, &raw); err != nil {
			return nil, fmt.Errorf("decode JSON payload: %w", err)
		}

	case codec.EncodingCBOR:
		c := codec.CBORCodec{}
		if err := c.Decode(payload, &raw); err != nil {
			return nil, fmt.Errorf("decode CBOR payload: %w", err)
		}

	default:
		return payload, nil
	}

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("pretty-print payload: %w", err)
	}

	return out, nil
}

// renderPayload is a convenience that never fails — it falls back to
// the raw payload bytes on error.
func renderPayload(r PayloadRenderer, evt event.Event) []byte {
	out, err := r.Render(evt.Payload(), evt.Encoding())
	if err != nil || out == nil {
		return evt.Payload()
	}

	return out
}

func csrfToken(r *http.Request) string {
	return r.FormValue("_csrf")
}

func csrfMeta(r *http.Request) string {
	return ""
}
