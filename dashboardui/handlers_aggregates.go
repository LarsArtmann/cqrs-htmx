package dashboardui

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
)

// ===== Aggregate Browser =====

// listStreams returns the first page of stream listings for the index pages
// (aggregates, time-travel, snapshots). When no StreamReader is configured or
// the lookup fails it returns nil so the page renders an empty state.
func (d *Dashboard) listStreams(r *http.Request) []listing.StreamListing {
	if d.cfg.StreamReader == nil {
		return nil
	}

	page, err := d.cfg.StreamReader.List(r.Context(), listing.ListOptions{Limit: uint(d.cfg.PageSize)})
	if err != nil || page == nil {
		return nil
	}

	return page.Items
}

func (d *Dashboard) aggregatesIndexHandler(w http.ResponseWriter, r *http.Request) {
	d.renderStreamIndex(w, r, "Aggregates", "/aggregates", d.renderAggregates)
}

func (d *Dashboard) aggregateDetailHandler(w http.ResponseWriter, r *http.Request) {
	ref, events, ok := d.loadStreamFromRequest(w, r)
	if !ok {
		return
	}

	streamType, streamID := string(ref.Type), ref.ID.String()

	p := d.page("Aggregate: "+streamType+"/"+truncate(streamID, titleIDWidth), "/aggregates", r)

	link := func(href, label string) string {
		return fmt.Sprintf(`<a href="%s" style="color:var(--accent);text-decoration:none">%s</a>`, href, label)
	}

	html := d.renderAggregateDetail(p, ref, events, link)
	renderPage(w, r, html)
}

func (d *Dashboard) renderAggregateDetail(
	p pageData,
	ref id.StreamRef,
	events []event.Event,
	link func(string, string) string,
) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		fmt.Fprintf(&b, `<div style="margin-bottom:24px">`)
		fmt.Fprintf(
			&b,
			`<h2 style="margin:0 0 4px">%s: <code>%s</code></h2>`,
			esc(string(ref.Type)),
			esc(ref.ID.String()),
		)
		fmt.Fprintf(&b, `<div style="color:var(--muted);font-size:0.88em">%d events · current version %s</div>`,
			len(events), latestVersion(events))
		b.WriteString(`</div>`)

		if d.caps.EventSource && len(events) > 0 {
			maxVer := events[len(events)-1].Version().Int()

			fmt.Fprintf(
				&b,
				`<div style="margin-bottom:16px"><a href="%s/time-travel/%s/%s" style="display:inline-flex;align-items:center;gap:6px;padding:6px 12px;border:1px solid var(--border);border-radius:6px;text-decoration:none;color:var(--accent);font-size:0.85em">Inspect time-travel for this aggregate</a></div>`,
				p.BasePath,
				esc(string(ref.Type)),
				esc(ref.ID.String()),
			)

			_ = maxVer
		}

		if len(events) == 0 {
			b.WriteString(
				`<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No events</h3><p>This aggregate has no recorded events.</p></div>`,
			)

			return b.String()
		}

		b.WriteString(`<h4 style="margin-bottom:8px">Event Timeline</h4>`)
		b.WriteString(`<table style="width:100%;border-collapse:collapse">`)
		b.WriteString(`<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">`)
		b.WriteString(`<th style="padding:8px">Version</th><th style="padding:8px">Type</th>`)
		b.WriteString(`<th style="padding:8px">Occurred At</th><th style="padding:8px">Event ID</th>`)
		b.WriteString(`</tr></thead><tbody>`)

		for _, evt := range events {
			fmt.Fprintf(
				&b, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-weight:600">%s</td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px">%s</td>
			</tr>`,
				esc(evt.Version().String()),
				link(fmt.Sprintf("%s/events/%s", p.BasePath, esc(evt.ID().String())), esc(string(evt.Type()))),
				esc(evt.OccurredAt().Format("2006-01-02 15:04:05")),
				fmt.Sprintf(`<code style="font-size:0.8em">%s</code>`, truncate(evt.ID().String(), eventIDWidth)),
			)
		}

		b.WriteString(`</tbody></table>`)

		return b.String()
	})
}

func (d *Dashboard) renderAggregates(p pageData, listings []listing.StreamListing) string {
	return d.renderLayout(p, func() string {
		if len(listings) == 0 {
			return `<div style="padding:40px;text-align:center;color:#64748b"><h3>No aggregates found</h3></div>`
		}

		var (
			rows      string
			rowsSb131 strings.Builder
		)
		for _, l := range listings {
			fmt.Fprintf(&rowsSb131, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px">%d</td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
			</tr>`,
				truncate(l.ID.String(), listIDWidth),
				l.Type,
				l.Version.String(),
				l.EventCount,
				l.LastEventAt.Format("2006-01-02 15:04:05"))
		}

		rows += rowsSb131.String()

		return fmt.Sprintf(`
			<h3 style="margin-bottom:12px">Aggregates</h3>
			<table style="width:100%%;border-collapse:collapse">
				<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">
					<th style="padding:8px">ID</th>
					<th style="padding:8px">Type</th>
					<th style="padding:8px">Version</th>
					<th style="padding:8px">Events</th>
					<th style="padding:8px">Last Event</th>
				</tr></thead>
				<tbody>%s</tbody>
			</table>`, rows)
	})
}
