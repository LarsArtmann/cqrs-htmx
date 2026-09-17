package dashboardui

import (
	"net/http"
	"sort"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/templ-components/display"
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

// eventSortHeader builds a typed sortable header for the events table,
// preserving the existing server-side sort contract (?sort=&dir= toggling)
// plus any active filter params. The library renders the aria-sort value and
// direction indicator from SortDirection.
func eventSortHeader(basePath, label, column string, s sortState, extraParams string) display.TableHeader {
	direction := display.SortNone
	next := sortAsc

	if s.Column == column {
		if s.Direction == sortAsc {
			direction = display.SortAsc
			next = sortDesc
		} else {
			direction = display.SortDesc
			next = sortAsc
		}
	}

	href := basePath + "/events?sort=" + column + "&dir=" + next
	if extraParams != "" {
		href += "&" + extraParams
	}

	return display.TableHeader{Label: label, Sortable: true, SortDirection: direction, Href: href}
}
