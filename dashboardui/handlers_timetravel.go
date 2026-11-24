package dashboardui

import (
	"net/http"
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/templ-components/icons"
)

// ===== Time-Travel =====

const maxVersionLinks = 20 // show individual version links up to this threshold

// streamListPageConfig holds the display parameters for a stream-listing index
// page (time-travel and snapshots share the same table layout). Extracting
// this avoids duplicating the row/table/pagination rendering.
type streamListPageConfig struct {
	subtitle   string
	emptyTitle string
	emptyMsg   string
	pagePath   string // pagination base path (e.g., "/time-travel")
	linkPath   string // detail link path prefix (e.g., "/time-travel")
	linkText   string // link label (e.g., "Inspect")
}

func (d *Dashboard) timeTravelIndexHandler(w http.ResponseWriter, r *http.Request) {
	d.renderStreamIndex(w, r, "Time Travel", "/time-travel", timeTravelIndexPage)
}

// timeTravelRow is one rendered row of the time-travel event table: the
// event, its display-ready payload, and whether it is the STEP the selected
// version represents (the last event the view includes — the delta that got
// the aggregate from v-1 to v).
type timeTravelRow struct {
	evt     event.Event
	payload string
	isStep  bool
}

// rowClass returns the row's CSS class; the step row is visually
// highlighted and its payload expanded.
func (r timeTravelRow) rowClass() string {
	if r.isStep {
		return "step-row"
	}

	return ""
}

// buildTimeTravelRows renders every event's payload for inline display so a
// version inspection shows WHAT happened at each step, not just event types
// and timestamps.
func buildTimeTravelRows(config Config, events []event.Event, currentVersion event.Version) []timeTravelRow {
	rows := make([]timeTravelRow, len(events))
	for i, evt := range events {
		rows[i] = timeTravelRow{
			evt:     evt,
			payload: string(payloadBytes(config, evt)),
			isStep:  evt.Version() == currentVersion,
		}
	}

	return rows
}

func (d *Dashboard) timeTravelDetailHandler(w http.ResponseWriter, r *http.Request) {
	ref, allEvents, ok := d.loadStreamFromRequest(w, r)
	if !ok {
		return
	}

	if len(allEvents) == 0 {
		p := d.page("Time Travel: "+string(ref.Type), "/time-travel", r)
		renderPage(
			w,
			r,
			layout(
				p,
				emptyStatePanel(
					icons.Inbox,
					p.empty[EmptyStateTimeTravelEvents].Title,
					p.empty[EmptyStateTimeTravelEvents].Message,
				),
			),
		)

		return
	}

	maxVersion := allEvents[len(allEvents)-1].Version()

	requestedVersion := maxVersion

	if v := r.URL.Query().Get("v"); v != "" {
		if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
			requestedVersion = event.Version(parsed)
		}
	}

	if requestedVersion > maxVersion {
		requestedVersion = maxVersion
	}

	if requestedVersion < event.Version(1) {
		requestedVersion = event.Version(1)
	}

	eventsToVersion, err := d.config.EventSource.LoadToVersion(r.Context(), ref, requestedVersion)
	if err != nil {
		d.renderError(w, r, http.StatusInternalServerError, "failed to load version")

		return
	}

	p := d.page("Time Travel: "+streamTitlePath(ref), "/time-travel", r)
	renderPage(w, r, timeTravelDetailPage(
		p,
		ref,
		buildTimeTravelRows(d.config, eventsToVersion, requestedVersion),
		requestedVersion,
		maxVersion,
	))
}
