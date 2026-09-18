package cqrshtmx

import (
	"encoding/json/v2"
	"net/http"

	etag "github.com/larsartmann/go-etag/server"
)

// ProjectionStatusEntry represents the health of a single projection worker.
// It mirrors the fields of projectionhost.WorkerState so that any system
// exposing projection health can serve a consistent JSON shape.
//
// LagMillis is in milliseconds (not time.Duration) because encoding/json/v2
// does not provide a default representation for time.Duration.
type ProjectionStatusEntry struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Checkpoint string `json:"checkpoint"`
	Processed  int64  `json:"processed"`
	Errors     int64  `json:"errors"`
	Restarts   int    `json:"restarts"`
	LagMillis  int64  `json:"lagMs"`
	LastError  string `json:"lastError,omitempty"`
}

// ProjectionStatusProvider reports the current health of registered
// projections. Implementations include usermgmt.Service and any consumer
// type that wraps a *projectionhost.Host.
type ProjectionStatusProvider interface {
	ProjectionStatuses() []ProjectionStatusEntry
}

// ProjectionStatusHandler returns an http.HandlerFunc that serves live
// projection health as JSON. The data is recomputed on every request (it
// changes as projections process events), so the response uses no-cache
// semantics with a per-request FNV-1a ETag for conditional GET support.
//
//	mux.Handle("GET /health/projections",
//	    cqrshtmx.ProjectionStatusHandler(svc))
//
// The provider (e.g. *usermgmt.Service) must implement
// [ProjectionStatusProvider]. If the provider is nil, the handler returns
// 503 with an error body.
//
// Conditional evaluation (If-None-Match) is owned by the go-etag middleware
// wrapping the inner handler: SkipIfPresent adopts the per-request FNV-1a
// ETag the handler sets, buffers the JSON body, and serves 304 on a weak
// match (RFC 7232 §2.3.2).
func ProjectionStatusHandler(provider ProjectionStatusProvider) http.HandlerFunc {
	cfg := etag.DefaultETagConfig()
	cfg.SkipIfPresent = true

	return etag.New(cfg)(serveProjectionStatus(provider)).ServeHTTP
}

func serveProjectionStatus(provider ProjectionStatusProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if provider == nil {
			w.Header().Set("Content-Type", ContentTypeJSON)
			w.WriteHeader(http.StatusServiceUnavailable)
			writeAll(w, []byte(`{"error":"no projection status provider configured"}`))

			return
		}

		statuses := provider.ProjectionStatuses()

		data, err := json.Marshal(statuses)
		if err != nil {
			w.Header().Set("Content-Type", ContentTypeJSON)
			w.WriteHeader(http.StatusInternalServerError)
			writeAll(w, []byte(`{"error":"failed to serialize projection status"}`))

			return
		}

		tag := `"` + hashTag(data) + `"`

		w.Header().Set("Content-Type", ContentTypeJSON)
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("ETag", tag)

		writeAll(w, data)
	}
}
