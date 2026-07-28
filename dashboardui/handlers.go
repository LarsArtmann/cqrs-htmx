package dashboardui

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	errorfamily "github.com/larsartmann/go-error-family"
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

func streamPathValues(r *http.Request) (string, string) {
	return r.PathValue("type"), r.PathValue("id")
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
	_ = id.NewAggregateID
	_ = event.Type("")
)
