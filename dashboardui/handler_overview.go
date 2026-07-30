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
	titleIDWidth      = 12
	listIDWidth       = 24
	eventIDWidth      = 20
	eventTypeWidth    = 30
	snapshotIDWidth   = 16
	errorDisplayWidth = 60
)

const (
	recentEventsLimit  = 5
	overviewCountLimit = 500
)

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

func (d *Dashboard) overviewStats(ctx context.Context) overviewStats { //nolint:cyclop // stat computation
	stats := overviewStats{
		TotalAggregates: "0",
		TotalEvents:     "0",
	}

	if d.cfg.StreamReader != nil {
		page, err := d.cfg.StreamReader.List(ctx, listing.ListOptions{Limit: uint(d.cfg.PageSize)})
		if err == nil && page != nil {
			stats.TotalAggregates = strconv.Itoa(len(page.Items))
			if page.HasMore {
				stats.TotalAggregates += "+"
			}
		}
	}

	if d.cfg.SeekableJournal != nil {
		events, err := d.cfg.SeekableJournal.ReadFrom(ctx, id.EventID{}, overviewCountLimit)
		if err == nil {
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

			stats.TotalEvents = strconv.Itoa(len(events))
			if len(events) >= overviewCountLimit {
				stats.TotalEvents += "+"
			}
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

func (d *Dashboard) renderOverview(p pageData, stats overviewStats) string {
	var b strings.Builder

	b.WriteString(d.renderLayout(p, func() string {
		var inner strings.Builder

		inner.WriteString(`<div class="stat-grid">`)
		statCard(&inner, stats.TotalEvents, "Events", "")
		statCard(&inner, stats.TotalAggregates, "Aggregates", "")

		if len(stats.Projections) > 0 {
			active := 0

			for _, pr := range stats.Projections {
				if pr.StatusKind == statusGood {
					active++
				}
			}

			statCard(&inner, fmt.Sprintf("%d/%d", active, len(stats.Projections)), "Projections", "ok")
		}

		inner.WriteString(`</div>`)

		if len(stats.Projections) > 0 {
			inner.WriteString(`<div class="panel" id="projection-health" hx-get="`)
			inner.WriteString(p.BasePath)
			inner.WriteString(`/-/partials/projection-health" hx-trigger="every 10s" hx-swap="outerHTML">`)
			inner.WriteString(`<div class="panel-title">Projection Health</div>`)
			inner.WriteString(`<table class="data-table"><thead><tr>`)
			inner.WriteString(`<th scope="col">Name</th><th scope="col">Status</th>`)
			inner.WriteString(`<th scope="col">Lag</th><th scope="col">Processed</th><th scope="col">Errors</th>`)
			inner.WriteString(`</tr></thead><tbody>`)

			for _, pr := range stats.Projections {
				inner.WriteString(renderProjectionRow(pr))
			}

			inner.WriteString(`</tbody></table></div>`)
		}

		if len(stats.RecentEvents) > 0 {
			inner.WriteString(`<h3>Recent Events</h3>`)
			inner.WriteString(`<table class="data-table"><thead><tr>`)
			inner.WriteString(`<th scope="col">Time</th><th scope="col">Type</th>`)
			inner.WriteString(`<th scope="col">Stream</th><th scope="col">Version</th>`)
			inner.WriteString(`</tr></thead><tbody>`)

			for _, e := range stats.RecentEvents {
				fmt.Fprintf(
					&inner,
					`<tr><td class="mono">%s</td><td><code>%s</code></td><td class="mono">%s</td><td>%s</td></tr>`,
					esc(e.Time),
					esc(e.Type),
					esc(truncate(e.StreamID, eventIDWidth)),
					esc(e.Version),
				)
			}

			inner.WriteString(`</tbody></table>`)
		}

		return inner.String()
	}))

	return b.String()
}

func renderProjectionRow(p projectionStat) string {
	badgeClass := "badge badge-neutral"

	switch p.StatusKind {
	case statusGood:
		badgeClass = "badge badge-ok"
	case statusWarn:
		badgeClass = "badge badge-warn"
	case statusBad:
		badgeClass = "badge badge-err"
	}

	return fmt.Sprintf(
		`<tr><td>%s</td><td><span class="%s">%s</span></td><td class="mono">%s</td><td>%d</td><td>%d</td></tr>`,
		esc(p.Name),
		badgeClass,
		esc(p.Status),
		esc(p.Lag),
		p.Processed,
		p.Errors,
	)
}

func statCard(b *strings.Builder, value, label, variant string) {
	classes := "stat-card"
	if variant != "" {
		classes += " " + variant
	}

	fmt.Fprintf(b, `<div class="%s">`, classes)
	fmt.Fprintf(b, `<div class="stat-card-value">%s</div>`, esc(value))
	fmt.Fprintf(b, `<div class="stat-card-label">%s</div>`, esc(label))
	b.WriteString(`</div>`)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}

	return s[:n] + "..."
}

func esc(s string) string {
	return html.EscapeString(s)
}

func metaRow(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, `<tr><td class="meta-key">%s</td><td class="meta-val">%s</td></tr>`, key, value)
}
