package dashboardui

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func (d *Dashboard) eventsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Events", "/events", r)

	filters := parseEventFilter(r)
	sortBy := parseSort(r)

	if format := parseFormat(r); format != formatHTML {
		events, err := d.loadFilteredEvents(r.Context(), id.EventID{}, filters, exportLimit)
		if err != nil {
			d.renderError(w, r, http.StatusInternalServerError, "failed to load events for export")

			return
		}

		sortEvents(events, sortBy)

		switch format {
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

	renderPage(w, r, eventsPage(p, events, paginationState{
		HasNext:     hasNext,
		NextCursor:  nextCursor,
		PageSize:    pageSize,
		HasPrev:     hasPrev,
		After:       afterCursor,
		PrevHistory: prevHistory,
	}.WithCountInfo(len(events)), filters, sortBy))
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
	renderPage(w, r, eventDetailPage(p, evt, prevID, nextID, payloadBytes(d.config.PayloadRenderer, evt)))
}
