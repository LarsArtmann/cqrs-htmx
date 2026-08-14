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

// postForm issues an HTMX POST with form values and returns the recorder.
func postForm(h http.Handler, target string, form url.Values) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	h.ServeHTTP(rec, req)
	return rec
}

func TestFlow_TenantCreateSuspendReactivateDelete(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	// Create.
	rec := postForm(h, "/admin/tenants", url.Values{"name": {"omega"}, "display_name": {"Omega"}})
	if rec.Header().Get("HX-Redirect") != "/admin/tenants/omega" {
		t.Fatalf("create HX-Redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	tenant, err := svc.GetTenant(ctx, identitymodel.NewTenantID("omega"))
	if err != nil || tenant == nil {
		t.Fatalf("tenant not created: %v", err)
	}

	// Suspend.
	postForm(h, "/admin/tenants/omega/suspend", nil)
	if t2, _ := svc.GetTenant(ctx, identitymodel.NewTenantID("omega")); !t2.Suspended {
		t.Error("tenant should be suspended")
	}

	// Reactivate.
	postForm(h, "/admin/tenants/omega/reactivate", nil)
	if t2, _ := svc.GetTenant(ctx, identitymodel.NewTenantID("omega")); t2.Suspended {
		t.Error("tenant should be reactivated")
	}

	// Delete.
	postForm(h, "/admin/tenants/omega/delete", nil)
	if rec2 := httptest.NewRecorder(); rec2.Code == 0 {
		_ = rec2
	}
	if _, err := svc.GetTenant(ctx, identitymodel.NewTenantID("omega")); err == nil {
		t.Error("tenant should be deleted (GetTenant should fail)")
	}
}

func TestFlow_UserDelete(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-goner"), Email: "goner@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	id := target.User.ID.Get().String()

	rec := postForm(h, "/admin/users/"+id+"/delete", nil)
	if rec.Header().Get("HX-Redirect") != "/admin/users" {
		t.Errorf("delete HX-Redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	if _, err := svc.GetUser(ctx, target.User.ID); err == nil {
		t.Error("user should be deleted")
	}
}

func TestFlow_MemberAddAndRemove(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	tenant, err := svc.CreateTenant(ctx, usermgmt.CreateTenantRequest{
		ID: identitymodel.NewTenantID("beta"), Name: "beta", DisplayName: "Beta",
	})
	if err != nil {
		t.Fatal(err)
	}
	member, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-member1"), Email: "member1@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Add member by email.
	rec := postForm(h, "/admin/tenants/beta/members", url.Values{
		"email": {"member1@example.com"}, "role": {"admin"},
	})
	if rec.Header().Get("HX-Redirect") != "/admin/tenants/beta" {
		t.Errorf("add member HX-Redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	members := svc.TenantMembers(ctx, tenant.ID)
	if len(members) != 1 || members[0].ActorID.String() != member.User.ID.Get().String() {
		t.Fatalf("expected 1 member, got %+v", members)
	}

	// Remove member by actor prefix.
	actor := members[0].ActorID.PrefixedString()
	rec2 := postForm(h, "/admin/tenants/beta/members/"+actor+"/delete", nil)
	if rec2.Header().Get("HX-Redirect") != "/admin/tenants/beta" {
		t.Errorf("remove member HX-Redirect = %q", rec2.Header().Get("HX-Redirect"))
	}
	if got := svc.TenantMembers(ctx, tenant.ID); len(got) != 0 {
		t.Fatalf("expected 0 members after remove, got %d", len(got))
	}
}

func TestFlow_AddMemberUnknownEmail(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	if _, err := svc.CreateTenant(context.Background(), usermgmt.CreateTenantRequest{
		ID: identitymodel.NewTenantID("gamma"), Name: "gamma",
	}); err != nil {
		t.Fatal(err)
	}
	rec := postForm(h, "/admin/tenants/gamma/members", url.Values{
		"email": {"nobody@example.com"}, "role": {"user"},
	})
	// Unknown email -> error toast + redirect back (no member added).
	if rec.Header().Get("HX-Redirect") != "/admin/tenants/gamma" {
		t.Errorf("HX-Redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	if !strings.Contains(rec.Header().Get("HX-Trigger"), "adminui:toast") {
		t.Error("expected error toast header")
	}
	if got := svc.TenantMembers(context.Background(), identitymodel.NewTenantID("gamma")); len(got) != 0 {
		t.Errorf("expected no members, got %d", len(got))
	}
}

func TestSafeRedirectPath_RejectsOpenRedirect(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"/admin/users":        "/admin/users",
		"/":                   "/",
		"//evil.com":          "/",
		"https://evil.com":    "/",
		"//evil.com/x":        "/",
		"":                    "/",
		"relative":            "/",
		"/admin/tenants/acme": "/admin/tenants/acme",
	}
	for in, want := range cases {
		if got := safeRedirectPath(in); got != want {
			t.Errorf("safeRedirectPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFlow_UpdateMemberRole(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	tenant, err := svc.CreateTenant(ctx, usermgmt.CreateTenantRequest{
		ID: identitymodel.NewTenantID("delta"), Name: "delta", DisplayName: "Delta",
	})
	if err != nil {
		t.Fatal(err)
	}
	member, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-role1"), Email: "role1@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddMember(
		ctx,
		identitymodel.ActorIDFromUser(member.User.ID),
		tenant.ID,
		[]identitymodel.Role{identitymodel.RoleViewer},
	); err != nil {
		t.Fatal(err)
	}

	// Change role from viewer to admin via inline edit endpoint.
	actor := identitymodel.ActorIDFromUser(member.User.ID).PrefixedString()
	rec := postForm(h, "/admin/tenants/delta/members/"+actor, url.Values{"role": {"admin"}})
	if rec.Header().Get("HX-Redirect") != "/admin/tenants/delta" {
		t.Fatalf("update role HX-Redirect = %q", rec.Header().Get("HX-Redirect"))
	}

	members := svc.TenantMembers(ctx, tenant.ID)
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if len(members[0].Roles) != 1 || members[0].Roles[0] != identitymodel.RoleAdmin {
		t.Errorf("role = %v, want [admin]", members[0].Roles)
	}
}
