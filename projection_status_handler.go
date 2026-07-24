package cqrshtmx

import (
	"encoding/json/v2"
	"net/http"
	"time"
)

// ProjectionStatusEntry represents the health of a single projection worker.
// It mirrors the fields of projectionhost.WorkerState so that any system
// exposing projection health can serve a consistent JSON shape.
type ProjectionStatusEntry struct {
	Name       string        `json:"name"`
	Status     string        `json:"status"`
	Checkpoint string        `json:"checkpoint"`
	Processed  int64         `json:"processed"`
	Errors     int64         `json:"errors"`
	Restarts   int           `json:"restarts"`
	Lag        time.Duration `json:"lag"`
	LastError  string        `json:"lastError,omitempty"`
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
func ProjectionStatusHandler(provider ProjectionStatusProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if provider == nil {
			w.Header().Set("Content-Type", ContentTypeJSON)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"no projection status provider configured"}`))

			return
		}

		statuses := provider.ProjectionStatuses()

		data, err := json.Marshal(statuses)
		if err != nil {
			w.Header().Set("Content-Type", ContentTypeJSON)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"failed to serialize projection status"}`))

			return
		}

		etag := `"` + hashTag(data) + `"`

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("ETag", etag)

		if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)

			return
		}

		_, _ = w.Write(data)
	}
}
