package dashboardui

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"runtime"
)

// healthzHandler is a liveness probe. Always returns 200 if the process
// is running and the dashboard has not been closed.
func (d *Dashboard) healthzHandler(w http.ResponseWriter, _ *http.Request) {
	select {
	case <-d.done:
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{jsonKeyStatus: "shutting_down"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{jsonKeyStatus: "ok"})
	}
}

// readyzHandler is a readiness probe. Returns 200 when the dashboard has
// at least one data source configured and has not been closed.
func (d *Dashboard) readyzHandler(w http.ResponseWriter, _ *http.Request) {
	select {
	case <-d.done:
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{jsonKeyStatus: "shutting_down", jsonKeyReady: false})

		return
	default:
	}

	if !d.caps.HasEventRead() && !d.caps.EventSource {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			jsonKeyStatus: "no_data_source",
			jsonKeyReady:  false,
		})

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{jsonKeyStatus: "ready", jsonKeyReady: true})
}

// versionzHandler returns build and configuration metadata.
func (d *Dashboard) versionzHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, versionInfo{
		Module:       modulePath,
		GoVersion:    runtime.Version(),
		Capabilities: d.caps,
		ReadOnly:     d.config.ReadOnly,
		BasePath:     d.config.BasePath,
		Title:        d.config.Title,
	})
}

type versionInfo struct {
	Module       string       `json:"module"`
	GoVersion    string       `json:"goVersion"`
	Capabilities Capabilities `json:"capabilities"`
	ReadOnly     bool         `json:"readOnly"`
	BasePath     string       `json:"basePath"`
	Title        string       `json:"title"`
}

const modulePath = "github.com/larsartmann/cqrs-htmx/dashboardui/v4"

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	body, err := json.Marshal(v)
	if err != nil {
		_, _ = fmt.Fprint(w, `{"error":"marshal_failed"}`)

		return
	}

	_, _ = w.Write(body)
}
