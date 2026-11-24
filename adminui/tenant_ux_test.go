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
	// The "drop" assertion reads the TABLE ONLY (the HTMX partial): the full
	// page legitimately repeats every tenant name in the header switcher,
	// which is page chrome, not the search result.
	prec := httptest.NewRecorder()
	preq := httptest.NewRequest(http.MethodGet, "/admin/tenants?q=globex+corp", nil)
	preq.Header.Set("HX-Request", "true")
	h.ServeHTTP(prec, preq)
	if strings.Contains(prec.Body.String(), ">acme<") {
		t.Error("search should drop the non-matching tenant from the table")
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
	got := rec.Body.String()
	if strings.Contains(got, "<!DOCTYPE html>") || strings.Contains(got, `id="tenants-table"`) {
		t.Errorf("HTMX partial must be the fragment only, got: %.120s", got)
	}
}

// TestMembersCard_AddMemberCombobox pins the add-member affordances (T08):
// the field is a combobox (combobox roles + the listbox region it controls)
// fed per keystroke by /users/suggest — no datalist, no directory shipped
// with the page — and the role control is an unchecked checkbox group.
func TestMembersCard_AddMemberCombobox(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	mustCreateTenant(t, svc, "acme")
	mustRegister(t, svc, "u-known", "known@acme.com")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/acme", nil))
	body := rec.Body.String()

	if !strings.Contains(body, `role="combobox"`) {
		t.Error("add-member input should carry combobox semantics")
	}
	if !strings.Contains(body, `role="listbox"`) {
		t.Error("the combobox should control a listbox region")
	}
	if !strings.Contains(body, "users/suggest") {
		t.Error("the combobox should point its fetches at /users/suggest")
	}
	if strings.Contains(body, "<datalist") || strings.Contains(body, `list="admin-member-emails"`) {
		t.Error("the 200-email datalist must be retired (T08 F053)")
	}
	if !strings.Contains(body, `role="group"`) || !strings.Contains(body, `name="roles"`) {
		t.Error("role control should be a named checkbox group")
	}
	// The library renders Checked as a bare ` checked` attribute (its class
	// tokens only ever contain the `:checked:` pseudo variant, which never has
	// the leading space), so a fresh tenant with zero members must not contain
	// one anywhere on the page.
	if strings.Contains(body, " checked") {
		t.Error("a fresh tenant's role checkboxes must all start unchecked")
	}
}

// TestMembersCard_SuggestEndpointFeedsTheField proves the combobox's backend
// contract (T08): the suggest endpoint answers directory-wide per keystroke —
// including existing members — and the duplicate add remains the members
// POST's honest refusal, exactly as it was under the datalist.
func TestMembersCard_SuggestEndpointFeedsTheField(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	tenant := mustCreateTenant(t, svc, "acme")
	member := mustRegister(t, svc, "u-in", "in@acme.com")
	_ = mustRegister(t, svc, "u-out", "out@acme.com")
	mustAddMember(t, svc, member.User.ID, tenant.ID, identitymodel.RoleUser)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/suggest?q=out", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /users/suggest: status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"out@acme.com"`) {
		t.Errorf("suggest should offer the non-member, got: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/suggest?q=in@", nil))
	if !strings.Contains(rec.Body.String(), `"in@acme.com"`) {
		t.Error("suggest is directory-wide; the members POST is the duplicate gate")
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
