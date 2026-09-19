package dashboardui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/forms"
	"github.com/larsartmann/templ-components/icons"
	"github.com/larsartmann/templ-components/utils"
)

func (d *Dashboard) eventsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Events", "/events", r)

	filters := parseEventFilter(r)
	sortBy := parseSort(r)

	if fmt := parseFormat(r); fmt != formatHTML {
		events, err := d.loadFilteredEvents(r.Context(), id.EventID{}, filters, exportLimit)
		if err != nil {
			d.renderError(w, r, http.StatusInternalServerError, "failed to load events for export")

			return
		}

		sortEvents(events, sortBy)

		switch fmt {
		case formatCSV:
			exportEventsCSV(w, events)
		case formatJSON:
			exportEventsJSON(w, events)
		case formatHTML:
			// handled below
		}

		return
	}

	pageSize := parsePageSize(r, d.config.PageSize)
	afterCursor, prevHistory, hasPrev := parseCursorParams(r)
	afterID, _ := id.ParseEventID(afterCursor)

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
		d.renderError(w, r, http.StatusInternalServerError, "failed to load events")

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

	html := d.renderEvents(r.Context(), p, events, paginationState{
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
		d.renderError(w, r, http.StatusBadRequest, "invalid event ID")

		return
	}

	evt, err := d.loadEventByID(r.Context(), eventID)
	if err != nil {
		d.renderError(w, r, http.StatusNotFound, "event not found")

		return
	}

	prevID, nextID := d.findEventNeighbors(r.Context(), eventID)

	p := d.page("Event: "+truncate(string(evt.Type()), eventTypeWidth), "/events", r)
	html := d.renderEventDetail(r.Context(), p, evt, prevID, nextID)
	renderPage(w, r, html)
}

func (d *Dashboard) renderEventDetail(
	ctx context.Context,
	p pageData,
	evt event.Event,
	prevID, nextID string,
) string {
	return d.renderLayout(ctx, p, func() string {
		var b strings.Builder

		payload := renderPayload(d.config.PayloadRenderer, evt)
		meta := evt.Metadata()

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(
			&b,
			`<h2><code>%s</code> %s %s</h2>`,
			esc(
				string(evt.Type()),
			),
			badgeHTML(ctx, fmt.Sprintf("schema v%d", evt.SchemaVersion()), display.BadgeNeutral),
			encodingBadge(ctx, string(evt.Encoding())),
		)
		fmt.Fprintf(
			&b,
			`<div class="page-subtitle mono">%s %s</div>`,
			esc(evt.ID().String()),
			copyButtonHTML(ctx, evt.ID().String(), ""),
		)

		if prevID != "" || nextID != "" {
			b.WriteString(`<div class="filter-bar section-gap">`)

			if prevID != "" {
				b.WriteString(buttonLink(
					ctx,
					"← Previous",
					p.BasePath+"/events/"+esc(prevID),
					"",
					display.ButtonSecondary,
					false,
				))
			} else {
				b.WriteString(
					buttonLink(ctx, "← Previous", "#", "Previous event (disabled)", display.ButtonSecondary, true),
				)
			}

			if nextID != "" {
				b.WriteString(buttonLink(
					ctx,
					"Next →",
					p.BasePath+"/events/"+esc(nextID),
					"",
					display.ButtonOutlineInfo,
					false,
				))
			} else {
				b.WriteString(buttonLink(ctx, "Next →", "#", "Next event (disabled)", display.ButtonOutlineInfo, true))
			}

			b.WriteString(`</div>`)
		}

		b.WriteString(`</div>`)

		b.WriteString(`<div class="two-col-grid">`)

		items := eventMetaItems(evt, meta)

		b.WriteString(`<div><h3>Metadata</h3>`)
		b.WriteString(definitionListHTML(ctx, items))

		if len(meta.Custom) > 0 {
			customItems := make([]display.DefinitionItem, 0, len(meta.Custom))

			for k, v := range meta.Custom {
				customItems = append(customItems, defItem(string(k), v))
			}

			b.WriteString(`<h3>Custom Metadata</h3>`)
			b.WriteString(definitionListHTML(ctx, customItems))
		}

		b.WriteString(`</div>`)

		b.WriteString(`<div><h3>Payload</h3>`)
		b.WriteString(`<div class="filter-bar">`)
		b.WriteString(copyButtonHTML(ctx, string(payload), "Copy Payload"))
		b.WriteString(buttonSubmit(
			ctx,
			"Download JSON",
			"",
			display.ButtonSecondary,
			templ.Attributes{"data-download-payload": esc(evt.ID().String())},
		))
		b.WriteString(`</div>`)
		fmt.Fprintf(
			&b,
			`<pre class="code-block" id="event-payload"><code>%s</code></pre>`,
			esc(string(payload)),
		)
		b.WriteString(`</div>`)

		b.WriteString(`</div>`)

		return b.String()
	})
}

