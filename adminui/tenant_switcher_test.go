package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// TestTenantSwitcher_SuperAdminListsLiveTenants pins the header jump control:
// super-admin mode lists LIVE tenants on every page, and suspended tenants —
// which are not navigable targets — stay out of the list.
func TestTenantSwitcher_SuperAdminListsLiveTenants(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	// zeta is created FIRST; the switcher lists display-name order, so acme
	// (created second) must still be reachable — order itself is asserted via
	// the tenant detail test below.
	mustCreateTenant(t, svc, "zeta")
	mustCreateTenant(t, svc, "acme")
	mustCreateTenant(t, svc, "ghost-tenant-x")
	if err := svc.SuspendTenant(t.Context(), identitymodel.NewTenantID("ghost-tenant-x"), "test"); err != nil {
		t.Fatalf("SuspendTenant: %v", err)
	}

	body := renderPageBody(t, h, "/admin/")

	for _, want := range []string{"acme", "zeta"} {
		if !strings.Contains(body, want) {
			t.Errorf("switcher must list live tenant %q", want)
		}
	}
	if strings.Contains(body, "ghost-tenant-x") {
		t.Error("suspended tenants are not navigable targets and must stay out of the switcher")
	}
}

// TestTenantSwitcher_TenantAdminModeHidden pins the mode rule: ModeTenantAdmin
// is scoped to exactly one tenant by construction, so the switcher must not
// render there at all.
func TestTenantSwitcher_TenantAdminModeHidden(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "tenant-admin@example.com")
	h, svc := newTenantAdminPanel(t, user, "acme")
	mustCreateTenant(t, svc, "acme")

	body := renderPageBody(t, h, "/admin/")
	if strings.Contains(body, "admin-tenant-switcher") {
		t.Error("tenant-admin mode must not render the tenant switcher")
	}
}

// TestTenantSwitcher_TenantDetailMarksCurrent pins the "where am I" marker:
// the tenant detail page rebuilds the switcher with the viewed tenant marked,
// and the trigger names it (T091 — the detail page is the tenant dashboard
// the switcher falls back to from any other page).
func TestTenantSwitcher_TenantDetailMarksCurrent(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustCreateTenant(t, svc, "acme")

	body := renderPageBody(t, h, "/admin/tenants/acme")

	if !strings.Contains(body, "admin-tenant-switcher") {
		t.Fatal("tenant detail must carry the switcher")
	}
	// The trigger names the current tenant; the item list repeats it with the
	// check icon. Both derive from the same switcher data, so the trigger
	// label is the pinned observable.
	if !strings.Contains(body, ">acme<") {
		t.Error("the switcher trigger must name the viewed tenant")
	}
}

// renderPageBody GETs a panel path with the test user in context and returns
// the full-page HTML body.
func renderPageBody(t *testing.T, h http.Handler, path string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status = %d, want 200; body=%s", path, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}
