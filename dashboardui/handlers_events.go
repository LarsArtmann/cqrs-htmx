package dashboardui

import (
	"io"
	"net/http"
	"net/url"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// maxEventFilterBody caps the search rungs' request body (the htmx form
// serialization of three short filter inputs; adminui's search cap is the
// same bound). A larger body is refused before any parse work.
const maxEventFilterBody = 64 << 10

func (d *Dashboard) eventsIndexHandler(w http.ResponseWriter, r *http.Request) {
	d.renderEvents(w, r, parseEventFilter(r), parseSort(r))
}

// eventsSearchHandler is the QUERY ladder surface for the events filter
// (RFC 10008): the same read as GET /events, with the filter criteria
// traveling as the request body instead of URL params. The native QUERY route
// and its POST fallback twin register the SAME handler, so every rung of the
// ladder answers byte-identically; the client is htmx + the embedded
// cqrs-query extension, which upgrade the METHOD but keep htmx's urlencoded
// form serialization for the body (the adminui search precedent). GET /events
// stays the bookmarkable URL-params form.
func (d *Dashboard) eventsSearchHandler(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseEventFilterBody(r)
	if !ok {
		d.renderError(w, r, http.StatusBadRequest, "invalid event filter body")

		return
	}

	d.renderEvents(w, r, filters, parseSort(r))
}

// parseEventFilterBody reads the filter criteria from a urlencoded request
// body. Content-Type-lenient per the library principle (never enforce
// defaults consumers might disagree with); a consumer stacking RFC
// 10008-strict rejection wraps the route with RequireContentType.
func parseEventFilterBody(r *http.Request) (eventFilter, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxEventFilterBody+1))
	if err != nil || len(body) > maxEventFilterBody {
		return eventFilter{}, false
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return eventFilter{}, false
	}

	return eventFilter{
		Type:       values.Get("type"),
		StreamType: values.Get("streamType"),
		StreamID:   values.Get("streamID"),
	}, true
}

// renderEvents is the shared events-read core behind GET /events and the
// QUERY search rungs: one load-and-render path so every rung of the ladder
// answers the same page.
func (d *Dashboard) renderEvents(w http.ResponseWriter, r *http.Request, filters eventFilter, sortBy sortState) {
	if format := parseFormat(r); format != formatHTML {
		d.exportEvents(w, r, format, filters, sortBy)

		return
	}

	p := d.page("Events", "/events", r)

	pageSize := parsePageSize(r, d.config.PageSize)
	afterCursor, prevHistory, hasPrev := parseCursorParams(r)

	afterID, okCursor := parseAfterCursor(d, w, r, afterCursor, id.ParseEventID)
	if !okCursor {
		return
	}

	// The sorted-only view (no filters) is a WINDOW over the most recent
	// filterScanLimit events: cursors are ignored, so the whole window is
	// shown and pagination controls would loop on the same data forever.
	windowed := sortBy.Active() && !filters.Active()

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

	// The window badge tells the truth about what the scan actually returned:
	// the fetch above caps at filterScanLimit, but a smaller journal (or a
	// smaller store-side cap) returns fewer — announcing the cap would claim
	// rows that were never loaded.
	windowCount := 0
	if windowed {
		windowCount = len(events)
	}

	hasNext, events, nextCursor := pageEvents(events, pageSize, windowed)

	sortEvents(events, sortBy)

	state := paginationState{
		HasNext:     hasNext,
		NextCursor:  nextCursor,
		PageSize:    pageSize,
		HasPrev:     hasPrev,
		After:       afterCursor,
		PrevHistory: prevHistory,
	}.WithCountInfo(len(events))

	if windowed {
		// The window is not the whole journal: no next/prev pages (cursors
		// are ignored in this branch) and no "of Z" total (the count would
		// claim completeness for a truncated view).
		state.HasNext = false
		state.HasPrev = false
		state.NextCursor = ""
		state.TotalCount = ""
	}

	renderPage(w, r, eventsPage(p, events, state, filters, sortBy, windowCount))
}

// pageEvents applies the pagination cut and derives the Next cursor.
// hasNext reports whether more rows follow; the windowed sorted view shows
// the WHOLE scan window instead (a pageSize cut there would strand every
// event past page one behind a disabled pagination bar). The cursor comes
// from the PRE-SORT journal-order tail — the sorted slice's last element is
// an arbitrary journal position, and a cursor taken from it re-serves
// overlapping pages.
func pageEvents(
	events []event.Event,
	pageSize int,
	windowed bool,
) (bool, []event.Event, string) {
	// The windowed view shows the WHOLE fetched window (the fetch already caps
	// at filterScanLimit); a pageSize cut there would strand every event past
	// page one behind a disabled pagination bar.
	hasNext := !windowed && len(events) > pageSize
	if hasNext {
		events = events[:pageSize]
	}

	var nextCursor string
	if hasNext && len(events) > 0 {
		nextCursor = events[len(events)-1].ID().String()
	}

	return hasNext, events, nextCursor
}

// exportEvents writes the filtered event journal in a non-HTML export format
// (CSV or JSON). The window is the NEWEST exportLimit matching events (the
// journal is paged once keeping a rolling tail — a head read would export
// the OLDEST events and silently drop everything recent); HTML is routed to
// the paginated index view by the caller instead.
func (d *Dashboard) exportEvents(
	w http.ResponseWriter,
	r *http.Request,
	format responseFormat,
	filters eventFilter,
	sortBy sortState,
) {
	events, _, err := d.loadTailEvents(r.Context(), filters, exportLimit)
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
		// unreachable: callers guard on format != formatHTML
	}
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
		d.renderLoadError(w, r, "event", err)

		return
	}

	prevID, nextID := d.findEventNeighbors(r.Context(), eventID)

	p := d.page("Event: "+truncate(string(evt.Type()), eventTypeWidth), "/events", r)
	renderPage(w, r, eventDetailPage(p, evt, prevID, nextID, payloadBytes(d.config, evt)))
}
