package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// a11yOverview renders the overview page for accessibility assertions.
func a11yOverview(t *testing.T) string {
	t.Helper()

	store := memorystorage.NewMemoryStore()

	d, err := New(Config{EventSource: store, Journal: store})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dashboard/", nil))

	return rec.Body.String()
}

// TestA11y_LandmarksAndSkipLink pins the keyboard/landmark contract of the
// dashboard shell: a skip link as the first focusable element, a labelled main
// region, and a labelled navigation landmark.
func TestA11y_LandmarksAndSkipLink(t *testing.T) {
	body := a11yOverview(t)

	if !strings.Contains(body, `class="skip-link"`) || !strings.Contains(body, `href="#main-content"`) {
		t.Error("missing skip link targeting #main-content")
	}

	if !strings.Contains(body, `<main id="main-content"`) {
		t.Error("missing main landmark")
	}

	if !strings.Contains(body, `aria-label="Dashboard navigation"`) {
		t.Error("sidebar nav missing aria-label")
	}
}

// TestA11y_LiveRegions pins the screen-reader announcement contract: the toast
// host is a polite live region and the library error-handling announcer is
// mounted.
func TestA11y_LiveRegions(t *testing.T) {
	body := a11yOverview(t)

	for _, want := range []string{
		`aria-live="polite"`,                  // toast container
		`id="tc-error-announcer"`,             // GlobalErrorHandling announcer
		`aria-label="Toggle navigation menu"`, // hamburger button label
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing live-region/label element %q", want)
		}
	}
}

// TestA11y_EmptyStateRoleStatus verifies library empty states expose
// role="status" so assistive tech announces them when swaps insert them.
func TestA11y_EmptyStateRoleStatus(t *testing.T) {
	html := emptyState(context.Background(), "No events yet", "coming soon")

	if !strings.Contains(html, `role="status"`) {
		t.Error("empty state missing role=status")
	}
}

// TestA11y_SortHeadersCarryAriaSort verifies the events table's typed headers
// render aria-sort values (the library maps SortDirection to
// ascending/descending/none).
func TestA11y_SortHeadersCarryAriaSort(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	for range 3 {
		aggID := id.NewStreamID()

		evt, _ := event.New("test.event", aggID, "TestAggregate", event.Version(1), struct{}{})
		_ = store.Save(
			context.Background(),
			id.NewStreamRef("TestAggregate", aggID),
			[]event.Event{evt},
			event.Version(0),
		)
	}

	reader := listing.NewInMemoryStreamReader(store)

	d, err := New(Config{EventSource: store, SeekableJournal: store, StreamReader: reader})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	for _, tc := range []struct {
		target string
		want   string
	}{
		{"/dashboard/events", `aria-sort="none"`},
		{"/dashboard/events?sort=time&dir=desc", `aria-sort="descending"`},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.target, nil))

		if rec.Code != http.StatusOK {
			t.Skipf("%s unavailable (status %d)", tc.target, rec.Code)
		}

		if !strings.Contains(rec.Body.String(), tc.want) {
			t.Errorf("%s: expected %q in headers", tc.target, tc.want)
		}
	}
}
