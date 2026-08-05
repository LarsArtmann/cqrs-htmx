package dashboardui

import (
	"net/http"
	"sort"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// sortState tracks the active sort column and direction for table headers.
type sortState struct {
	Column    string // e.g., "time", "type", "version"
	Direction string // sortAsc or sortDesc
}

const (
	sortAsc  = "asc"
	sortDesc = "desc"
)

func (s sortState) Active() bool {
	return s.Column != ""
}

// sortParam returns the query-string fragment for this sort state.
func (s sortState) extraParams() string {
	if !s.Active() {
		return ""
	}

	return "sort=" + s.Column + "&dir=" + s.Direction
}

// parseSort extracts the sort column and direction from query params.
func parseSort(r *http.Request) sortState {
	col := r.URL.Query().Get("sort")
	if col == "" {
		return sortState{}
	}

	dir := r.URL.Query().Get("dir")
	if dir != sortAsc && dir != sortDesc {
		dir = sortAsc
	}

	return sortState{Column: col, Direction: dir}
}

// sortEvents sorts a slice of events in-memory by the given column/direction.
func sortEvents(events []event.Event, s sortState) {
	if !s.Active() {
		return
	}

	switch s.Column {
	case "time":
		sort.SliceStable(events, func(i, j int) bool {
			if s.Direction == sortAsc {
				return events[i].OccurredAt().Before(events[j].OccurredAt())
			}

			return events[i].OccurredAt().After(events[j].OccurredAt())
		})
	case "type":
		sort.SliceStable(events, func(i, j int) bool {
			a, b := string(events[i].Type()), string(events[j].Type())
			if s.Direction == sortAsc {
				return a < b
			}

			return a > b
		})
	case "streamType":
		sort.SliceStable(events, func(i, j int) bool {
			a, b := string(events[i].StreamType()), string(events[j].StreamType())
			if s.Direction == sortAsc {
				return a < b
			}

			return a > b
		})
	case "version":
		sort.SliceStable(events, func(i, j int) bool {
			a, b := events[i].Version().UInt64(), events[j].Version().UInt64()
			if s.Direction == sortAsc {
				return a < b
			}

			return a > b
		})
	}
}

// sortHeader renders a clickable sortable column header with an indicator.
func sortHeader(basePath, label, column string, s sortState, extraParams string) string {
	direction := sortAsc
	indicator := ""

	if s.Column == column {
		if s.Direction == sortAsc {
			indicator = " \u25B2" // ▲
			direction = sortDesc
		} else {
			indicator = " \u25BC" // ▼
			direction = sortAsc
		}
	}

	query := "sort=" + column + "&dir=" + direction
	if extraParams != "" {
		query += "&" + extraParams
	}

	return `<th scope="col"><a href="` + basePath + "/events?" + query + `" class="sort-header">` +
		label + indicator + `</a></th>`
}
