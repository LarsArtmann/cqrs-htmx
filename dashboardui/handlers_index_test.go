package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
)

// fakeStreamReader implements listing.StreamReader for index handler tests.
type fakeStreamReader struct {
	items []listing.StreamListing
}

func (f *fakeStreamReader) List(
	_ context.Context,
	_ listing.ListOptions,
) (*listing.Page[listing.StreamListing], error) {
	return &listing.Page[listing.StreamListing]{Items: f.items, HasMore: false}, nil
}

func (f *fakeStreamReader) ListWithStatus(
	_ context.Context,
	_ listing.ListOptions,
) (*listing.Page[listing.StreamStatus], error) {
	statuses := make([]listing.StreamStatus, len(f.items))

	for i, item := range f.items {
		statuses[i] = listing.StreamStatus{Ref: item, Status: event.TombstoneActive}
	}

	return &listing.Page[listing.StreamStatus]{Items: statuses, HasMore: false}, nil
}

var _ listing.StreamReader = (*fakeStreamReader)(nil)

// cursorStreamReader is a listing.StreamReader that respects Limit and After
// for cursor-based pagination tests. Items must be pre-sorted by ID.
type cursorStreamReader struct {
	items []listing.StreamListing
}

func (c *cursorStreamReader) List(
	_ context.Context,
	opts listing.ListOptions,
) (*listing.Page[listing.StreamListing], error) {
	start := 0

	if opts.After.String() != "" {
		for i, item := range c.items {
			if item.ID.String() == opts.After.String() {
				start = i + 1

				break
			}
		}
	}

	end := start + int(opts.Limit)
	if end > len(c.items) {
		end = len(c.items)
	}

	if start >= len(c.items) {
		return &listing.Page[listing.StreamListing]{Items: nil, HasMore: false}, nil
	}

	page := listing.Page[listing.StreamListing]{
		Items:   c.items[start:end],
		HasMore: end < len(c.items),
	}

	return &page, nil
}

func (c *cursorStreamReader) ListWithStatus(
	ctx context.Context,
	opts listing.ListOptions,
) (*listing.Page[listing.StreamStatus], error) {
	page, err := c.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	statuses := make([]listing.StreamStatus, len(page.Items))
	for i, item := range page.Items {
		statuses[i] = listing.StreamStatus{Ref: item, Status: event.TombstoneActive}
	}

	return &listing.Page[listing.StreamStatus]{Items: statuses, HasMore: page.HasMore}, nil
}

var _ listing.StreamReader = (*cursorStreamReader)(nil)

func nStreamListings(n int) []listing.StreamListing {
	items := make([]listing.StreamListing, n)
	for i := range items {
		items[i] = listing.StreamListing{
			ID: id.NewStreamID(), Type: "user", Version: event.Version(i + 1),
			EventCount: uint(i + 1), LastEventAt: time.Now(),
		}
	}

	return items
}

func streamListings() []listing.StreamListing {
	return []listing.StreamListing{
		{
			ID: id.NewStreamID(), Type: "user", Version: event.Version(3),
			EventCount: 3, LastEventAt: time.Time{},
		},
		{
			ID: id.NewStreamID(), Type: "tenant", Version: event.Version(1),
			EventCount: 1, LastEventAt: time.Time{},
		},
	}
}

// --- Time-Travel Index Handler ---

func TestTimeTravelIndexHandler_NoStreamReader(t *testing.T) {
	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/time-travel", nil)
	d.timeTravelIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "No aggregates found") {
		t.Fatalf("expected empty state, got body: %s", w.Body.String())
	}
}

func TestTimeTravelIndexHandler_WithListings(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal:      &stubJournal{},
		StreamReader: &fakeStreamReader{items: streamListings()},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/time-travel", nil)
	d.timeTravelIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	if !strings.Contains(body, "user") {
		t.Fatalf("expected stream type 'user' in body")
	}

	if !strings.Contains(body, "tenant") {
		t.Fatalf("expected stream type 'tenant' in body")
	}

	if !strings.Contains(body, "/time-travel/user/") {
		t.Fatalf("expected time-travel detail link")
	}
}

// --- Snapshots Index Handler ---

func TestSnapshotsIndexHandler_NoStreamReader(t *testing.T) {
	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/snapshots", nil)
	d.snapshotsIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "No aggregates found") {
		t.Fatalf("expected empty state, got body: %s", w.Body.String())
	}
}

func TestSnapshotsIndexHandler_WithListings(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal: &stubJournal{},
		StreamReader: &fakeStreamReader{items: []listing.StreamListing{
			{
				ID: id.NewStreamID(), Type: "user", Version: event.Version(5),
				EventCount: 5, LastEventAt: time.Time{},
			},
		}},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/snapshots", nil)
	d.snapshotsIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	if !strings.Contains(body, "user") {
		t.Fatalf("expected stream type 'user' in body")
	}

	if !strings.Contains(body, "/snapshots/user/") {
		t.Fatalf("expected snapshot detail link")
	}
}

func TestSnapshotsIndexHandler_RendersVersion(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal: &stubJournal{},
		StreamReader: &fakeStreamReader{items: []listing.StreamListing{
			{
				ID: id.NewStreamID(), Type: "order", Version: event.Version(42),
				EventCount: 42, LastEventAt: time.Time{},
			},
		}},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/snapshots", nil)
	d.snapshotsIndexHandler(w, r)

	if !strings.Contains(w.Body.String(), "42") {
		t.Fatalf("expected version 42 in body")
	}
}

// --- Time-Travel & Snapshots Pagination ---

func TestTimeTravelIndexHandler_Pagination(t *testing.T) {
	items := nStreamListings(5)
	d := mustTestDashboardWithConfig(t, Config{
		Journal:      &stubJournal{},
		StreamReader: &cursorStreamReader{items: items},
		PageSize:     2,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/time-travel", nil)
	d.timeTravelIndexHandler(w, r)

	body := w.Body.String()

	if !strings.Contains(body, "Next") {
		t.Fatalf("expected Next link with 5 items and PageSize 2, got: %s", body)
	}
}

func TestSnapshotsIndexHandler_Pagination(t *testing.T) {
	items := nStreamListings(5)
	d := mustTestDashboardWithConfig(t, Config{
		Journal:      &stubJournal{},
		StreamReader: &cursorStreamReader{items: items},
		PageSize:     2,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/snapshots", nil)
	d.snapshotsIndexHandler(w, r)

	body := w.Body.String()

	if !strings.Contains(body, "Next") {
		t.Fatalf("expected Next link with 5 items and PageSize 2, got: %s", body)
	}
}
