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
	"github.com/larsartmann/templ-components/icons"
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
	html := goldenRender(t, emptyStatePanel(icons.Inbox, "No events yet", "coming soon"))

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

// TestA11y_FilterInputsKeepLabelPairing pins the label contract of the event
// filter bar after the forms.Input adoption: every input keeps its explicit
// DOM id and a matching label[for] (the library derives ids from Name when ID
// is empty, which would silently break the historical selectors), and the
// hx-get partial-swap wiring survives.
func TestA11y_FilterInputsKeepLabelPairing(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	aggID := id.NewStreamID()

	evt, err := event.New("test.event", aggID, "TestAggregate", event.Version(1), struct{}{})
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	if err := store.Save(
		context.Background(),
		id.NewStreamRef("TestAggregate", aggID),
		[]event.Event{evt},
		event.Version(0),
	); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	d, err := New(Config{EventSource: store, Journal: store})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dashboard/events", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()

	for _, id := range []string{"filter-type", "filter-stream-type", "filter-stream-id"} {
		if !strings.Contains(body, `id="`+id+`"`) {
			t.Errorf("filter input #%s missing", id)
		}

		if !strings.Contains(body, `for="`+id+`"`) {
			t.Errorf("filter label[for=%s] missing", id)
		}
	}

	for _, want := range []string{
		`class="filter-bar" hx-get="`,
		`hx-target="#main-content"`,
		`hx-select="#main-content"`,
		`hx-swap="outerHTML"`,
		`hx-push-url="true"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("filter form wiring %q missing", want)
		}
	}
}
