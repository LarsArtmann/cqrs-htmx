package cqrshtmx

import (
	"encoding/json/v2"
	"net/http"
	"sync"
)

// ReadinessCheck is a single health check function. Return nil if healthy,
// or an error describing the failure. The error message is included in the
// 503 response body so operators can identify which check failed.
type ReadinessCheck func() error

// readinessResult is the JSON response body for ReadinessHandler.
type readinessResult struct {
	Status string                     `json:"status"`
	Checks map[string]readinessDetail `json:"checks,omitempty"`
}

type readinessDetail struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ReadinessHandler returns an http.HandlerFunc that runs all checks in
// parallel and returns 200 OK when every check passes, or 503 Service
// Unavailable when any check fails. Each check name keys into the response
// body so operators can identify failing subsystems.
//
//	mux.Handle("/ready", cqrshtmx.ReadinessHandler(
//	    cqrshtmx.NewNamedCheck("event-store", func() error { return db.Ping() }),
//	    cqrshtmx.NewNamedCheck("projections", func() error {
//	        for _, ws := range host.Status() {
//	            if ws.Status == "failed" { return ws.Err() }
//	        }
//
//	        return nil
//	    }),
//	))
func ReadinessHandler(checks ...NamedCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if len(checks) == 0 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))

			return
		}

		result := readinessResult{
			Status: "ok",
			Checks: make(map[string]readinessDetail, len(checks)),
		}

		var mu sync.Mutex
		var wg sync.WaitGroup

		allOK := true

		for _, check := range checks {
			wg.Add(1)

			go func(namedCheck NamedCheck) {
				defer wg.Done()

				detail := readinessDetail{Status: "ok"} //nolint:exhaustruct // Error is omitempty; zero value is correct when check passes

				if err := namedCheck.Check(); err != nil {
					detail.Status = "fail"
					detail.Error = err.Error()

					mu.Lock()
					allOK = false
					mu.Unlock()
				}

				mu.Lock()
				result.Checks[namedCheck.Name] = detail
				mu.Unlock()
			}(check)
		}

		wg.Wait()

		if !allOK {
			result.Status = "degraded"

			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		body, _ := json.Marshal(result)
		_, _ = w.Write(body)
	}
}

// NamedCheck pairs a human-readable name with a ReadinessCheck.
type NamedCheck struct {
	Name  string
	Check ReadinessCheck
}

// NewNamedCheck creates a NamedCheck from a name and check function.
// Convenience wrapper for ReadinessHandler callers.
func NewNamedCheck(name string, check ReadinessCheck) NamedCheck {
	return NamedCheck{Name: name, Check: check}
}

// DebugHandler returns an http.HandlerFunc that serializes the provided
// info map as JSON. Use for ad-hoc debug endpoints that expose system
// metadata (version, config summary, uptime). The info is captured at
// handler construction time; for live data, use a handler closure.
//
//	mux.Handle("/debug", cqrshtmx.DebugHandler(map[string]any{
//	    "version":   "1.0.0",
//	    "goVersion": runtime.Version(),
//	    "module":    "myapp",
//	}))
func DebugHandler(info map[string]any) http.HandlerFunc {
	body, err := json.Marshal(info)
	if err != nil {
		body = []byte(`{"error":"failed to marshal debug info"}`)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")

		_, _ = w.Write(body)
	}
}
