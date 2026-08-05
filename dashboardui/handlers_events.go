package dashboardui

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func (d *Dashboard) eventsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Events", "/events", r)

	pageSize := parsePageSize(r, d.config.PageSize)
	afterCursor, prevHistory, hasPrev := parseCursorParams(r)
	afterID, _ := id.ParseEventID(afterCursor)
	filters := parseEventFilter(r)
	sortBy := parseSort(r)

	var events []event.Event

	var err error

	if filters.Active() {
		events, err = d.loadFilteredEvents(r.Context(), afterID, filters, pageSize)
	} else if sortBy.Active() {
		// When sorting is active, load up to filterScanLimit events for in-memory sort.
		events, err = d.loadFilteredEvents(r.Context(), id.EventID{}, eventFilter{}, filterScanLimit)
	} else {
		events, err = d.loadRecentEvents(r.Context(), afterID, pageSize+1)
	}

	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "failed to load events")

		return
	}

	hasNext := len(events) > pageSize
	if hasNext {
		events = events[:pageSize]
	}

	sortEvents(events, sortBy)

	var nextCursor string
	if hasNext && len(events) > 0 {
		nextCursor = events[len(events)-1].ID().String()
	}

	html := d.renderEvents(p, events, paginationState{
		HasNext:     hasNext,
		NextCursor:  nextCursor,
		PageSize:    pageSize,
		HasPrev:     hasPrev,
		After:       afterCursor,
		PrevHistory: prevHistory,
	}.WithCountInfo(len(events)), filters, sortBy)
	renderPage(w, r, html)
}

func (d *Dashboard) eventDetailHandler(w http.ResponseWriter, r *http.Request) {
	eventIDStr := r.PathValue("id")

	eventID, err := id.ParseEventID(eventIDStr)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "invalid event ID")

		return
	}

	evt, err := d.loadEventByID(r.Context(), eventID)
	if err != nil {
		renderError(w, r, http.StatusNotFound, "event not found")

		return
	}

	prevID, nextID := d.findEventNeighbors(r.Context(), eventID)

	p := d.page("Event: "+truncate(string(evt.Type()), eventTypeWidth), "/events", r)
	html := d.renderEventDetail(p, evt, prevID, nextID)
	renderPage(w, r, html)
}

func (d *Dashboard) renderEventDetail(p pageData, evt event.Event, prevID, nextID string) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		payload := renderPayload(d.config.PayloadRenderer, evt)
		meta := evt.Metadata()

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(
			&b,
			`<h2><code>%s</code> <span class="badge badge-neutral">schema v%d</span> <span class="badge %s">%s</span></h2>`,
			esc(
				string(evt.Type()),
			),
			evt.SchemaVersion(),
			encodingBadgeClass(string(evt.Encoding())),
			esc(string(evt.Encoding())),
		)
		fmt.Fprintf(&b, `<div class="page-subtitle mono copyable" data-copyable="%s" title="Click to copy">%s</div>`,
			esc(evt.ID().String()), esc(evt.ID().String()))

		if prevID != "" || nextID != "" {
			b.WriteString(`<div class="filter-bar section-gap">`)

			if prevID != "" {
				fmt.Fprintf(&b, `<a href="%s/events/%s" class="btn">← Previous</a>`, p.BasePath, esc(prevID))
			} else {
				b.WriteString(`<span class="btn" aria-disabled="true">← Previous</span>`)
			}

			if nextID != "" {
				fmt.Fprintf(&b, `<a href="%s/events/%s" class="btn btn-accent">Next →</a>`, p.BasePath, esc(nextID))
			} else {
				b.WriteString(`<span class="btn" aria-disabled="true">Next →</span>`)
			}

			b.WriteString(`</div>`)
		}

		b.WriteString(`</div>`)

		b.WriteString(`<div class="two-col-grid">`)

		b.WriteString(`<div><h3>Metadata</h3><table class="meta-table">`)
		metaRow(&b, "Stream Type", esc(string(evt.StreamType())))
		metaRowCopyable(&b, "Stream ID", esc(evt.StreamID().String()), evt.StreamID().String())
		metaRow(&b, "Version", esc(evt.Version().String()))
		metaRow(&b, "Schema Version", esc(fmt.Sprintf("%d", evt.SchemaVersion())))
		metaRow(&b, "Encoding", esc(string(evt.Encoding())))
		metaRow(&b, "Occurred At", esc(evt.OccurredAt().Format(time.RFC3339)))

		if corrID := meta.CorrelationID.String(); corrID != "" {
			metaRowCopyable(&b, "Correlation ID", esc(corrID), corrID)
		}

		if causID := meta.CausationID.String(); causID != "" {
			metaRowCopyable(&b, "Causation ID", esc(causID), causID)
		}

		if userID := meta.UserID.String(); userID != "" {
			metaRowCopyable(&b, "User ID", esc(userID), userID)
		}

		if reqID := meta.RequestID.String(); reqID != "" {
			metaRowCopyable(&b, "Request ID", esc(reqID), reqID)
		}

		if deadline, ok := evt.Deadline(); ok {
			metaRow(&b, "Deadline", esc(deadline.Format(time.RFC3339)))
		}

		b.WriteString(`</table></div>`)

		if len(meta.Custom) > 0 {
			b.WriteString(`<h3>Custom Metadata</h3><table class="meta-table">`)

			for k, v := range meta.Custom {
				metaRow(&b, esc(string(k)), esc(v))
			}

			b.WriteString(`</table></div>`)
		}

		b.WriteString(`</div>`)

		b.WriteString(`<div><h3>Payload</h3>`)
		fmt.Fprintf(
			&b,
			`<div class="filter-bar"><button class="btn" onclick="copyPayload()">Copy</button><button class="btn" onclick="downloadPayload('%s')">Download JSON</button></div>`,
			esc(evt.ID().String()),
		)
		fmt.Fprintf(&b, `<pre class="code-block" id="event-payload"><code>%s</code></pre>`, esc(string(payload)))
		b.WriteString(`</div>`)

		b.WriteString(`</div>`)

		return b.String()
	})
}

