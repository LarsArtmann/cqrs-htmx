package dashboardui

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"unicode/utf8"

	"github.com/larsartmann/templ-components/display"
)

// Display truncation widths for IDs shown in the dashboard UI.
const (
	titleIDWidth      = 12
	listIDWidth       = 24
	eventIDWidth      = 20
	eventTypeWidth    = 30
	snapshotIDWidth   = 16
	errorDisplayWidth = 60
)

func (d *Dashboard) overviewHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Overview", "/", r)
	stats := d.overviewStats(r.Context())
	renderPage(w, r, overviewPage(p, stats))
}

// projectionHealthPartialHandler returns just the projection health panel
// for HTMX polling. Registered at GET /-/partials/projection-health.
func (d *Dashboard) projectionHealthPartialHandler(w http.ResponseWriter, r *http.Request) {
	projs := buildProjectionStats(d.config.ProjectionHost)
	writeHTML(w, r, projectionHealthPanel(d.config.BasePath, projs), "projection health partial")
}

// statusKindToStatus maps an internal health kind to the status word the
// library's display.StatusBadge understands ("healthy"/"degraded"/"error").
// The empty result means "unknown kind" — the statusBadge component keeps
// the raw status text as an explicitly neutral badge.
func statusKindToStatus(kind string) string {
	switch kind {
	case statusGood:
		return "healthy"
	case statusWarn:
		return "degraded"
	case statusBad:
		return "error"
	default:
		return ""
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}

	cut := s[:n]
	// Back off while the cut would split a multi-byte rune: a mid-rune cut
	// is invalid UTF-8 and renders as mojibake in the truncated cell.
	for len(cut) > 0 {
		r, size := utf8.DecodeLastRuneInString(cut)
		if r != utf8.RuneError || size != 1 {
			break
		}

		cut = cut[:len(cut)-1]
	}

	return cut + "..."
}

// pathSeg escapes a free-text URL path segment (projection name, stream
// name). Go's ServeMux decodes the escape again in PathValue, so names with
// spaces or slashes round-trip through routes that would otherwise 404 or
// mis-split.
func pathSeg(s string) string {
	return url.PathEscape(s)
}

func esc(s string) string {
	return html.EscapeString(s)
}

// projectionsActiveText builds the "active/total" value for the projections
// stat card (healthy projections over total).
func projectionsActiveText(stats overviewStats) string {
	active := 0

	for _, pr := range stats.Projections {
		if pr.StatusKind == statusGood {
			active++
		}
	}

	return fmt.Sprintf("%d/%d", active, len(stats.Projections))
}

// sseHubValue picks the SSE-clients stat value: subscriber count, or the
// lifecycle state when the hub is closing down.
func sseHubValue(stats overviewStats) string {
	switch {
	case stats.SSEHub.Closed:
		return "closed"
	case stats.SSEHub.Draining:
		return "draining"
	default:
		return strconv.Itoa(stats.SSEHub.Subscribers)
	}
}

// sseHubTone maps the SSE hub lifecycle state to a stat tone.
func sseHubTone(stats overviewStats) display.StatTone {
	switch {
	case stats.SSEHub.Closed:
		return display.StatToneRed
	case stats.SSEHub.Draining:
		return display.StatToneYellow
	default:
		return display.StatToneGreen
	}
}
