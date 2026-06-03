package cqrshtmx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	httputil "github.com/larsartmann/httputil"
)

// WriteJSON encodes v as JSON and writes it to w with the given HTTP status code
// and Content-Type: application/json header. Returns any encoding error.
// Buffers the JSON before writing headers so a failed encode doesn't commit
// a success status code.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return fmt.Errorf("encode JSON response (status %d): %w", status, err)
	}
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
	return nil
}

// ClientIP delegates to httputil.ClientIP for client IP extraction.
//
// Deprecated: Import github.com/larsartmann/httputil directly and use
// httputil.ClientIP(r) instead. This re-export will be removed in a future
// version.
func ClientIP(r *http.Request) string {
	return httputil.ClientIP(r)
}
