package cqrshtmx

import (
	"bytes"
	"encoding/json/v2"
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
	_, _ = buf.WriteTo(w)
	return nil
}
