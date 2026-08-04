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
	HealthStatus    string
	HealthKind      string
}

type projectionStat struct {
	Name       string
	Status     string
	Lag        string
	Processed  int64
	Errors     int64
	StatusKind string
	Restarts   int
	Checkpoint string
	LastError  string
}

type recentEvent struct {
	Time string `json:"time"`
	Type string `json:"type"`
	//cqrs-lint:ignore(A032) display DTO; branded IDs add no value in view models
	StreamID string `json:"streamId"`
	//cqrs-lint:ignore(A032) display DTO; branded IDs add no value in view models
	StreamType string `json:"streamType"`
	Version    string `json:"version"`
	//cqrs-lint:ignore(A032) display DTO; branded IDs add no value in view models
	EventID   string    `json:"eventId"`
	OccurredAt time.Time `json:"-"`
}

//nolint:cyclop,gocognit // multi-source aggregation
func (d *Dashboard) overviewStats(ctx context.Context) overviewStats {
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

	if d.cfg.SeekableJournal != nil { //nolint:nestif // optional data source branching
		events, err := d.cfg.SeekableJournal.ReadFrom(ctx, id.EventID{}, overviewCountLimit)
		if err == nil {
			for i, evt := range events {
				if i >= recentEventsLimit {
					break
				}

				stats.RecentEvents = append(stats.RecentEvents, recentEvent{
					Time:       evt.OccurredAt().Format(time.RFC3339),
					Type:       string(evt.Type()),
					StreamID:   evt.StreamID().String(),
					StreamType: string(evt.StreamType()),
					Version:    evt.Version().String(),
					EventID:    evt.ID().String(),
					OccurredAt: evt.OccurredAt(),
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
					Time:       evt.OccurredAt().Format(time.RFC3339),
					Type:       string(evt.Type()),
					StreamID:   evt.StreamID().String(),
					StreamType: string(evt.StreamType()),
					Version:    evt.Version().String(),
					EventID:    evt.ID().String(),
					OccurredAt: evt.OccurredAt(),
				})
			}
		}
	}

	if d.cfg.ProjectionHost != nil {
		stats.Projections = buildProjectionStats(d.cfg.ProjectionHost)

		totalErrors := int64(0)
		anyBad := false
		anyWarn := false

		for _, pr := range stats.Projections {
			totalErrors += pr.Errors
			switch pr.StatusKind {
			case statusBad:
				anyBad = true
			case statusWarn:
				anyWarn = true
			}
		}

		if totalErrors > 0 {
			stats.DLQCount = strconv.FormatInt(totalErrors, 10)
		}

		switch {
		case anyBad:
			stats.HealthStatus = "Unhealthy"
			stats.HealthKind = statusBad
		case anyWarn:
			stats.HealthStatus = "Degraded"
			stats.HealthKind = statusWarn
		case len(stats.Projections) > 0:
			stats.HealthStatus = "Healthy"
			stats.HealthKind = statusGood
		}
	}

	return stats
}

func projectionStatusKind(status string) string {
	switch strings.ToLower(status) {
	case statusRunning, "live":
		return statusGood
	case "idle", "backoff", "draining":
		return statusWarn
	case "stopped", statusFailed:
		return statusBad
	default:
		return statusNeutral
	}
}

