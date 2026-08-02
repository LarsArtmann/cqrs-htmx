package dashboardui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// eventFilter holds in-memory filter criteria for the event browser.
type eventFilter struct {
	Type       string
	StreamType string
	//cqrs-lint:ignore(A032) filter field for HTMX form input
	StreamID   string
}

func (filter eventFilter) Active() bool {
	return filter.Type != "" || filter.StreamType != "" || filter.StreamID != ""
}

func (filter eventFilter) Matches(evt event.Event) bool {
	if filter.Type != "" && string(evt.Type()) != filter.Type {
		return false
	}

	if filter.StreamType != "" && string(evt.StreamType()) != filter.StreamType {
		return false
	}

	if filter.StreamID != "" && evt.StreamID().String() != filter.StreamID {
		return false
	}

	return true
}

func parseEventFilter(r *http.Request) eventFilter {
	q := r.URL.Query()

	return eventFilter{
		Type:       q.Get("type"),
		StreamType: q.Get("streamType"),
		StreamID:   q.Get("streamID"),
	}
}

// filterExtraParams builds a query-string fragment preserving active filters
// across pagination links (e.g., "type=user.created&streamType=User").
func (filter eventFilter) extraParams() string {
	if !filter.Active() {
		return ""
	}

	var parts []string
	if filter.Type != "" {
		parts = append(parts, "type="+filter.Type)
	}

	if filter.StreamType != "" {
		parts = append(parts, "streamType="+filter.StreamType)
	}

	if filter.StreamID != "" {
		parts = append(parts, "streamID="+filter.StreamID)
	}

	return strings.Join(parts, "&")
}

const filterScanLimit = 500

func (d *Dashboard) eventsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Events", "/events", r)

	pageSize := parsePageSize(r, d.cfg.PageSize)
	after := r.URL.Query().Get("after")
	afterID, _ := id.ParseEventID(after)
	filters := parseEventFilter(r)

	var events []event.Event

	var err error

	if filters.Active() {
		events, err = d.loadFilteredEvents(r.Context(), afterID, filters, pageSize)
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

	var nextCursor string
	if hasNext && len(events) > 0 {
		nextCursor = events[len(events)-1].ID().String()
	}

	html := d.renderEvents(p, events, paginationState{
		HasNext:    hasNext,
		NextCursor: nextCursor,
		PageSize:   pageSize,
		HasPrev:    after != "",
	}, filters)
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

//nolint:cyclop // journal fallback
func (d *Dashboard) loadEventByID(ctx context.Context, eventID id.EventID) (event.Event, error) {
	if d.cfg.EventByIDLoader != nil {
		evt, err := d.cfg.EventByIDLoader.LoadByEventID(ctx, eventID)
		if err != nil {
			var zero event.Event

			return zero, errorfamily.WrapInfrastructure(err,
				"dashboardui.event_detail.load_failed", "load event by ID")
		}

		return evt, nil
	}

	if d.cfg.SeekableJournal != nil {
		const scanLimit = 5000

		var after id.EventID

		for {
			batch, err := d.cfg.SeekableJournal.ReadFrom(ctx, after, scanLimit)
			if err != nil {
				return nil, errorfamily.WrapInfrastructure(err,
					"dashboardui.event_detail.scan_failed", "scan journal for event")
			}

			for _, evt := range batch {
				if evt.ID() == eventID {
					return evt, nil
				}
			}

			if len(batch) < scanLimit {
				break
			}

			after = batch[len(batch)-1].ID()
		}

		return nil, errorfamily.Newf(event.Rejection,
			"dashboardui.event_detail.not_found", "event %s not found in journal scan", eventID)
	}

	if d.cfg.Journal != nil {
		all, err := d.cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.event_detail.read_failed", "read journal")
		}

		for _, evt := range all {
			if evt.ID() == eventID {
				return evt, nil
			}
		}
	}

	return nil, errorfamily.Newf(event.Infrastructure,
		"dashboardui.event_detail.no_source", "no event source available to load event %s", eventID)
}

