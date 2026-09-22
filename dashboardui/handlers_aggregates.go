package dashboardui

import (
	"net/http"
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
)

// ===== Aggregate Browser =====

func (d *Dashboard) aggregatesIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Aggregates", "/aggregates", r)

	pageSize := parsePageSize(r, d.config.PageSize)
	afterCursor, prevHistory, hasPrev := parseCursorParams(r)

	var (
		listings []listing.StreamListing
		hasMore  bool
	)

	if d.config.StreamReader != nil { //nolint:nestif // optional data source branching
		opts := listing.ListOptions{Limit: uint(pageSize + 1)}

		if afterCursor != "" {
			parsed, err := id.ParseStreamID(afterCursor)
			if err == nil {
				opts.After = parsed
			}
		}

		page, err := d.config.StreamReader.List(r.Context(), opts)
		if err == nil && page != nil {
			hasMore = len(page.Items) > pageSize
			if hasMore {
				listings = page.Items[:pageSize]
			} else {
				listings = page.Items
			}
		}
	}

	var nextCursor string
	if hasMore && len(listings) > 0 {
		nextCursor = listings[len(listings)-1].ID.String()
	}

	renderPage(w, r, aggregatesPage(p, listings, paginationState{
		HasNext:     hasMore,
		NextCursor:  nextCursor,
		PageSize:    pageSize,
		HasPrev:     hasPrev,
		After:       afterCursor,
		PrevHistory: prevHistory,
	}.WithCountInfo(len(listings))))
}

func (d *Dashboard) aggregateDetailHandler(w http.ResponseWriter, r *http.Request) {
	ref, events, ok := d.loadStreamFromRequest(w, r)
	if !ok {
		return
	}

	p := d.page("Aggregate: "+streamTitlePath(ref), "/aggregates", r)

	page := d.aggregateTimelinePagination(r)
	pagedEvents, hasNext := paginateEventsByVersion(events, page)

	renderPage(w, r, aggregateDetailPage(
		p,
		ref,
		events,
		pagedEvents,
		timelinePageState(events, pagedEvents, page, hasNext),
	))
}

// aggregateTimelinePagination computes the pagination state for the event
// timeline on the aggregate detail page. Uses version number as cursor
// (?after=3 means start after version 3). The timeline is paginated in-memory
// since EventSource.Load returns all events for a single stream.
func (d *Dashboard) aggregateTimelinePagination(r *http.Request) paginationState {
	pageSize := parsePageSize(r, d.config.PageSize)
	afterCursor, prevHistory, _ := parseCursorParams(r)

	afterVersion := uint64(0)
	if afterCursor != "" {
		afterVersion, _ = strconv.ParseUint(afterCursor, 10, 64)
	}

	return paginationState{
		PageSize:    pageSize,
		After:       afterCursor,
		PrevHistory: prevHistory,
		HasPrev:     afterVersion > 0,
	}
}

// paginateEventsByVersion slices the events array for in-memory pagination.
// Uses the After cursor as a version number: events with version > After are
// returned, up to PageSize+1 (the +1 is for HasMore detection).
func paginateEventsByVersion(events []event.Event, page paginationState) ([]event.Event, bool) {
	afterVersion := uint64(0)
	if page.After != "" {
		afterVersion, _ = strconv.ParseUint(page.After, 10, 64)
	}

	start := 0

	for i, evt := range events {
		if evt.Version().UInt64() > afterVersion {
			start = i

			break
		}

		if i == len(events)-1 {
			start = len(events)
		}
	}

	if start >= len(events) {
		return nil, false
	}

	end := min(start+page.PageSize+1, len(events))
	paged := events[start:end]

	hasMore := len(paged) > page.PageSize
	if hasMore {
		paged = paged[:page.PageSize]
	}

	return paged, hasMore
}
