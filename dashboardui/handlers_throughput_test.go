package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// throughputPanelEvent builds one journaled event at a fixed time.
func throughputPanelEvent(t *testing.T, at time.Time) event.Event {
	t.Helper()

	evt, err := event.New(
		"user.created", id.NewStreamID(), "User", event.Version(1),
		struct{}{},
		event.WithOccurredAt(at),
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	return evt
}

func TestThroughputPanel_RendersChart(t *testing.T) {
	t.Parallel()

	d := mustTestDashboardWithConfig(t, Config{Journal: &fakeSeekableJournal{
		events: []event.Event{
			throughputPanelEvent(t, time.Now().Add(-90*time.Minute)),
			throughputPanelEvent(t, time.Now().Add(-30*time.Minute)),
			throughputPanelEvent(t, time.Now().Add(-1*time.Minute)),
		},
	}})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/throughput", nil)
	d.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /throughput: got %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{"Event Throughput", "<svg", "3 in window"} {
		if !strings.Contains(body, want) {
			t.Errorf("throughput page missing %q:\n%s", want, body)
		}
	}
}

func TestThroughputPanel_EmptyJournalRendersEmptyState(t *testing.T) {
	t.Parallel()

	d := mustTestDashboardWithConfig(t, Config{Journal: &fakeSeekableJournal{}})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/throughput", nil)
	d.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /throughput: got %d, want 200", rec.Code)
	}

	if body := rec.Body.String(); !strings.Contains(body, "No events to chart") {
		t.Errorf("empty journal should render the empty state, got:\n%s", body)
	}
}

func TestThroughputRoute_GatedByEventRead(t *testing.T) {
	t.Parallel()

	// EventSource-only: no Journal or SeekableJournal, so HasEventRead is
	// false and the panel must be absent from the manifest.
	eventSourceOnly := mustTestDashboardWithConfig(t, Config{
		EventSource: &fakeEventSource{},
	})
	for _, route := range eventSourceOnly.Routes() {
		if route.Pattern == "/throughput" {
			t.Error("EventSource-only manifest must not serve /throughput")
		}
	}

	// Journal-bearing: present.
	withJournal := mustTestDashboardWithConfig(t, Config{Journal: &fakeSeekableJournal{}})
	found := false

	for _, route := range withJournal.Routes() {
		if route.Pattern == "/throughput" {
			found = true
		}
	}

	if !found {
		t.Error("journal-bearing manifest must serve /throughput")
	}
}

func TestParseThroughputBuckets(t *testing.T) {
	t.Parallel()

	req := func(query string) *http.Request {
		return httptest.NewRequest(http.MethodGet, "/throughput"+query, nil)
	}

	cases := []struct {
		query string
		want  int
	}{
		{"", throughputBuckets},
		{"?buckets=24", 24},
		{"?buckets=2", minThroughputBuckets},   // clamped up
		{"?buckets=500", maxThroughputBuckets}, // clamped down
		{"?buckets=abc", throughputBuckets},    // unparseable falls back
	}

	for _, tc := range cases {
		if got := parseThroughputBuckets(req(tc.query), throughputBuckets); got != tc.want {
			t.Errorf("parseThroughputBuckets(%q) = %d, want %d", tc.query, got, tc.want)
		}
	}
}