func (d *Dashboard) renderEvents(
	p pageData,
	events []event.Event,
	page paginationState,
	filter eventFilter,
	sortBy sortState,
) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder
		b.WriteString(`<div class="page-header"><h2>Event Stream</h2></div>`)

		b.WriteString(renderEventFilterBar(p.BasePath, filter))

		if len(events) == 0 {
			if filter.Active() {
				return emptyState(
					"No matching events",
					"No events match the current filters. Try adjusting or clearing them.",
				)
			}

			return emptyState("No events yet", "Events will appear here as they are committed to the store.")
		}

		var rows strings.Builder

		for _, evt := range events {
			fmt.Fprintf(
				&rows,
				`<tr><td class="mono">%s</td><td><a href="%s/events/%s"><code>%s</code></a></td><td class="mono copyable" data-copyable="%s" title="Click to copy">%s</td><td>%s</td><td>%s</td></tr>`,
				esc(evt.OccurredAt().Format("2006-01-02 15:04:05")),
				p.BasePath,
				esc(evt.ID().String()),
				esc(string(evt.Type())),
				esc(evt.StreamID().String()),
				esc(truncate(evt.StreamID().String(), listIDWidth)),
				esc(string(evt.StreamType())),
				esc(evt.Version().String()),
			)
		}

		combinedParams := filter.ExtraParams()
		if sp := sortBy.extraParams(); sp != "" {
			if combinedParams != "" {
				combinedParams += "&" + sp
			} else {
				combinedParams = sp
			}
		}

		fmt.Fprintf(
			&b,
			`<div class="table-scroll"><table class="data-table"><thead><tr>%s%s%s%s%s</tr></thead><tbody>%s</tbody></table></div>`,
			sortHeader(p.BasePath, "Time", "time", sortBy, filter.ExtraParams()),
			sortHeader(p.BasePath, "Type", "type", sortBy, filter.ExtraParams()),
			`<th scope="col">Stream ID</th>`,
			sortHeader(p.BasePath, "Stream Type", "streamType", sortBy, filter.ExtraParams()),
			sortHeader(p.BasePath, "Version", "version", sortBy, filter.ExtraParams()),
			rows.String(),
		)

		b.WriteString(renderPagination(p.BasePath, "/events", page, combinedParams))

		return b.String()
	})
}

// renderEventFilterBar renders the filter form with current values pre-filled.
// The form uses hx-get for partial content swapping (no full page reload).
func renderEventFilterBar(basePath string, filter eventFilter) string {
	return fmt.Sprintf(
		`<form class="filter-bar" hx-get="%s/events" hx-target="#main-content" hx-select="#main-content" hx-swap="outerHTML" hx-push-url="true">`+
			`<label for="filter-type">Type</label><input id="filter-type" type="text" name="type" value="%s" placeholder="event.type"/>`+
			`<label for="filter-stream-type">Stream Type</label><input id="filter-stream-type" type="text" name="streamType" value="%s" placeholder="User"/>`+
			`<label for="filter-stream-id">Stream ID</label><input id="filter-stream-id" type="text" name="streamID" value="%s" placeholder="01H..."/>`+
			`<button type="submit" class="btn btn-accent">Filter</button>`+
			`<a href="%s/events" class="btn">Clear</a>`+
			`</form>`,
		esc(basePath),
		esc(filter.Type),
		esc(filter.StreamType),
		esc(filter.StreamID),
		esc(basePath),
	)
}
