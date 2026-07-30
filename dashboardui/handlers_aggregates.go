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
	p := d.page("Aggregates", "/aggregates", r)

	pageSize := parsePageSize(r, d.cfg.PageSize)

	var (
		listings []listing.StreamListing
		hasMore  bool
	)

	if d.cfg.StreamReader != nil {
		opts := listing.ListOptions{Limit: uint(pageSize + 1)}

		if after := r.URL.Query().Get("after"); after != "" {
			parsed, err := id.ParseStreamID(after)
			if err == nil {
				opts.After = parsed
			}
		}

		page, err := d.cfg.StreamReader.List(r.Context(), opts)
		if err == nil && page != nil {
			hasMore = len(page.Items) > pageSize
			if hasMore {
				listings = page.Items[:pageSize]
			} else {
				listings = page.Items
			}
		}
	}

	hasPrev := r.URL.Query().Get("after") != ""

	var nextCursor string
	if hasMore && len(listings) > 0 {
		nextCursor = listings[len(listings)-1].ID.String()
	}

	html := d.renderAggregates(p, listings, paginationState{
		HasNext:    hasMore,
		NextCursor: nextCursor,
		PageSize:   pageSize,
		HasPrev:    hasPrev,
	})
	renderPage(w, r, html)
}

func (d *Dashboard) aggregateDetailHandler(w http.ResponseWriter, r *http.Request) {
	ref, events, ok := d.loadStreamFromRequest(w, r)
	if !ok {
		return
	}

	p := d.page("Aggregate: "+streamTitlePath(ref), "/aggregates", r)
	html := d.renderAggregateDetail(p, ref, events)
	renderPage(w, r, html)
}

func (d *Dashboard) renderAggregateDetail(
	p pageData,
	ref id.StreamRef,
	events []event.Event,
) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(
			&b,
			`<h2>%s: <code class="copyable" data-copyable="%s" title="Click to copy">%s</code></h2>`,
			esc(string(ref.Type)),
			esc(ref.ID.String()),
			esc(ref.ID.String()),
		)
		fmt.Fprintf(
			&b,
			`<div class="page-subtitle">%d events · current version %s</div>`,
			len(events),
			latestVersion(events),
		)
		b.WriteString(`</div>`)

		if d.caps.EventSource && len(events) > 0 {
			fmt.Fprintf(
				&b,
				`<div class="section-gap"><a href="%s/time-travel/%s/%s" class="btn btn-accent">Inspect time-travel for this aggregate</a></div>`,
				p.BasePath,
				esc(string(ref.Type)),
				esc(ref.ID.String()),
			)
		}

		if len(events) == 0 {
			return emptyState("No events", "This aggregate has no recorded events.")
		}

		var rows strings.Builder

		for _, evt := range events {
			fmt.Fprintf(
				&rows,
				`<tr><td class="cell-emph">%s</td><td><a href="%s/events/%s"><code>%s</code></a></td><td class="mono">%s</td><td><code class="mono copyable" data-copyable="%s" title="Click to copy">%s</code></td></tr>`,
				esc(evt.Version().String()),
				p.BasePath,
				esc(evt.ID().String()),
				esc(string(evt.Type())),
				esc(evt.OccurredAt().Format("2006-01-02 15:04:05")),
				esc(evt.ID().String()),
				truncate(evt.ID().String(), eventIDWidth),
			)
		}

		b.WriteString(`<h3>Event Timeline</h3>`)
		fmt.Fprintf(
			&b,
			`<div class="table-scroll"><table class="data-table"><thead><tr><th scope="col">Version</th><th scope="col">Type</th><th scope="col">Occurred At</th><th scope="col">Event ID</th></tr></thead><tbody>%s</tbody></table></div>`,
			rows.String(),
		)

		return b.String()
	})
}

func (d *Dashboard) renderAggregates(p pageData, listings []listing.StreamListing, pg paginationState) string {
	return d.renderLayout(p, func() string {
		if len(listings) == 0 {
			return emptyState("No aggregates found", "")
		}

		var rows strings.Builder

		for _, l := range listings {
			fmt.Fprintf(
				&rows,
				`<tr><td class="mono">%s</td><td>%s</td><td>%s</td><td>%d</td><td class="mono">%s</td></tr>`,
				esc(truncate(l.ID.String(), listIDWidth)),
				esc(string(l.Type)),
				esc(l.Version.String()),
				l.EventCount,
				esc(l.LastEventAt.Format("2006-01-02 15:04:05")),
			)
		}

		var b strings.Builder
		fmt.Fprintf(
			&b,
			`<h2>Aggregates</h2><div class="table-scroll"><table class="data-table"><thead><tr><th scope="col">ID</th><th scope="col">Type</th><th scope="col">Version</th><th scope="col">Events</th><th scope="col">Last Event</th></tr></thead><tbody>%s</tbody></table></div>`,
			rows.String(),
		)
		b.WriteString(renderPagination(p.BasePath, "/aggregates", pg, ""))

		return b.String()
	})
}