// healthKindToVariant maps an internal health kind (statusGood/statusWarn/...)
// to the CSS stat-card variant class (ok/warn/err).
func healthKindToVariant(kind string) string {
	switch kind {
	case statusGood:
		return "ok"
	case statusWarn:
		return "warn"
	case statusBad:
		return "err"
	default:
		return ""
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

		if stats.HealthStatus != "" {
			statCard(&inner, stats.HealthStatus, "System Health", healthKindToVariant(stats.HealthKind))
		}

		if stats.DLQCount != "" {
			statCard(&inner, stats.DLQCount, "Dead Letters", "err")
		}

		inner.WriteString(`</div>`)

		if len(stats.Projections) > 0 {
			inner.WriteString(renderProjectionHealthPanel(p.BasePath, stats.Projections))
		}

		if len(stats.RecentEvents) > 0 {
			inner.WriteString(`<h2>Recent Events</h2>`)
			inner.WriteString(`<div class="table-scroll"><table class="data-table"><thead><tr>`)
			inner.WriteString(`<th scope="col">Time</th><th scope="col">Type</th>`)
			inner.WriteString(`<th scope="col">Stream</th><th scope="col">Version</th>`)
			inner.WriteString(`</tr></thead><tbody>`)

			for _, e := range stats.RecentEvents {
				timeDisplay := esc(e.Time)
				if !e.OccurredAt.IsZero() {
					timeDisplay = esc(relativeTime(e.OccurredAt))
				}

				streamCell := esc(truncate(e.StreamID, eventIDWidth))
				if e.StreamType != "" {
					streamCell = fmt.Sprintf(
						`<a href="%s/aggregates/%s/%s" class="mono copyable" data-copyable="%s" title="Click to copy">%s</a>`,
						p.BasePath,
						esc(e.StreamType),
						esc(e.StreamID),
						esc(e.StreamID),
						esc(truncate(e.StreamID, eventIDWidth)),
					)
				}

				fmt.Fprintf(
					&inner,
					`<tr><td class="mono" title="%s">%s</td><td><a href="%s/events/%s"><code>%s</code></a></td><td>%s</td><td>%s</td></tr>`,
					esc(e.Time),
					timeDisplay,
					p.BasePath,
					esc(e.EventID),
					esc(e.Type),
					streamCell,
					esc(e.Version),
				)
			}

			inner.WriteString(`</tbody></table></div>`)
		}

		return inner.String()
	}))

	return b.String()
}

func renderProjectionRow(p projectionStat) string {
	badgeClass := badgeNeutral

	switch p.StatusKind {
	case statusGood:
		badgeClass = badgeOK
	case statusWarn:
		badgeClass = badgeWarn
	case statusBad:
		badgeClass = badgeErr
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

// metaRowCopyable renders a metadata row where the value is click-to-copy.
// The rawValue is placed in data-copyable for clipboard; displayValue is shown.
func metaRowCopyable(b *strings.Builder, key, displayValue, rawValue string) {
	fmt.Fprintf(
		b,
		`<tr><td class="meta-key">%s</td><td class="meta-val copyable" data-copyable="%s" title="Click to copy">%s</td></tr>`,
		key,
		esc(rawValue),
		displayValue,
	)
}

// projectionHealthPartialHandler returns just the projection health panel HTML
// for HTMX polling. Registered at GET /-/partials/projection-health.
func (d *Dashboard) projectionHealthPartialHandler(w http.ResponseWriter, r *http.Request) {
	projs := buildProjectionStats(d.cfg.ProjectionHost)
	html := renderProjectionHealthPanel(d.cfg.BasePath, projs)
	writeHTML(w, r, html, "projection health partial")
}

// renderProjectionHealthPanel renders the projection health panel div with
// HTMX polling attributes and the table inside. Used by both the overview page
// and the projection-health partial endpoint.
func renderProjectionHealthPanel(basePath string, projs []projectionStat) string {
	var b strings.Builder

	b.WriteString(`<div class="panel" id="projection-health" hx-get="`)
	b.WriteString(basePath)
	b.WriteString(`/-/partials/projection-health" hx-trigger="every 10s" hx-swap="outerHTML">`)
	b.WriteString(`<div class="panel-title">Projection Health</div>`)
	b.WriteString(`<div class="table-scroll"><table class="data-table"><thead><tr>`)
	b.WriteString(`<th scope="col">Name</th><th scope="col">Status</th>`)
	b.WriteString(`<th scope="col">Lag</th><th scope="col">Processed</th><th scope="col">Errors</th>`)
	b.WriteString(`</tr></thead><tbody>`)

	for _, pr := range projs {
		b.WriteString(renderProjectionRow(pr))
	}

	b.WriteString(`</tbody></table></div></div>`)

	return b.String()
}
