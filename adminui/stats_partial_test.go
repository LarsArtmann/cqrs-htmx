package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
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

// TestDashboardQuickActions_ModeGate pins the mode honesty of the dashboard
// quick actions. Super-admin sees "Create user" / "Create tenant" (links to
// the existing /users/new and /tenants/new forms) on the page AND in the
// polled stats partial — the row renders inside the region, so a 30s refresh
// keeps it instead of dropping it. Tenant-admin mode serves no creation
// route, so it must render neither action anywhere: a dead control would be
// a lie.
func TestDashboardQuickActions_ModeGate(t *testing.T) {
	admin := mustUser(t, "admin@example.com")
	superPanel, _ := newTestPanel(t, admin)

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	tenantPanel, err := New(Config{
		Service:    svc,
		Mode:       ModeTenantAdmin,
		TenantID:   identitymodel.NewTenantID("acme"),
		Authorizer: RequireAuthenticated(),
	})
	if err != nil {
		t.Fatal(err)
	}
	ops := mustUser(t, "ops@acme.com")
	mux := http.NewServeMux()
	mux.Handle("/admin/", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(usermgmt.WithUser(r.Context(), ops)))
		})
	}(http.StripPrefix("/admin", tenantPanel.Handler())))

	for _, path := range []string{"/admin/", "/admin/partials/stats"} {
		rec := httptest.NewRecorder()
		superPanel.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("super-admin %s: status %d", path, rec.Code)
		}
		body := rec.Body.String()
		for _, want := range []string{
			"Create user",
			"Create tenant",
			`href="/admin/users/new"`,
			`href="/admin/tenants/new"`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("super-admin %s missing %q", path, want)
			}
		}

		rec2 := httptest.NewRecorder()
		mux.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, path, nil))
		if rec2.Code != http.StatusOK {
			t.Fatalf("tenant-admin %s: status %d", path, rec2.Code)
		}
		body2 := rec2.Body.String()
		for _, banned := range []string{"Create user", "Create tenant", "/users/new", "/tenants/new"} {
			if strings.Contains(body2, banned) {
				t.Errorf("tenant-admin %s must not render %q", path, banned)
			}
		}
	}
}
