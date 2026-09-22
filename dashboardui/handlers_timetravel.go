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

func (d *Dashboard) timeTravelDetailHandler(w http.ResponseWriter, r *http.Request) {
	ref, allEvents, ok := d.loadStreamFromRequest(w, r)
	if !ok {
		return
	}

	if len(allEvents) == 0 {
		p := d.page("Time Travel: "+string(ref.Type), "/time-travel", r)
		renderPage(w, r, layout(p, emptyStatePanel(icons.Inbox, "No events", "")))

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

	eventsToVersion, err := d.config.EventSource.LoadToVersion(r.Context(), ref, requestedVersion)
	if err != nil {
		d.renderError(w, r, http.StatusInternalServerError, "failed to load version")

		return
	}

	p := d.page("Time Travel: "+streamTitlePath(ref), "/time-travel", r)
	renderPage(w, r, timeTravelDetailPage(
		p,
		ref,
		eventsToVersion,
		requestedVersion,
		maxVersion,
	))
}
