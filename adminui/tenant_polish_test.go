package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// TestTenantsList_StatusFilter pins the lifecycle chips' server side: the
// status query param narrows the TABLE to that lifecycle state, and unknown
// values degrade to the unfiltered list rather than an empty one (a
// hand-edited URL must not read as "no tenants exist"). Asserted through the
// HTMX partial: the full page legitimately repeats every tenant name in the
// header switcher.
func TestTenantsList_StatusFilter(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustCreateTenant(t, svc, "acme")
	mustCreateTenant(t, svc, "ghost")
	if err := svc.SuspendTenant(t.Context(), identitymodel.NewTenantID("ghost"), "test"); err != nil {
		t.Fatalf("SuspendTenant: %v", err)
	}

	getPartial := func(query string) string {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/tenants"+query, nil)
		req.Header.Set("HX-Request", "true")
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d", query, rec.Code)
		}
		return rec.Body.String()
	}

	if body := getPartial("?status=active"); strings.Contains(body, "ghost") {
		t.Error("status=active must drop the suspended tenant")
	}
	if body := getPartial("?status=suspended"); !strings.Contains(body, "ghost") || strings.Contains(body, ">acme<") {
		t.Error("status=suspended must keep only the suspended tenant")
	}
	if body := getPartial("?status=bogus"); !strings.Contains(body, "acme") || !strings.Contains(body, "ghost") {
		t.Error("an unknown status must degrade to the unfiltered list")
	}
}

// TestTenantsList_SortByMembers pins the ordering toggle: sort=members orders
// by member count descending, with display name as the stable tiebreak.
func TestTenantsList_SortByMembers(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustCreateTenant(t, svc, "lonely") // zero members
	mustCreateTenant(t, svc, "busy")   // two members
	u1 := mustRegister(t, svc, "m1", "m1@acme.dev")
	u2 := mustRegister(t, svc, "m2", "m2@acme.dev")
	mustAddMember(t, svc, u1.User.ID, identitymodel.NewTenantID("busy"), identitymodel.RoleViewer)
	mustAddMember(t, svc, u2.User.ID, identitymodel.NewTenantID("busy"), identitymodel.RoleViewer)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants?sort=members", nil))
	body := rec.Body.String()
	if strings.Index(body, "busy") > strings.Index(body, "lonely") {
		t.Error("sort=members must put the tenant with more members first")
	}

	// Default (no param) stays display-name order: busy before lonely.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants", nil))
	body = rec.Body.String()
	if strings.Index(body, "busy") > strings.Index(body, "lonely") {
		t.Error("default sort must be display-name order (busy before lonely)")
	}
}

// TestTenantDetail_ImpactConfirms pins the preflight truth: the destructive
// actions name their blast radius (member count) in the confirm dialog, so
// the admin sees the impact before confirming something irreversible.
func TestTenantDetail_ImpactConfirms(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustCreateTenant(t, svc, "acme")
	u1 := mustRegister(t, svc, "m1", "m1@acme.dev")
	u2 := mustRegister(t, svc, "m2", "m2@acme.dev")
	mustAddMember(t, svc, u1.User.ID, identitymodel.NewTenantID("acme"), identitymodel.RoleViewer)
	mustAddMember(t, svc, u2.User.ID, identitymodel.NewTenantID("acme"), identitymodel.RoleViewer)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/acme", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant detail: status %d", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		"Its 2 member(s) lose access",
		"revoke its 2 membership(s)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("detail page must carry the impact confirm %q", want)
		}
	}
}

// TestTenantDetail_AuditShortcut documents the plan's T104 gap: the audit
// log records NO tenant lifecycle entries (verified empirically — a
// CreateTenant dispatch produces zero entries; commandAuditMiddleware covers
// user/session commands only), so a per-tenant audit shortcut would land on
// an empty search — a dead-end link this plan's philosophy forbids. The
// audit-filter extension that would have made it land was built and
// reverted in the same change. The link ships when the audit log gains
// tenant attribution — the usermgmt ask in TODO_LIST item (h), D2-gated.
func TestTenantDetail_AuditShortcut(t *testing.T) {
	t.Skip("audit log has no tenant attribution yet — TODO_LIST item (h), D2-gated")
}
