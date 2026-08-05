package core

import (
	"context"
	"net/http"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// FilterScanLimit is the maximum number of events scanned for in-memory
// filtering when event-type/stream-type/stream-ID filters are active.
const FilterScanLimit = 500

// EventFilter holds in-memory filter criteria for the event browser.
type EventFilter struct {
	Type       string
	StreamType string
	StreamID   string
}

// Active returns true if any filter field is set.
func (f EventFilter) Active() bool {
	return f.Type != "" || f.StreamType != "" || f.StreamID != ""
}

// Matches returns true if the event matches all active filter criteria.
func (f EventFilter) Matches(evt event.Event) bool {
	if f.Type != "" && string(evt.Type()) != f.Type {
		return false
	}

	if f.StreamType != "" && string(evt.StreamType()) != f.StreamType {
		return false
	}

	if f.StreamID != "" && evt.StreamID().String() != f.StreamID {
		return false
	}

	return true
}

// ExtraParams builds a query-string fragment preserving active filters
// across pagination links (e.g., "type=user.created&streamType=User").
func (f EventFilter) ExtraParams() string {
	if !f.Active() {
		return ""
	}

	var parts []string
	if f.Type != "" {
		parts = append(parts, "type="+f.Type)
	}

	if f.StreamType != "" {
		parts = append(parts, "streamType="+f.StreamType)
	}

	if f.StreamID != "" {
		parts = append(parts, "streamID="+f.StreamID)
	}

	return strings.Join(parts, "&")
}

// ParseEventFilter extracts event filter criteria from the request query string.
func ParseEventFilter(r *http.Request) EventFilter {
	q := r.URL.Query()

	return EventFilter{
		Type:       q.Get("type"),
		StreamType: q.Get("streamType"),
		StreamID:   q.Get("streamID"),
	}
}

// LoadRecentEvents reads up to limit events starting after the given cursor.
// Uses SeekableJournal when available (efficient cursor-based reads), falls
// back to Journal.ReadAll when only a basic Journal is configured.
func LoadRecentEvents(ctx context.Context, cfg Config, after id.EventID, limit int) ([]event.Event, error) {
	if cfg.SeekableJournal != nil {
		events, err := cfg.SeekableJournal.ReadFrom(ctx, after, limit)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.recent_events.read_failed", "read recent events")
		}

		return events, nil
	}

	if cfg.Journal != nil {
		all, err := cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.recent_events.read_all_failed", "read all events")
		}

		if len(all) > limit {
			all = all[:limit]
		}

		return all, nil
	}

	return nil, nil
}

// LoadFilteredEvents reads a generous batch from the journal and applies
// in-memory filters, returning up to pageSize+1 results (the +1 is for
// HasMore detection). When filters are active we scan up to FilterScanLimit
// raw events per page to find matches.
func LoadFilteredEvents(
	ctx context.Context,
	cfg Config,
	after id.EventID,
	filter EventFilter,
	pageSize int,
) ([]event.Event, error) {
	rawLimit := max(FilterScanLimit, pageSize+1)

	var raw []event.Event

	var err error

	if cfg.SeekableJournal != nil {
		raw, err = cfg.SeekableJournal.ReadFrom(ctx, after, rawLimit)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.filtered_events.read_failed", "read events for filtering")
		}
	} else if cfg.Journal != nil {
		raw, err = cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.filtered_events.read_all_failed", "read all events for filtering")
		}

		if len(raw) > rawLimit {
			raw = raw[:rawLimit]
		}
	}

	var filtered []event.Event

	for _, evt := range raw {
		if filter.Matches(evt) {
			filtered = append(filtered, evt)
			if len(filtered) > pageSize {
				break
			}
		}
	}

	return filtered, nil
}

