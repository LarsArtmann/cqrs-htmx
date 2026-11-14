package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// TestUserDetail_TenantAccessCard pins the user-side half of tenant
// integration: the page names the tenants the person can act in (resolved from
// the Casbin domain to a tenant, not a raw domain string), their roles there,
// and offers the grant form.
func TestUserDetail_TenantAccessCard(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	tenant, err := svc.CreateTenant(ctx, usermgmt.CreateTenantRequest{
		ID: identitymodel.NewTenantID("acme"), Name: "acme", DisplayName: "Acme Corporation",
	})
	if err != nil {
		t.Fatal(err)
	}
	member := mustRegister(t, svc, "u-ada", "ada@acme.com")
	mustAddMember(t, svc, member.User.ID, tenant.ID, identitymodel.RoleOwner)

	body := renderUserDetail(t, h, member.User.ID.Get().String())

	for _, want := range []string{
		"Tenant access",
		"Acme Corporation", // the tenant's display name, not "acme" alone
		">acme<",           // its slug
		"owner",            // the role held there
		"Grant access",
		"Choose tenant…",
		"/users/" + member.User.ID.Get().String() + "/tenants/acme/delete",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("user detail missing %q", want)
		}
	}
}

// TestUserDetail_TenantAccessEmptyState proves the card is always present: a
// user with no memberships sees an explanation and the grant form, not a
// silently absent panel.
func TestUserDetail_TenantAccessEmptyState(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	lonely := mustRegister(t, svc, "u-lonely", "lonely@example.com")
	mustCreateTenant(t, svc, "acme")

	body := renderUserDetail(t, h, lonely.User.ID.Get().String())
	if !strings.Contains(body, "not a member of any tenant yet") {
		t.Error("expected the empty tenant-access explanation")
	}
	if !strings.Contains(body, "Grant access") {
		t.Error("the grant form should render even with no memberships")
	}
	if !strings.Contains(body, "Acme") && !strings.Contains(body, "acme") {
		t.Error("the tenant select should list live tenants")
	}
}

// TestUserGrantTenantAccess dispatches the membership grant from the user page
// and proves it lands in the durable membership store.
func TestUserGrantTenantAccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	tenant := mustCreateTenant(t, svc, "acme")
	target := mustRegister(t, svc, "u-1", "one@acme.com")
	userBase := "/admin/users/" + target.User.ID.Get().String()

	rec := postForm(h, userBase+"/tenants", url.Values{
		"tenant": {"acme"}, "role": {"owner"},
	})
	if got := rec.Header().Get("HX-Redirect"); got != userBase {
		t.Errorf("grant redirect = %q, want %q", got, userBase)
	}
	if !strings.Contains(rec.Header().Get("HX-Trigger"), "adminui:toast") {
		t.Error("expected a success toast")
	}

	members := svc.TenantMembers(ctx, tenant.ID)
	if len(members) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(members))
	}
	if !members[0].HasRole(identitymodel.RoleOwner) {
		t.Errorf("expected owner role, got %v", members[0].Roles)
	}

	// Re-granting replaces the role set (the durable AddMember contract).
	postForm(h, userBase+"/tenants", url.Values{"tenant": {"acme"}, "role": {"viewer"}})
	members = svc.TenantMembers(ctx, tenant.ID)
	if len(members) != 1 || !members[0].HasRole(identitymodel.RoleViewer) {
		t.Errorf("expected the role set to be replaced with viewer, got %+v", members)
	}
}

// TestUserGrantTenantAccess_RejectsBadInput proves the grant form cannot be
// used to mint authority: unknown roles and unknown tenants are refused before
// any command is dispatched.
func TestUserGrantTenantAccess_RejectsBadInput(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	tenant := mustCreateTenant(t, svc, "acme")
	target := mustRegister(t, svc, "u-1", "one@acme.com")
	userBase := "/admin/users/" + target.User.ID.Get().String()

	for name, form := range map[string]url.Values{
		"unknown role":   {"tenant": {"acme"}, "role": {"root"}},
		"unknown tenant": {"tenant": {"ghost"}, "role": {"owner"}},
		"missing role":   {"tenant": {"acme"}},
		"missing tenant": {"role": {"owner"}},
	} {
		rec := postForm(h, userBase+"/tenants", form)
		if !strings.Contains(rec.Header().Get("HX-Trigger"), "adminui:toast") {
			t.Errorf("%s: expected a refusal toast", name)
		}
		if got := svc.TenantMembers(ctx, tenant.ID); len(got) != 0 {
			t.Errorf("%s: expected no membership, got %+v", name, got)
		}
	}
}

// TestUserRevokeTenantAccess removes a membership from the user page and proves
// the durable store agrees.
func TestUserRevokeTenantAccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	tenant := mustCreateTenant(t, svc, "acme")
	target := mustRegister(t, svc, "u-1", "one@acme.com")
	mustAddMember(t, svc, target.User.ID, tenant.ID, identitymodel.RoleUser)

	userBase := "/admin/users/" + target.User.ID.Get().String()
	rec := postForm(h, userBase+"/tenants/acme/delete", nil)
	if got := rec.Header().Get("HX-Redirect"); got != userBase {
		t.Errorf("revoke redirect = %q, want %q", got, userBase)
	}
	if got := svc.TenantMembers(ctx, tenant.ID); len(got) != 0 {
		t.Fatalf("expected the membership to be gone, got %+v", got)
	}
}

func renderUserDetail(t *testing.T, h http.Handler, userID string) string {
	t.Helper()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/"+userID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET user detail: status %d; body=%s", rec.Code, rec.Body.String())
	}

	return rec.Body.String()
}
