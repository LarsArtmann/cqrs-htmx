package hx

import (
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// WriteAll writes data to w as the terminal response body (after headers and
// status have been committed via WriteHeader). Write failures at this point
// indicate a broken client connection — there is nothing the handler can do
// except log the failure for observability.
//
// This replaces the anti-pattern of silently discarding write errors with
// `_, _ = w.Write(data)`, which makes connection failures invisible.
func WriteAll(w io.Writer, data []byte) {
	if _, err := w.Write(data); err != nil {
		slog.Debug("cqrshtmx: response write failed", "error", err)
	}
}

// WriteAllString writes s to w, using io.StringWriter when available to
// avoid a []byte(s) allocation. Like WriteAll, logs failures at debug level.
func WriteAllString(w io.Writer, s string) {
	if sw, ok := w.(io.StringWriter); ok {
		if _, err := sw.WriteString(s); err != nil {
			slog.Debug("cqrshtmx: response write failed", "error", err)
		}

		return
	}

	WriteAll(w, []byte(s))
}

// MarshalJSONForResponse marshals v as JSON and writes the result to w.
// On marshal failure, writes a fallback 500 JSON error response.
// This is used by terminal handlers where the caller has already committed
// the status code and needs a guaranteed body.
func MarshalJSONForResponse(w http.ResponseWriter, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		WriteAll(w, []byte(`{"error":"failed to serialize response"}`))

		return
	}

	WriteAll(w, data)
}

// MarshalJSONOrFallback serialises v as JSON and returns it as a string. If
// marshalling fails (which should be impossible for the well-formed types that
// call this), it returns fallback verbatim — letting each caller ship a
// domain-correct minimum-valid JSON document instead of a generic error.
// Both transports (SSE ack frames and RFC 9457 problem details) degrade
// gracefully through it.
func MarshalJSONOrFallback(v any, fallback string) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fallback
	}

	return string(b)
}

// HashTag derives a short, stable cache tag from bytes using the FNV-1a 64-bit
// hash. It is not cryptographically significant — it only needs to change when
// the content changes, so stale assets/specs are refetched.
//
// Implemented inline (rather than hash/fnv) to avoid the hash.Hash.Write
// interface, which requires discarding an error return value that can never
// occur for a pure in-memory hash computation.
func HashTag(data []byte) string {
	const (
		offsetBasis uint64 = 14695981039346656037
		prime       uint64 = 1099511628211
	)

	hash := offsetBasis
	for _, b := range data {
		hash ^= uint64(b)
		hash *= prime
	}

	return strconv.FormatUint(hash, 16)
}
