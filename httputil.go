package cqrshtmx

import (
	"bytes"
	"encoding/json/v2"
	"log/slog"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// WriteJSON encodes v as JSON and writes it to w with the given HTTP status code
// and Content-Type: application/json header. Returns any encoding error.
// Buffers the JSON before writing headers so a failed encode doesn't commit
// a success status code.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	var buf bytes.Buffer
	if err := json.MarshalWrite(&buf, v); err != nil {
		return errorfamily.Wrapf(
			err,
			event.Infrastructure,
			"cqrshtmx.http.json_encode_failed",
			"encode JSON response (status %d)",
			status,
		)
	}

	buf.WriteByte('\n')
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("cqrshtmx: response write failed", "error", err)
	}

	return nil
}

// marshalJSONOrFallback serialises v as JSON and returns it as a string. If
// marshalling fails (which should be impossible for the well-formed types that
// call this), it returns fallback verbatim — letting each caller ship a
// domain-correct minimum-valid JSON document instead of a generic error.
// Used by CommandAck.ackJSON and StructuredError.JSON so both transports
// (SSE/WS ack frames and RFC 7807 problem details) degrade gracefully.
func marshalJSONOrFallback(v any, fallback string) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fallback
	}

	return string(b)
}
