package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// TestTenantDetail_MembersRenderAsPeople pins the core tenant-integration fix:
// a membership row names the PERSON (email, linked to their user page, plus
// their display name) instead of leading with the opaque "user:01HZ…" actor
// token an admin cannot map to anybody.
func TestTenantDetail_MembersRenderAsPeople(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	ctx := context.Background()
	tenant := mustCreateTenant(t, svc, "acme")
	member := mustRegister(t, svc, "u-ada", "ada@acme.com")
	if err := svc.ChangeDisplayName(ctx, member.User.ID, "Ada Lovelace"); err != nil {
		t.Fatalf("ChangeDisplayName: %v", err)
	}
	mustAddMember(t, svc, member.User.ID, tenant.ID, identitymodel.RoleOwner)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/acme", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant detail: status %d", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		"ada@acme.com", // the person, not the actor token
		"Ada Lovelace", // their display name
		"/admin/users/" + member.User.ID.Get().String(), // one click to their page
		"owner", // the role they hold, as a badge
		"Added", // when the membership started
	} {
		if !strings.Contains(body, want) {
			t.Errorf("tenant detail missing %q", want)
		}
	}
}

// TestTenantDetail_MetaCard proves the tenant identity block answers "which
// tenant is this" without the admin inferring it from the roster: id, slug,
// display name, status, and current member count are all on the page.
func TestTenantDetail_MetaCard(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	tenant := mustCreateTenant(t, svc, "globex")
	member := mustRegister(t, svc, "u-1", "one@globex.com")
	mustAddMember(t, svc, member.User.ID, tenant.ID, identitymodel.RoleViewer)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/globex", nil))
	body := rec.Body.String()

	for _, want := range []string{"Tenant ID", "globex", "Display name", "Status", "Members"} {
		if !strings.Contains(body, want) {
			t.Errorf("tenant meta card missing %q", want)
		}
	}
}

// TestTenantsIndex_SearchAndMemberCounts pins the roster overview: every row
// reports how many people the tenant scopes, and the search narrows by slug or
// display name while keeping the pagination URL honest.
func TestTenantsIndex_SearchAndMemberCounts(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	acme := mustCreateTenant(t, svc, "acme")
	_, err := svc.CreateTenant(context.Background(), usermgmt.CreateTenantRequest{
		ID: identitymodel.NewTenantID("globex"), Name: "globex", DisplayName: "Globex Corporation",
	})
	if err != nil {
		t.Fatalf("CreateTenant globex: %v", err)
	}
	first := mustRegister(t, svc, "u-a1", "a1@acme.com")
	second := mustRegister(t, svc, "u-a2", "a2@acme.com")
	mustAddMember(t, svc, first.User.ID, acme.ID, identitymodel.RoleUser)
	mustAddMember(t, svc, second.User.ID, acme.ID, identitymodel.RoleViewer)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("tenants index: status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"acme", "Globex Corporation", "2", "Members"} {
		if !strings.Contains(body, want) {
			t.Errorf("tenants index missing %q", want)
		}
	}
	// Search narrows to the display-name match (case-insensitive).
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants?q=globex+corp", nil))
	body = rec.Body.String()
	if !strings.Contains(body, "Globex Corporation") {
		t.Error("search should keep the matching tenant")
	}
	if strings.Contains(body, ">acme<") {
		t.Error("search should drop the non-matching tenant")
	}

	// A miss renders honest guidance instead of an empty table.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants?q=nomatch", nil))
	if !strings.Contains(rec.Body.String(), "No tenants match your search") {
		t.Error("expected the empty-search state")
	}

	// HTMX requests get the table fragment, not a document.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/tenants?q=acme", nil)
	req.Header.Set("HX-Request", "true")
	h.ServeHTTP(rec, req)
	if got := rec.Body.String(); strings.Contains(got, "<!DOCTYPE html>") || strings.Contains(got, `id="tenants-table"`) {
		t.Errorf("HTMX partial must be the fragment only, got: %.120s", got)
	}
}

// TestMembersCard_AddMemberAutocomplete pins the add-member affordances: the
// directory feeds a datalist so an admin picks a person, and the role select
// cannot silently default to the first role.
func TestMembersCard_AddMemberAutocomplete(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	mustCreateTenant(t, svc, "acme")
	mustRegister(t, svc, "u-known", "known@acme.com")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/acme", nil))
	body := rec.Body.String()

	if !strings.Contains(body, `list="admin-member-emails"`) {
		t.Error("add-member input should point at the directory datalist")
	}
	if !strings.Contains(body, `<datalist id="admin-member-emails">`) {
		t.Error("expected the directory datalist")
	}
	if !strings.Contains(body, `value="known@acme.com"`) {
		t.Error("datalist should carry directory emails")
	}
	if !strings.Contains(body, "Choose role…") {
		t.Error("role select should require an explicit choice")
	}
}

// TestTenantAdminMembers_NoUserLinks proves the tenant-admin mode does not link
// to a user directory it does not serve: the mode has no /users route, so a
// detail link there would 404.
func TestTenantAdminMembers_NoUserLinks(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "ops@acme.com")
	h, svc := newTenantAdminPanel(t, user, "acme")

	mustCreateTenant(t, svc, "acme")
	member := mustRegister(t, svc, "u-m1", "m1@acme.com")
	mustAddMember(t, svc, member.User.ID, identitymodel.NewTenantID("acme"), identitymodel.RoleUser)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members", nil))
	body := rec.Body.String()

	if !strings.Contains(body, "m1@acme.com") {
		t.Error("members page should resolve the member's email")
	}
	if strings.Contains(body, "/admin/users/") {
		t.Error("tenant-admin mode must not link to the absent user directory")
	}
}
