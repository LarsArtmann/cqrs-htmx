package dashboardui

import (
	"encoding/json/v2"
	"net/http"
	"runtime"
	"runtime/debug"
)

// healthzHandler is a liveness probe. Always returns 200 if the process
// is running and the dashboard has not been closed. With an EngineHealths
// provider configured, per-engine health rides along — liveness itself
// never depends on it (a sick engine is not a dead process).
func (d *Dashboard) healthzHandler(w http.ResponseWriter, r *http.Request) {
	select {
	case <-d.done:
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{jsonKeyStatus: "shutting_down"})
	default:
		payload := map[string]any{jsonKeyStatus: "ok"}
		if engines := d.engineHealthPayload(r); engines != nil {
			payload[jsonKeyEngines] = engines
		}

		writeJSON(w, http.StatusOK, payload)
	}
}

// readyzHandler is a readiness probe. Returns 200 when the dashboard has
// at least one data source configured and has not been closed.
func (d *Dashboard) readyzHandler(w http.ResponseWriter, r *http.Request) {
	select {
	case <-d.done:
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]any{jsonKeyStatus: "shutting_down", jsonKeyReady: false},
		)

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

	// Engine health gates readiness: a quarantined or erroring engine means
	// the system cannot serve queries yet, so the orchestrator must keep this
	// pod out of rotation.
	engines := d.engineHealthPayload(r)
	for _, engine := range engines {
		if !engine.Healthy {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				jsonKeyStatus:  "engine_unhealthy",
				jsonKeyReady:   false,
				jsonKeyEngines: engines,
			})

			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{jsonKeyStatus: "ready", jsonKeyReady: true})
}

// engineHealthPayload collects the provider's view; nil when no provider is
// configured (the v4 zero-config default — probes stay as they were).
func (d *Dashboard) engineHealthPayload(r *http.Request) []engineHealthView {
	if d.config.EngineHealths == nil {
		return nil
	}

	return d.config.EngineHealths(r.Context())
}

// versionzHandler returns build and configuration metadata. The module and
// version fields identify the dashboardui module (fork-honest via reflection);
// the vcs* fields identify the binary build the dashboard is served from.
func (d *Dashboard) versionzHandler(w http.ResponseWriter, _ *http.Request) {
	module := ownModulePath()

	var version string

	var stamp vcsBuildInfo

	if build, ok := debug.ReadBuildInfo(); ok {
		version = resolveModuleVersion(build, module)
		stamp = resolveVCS(build.Settings)
	}

	writeJSON(w, http.StatusOK, versionInfo{
		Module:       module,
		Version:      version,
		GoVersion:    runtime.Version(),
		VCSRevision:  stamp.Revision,
		VCSTime:      stamp.Time,
		VCSModified:  stamp.Modified,
		Capabilities: d.caps,
		ReadOnly:     d.config.ReadOnly,
		BasePath:     d.config.BasePath,
		Title:        d.config.Title,
	})
}

type versionInfo struct {
	Module       string       `json:"module"`
	Version      string       `json:"version,omitzero"`
	GoVersion    string       `json:"goVersion"`
	VCSRevision  string       `json:"vcsRevision,omitzero"`
	VCSTime      string       `json:"vcsTime,omitzero"`
	VCSModified  string       `json:"vcsModified,omitzero"`
	Capabilities Capabilities `json:"capabilities"`
	ReadOnly     bool         `json:"readOnly"`
	BasePath     string       `json:"basePath"`
	Title        string       `json:"title"`
}

const modulePath = "github.com/larsartmann/cqrs-htmx/dashboardui/v4"

func writeJSON(w http.ResponseWriter, status int, v any) {
	// Marshal BEFORE writing the status: a failed marshal must not leave a
	// 2xx/committed status on the wire with an error body after it.
	body, err := json.Marshal(v)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"marshal_failed"}`))

		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
