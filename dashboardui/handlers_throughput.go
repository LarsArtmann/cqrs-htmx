package dashboardui

import (
	"net/http"
	"strconv"
	"time"
)

// ===== Event Throughput =====

// throughputSpan is the charted window ending at the newest event; the
// bucket count divides it (24h / 12 = 2h buckets).
const (
	throughputSpan    = 24 * time.Hour
	throughputBuckets = 12

	// minThroughputBuckets / maxThroughputBuckets bound the ?buckets=
	// override so a URL cannot make the chart unreadable or the labels
	// vanish.
	minThroughputBuckets = 4
	maxThroughputBuckets = 48
)

// throughputIndexHandler serves the event-throughput panel: journal events
// bucketed into fixed time windows as a pure-SVG line chart (zero client
// JavaScript, CSP-safe by construction).
func (d *Dashboard) throughputIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Throughput", "/throughput", r)

	buckets := parseThroughputBuckets(r, throughputBuckets)

	data, err := d.loadEventThroughput(r.Context(), throughputSpan, buckets)
	if err != nil {
		d.renderError(w, r, http.StatusInternalServerError, "failed to load event throughput")

		return
	}

	renderPage(w, r, throughputPage(p, data))
}

// parseThroughputBuckets reads the ?buckets= override, clamped to a sane
// range; an unparseable value falls back to the default rather than erroring
// (the chart is a glance, not a form).
func parseThroughputBuckets(r *http.Request, fallback int) int {
	raw := r.URL.Query().Get("buckets")
	if raw == "" {
		return fallback
	}

	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return min(max(n, minThroughputBuckets), maxThroughputBuckets)
}
