package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// fakeHubStats is a static SSEHubStatsSource for the card tests.
type fakeHubStats struct {
	subs  int
	drops int64
}

func (f fakeHubStats) SubscriberCount() int { return f.subs }
func (f fakeHubStats) DroppedEvents() int64 { return f.drops }

// TestDashboard_SSEHubStatsCards pins the Config.SSEHubStats contract: when a
// source is wired, the dashboard (and its polled stats partial) carry the two
// operational cards — live subscribers and lifetime slow-consumer drops — and
// a nonzero drop count switches the drop card to the attention (yellow) tone.
func TestDashboard_SSEHubStatsCards(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	h, _ := newTestPanel(t, user, Config{
		Service:     svc,
		Authorizer:  RequireAuthenticated(),
		SSEHubStats: fakeHubStats{subs: 3, drops: 7},
	})

	for _, tc := range []struct {
		name string
		path string
	}{
		{"dashboard page", "/admin/"},
		{"polled stats partial", "/admin/partials/stats"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("%s: status %d, body %s", tc.name, rec.Code, rec.Body.String())
			}

			html := rec.Body.String()
			for _, want := range []string{"SSE live", "Dropped events", ">3<", ">7<"} {
				if !strings.Contains(html, want) {
					t.Errorf("%s missing %q", tc.name, want)
				}
			}

			if !strings.Contains(html, "bg-yellow-50") {
				t.Errorf("%s: nonzero drops must render the yellow attention tone", tc.name)
			}
		})
	}
}

// TestDashboard_SSEHubStatsAbsentByDefault: nil SSEHubStats (the default)
// adds no hub cards — the dashboard must not advertise an SSE feed the
// consumer never wired.
func TestDashboard_SSEHubStatsAbsentByDefault(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard: status %d", rec.Code)
	}

	if html := rec.Body.String(); strings.Contains(html, "SSE live") || strings.Contains(html, "Dropped events") {
		t.Error("hub stats cards rendered without Config.SSEHubStats — nil must add nothing")
	}
}

// TestDashboard_SSEHubStatsZeroDropsGreenTone: zero drops is the healthy
// state and renders green, not the attention tone.
func TestDashboard_SSEHubStatsZeroDropsGreenTone(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	h, _ := newTestPanel(t, user, Config{
		Service:     svc,
		Authorizer:  RequireAuthenticated(),
		SSEHubStats: fakeHubStats{subs: 0, drops: 0},
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard: status %d", rec.Code)
	}

	html := rec.Body.String()
	if !strings.Contains(html, ">0<") {
		t.Error("zero subscribers should render as 0")
	}

	if strings.Contains(html, "bg-yellow-50") {
		t.Error("zero drops must not render the yellow attention tone")
	}
}
