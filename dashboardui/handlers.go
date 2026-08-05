package dashboardui

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func streamRefFromRequest(r *http.Request) (id.StreamRef, error) {
	return StreamRefFromID(r.PathValue("type"), r.PathValue("id"))
}

func (d *Dashboard) loadStreamFromRequest(
	w http.ResponseWriter,
	r *http.Request,
) (id.StreamRef, []event.Event, bool) {
	ref, err := streamRefFromRequest(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "invalid stream reference")

		return id.StreamRef{}, nil, false
	}

	events, err := d.config.EventSource.Load(r.Context(), ref)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "failed to load aggregate")

		return id.StreamRef{}, nil, false
	}

	return ref, events, true
}

// streamTitlePath renders a "type/truncated-id" path for page titles.
func streamTitlePath(ref id.StreamRef) string {
	return string(ref.Type) + "/" + truncate(ref.ID.String(), titleIDWidth)
}

// latestVersion returns the last event version, or "0" for an empty stream.
func latestVersion(events []event.Event) string {
	if len(events) == 0 {
		return "0"
	}

	return events[len(events)-1].Version().String()
}
