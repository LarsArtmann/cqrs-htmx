package dashboardui

import (
	"net/http"
	"time"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4/core"
)

// engineStatsStaleThreshold flags engine RTT snapshots whose last sample is
// older than this — a quiet engine is not a dead one, but stale numbers next
// to live ones mislead.
const engineStatsStaleThreshold = 5 * time.Minute

// telemetryData is the aggregated view model for the telemetry page. Sections
// render only when their provider is configured; a provider error surfaces
// inline in its section (read-only panel: one failing provider must not
// blank the others).
type telemetryData struct {
	HasTopology  bool
	Topology     core.TopologyView
	TopologyErr  error
	HasPlacement bool
	Placements   []core.PlacementView
	PlacementErr error
	Stats        []core.EngineStatsView
}

// telemetryHandler renders the read-only telemetry page: system topology,
// query placements, and engine latency stats — one section per configured
// provider.
func (d *Dashboard) telemetryHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Telemetry", "/telemetry", r)

	var data telemetryData

	if d.config.Topology != nil {
		data.HasTopology = true
		data.Topology, data.TopologyErr = d.config.Topology(r.Context())
	}

	if d.config.Placements != nil {
		data.HasPlacement = true
		data.Placements, data.PlacementErr = d.config.Placements(r.Context())
	}

	if d.config.EngineStats != nil {
		data.Stats = d.config.EngineStats(r.Context())
	}

	renderPage(w, r, telemetryPage(p, data))
}
