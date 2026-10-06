package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// typedEvents seeds n events of one type on one stream (versions 1..n).
func typedEvents(t *testing.T, n int, eventType string) []event.Event {
	t.Helper()

	streamID := id.NewStreamID()
	events := make([]event.Event, 0, n)

	for i := range n {
		evt, err := event.New(
			event.Type(eventType),
			streamID,
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

// TestEventsIndex_SortedViewIsAWindow pins the M15 contract: the sorted-only
// view (no filters) shows the whole scan window with the truncation badge and
// NO pagination controls — cursors are ignored in this branch, so a Next link
// would reload and re-truncate the same window forever.
func TestEventsIndex_SortedViewIsAWindow(t *testing.T) {
	t.Parallel()

	// 60 events > default page size 50: the pre-M15 behavior rendered a
	// Next link into the same window.
	d := MustNew(Config{Journal: &fakeSeekableJournal{events: typedEvents(t, 60, "user.created")}})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events?sort=time", nil)
	d.eventsIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	if !strings.Contains(body, "sorted view: first 500 events") {
		t.Error("expected the sorted-window truncation badge")
	}

	if strings.Contains(body, "after=") {
		t.Error("sorted window must not emit cursor links")
	}

	if strings.Contains(body, `class="pagination"`) {
		t.Error("sorted window must not render the pagination bar")
	}
}

// TestEventsIndex_FilteredSortedViewStillPaginates pins that windowing only
// applies WITHOUT filters: filter+sort pages normally through cursors.
func TestEventsIndex_FilteredSortedViewStillPaginates(t *testing.T) {
	t.Parallel()

	d := MustNew(Config{Journal: &fakeSeekableJournal{events: typedEvents(t, 60, "user.created")}})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events?type=user.created&sort=type", nil)
	d.eventsIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	// 60 matches > pageSize 50 → a real Next link must exist.
	if !strings.Contains(body, "after=") {
		t.Error("filtered+sorted view should still paginate with cursors")
	}

	if !strings.Contains(body, "Next") {
		t.Error("filtered+sorted view should render the Next control")
	}
}

// TestEventsIndex_UnsortedViewPaginates guards the default branch: no sort,
// no filter — plain cursor pagination.
func TestEventsIndex_UnsortedViewPaginates(t *testing.T) {
	t.Parallel()

	d := MustNew(Config{Journal: &fakeSeekableJournal{events: typedEvents(t, 60, "user.created")}})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	d.eventsIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "after=") {
		t.Error("unfiltered/unsorted view should paginate with cursors")
	}
}
