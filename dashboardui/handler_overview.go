package dashboardui

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/utils"
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
	html := d.renderOverview(r.Context(), p, stats)
	renderPage(w, r, html)
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

func (d *Dashboard) renderOverview(ctx context.Context, p pageData, stats overviewStats) string {
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
			inner.WriteString(renderProjectionHealthPanel(ctx, p.BasePath, stats.Projections))
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

func renderProjectionRow(ctx context.Context, p projectionStat) string {
	var b strings.Builder

	fmt.Fprintf(&b, `<tr><td>%s</td><td>`, esc(p.Name))
	statusBadge(ctx, &b, p.StatusKind, p.Status)
	fmt.Fprintf(&b, `</td><td class="mono">%s</td><td>%d</td><td>%d</td></tr>`, esc(p.Lag), p.Processed, p.Errors)

	return b.String()
}

// statusKindToStatus maps an internal health kind to the status word the
// library's display.StatusBadge understands ("healthy"/"degraded"/"error").
// The empty result means "unknown kind" — the caller keeps the raw status
// text as an explicitly neutral badge.
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

// statusBadge renders a projection status as a templ-components badge (the
// hybrid adoption path: templ.Component.Render into the existing
// strings.Builder — no .templ conversion). Unknown kinds keep their raw
// status text styled as an explicitly neutral badge (old default behavior).
// Render only errors on writer failure, which strings.Builder cannot
// produce.
func statusBadge(ctx context.Context, b *strings.Builder, kind, statusText string) {
	if mapped := statusKindToStatus(kind); mapped != "" {
		_ = display.StatusBadge(mapped).Render(ctx, b)

		return
	}

	props := display.BadgeProps{
		BaseProps: utils.BaseProps{ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: ""},
		Text:      statusText,
		Type:      display.BadgeNeutral,
		Size:      display.BadgeSizeMD,
		Pill:      false,
		Dot:       true,
		Href:      "",
	}
	_ = display.Badge(props).Render(ctx, b)
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
	projs := buildProjectionStats(d.config.ProjectionHost)
	html := renderProjectionHealthPanel(r.Context(), d.config.BasePath, projs)
	writeHTML(w, r, html, "projection health partial")
}

// renderProjectionHealthPanel renders the projection health panel div with
// HTMX polling attributes and the table inside. Used by both the overview page
// and the projection-health partial endpoint.
func renderProjectionHealthPanel(ctx context.Context, basePath string, projs []projectionStat) string {
	var b strings.Builder

	b.WriteString(`<div class="panel" id="projection-health" hx-get="`)
	b.WriteString(basePath)
	b.WriteString(`/-/partials/projection-health" hx-trigger="every 10s, refresh" hx-swap="outerHTML">`)
	b.WriteString(`<div class="panel-title">Projection Health</div>`)
	b.WriteString(`<div class="table-scroll"><table class="data-table"><thead><tr>`)
	b.WriteString(`<th scope="col">Name</th><th scope="col">Status</th>`)
	b.WriteString(`<th scope="col">Lag</th><th scope="col">Processed</th><th scope="col">Errors</th>`)
	b.WriteString(`</tr></thead><tbody>`)

	for _, pr := range projs {
		b.WriteString(renderProjectionRow(ctx, pr))
	}

	b.WriteString(`</tbody></table></div></div>`)

	return b.String()
}
