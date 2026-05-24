package cqrshtmx

import (
	"encoding/json"
	"fmt"
	"net/http"

	httputil "github.com/larsartmann/httputil"
)

// WriteJSON encodes v as JSON and writes it to w with the given HTTP status code
// and Content-Type: application/json header. Returns any encoding error.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode JSON response: %w", err)
	}
	return nil
}

// ClientIP delegates to httputil.ClientIP for client IP extraction.
func ClientIP(r *http.Request) string {
	return httputil.ClientIP(r)
}
