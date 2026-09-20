package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// TestStatsPartialRendered verifies the polled stats endpoint returns the
// PolledRegion fragment itself (hx-swap=outerHTML requires the region in the
// response so polling continues) with the poll trigger wired.
func TestStatsPartialRendered(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/partials/stats", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("stats partial: status %d, body %s", rec.Code, rec.Body.String())
	}
	html := rec.Body.String()

	for _, want := range []string{
		`hx-get="/admin/partials/stats"`,
		`hx-trigger="every 30s"`,
		`hx-swap="outerHTML"`,
		`Updated`, // ShowTimestamp footer
		`Users`,   // a stat card label (super-admin mode)
	} {
		if !strings.Contains(html, want) {
			t.Errorf("stats partial missing %q", want)
		}
	}
	if strings.Contains(html, "<html") {
		t.Error("stats partial must be a fragment, not a full page")
	}
}

// TestStatsPartialAuthGate verifies the endpoint is session-gated like the
// dashboard page (401 without a user).
func TestStatsPartialAuthGate(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	panel.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/partials/stats", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated stats partial: status %d, want 401", rec.Code)
	}
}

// TestDashboardPollsStats verifies the dashboard page embeds the polled
// region pointing at the partial endpoint.
func TestDashboardPollsStats(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	html := rec.Body.String()
	if !strings.Contains(html, `hx-get="/admin/partials/stats"`) {
		t.Error("dashboard missing polled stats region")
	}
	if !strings.Contains(html, `every 30s`) {
		t.Error("dashboard stats region missing 30s poll trigger")
	}
}