func (d *Dashboard) renderEvents(
	ctx context.Context,
	p pageData,
	events []event.Event,
	page paginationState,
	filter eventFilter,
	sortBy sortState,
) string {
	return d.renderLayout(ctx, p, func() string {
		var b strings.Builder
		b.WriteString(`<div class="page-header"><h2>Event Stream</h2></div>`)

		b.WriteString(renderEventFilterBar(ctx, p.BasePath, filter))

		if len(events) == 0 {
			if filter.Active() {
				return emptyStateIcon(ctx, icons.Search,
					"No matching events",
					"No events match the current filters. Try adjusting or clearing them.",
				)
			}

			return emptyStateIcon(
				ctx,
				icons.QueueList,
				"No events yet",
				"Events will appear here as they are committed to the store.",
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

		headers := []display.TableHeader{
			eventSortHeader(p.BasePath, "Time", "time", sortBy, filter.ExtraParams()),
			eventSortHeader(p.BasePath, "Type", "type", sortBy, filter.ExtraParams()),
			{Label: "Stream ID", Sortable: false, SortDirection: "", Href: ""},
			eventSortHeader(p.BasePath, "Stream Type", "streamType", sortBy, filter.ExtraParams()),
			eventSortHeader(p.BasePath, "Version", "version", sortBy, filter.ExtraParams()),
		}

		rows := make([]display.TableRow, 0, len(events))

		for _, evt := range events {
			typeCell := fmt.Sprintf(
				`<a href="%s/events/%s"><code>%s</code></a>`,
				p.BasePath,
				esc(evt.ID().String()),
				esc(string(evt.Type())),
			)
			streamCell := fmt.Sprintf(
				`<span class="mono">%s</span> %s`,
				esc(truncate(evt.StreamID().String(), listIDWidth)),
				copyButtonHTML(ctx, evt.StreamID().String(), ""),
			)

			rows = append(rows, display.TableRow{
				Cells: []display.TableCell{
					textCell(evt.OccurredAt().Format("2006-01-02 15:04:05")),
					rawCell(typeCell),
					rawCell(streamCell),
					textCell(string(evt.StreamType())),
					textCell(evt.Version().String()),
				},
				Href: "",
			})
		}

		b.WriteString(tableHTML(ctx, headers, rows, "events-tbody"))

		b.WriteString(renderPagination(ctx, p.BasePath, "/events", page, combinedParams))
		b.WriteString(formatLinks(ctx, p.BasePath, "/events"))

		return b.String()
	})
}

// filterInput renders a library forms.Input field for the event filter bar.
// The explicit ID keeps the historical DOM ids (filter-type, filter-stream-type,
// filter-stream-id) so CSS, e2e probes, and hx-get serialization stay stable.
func filterInput(ctx context.Context, id, label, name, value, placeholder string) string {
	var b strings.Builder

	_ = forms.Input(forms.InputProps{
		BaseProps:    utils.BaseProps{ID: id, Class: "", Attrs: nil, AriaLabel: "", Nonce: ""},
		Type:         forms.InputText,
		Name:         name,
		Value:        value,
		Placeholder:  placeholder,
		Label:        label,
		Required:     false,
		Disabled:     false,
		ReadOnly:     false,
		AutoFocus:    false,
		MaxLength:    0,
		EnterKeyHint: "",
		Error:        "",
		HelpText:     "",
	}).Render(ctx, &b)

	return b.String()
}

// renderEventFilterBar renders the filter form with current values pre-filled.
// The form uses hx-get for partial content swapping (no full page reload).
func renderEventFilterBar(ctx context.Context, basePath string, filter eventFilter) string {
	return fmt.Sprintf(
		`<form class="filter-bar" hx-get="%s/events" hx-target="#main-content" hx-select="#main-content" hx-swap="outerHTML" hx-push-url="true">`+
			filterInput(
				ctx,
				"filter-type",
				"Type",
				"type",
				filter.Type,
				"event.type",
			)+
			filterInput(
				ctx,
				"filter-stream-type",
				"Stream Type",
				"streamType",
				filter.StreamType,
				"User",
			)+
			filterInput(
				ctx,
				"filter-stream-id",
				"Stream ID",
				"streamID",
				filter.StreamID,
				"01H...",
			)+
			buttonSubmit(
				ctx,
				"Filter",
				"",
				display.ButtonOutlineInfo,
				nil,
			)+
			buttonLink(
				ctx,
				"Clear",
				basePath+"/events",
				"",
				display.ButtonSecondary,
				false,
			)+
			`</form>`,
		esc(basePath),
	)
}

// monoSpan renders s as an inline mono span for definition list details.
func monoSpan(s string) string {
	return "<span class=\"mono\">" + s + "</span>"
}

// eventMetaItems builds the metadata definition items for an event detail
// page: stream/version/encoding facts plus correlation, causation, actor, and
// request IDs when present.
func eventMetaItems(evt event.Event, meta event.Metadata) []display.DefinitionItem {
	items := []display.DefinitionItem{
		defItem("Stream Type", string(evt.StreamType())),
		defItemCopy("Stream ID", monoSpan(esc(evt.StreamID().String())), evt.StreamID().String()),
		defItem("Version", evt.Version().String()),
		defItem("Schema Version", fmt.Sprintf("%d", evt.SchemaVersion())),
		defItem("Encoding", string(evt.Encoding())),
		defItem("Occurred At", evt.OccurredAt().Format(time.RFC3339)),
	}

	if corrID := meta.CorrelationID.String(); corrID != "" {
		items = append(items, defItemCopy("Correlation ID", monoSpan(esc(corrID)), corrID))
	}

	if causID := meta.CausationID.String(); causID != "" {
		items = append(items, defItemCopy("Causation ID", monoSpan(esc(causID)), causID))
	}

	if actorID := meta.ActorID; !actorID.IsZero() {
		actorPrefixed := actorID.PrefixedString()
		items = append(items, defItemCopy("Actor ID", monoSpan(esc(actorPrefixed)), actorPrefixed))
	}

	if reqID := meta.RequestID.String(); reqID != "" {
		items = append(items, defItemCopy("Request ID", monoSpan(esc(reqID)), reqID))
	}

	if deadline, ok := evt.Deadline(); ok {
		items = append(items, defItem("Deadline", deadline.Format(time.RFC3339)))
	}

	return items
}
