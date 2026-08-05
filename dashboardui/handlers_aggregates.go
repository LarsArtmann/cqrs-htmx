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

// listStreamsPaged loads a cursor-paginated page of stream listings for the
// stream-index pages (time-travel, snapshots). Returns the listings and the
// pagination state for rendering Prev/Next controls.
func (d *Dashboard) listStreamsPaged(r *http.Request) ([]listing.StreamListing, paginationState) {
	pageSize := parsePageSize(r, d.config.PageSize)
	afterCursor, prevHistory, hasPrev := parseCursorParams(r)

	if d.config.StreamReader == nil {
		return nil, paginationState{PageSize: pageSize} //nolint:exhaustruct // no data source: only page size matters
	}

	opts := listing.ListOptions{Limit: uint(pageSize + 1)}

	if afterCursor != "" {
		parsed, err := id.ParseStreamID(afterCursor)
		if err == nil {
			opts.After = parsed
		}
	}

	page, err := d.config.StreamReader.List(r.Context(), opts)
	if err != nil || page == nil {
		return nil, paginationState{PageSize: pageSize} //nolint:exhaustruct // error: only page size matters
	}

	hasMore := len(page.Items) > pageSize

	listings := page.Items
	if hasMore {
		listings = listings[:pageSize]
	}

	var nextCursor string
	if hasMore && len(listings) > 0 {
		nextCursor = listings[len(listings)-1].ID.String()
	}

	return listings, paginationState{
		HasNext:     hasMore,
		NextCursor:  nextCursor,
		PageSize:    pageSize,
		HasPrev:     hasPrev,
		After:       afterCursor,
		PrevHistory: prevHistory,
	}
}

func (d *Dashboard) aggregatesIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Aggregates", "/aggregates", r)

	pageSize := parsePageSize(r, d.config.PageSize)
	afterCursor, prevHistory, hasPrev := parseCursorParams(r)

	var (
		listings []listing.StreamListing
		hasMore  bool
	)

	if d.config.StreamReader != nil { //nolint:nestif // optional data source branching
		opts := listing.ListOptions{Limit: uint(pageSize + 1)}

		if afterCursor != "" {
			parsed, err := id.ParseStreamID(afterCursor)
			if err == nil {
				opts.After = parsed
			}
		}

		page, err := d.config.StreamReader.List(r.Context(), opts)
		if err == nil && page != nil {
			hasMore = len(page.Items) > pageSize
			if hasMore {
				listings = page.Items[:pageSize]
			} else {
				listings = page.Items
			}
		}
	}

	var nextCursor string
	if hasMore && len(listings) > 0 {
		nextCursor = listings[len(listings)-1].ID.String()
	}

	html := d.renderAggregates(p, listings, paginationState{
		HasNext:     hasMore,
		NextCursor:  nextCursor,
		PageSize:    pageSize,
		HasPrev:     hasPrev,
		After:       afterCursor,
		PrevHistory: prevHistory,
	})
	renderPage(w, r, html)
}

func (d *Dashboard) aggregateDetailHandler(w http.ResponseWriter, r *http.Request) {
	ref, events, ok := d.loadStreamFromRequest(w, r)
	if !ok {
		return
	}

	p := d.page("Aggregate: "+streamTitlePath(ref), "/aggregates", r)
	html := d.renderAggregateDetail(p, ref, events, d.aggregateTimelinePagination(r))
	renderPage(w, r, html)
}

// aggregateTimelinePagination computes the pagination state for the event
// timeline on the aggregate detail page. Uses version number as cursor
// (?after=3 means start after version 3). The timeline is paginated in-memory
// since EventSource.Load returns all events for a single stream.
func (d *Dashboard) aggregateTimelinePagination(r *http.Request) paginationState {
	pageSize := parsePageSize(r, d.config.PageSize)
	afterCursor, prevHistory, _ := parseCursorParams(r)

	afterVersion := uint64(0)
	if afterCursor != "" {
		afterVersion, _ = strconv.ParseUint(afterCursor, 10, 64)
	}

	return paginationState{
		PageSize:    pageSize,
		After:       afterCursor,
		PrevHistory: prevHistory,
		HasPrev:     afterVersion > 0,
	}
}

func (d *Dashboard) renderAggregateDetail(
	p pageData,
	ref id.StreamRef,
	events []event.Event,
	page paginationState,
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

		// In-memory pagination using version numbers as cursors.
		pagedEvents, hasNext := paginateEventsByVersion(events, page)

		var rows strings.Builder

		for _, evt := range pagedEvents {
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

		timelinePage := page
		timelinePage.HasNext = hasNext
		if hasNext && len(pagedEvents) > 0 {
			timelinePage.NextCursor = pagedEvents[len(pagedEvents)-1].Version().String()
		}

		b.WriteString(renderPagination(
			p.BasePath,
			"/aggregates/"+esc(string(ref.Type))+"/"+esc(ref.ID.String()),
			timelinePage,
			"",
		))

		return b.String()
	})
}

// paginateEventsByVersion slices the events array for in-memory pagination.
// Uses the After cursor as a version number: events with version > After are
// returned, up to PageSize+1 (the +1 is for HasMore detection).
func paginateEventsByVersion(events []event.Event, page paginationState) ([]event.Event, bool) {
	afterVersion := uint64(0)
	if page.After != "" {
		afterVersion, _ = strconv.ParseUint(page.After, 10, 64)
	}

	start := 0
	for i, evt := range events {
		if evt.Version().UInt64() > afterVersion {
			start = i
			break
		}
		if i == len(events)-1 {
			start = len(events)
		}
	}

	if start >= len(events) {
		return nil, false
	}

	end := min(start+page.PageSize+1, len(events))
	paged := events[start:end]
	hasMore := len(paged) > page.PageSize
	if hasMore {
		paged = paged[:page.PageSize]
	}

	return paged, hasMore
}

func (d *Dashboard) renderAggregates(p pageData, listings []listing.StreamListing, page paginationState) string {
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
		b.WriteString(renderPagination(p.BasePath, "/aggregates", page, ""))

		return b.String()
	})
}
