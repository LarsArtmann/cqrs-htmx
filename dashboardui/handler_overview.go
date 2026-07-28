package dashboardui

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
)

// Projection status kinds drive the color-coding of projection health in the UI.
const (
	statusGood    = "good"
	statusWarn    = "warn"
	statusBad     = "bad"
	statusNeutral = "neutral"
)

// Display truncation widths for IDs shown in the dashboard UI.
const (
	titleIDWidth       = 12 // page-title stream/aggregate ID truncation
	listIDWidth        = 24 // list-row ID truncation
	eventIDWidth       = 20 // event ID truncation in tables
	eventTypeWidth     = 30 // event type label truncation
	snapshotIDWidth    = 16 // snapshot detail streamID truncation
	errorDisplayWidth  = 60 // error message truncation in DLQ
)

// recentEventsLimit is how many recent events the overview card shows.
const recentEventsLimit = 5

func (d *Dashboard) overviewHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Overview", "/", r)

	stats := d.overviewStats(r.Context())

	html := d.renderOverview(p, stats)
	renderPage(w, r, html)
}

type overviewStats struct {
	TotalAggregates string
	TotalEvents     string
	Projections     []projectionStat
	DLQCount        string
	RecentEvents    []recentEvent
}

type projectionStat struct {
	Name       string
	Status     string
	Lag        string
	Processed  int64
	Errors     int64
	StatusKind string
}

type recentEvent struct {
	Time     string
	Type     string
	StreamID string
	Version  string
	EventID  string
}

func (d *Dashboard) overviewStats(ctx context.Context) overviewStats {
	stats := overviewStats{}

	if d.cfg.StreamReader != nil {
		page, err := d.cfg.StreamReader.List(ctx, listing.ListOptions{Limit: 1})
		if err == nil && page != nil {
			stats.TotalAggregates = strconv.Itoa(len(page.Items))
			if page.HasMore {
				stats.TotalAggregates += "+"
			}
		}
	}

	if d.cfg.SeekableJournal != nil {
		events, err := d.cfg.SeekableJournal.ReadFrom(ctx, id.EventID{}, recentEventsLimit)
		if err == nil {
			for _, evt := range events {
				stats.RecentEvents = append(stats.RecentEvents, recentEvent{
					Time:     evt.OccurredAt().Format(time.RFC3339),
					Type:     string(evt.Type()),
					StreamID: evt.StreamID().String(),
					Version:  evt.Version().String(),
					EventID:  evt.ID().String(),
				})
			}

			stats.TotalEvents = fmt.Sprintf("%d+", len(events))
		}
	} else if d.cfg.Journal != nil {
		events, err := d.cfg.Journal.ReadAll(ctx)
		if err == nil {
			stats.TotalEvents = strconv.Itoa(len(events))
			for i, evt := range events {
				if i >= recentEventsLimit {
					break
				}

				stats.RecentEvents = append(stats.RecentEvents, recentEvent{
					Time:     evt.OccurredAt().Format(time.RFC3339),
					Type:     string(evt.Type()),
					StreamID: evt.StreamID().String(),
					Version:  evt.Version().String(),
					EventID:  evt.ID().String(),
				})
			}
		}
	}

	if d.cfg.ProjectionHost != nil {
		lagPerProj := d.cfg.ProjectionHost.LagPerProjection()
		for _, ws := range d.cfg.ProjectionHost.Status() {
			lag := lagPerProj[ws.Name]
			stats.Projections = append(stats.Projections, projectionStat{
				Name:       ws.Name,
				Status:     string(ws.Status),
				Lag:        lag.String(),
				Processed:  ws.Processed,
				Errors:     ws.Errors,
				StatusKind: projectionStatusKind(string(ws.Status)),
			})
		}
	}

	return stats
}

func projectionStatusKind(status string) string {
	switch strings.ToLower(status) {
	case "running", "live":
		return statusGood
	case "idle", "backoff", "draining":
		return statusWarn
	case "stopped", "failed":
		return statusBad
	default:
		return statusNeutral
	}
}

