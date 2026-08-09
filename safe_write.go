package cqrshtmx

import (
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
)

// writeAll writes data to w as the terminal response body (after headers and
// status have been committed via WriteHeader). Write failures at this point
// indicate a broken client connection — there is nothing the handler can do
// except log the failure for observability.
//
// This replaces the anti-pattern of silently discarding write errors with
// `_, _ = w.Write(data)`, which makes connection failures invisible.
func writeAll(w io.Writer, data []byte) {
	if _, err := w.Write(data); err != nil {
		slog.Debug("cqrshtmx: response write failed", "error", err)
	}
}

// writeAllString writes s to w, using io.StringWriter when available to
// avoid a []byte(s) allocation. Like writeAll, logs failures at debug level.
func writeAllString(w io.Writer, s string) {
	if sw, ok := w.(io.StringWriter); ok {
		if _, err := sw.WriteString(s); err != nil {
			slog.Debug("cqrshtmx: response write failed", "error", err)
		}

		return
	}

	writeAll(w, []byte(s))
}

// marshalJSONForResponse marshals v as JSON and writes the result to w.
// On marshal failure, writes a fallback 500 JSON error response.
// This is used by terminal handlers where the caller has already committed
// the status code and needs a guaranteed body.
func marshalJSONForResponse(w http.ResponseWriter, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		writeAll(w, []byte(`{"error":"failed to serialize response"}`))

		return
	}

	writeAll(w, data)
}
