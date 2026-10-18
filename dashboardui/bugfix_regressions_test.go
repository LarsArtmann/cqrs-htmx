package dashboardui

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
)

// alternatingEvents builds n events alternating between two types so the
// sorted order differs from journal order (same-type sorts are stable
// no-ops and cannot expose cursor-ordering bugs).
func alternatingEvents(t *testing.T, n int, typeA, typeB string) []event.Event {
	t.Helper()

	events := make([]event.Event, 0, n)

	for i := range n {
		eventType := typeA
		if i%2 == 1 {
			eventType = typeB
		}

		evt, err := event.New(
			event.Type(eventType),
			id.NewStreamID(),
			"TestAgg",
			event.Version(i+1),
			map[string]string{"k": "v"},
		)
		if err != nil {
			t.Fatalf("event.New: %v", err)
		}

		events = append(events, evt)
	}

	return events
}

// TestEventsIndex_SortedWindowShowsWholeWindow pins that the sorted-only
// window renders every windowed row: the pageSize truncation used to cut it
// to the page size, stranding the tail behind a disabled pagination bar
// while the badge claimed "first 500 events".
func TestEventsIndex_SortedWindowShowsWholeWindow(t *testing.T) {
	t.Parallel()

	d := MustNew(Config{Journal: &fakeSeekableJournal{events: typedEvents(t, 60, "user.created")}})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events?sort=time", nil)
	d.eventsIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	for version := 51; version <= 60; version++ {
		if !strings.Contains(body, "<td>"+strconv.Itoa(version)+"</td>") {
			t.Errorf("sorted window should show event version %d past the page size", version)
		}
	}
}

// eventIDsOnPage extracts the event detail IDs linked in a rendered page.
var eventDetailHref = regexp.MustCompile(`events/([0-9A-HJKMNP-TV-Z]{26})`)

func eventIDsOnPage(body string) map[string]bool {
	ids := make(map[string]bool)

	for _, match := range eventDetailHref.FindAllStringSubmatch(body, -1) {
		ids[match[1]] = true
	}

	return ids
}

// TestEventsIndex_FilteredSortedCursorAdvancesWithoutOverlap pins that the
// Next cursor comes from the PRE-SORT journal-order tail: a cursor taken
// from the sorted slice re-served overlapping pages with duplicates.
func TestEventsIndex_FilteredSortedCursorAdvancesWithoutOverlap(t *testing.T) {
	t.Parallel()

	events := alternatingEvents(t, 60, "user.alpha", "user.beta")
	d := MustNew(Config{Journal: &fakeSeekableJournal{events: events}})

	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/events?sort=type&limit=25", nil)
	d.eventsIndexHandler(w1, r1)

	if w1.Code != http.StatusOK {
		t.Fatalf("page 1: expected 200, got %d", w1.Code)
	}

	body1 := w1.Body.String()

	after := regexp.MustCompile(`after=([0-9A-HJKMNP-TV-Z]{26})`).FindStringSubmatch(body1)
	if after == nil {
		t.Fatal("page 1 must emit a Next cursor")
	}

	page1IDs := eventIDsOnPage(body1)
	if len(page1IDs) != 25 {
		t.Fatalf("page 1 linked %d events, want 25", len(page1IDs))
	}

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/events?sort=type&limit=25&after="+after[1], nil)
	d.eventsIndexHandler(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("page 2: expected 200, got %d", w2.Code)
	}

	body2 := w2.Body.String()

	page2IDs := eventIDsOnPage(body2)
	if len(page2IDs) != 25 {
		t.Fatalf("page 2 linked %d events, want 25", len(page2IDs))
	}

	for id := range page2IDs {
		if page1IDs[id] {
			t.Fatal("page 2 re-served an event already shown on page 1 (cursor taken from the sorted slice)")
		}
	}

	if strings.Contains(body2, "after=") {
		t.Error("page 2 is the last page; it must not emit another Next cursor")
	}
}

// TestEventsExport_PreservesFilters pins that CSV/JSON exports honor the
// active type filter (they used to silently export the whole journal).
func TestEventsExport_PreservesFilters(t *testing.T) {
	t.Parallel()

	events := alternatingEvents(t, 6, "user.alpha", "user.beta")
	d := MustNew(Config{Journal: &fakeSeekableJournal{events: events}})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events?type=user.alpha&format=csv", nil)
	d.eventsIndexHandler(w, r)

	body := w.Body.String()

	if !strings.Contains(body, "user.alpha") {
		t.Error("filtered export should contain matching events")
	}

	if strings.Contains(body, "user.beta") {
		t.Error("filtered export must exclude non-matching events")
	}
}