// findEventNeighbors scans recent events to find the previous and next event IDs
// relative to eventID. Returns empty strings if not found or at the boundary.
// This is a best-effort scan limited to the most recent batch of events.
func (d *Dashboard) findEventNeighbors(ctx context.Context, eventID id.EventID) (string, string) {
	const neighborScanLimit = 500

	var events []event.Event

	var err error

	var prevID, nextID string

	if d.cfg.SeekableJournal != nil {
		events, err = d.cfg.SeekableJournal.ReadFrom(ctx, id.EventID{}, neighborScanLimit)
	} else if d.cfg.Journal != nil {
		events, err = d.cfg.Journal.ReadAll(ctx)
		if err == nil && len(events) > neighborScanLimit {
			events = events[:neighborScanLimit]
		}
	}

	if err != nil || len(events) == 0 {
		return "", ""
	}

	for i, evt := range events {
		if evt.ID() == eventID {
			if i > 0 {
				prevID = events[i-1].ID().String()
			}

			if i < len(events)-1 {
				nextID = events[i+1].ID().String()
			}

			return prevID, nextID
		}
	}

	return "", ""
}

func (d *Dashboard) renderEventDetail(p pageData, evt event.Event, prevID, nextID string) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		payload := renderPayload(d.cfg.PayloadRenderer, evt)
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
		fmt.Fprintf(&b, `<pre class="code-block"><code>%s</code></pre>`, esc(string(payload)))
		b.WriteString(`</div>`)

		b.WriteString(`</div>`)

		return b.String()
	})
}

func (d *Dashboard) loadRecentEvents(ctx context.Context, after id.EventID, limit int) ([]event.Event, error) {
	if d.cfg.SeekableJournal != nil {
		events, err := d.cfg.SeekableJournal.ReadFrom(ctx, after, limit)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.recent_events.read_failed", "read recent events")
		}

		return events, nil
	}

	if d.cfg.Journal != nil {
		all, err := d.cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.recent_events.read_all_failed", "read all events")
		}

		if len(all) > limit {
			all = all[:limit]
		}

		return all, nil
	}

	return nil, nil
}

// loadFilteredEvents reads a generous batch from the journal and applies
// in-memory filters, returning up to pageSize+1 results (the +1 is for
// HasMore detection). When filters are active we scan up to filterScanLimit
// raw events per page to find matches.
func (d *Dashboard) loadFilteredEvents(
	ctx context.Context,
	after id.EventID,
	filter eventFilter,
	pageSize int,
) ([]event.Event, error) {
	rawLimit := max(filterScanLimit, pageSize+1)

	var raw []event.Event

	var err error

	if d.cfg.SeekableJournal != nil {
		raw, err = d.cfg.SeekableJournal.ReadFrom(ctx, after, rawLimit)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.filtered_events.read_failed", "read events for filtering")
		}
	} else if d.cfg.Journal != nil {
		raw, err = d.cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.filtered_events.read_all_failed", "read all events for filtering")
		}

		if len(raw) > rawLimit {
			raw = raw[:rawLimit]
		}
	}

	var filtered []event.Event

	for _, evt := range raw {
		if filter.Matches(evt) {
			filtered = append(filtered, evt)
			if len(filtered) > pageSize {
				break
			}
		}
	}

	return filtered, nil
}

func (d *Dashboard) renderEvents(p pageData, events []event.Event, page paginationState, filter eventFilter) string {
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

		fmt.Fprintf(
			&b,
			`<div class="table-scroll"><table class="data-table"><thead><tr><th scope="col">Time</th><th scope="col">Type</th><th scope="col">Stream ID</th><th scope="col">Stream Type</th><th scope="col">Version</th></tr></thead><tbody>%s</tbody></table></div>`,
			rows.String(),
		)

		b.WriteString(renderPagination(p.BasePath, "/events", page, filter.extraParams()))

		return b.String()
	})
}

// renderEventFilterBar renders the filter form with current values pre-filled.
func renderEventFilterBar(basePath string, filter eventFilter) string {
	return fmt.Sprintf(
		`<form method="GET" action="%s/events" class="filter-bar">`+
			`<label for="filter-type">Type</label><input id="filter-type" type="text" name="type" value="%s" placeholder="event.type"/>`+
			`<label for="filter-stream-type">Stream Type</label><input id="filter-stream-type" type="text" name="streamType" value="%s" placeholder="User"/>`+
			`<label for="filter-stream-id">Stream ID</label><input id="filter-stream-id" type="text" name="streamID" value="%s" placeholder="01H..."/>`+
			`<button type="submit" class="btn btn-accent">Filter</button>`+
			`<a href="%s/events" class="btn">Clear</a>`+
			`</form>`,
		esc(basePath), esc(filter.Type), esc(filter.StreamType), esc(filter.StreamID), esc(basePath),
	)
}