// LoadEventByID finds a single event by its EventID. Uses EventByIDLoader
// when available (O(1)), falls back to scanning the SeekableJournal, then
// the Journal.
func LoadEventByID(ctx context.Context, cfg Config, eventID id.EventID) (event.Event, error) { //nolint:cyclop // journal fallback
	if cfg.EventByIDLoader != nil {
		evt, err := cfg.EventByIDLoader.LoadByEventID(ctx, eventID)
		if err != nil {
			var zero event.Event

			return zero, errorfamily.WrapInfrastructure(err,
				"dashboardui.event_detail.load_failed", "load event by ID")
		}

		return evt, nil
	}

	if cfg.SeekableJournal != nil {
		const scanLimit = 5000

		var after id.EventID

		for {
			batch, err := cfg.SeekableJournal.ReadFrom(ctx, after, scanLimit)
			if err != nil {
				return nil, errorfamily.WrapInfrastructure(err,
					"dashboardui.event_detail.scan_failed", "scan journal for event")
			}

			for _, evt := range batch {
				if evt.ID() == eventID {
					return evt, nil
				}
			}

			if len(batch) < scanLimit {
				break
			}

			after = batch[len(batch)-1].ID()
		}

		return nil, errorfamily.Newf(event.Rejection,
			"dashboardui.event_detail.not_found", "event %s not found in journal scan", eventID)
	}

	if cfg.Journal != nil {
		all, err := cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.event_detail.read_failed", "read journal")
		}

		for _, evt := range all {
			if evt.ID() == eventID {
				return evt, nil
			}
		}
	}

	return nil, errorfamily.Newf(event.Infrastructure,
		"dashboardui.event_detail.no_source", "no event source available to load event %s", eventID)
}

// FindEventNeighbors scans recent events to find the previous and next event
// IDs relative to eventID. Returns empty strings if not found or at the
// boundary. This is a best-effort scan limited to the most recent batch.
func FindEventNeighbors(ctx context.Context, cfg Config, eventID id.EventID) (string, string) {
	const neighborScanLimit = 500

	var events []event.Event

	var err error

	var prevID, nextID string

	if cfg.SeekableJournal != nil {
		events, err = cfg.SeekableJournal.ReadFrom(ctx, id.EventID{}, neighborScanLimit)
	} else if cfg.Journal != nil {
		events, err = cfg.Journal.ReadAll(ctx)
		if err == nil && len(events) > neighborScanLimit {
			events = events[:neighborScanLimit]
		}
	}

	if err != nil || len(events) == 0 {
		return "", ""
	}

	for i, evt := range events {
		if evt.ID() == eventID {
			if i > 0 {
				prevID = events[i-1].ID().String()
			}

			if i < len(events)-1 {
				nextID = events[i+1].ID().String()
			}

			return prevID, nextID
		}
	}

	return "", ""
}

// ListStreamsPaged loads a cursor-paginated page of stream listings.
// Returns the listings and the pagination state for rendering Prev/Next
// controls.
func ListStreamsPaged(r *http.Request, cfg Config) ([]listing.StreamListing, PageState) {
	pageSize := ParsePageSize(r, cfg.PageSize)
	afterCursor, prevHistory, hasPrev := ParseCursorParams(r)

	if cfg.StreamReader == nil {
		return nil, PageState{PageSize: pageSize}
	}

	opts := listing.ListOptions{Limit: uint(pageSize + 1)}

	if afterCursor != "" {
		parsed, err := id.ParseStreamID(afterCursor)
		if err == nil {
			opts.After = parsed
		}
	}

	page, err := cfg.StreamReader.List(r.Context(), opts)
	if err != nil || page == nil {
		return nil, PageState{PageSize: pageSize}
	}

	hasMore := len(page.Items) > pageSize

	listings := page.Items
	if hasMore {
		listings = listings[:pageSize]
	}

	var nextCursor string
	if hasMore && len(listings) > 0 {
		nextCursor = listings[len(listings)-1].ID.String()
	}

	return listings, PageState{
		HasNext:     hasMore,
		NextCursor:  nextCursor,
		PageSize:    pageSize,
		HasPrev:     hasPrev,
		After:       afterCursor,
		PrevHistory: prevHistory,
	}
}
