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
		http.Error(w, "invalid stream reference: "+err.Error(), http.StatusBadRequest)

		return id.StreamRef{}, nil, false
	}

	events, err := d.cfg.EventSource.Load(r.Context(), ref)
	if err != nil {
		http.Error(w, "failed to load aggregate: "+err.Error(), http.StatusInternalServerError)

		return id.StreamRef{}, nil, false
	}

	return ref, events, true
}
// latestVersion returns the last event version, or "0" for an empty stream.
func latestVersion(events []event.Event) string {
	if len(events) == 0 {
		return "0"
	}

	return events[len(events)-1].Version().String()
}

// Ensure we use the imports.
var (
	_ = id.NewStreamID
	_ = event.Type("")
)