// TestEventsPage_ExportLinksCarryFilters pins the export link plumbing: the
// rendered links carry the active filter so clicking them exports what the
// user sees.
func TestEventsPage_ExportLinksCarryFilters(t *testing.T) {
	t.Parallel()

	events := alternatingEvents(t, 4, "user.alpha", "user.beta")
	d := MustNew(Config{Journal: &fakeSeekableJournal{events: events}})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events?type=user.alpha", nil)
	d.eventsIndexHandler(w, r)

	for _, format := range []string{"csv", "json"} {
		want := "type=user.alpha&amp;format=" + format
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("export %s link must carry the active filter (%q missing)", format, want)
		}
	}
}

// readAllOnlyCommandJournal models a CommandJournal without the Seekable
// variant, exercising the ReadAll fallback cursor path.
type readAllOnlyCommandJournal struct{ cmds []*command.PersistedCommand }

func (j readAllOnlyCommandJournal) ReadAll(_ context.Context) ([]*command.PersistedCommand, error) {
	return j.cmds, nil
}

// TestCommandsIndex_JournalFallbackCursorAdvances pins the ReadAll fallback:
// without the in-memory seek the Next link re-served page one forever.
func TestCommandsIndex_JournalFallbackCursorAdvances(t *testing.T) {
	t.Parallel()

	cmds := make([]*command.PersistedCommand, 0, 12)
	for range 12 {
		cmds = append(cmds, makeTestCommand(t))
	}

	d, err := New(Config{
		CommandJournal: readAllOnlyCommandJournal{cmds: cmds},
		Journal:        &fakeSeekableJournal{events: typedEvents(t, 1, "boot")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/commands?limit=10", nil)
	d.commandsIndexHandler(w1, r1)

	after := regexp.MustCompile(`after=([0-9A-HJKMNP-TV-Z]{26})`).FindStringSubmatch(w1.Body.String())
	if after == nil {
		t.Fatal("fallback page 1 must emit a Next cursor")
	}

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/commands?limit=10&after="+after[1], nil)
	d.commandsIndexHandler(w2, r2)

	body2 := w2.Body.String()

	if !strings.Contains(body2, cmds[11].ID().String()) {
		t.Error("page 2 must advance past the cursor (last command missing)")
	}

	if strings.Contains(body2, cmds[0].ID().String()) || strings.Contains(body2, cmds[9].ID().String()) {
		t.Error("page 2 re-served page-1 commands (fallback ignored the after cursor)")
	}
}

// errorStreamReader fails every List call.
type errorStreamReader struct{}

func (errorStreamReader) List(
	_ context.Context,
	_ listing.ListOptions,
) (*listing.Page[listing.StreamListing], error) {
	return nil, errors.New("store down")
}

func (errorStreamReader) ListWithStatus(
	_ context.Context,
	_ listing.ListOptions,
) (*listing.Page[listing.StreamStatus], error) {
	return nil, errors.New("store down")
}

// TestAggregatesIndex_ReaderErrorRendersErrorPanel pins that a failing
// StreamReader surfaces an error page instead of the "No aggregates" empty
// state (an outage must not masquerade as an empty store).
func TestAggregatesIndex_ReaderErrorRendersErrorPanel(t *testing.T) {
	t.Parallel()

	d, err := New(Config{
		StreamReader: errorStreamReader{},
		Journal:      &fakeSeekableJournal{events: typedEvents(t, 1, "boot")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/aggregates", nil)
	d.aggregatesIndexHandler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on reader failure, got %d", w.Code)
	}
}

// TestAggregatesIndex_MalformedCursorRejected pins the 400 contract shared
// by every other paginated index.
func TestAggregatesIndex_MalformedCursorRejected(t *testing.T) {
	t.Parallel()

	d, err := New(Config{
		StreamReader: errorStreamReader{},
		Journal:      &fakeSeekableJournal{events: typedEvents(t, 1, "boot")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/aggregates?after=not-a-stream-id", nil)
	d.aggregatesIndexHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed cursor, got %d", w.Code)
	}
}

// TestConfig_NegativePageSizeClamped pins that a negative PageSize cannot
// reach the handlers (it used to panic the first paginated request on a
// negative slice bound).
func TestConfig_NegativePageSizeClamped(t *testing.T) {
	t.Parallel()

	config, err := Config{
		Journal:  &fakeSeekableJournal{events: typedEvents(t, 3, "user.created")},
		PageSize: -5,
	}.Validate()
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if config.PageSize <= 0 {
		t.Errorf("PageSize = %d, want a positive default", config.PageSize)
	}

	d := MustNew(Config{
		Journal:  &fakeSeekableJournal{events: typedEvents(t, 3, "user.created")},
		PageSize: -5,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	d.eventsIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("negative-configured dashboard should serve events, got %d", w.Code)
	}
}