// renderOverview produces the overview page HTML.
// This is intentionally simple Go-generated HTML for the initial version.
// Future iterations will use templ components from templ-components.
func (d *Dashboard) renderOverview(p pageData, stats overviewStats) string {
	var b strings.Builder

	b.WriteString(d.renderLayout(p, func() string {
		var inner strings.Builder

		inner.WriteString(
			`<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:16px;margin-bottom:24px">`,
		)

		statCard(&inner, stats.TotalEvents, "Events", "#1d3557")
		statCard(&inner, stats.TotalAggregates, "Aggregates", "#1d3557")

		if len(stats.Projections) > 0 {
			active := 0

			for _, p := range stats.Projections {
				if p.StatusKind == statusGood {
					active++
				}
			}

			statCard(&inner, fmt.Sprintf("%d/%d", active, len(stats.Projections)), "Projections", "#16a34a")
		}

		inner.WriteString(`</div>`)

		if len(stats.Projections) > 0 {
			inner.WriteString(`<h3 style="margin-bottom:12px">Projection Health</h3>`)
			inner.WriteString(`<table style="width:100%;border-collapse:collapse;margin-bottom:24px">`)
			inner.WriteString(`<thead><tr style="text-align:left;border-bottom:2px solid #e6e8ec">`)
			inner.WriteString(`<th style="padding:8px">Name</th><th style="padding:8px">Status</th>`)
			inner.WriteString(`<th style="padding:8px">Lag</th><th style="padding:8px">Processed</th>`)
			inner.WriteString(`<th style="padding:8px">Errors</th></tr></thead><tbody>`)

			for _, p := range stats.Projections {
				color := "#64748b"

				switch p.StatusKind {
				case statusGood:
					color = "#16a34a"
				case statusWarn:
					color = "#d97706"
				case statusBad:
					color = "#dc2626"
				}

				fmt.Fprintf(&inner, `<tr style="border-bottom:1px solid #e6e8ec">
					<td style="padding:8px">%s</td>
					<td style="padding:8px"><span style="color:%s;font-weight:600">%s</span></td>
					<td style="padding:8px">%s</td>
					<td style="padding:8px">%d</td>
					<td style="padding:8px">%d</td>
				</tr>`, p.Name, color, p.Status, p.Lag, p.Processed, p.Errors)
			}

			inner.WriteString(`</tbody></table>`)
		}

		if len(stats.RecentEvents) > 0 {
			inner.WriteString(`<h3 style="margin-bottom:12px">Recent Events</h3>`)
			inner.WriteString(`<table style="width:100%;border-collapse:collapse">`)
			inner.WriteString(`<thead><tr style="text-align:left;border-bottom:2px solid #e6e8ec">`)
			inner.WriteString(`<th style="padding:8px">Time</th><th style="padding:8px">Type</th>`)
			inner.WriteString(`<th style="padding:8px">Stream</th><th style="padding:8px">Version</th>`)
			inner.WriteString(`</tr></thead><tbody>`)

			for _, e := range stats.RecentEvents {
				fmt.Fprintf(&inner, `<tr style="border-bottom:1px solid #e6e8ec">
					<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
					<td style="padding:8px"><code>%s</code></td>
					<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
					<td style="padding:8px">%s</td>
				</tr>`, e.Time, e.Type, truncate(e.StreamID, eventIDWidth), e.Version)
			}

			inner.WriteString(`</tbody></table>`)
		}

		return inner.String()
	}))

	return b.String()
}

func statCard(b *strings.Builder, value, label, color string) {
	fmt.Fprintf(
		b,
		`<div style="border:2px solid #111;padding:20px;text-align:center;background:%s;color:white">`,
		color,
	)
	fmt.Fprintf(b, `<div style="font-size:2.5rem;font-weight:900;line-height:1">%s</div>`, value)
	fmt.Fprintf(
		b,
		`<div style="font-size:0.7rem;font-weight:700;text-transform:uppercase;letter-spacing:0.12em;margin-top:6px">%s</div>`,
		label,
	)
	b.WriteString(`</div>`)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}

	return s[:n] + "..."
}

// esc wraps html.EscapeString for terse call sites.
func esc(s string) string {
	return html.EscapeString(s)
}

// metaRow writes a key-value row into a metadata table.
func metaRow(b *strings.Builder, key, value string) {
	fmt.Fprintf(
		b,
		`<tr style="border-bottom:1px solid var(--border)"><td style="padding:6px 8px;color:var(--muted);font-weight:500">%s</td><td style="padding:6px 8px;font-family:monospace;font-size:0.85em">%s</td></tr>`,
		key,
		value,
	)
}
