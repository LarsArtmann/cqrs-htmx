package dashboardui

import (
	"fmt"
	"time"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4/core"
	"github.com/larsartmann/templ-components/display"
)

// Core telemetry view aliases — the templ layer references the unqualified
// names, mirroring the Capabilities alias pattern.
type (
	topologyView     = core.TopologyView
	placementView    = core.PlacementView
	engineStatsView  = core.EngineStatsView
	engineHealthView = core.EngineHealthView
)

// instanceHealthBadgeType maps a topology instance's health string onto a
// badge tone. Unknown strings render neutral — telemetry is read-only and
// must not invent severity.
func instanceHealthBadgeType(health string) display.BadgeType {
	switch health {
	case "healthy", "ok":
		return display.BadgeSuccess
	case "degraded", "warn", "warning":
		return display.BadgeWarning
	case "unhealthy", "error", "down":
		return display.BadgeError
	default:
		return display.BadgeNeutral
	}
}

func engineStatCardID(stat core.EngineStatsView) string {
	return "stat-engine-" + stat.Name
}

// engineRTTValue renders the card value: the EWMA when live samples exist,
// an explicit marker otherwise.
func engineRTTValue(stat core.EngineStatsView) string {
	if !stat.HasRTT {
		return "no tracker"
	}

	if stat.Samples == 0 {
		return "no samples"
	}

	return stat.EWMA.Round(time.Microsecond).String()
}

// engineStatTone colors the card by freshness: live data green, stale
// yellow, no tracker neutral.
func engineStatTone(stat core.EngineStatsView) display.StatTone {
	switch {
	case !stat.HasRTT:
		return display.StatToneBlue
	case stat.Stale(engineStatsStaleThreshold):
		return display.StatToneYellow
	default:
		return display.StatToneGreen
	}
}

// formatEstLatencyMs renders the planner's latency estimate in a stable,
// compact form.
func formatEstLatencyMs(latencyMs float64) string {
	switch {
	case latencyMs <= 0:
		return "—"
	case latencyMs < 1:
		return fmt.Sprintf("%.3fms", latencyMs)
	case latencyMs < 100:
		return fmt.Sprintf("%.1fms", latencyMs)
	default:
		return fmt.Sprintf("%.0fms", latencyMs)
	}
}

// formatLastSample renders the last RTT sample time; zero means "never".
func formatLastSample(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	return t.Format("2006-01-02 15:04:05")
}

func engineFreshnessLabel(stat core.EngineStatsView) string {
	switch {
	case !stat.HasRTT:
		return "no tracker"
	case stat.Stale(engineStatsStaleThreshold):
		return "stale"
	default:
		return "fresh"
	}
}

func engineFreshnessBadgeType(stat core.EngineStatsView) display.BadgeType {
	switch {
	case !stat.HasRTT:
		return display.BadgeNeutral
	case stat.Stale(engineStatsStaleThreshold):
		return display.BadgeWarning
	default:
		return display.BadgeSuccess
	}
}
