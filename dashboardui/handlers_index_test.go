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
		Journal:      &stubJournal{},
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
		Journal:      &stubJournal{},
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
