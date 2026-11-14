package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// TestTenantAccess_MemberOfEveryTenant pins the grant form's dead-end guard:
// when the user belongs to every live tenant, the tenant select would offer
// nothing, so the form is replaced by an honest state instead of a disabled
// select nobody can use.
func TestTenantAccess_MemberOfEveryTenant(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	mustCreateTenant(t, svc, "acme")
	target := mustRegister(t, svc, "u-all", "all@acme.com")
	userBase := "/admin/users/" + target.User.ID.Get().String()

	// The user joins the only tenant there is.
	rec := postForm(h, userBase+"/tenants", url.Values{
		"tenant": {"acme"}, "roles": {"viewer"},
	})
	if rec.Header().Get("HX-Redirect") != userBase {
		t.Fatalf("grant redirect = %q", rec.Header().Get("HX-Redirect"))
	}

	body := renderUserDetail(t, h, target.User.ID.Get().String())
	if !strings.Contains(body, "member of every live tenant") {
		t.Error("expected the member-of-every-tenant state")
	}
	if strings.Contains(body, "Grant access") {
		t.Error("the grant form must be gone when every tenant is joined")
	}
}

// TestUsersList_EmptyStateCTA pins the true-empty directory state: it invites
// creating the first user instead of dead-ending. The search-miss variant
// keeps its clear-search escape hatch. Rendered directly: a mounted panel
// always has at least the session user, so the empty branch is only reachable
// at the component level (and by consumers with a custom session source).
func TestUsersList_EmptyStateCTA(t *testing.T) {
	t.Parallel()

	render := func(d usersListData) string {
		var b strings.Builder
		if err := usersTableContent(d).Render(context.Background(), &b); err != nil {
			t.Fatalf("render: %v", err)
		}
		return b.String()
	}

	empty := render(usersListData{BasePath: "/admin"})
	if !strings.Contains(empty, "Create the first user") {
		t.Error("empty directory should offer the create-user CTA")
	}
	if !strings.Contains(empty, `href="/admin/users/new"`) {
		t.Error("CTA must link at the new-user form")
	}

	search := render(usersListData{BasePath: "/admin", Search: "ada"})
	if strings.Contains(search, "Create the first user") {
		t.Error("a search miss is not an empty directory")
	}
	if !strings.Contains(search, "No users match your search") {
		t.Error("search miss should keep the search-aware empty state")
	}
}

// TestUnknownEmailBanner_AlertSemantics pins the a11y contract of the inline
// add-member banner: the library's warning Alert renders role="alert", so a
// screen reader announces the miss instead of silently swallowing it.
func TestUnknownEmailBanner_AlertSemantics(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	mustCreateTenant(t, svc, "acme")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/acme?member_error=unknown&member_email=nobody@nowhere.com", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "That account does not exist yet") {
		t.Fatal("expected the unknown-email banner to render")
	}
	if !strings.Contains(body, `role="alert"`) {
		t.Error("the warning banner must carry role=alert semantics")
	}
}

// TestSyncTargets pins the data-sync-target markers the frontend-sync
// protocol (ADR-0056) keys patches on: the tenant identity card and the
// user's tenant-access card are exactly the regions whose content moves
// outside a browser navigation.
func TestSyncTargets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	tenant := mustCreateTenant(t, svc, "acme")
	target := mustRegister(t, svc, "u-sync", "sync@acme.com")
	if err := svc.AddMember(ctx, identitymodel.ActorIDFromUser(target.User.ID), tenant.ID,
		[]identitymodel.Role{identitymodel.RoleViewer}); err != nil {
		t.Fatal(err)
	}

	tenantBody := getBody(t, h, "/admin/tenants/acme")
	if !strings.Contains(tenantBody, "data-sync-target") {
		t.Error("tenant detail: the meta card must carry data-sync-target")
	}

	userBody := renderUserDetail(t, h, target.User.ID.Get().String())
	if got := strings.Count(userBody, "data-sync-target"); got != 2 {
		t.Errorf("user detail: expected 2 sync targets (tenant access + danger zone), got %d", got)
	}
}

// TestLastAdminRemovalConfirmEscalates pins the last-admin guard on BOTH
// removal surfaces: when the member holds the tenant's only admin-tier role,
// the confirm dialog says the tenant would be left without an admin — the
// passkey case's blast-radius escalation, applied to tenants. A second
// admin-tier holder removes the escalation.
func TestLastAdminRemovalConfirmEscalates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	tenant := mustCreateTenant(t, svc, "acme")
	owner := mustRegister(t, svc, "u-owner", "owner@acme.com")
	viewer := mustRegister(t, svc, "u-viewer", "viewer@acme.com")

	if err := svc.AddMember(ctx, identitymodel.ActorIDFromUser(owner.User.ID), tenant.ID,
		[]identitymodel.Role{identitymodel.RoleOwner}); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddMember(ctx, identitymodel.ActorIDFromUser(viewer.User.ID), tenant.ID,
		[]identitymodel.Role{identitymodel.RoleViewer}); err != nil {
		t.Fatal(err)
	}

	// Roster surface: the only owner is about to be removed.
	body := getBody(t, h, "/admin/tenants/acme")
	if !strings.Contains(body, "They hold the only admin or owner role here") {
		t.Error("roster: removal of the sole admin-tier member must escalate its confirm")
	}

	// User surface: revoking that owner's access escalates the same way.
	userBody := renderUserDetail(t, h, owner.User.ID.Get().String())
	if !strings.Contains(userBody, "They hold the only admin or owner role there") {
		t.Error("user page: revoke of the sole admin-tier membership must escalate its confirm")
	}

	// A second admin-tier holder removes the escalation on both surfaces.
	second := mustRegister(t, svc, "u-owner2", "owner2@acme.com")
	if err := svc.AddMember(ctx, identitymodel.ActorIDFromUser(second.User.ID), tenant.ID,
		[]identitymodel.Role{identitymodel.RoleAdmin}); err != nil {
		t.Fatal(err)
	}

	body = getBody(t, h, "/admin/tenants/acme")
	if strings.Contains(body, "They hold the only admin or owner role here") {
		t.Error("roster: escalation must disappear once another admin-tier member exists")
	}
	userBody = renderUserDetail(t, h, owner.User.ID.Get().String())
	if strings.Contains(userBody, "They hold the only admin or owner role there") {
		t.Error("user page: escalation must disappear once another admin-tier member exists")
	}
}

// getBody fetches a panel page body (the GET half of the postForm helper).
func getBody(t *testing.T, h http.Handler, path string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d", path, rec.Code)
	}
	return rec.Body.String()
}
