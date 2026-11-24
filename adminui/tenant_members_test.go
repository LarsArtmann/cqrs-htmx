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

func TestPanel_TenantAdminMembersIndex(t *testing.T) {
	user := mustUser(t, "ops@acme.com")
	h, svc := newTenantAdminPanel(t, user, "acme")

	mustCreateTenant(t, svc, "acme")
	member := mustRegister(t, svc, "u-m1", "m1@acme.com")
	mustAddMember(t, svc, member.User.ID, identitymodel.NewTenantID("acme"), identitymodel.RoleUser)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("members index: status %d, body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "acme") {
		t.Error("members page should show tenant name")
	}
	actorPrefix := identitymodel.ActorIDFromUser(member.User.ID).PrefixedString()
	if !strings.Contains(body, actorPrefix) {
		t.Errorf("members page should show actor %q", actorPrefix)
	}
}

func TestPanel_TenantAdminMembersEmpty(t *testing.T) {
	user := mustUser(t, "ops@acme.com")
	h, _ := newTenantAdminPanel(t, user, "acme")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("members empty: status %d", rec.Code)
	}
}

func TestPanel_TenantAdminAddMember(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "ops@acme.com")
	h, svc := newTenantAdminPanel(t, user, "acme")

	mustCreateTenant(t, svc, "acme")
	mustRegister(t, svc, "u-add1", "add1@acme.com")

	rec := postForm(h, "/admin/members", url.Values{
		"email": {"add1@acme.com"}, "roles": {"user"},
	})
	if rec.Header().Get("HX-Redirect") != "/admin/members" {
		t.Errorf("add member redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	members := svc.TenantMembers(ctx, identitymodel.NewTenantID("acme"))
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
}

func TestPanel_TenantAdminAddMemberUnknownEmail(t *testing.T) {
	user := mustUser(t, "ops@acme.com")
	h, svc := newTenantAdminPanel(t, user, "acme")

	mustCreateTenant(t, svc, "acme")

	rec := postForm(h, "/admin/members", url.Values{
		"email": {"nobody@nowhere.com"}, "roles": {"user"},
	})
	// Unknown email -> redirect back carrying the typed address, so the page
	// can explain the miss inline instead of losing what the admin typed.
	redirect := rec.Header().Get("HX-Redirect")
	if !strings.HasPrefix(redirect, "/admin/members?") ||
		!strings.Contains(redirect, "member_error=unknown") ||
		!strings.Contains(redirect, "member_email=nobody%40nowhere.com") {
		t.Errorf("add member redirect = %q", redirect)
	}
	if !strings.Contains(rec.Header().Get("HX-Trigger"), "adminui:toast") {
		t.Error("expected error toast for unknown email")
	}

	// The redirect target renders the guidance and keeps the typed email.
	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, redirect, nil))
	if page.Code != http.StatusOK {
		t.Fatalf("error page status = %d", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, "That account does not exist yet") {
		t.Error("expected inline unknown-email banner")
	}
	if !strings.Contains(body, `value="nobody@nowhere.com"`) {
		t.Error("expected the typed email to be preserved in the form")
	}
}

func TestPanel_TenantAdminRemoveMember(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "ops@acme.com")
	h, svc := newTenantAdminPanel(t, user, "acme")

	mustCreateTenant(t, svc, "acme")
	member := mustRegister(t, svc, "u-rm1", "rm1@acme.com")
	mustAddMember(t, svc, member.User.ID, identitymodel.NewTenantID("acme"), identitymodel.RoleUser)

	actor := identitymodel.ActorIDFromUser(member.User.ID).PrefixedString()
	rec := postForm(h, "/admin/members/"+actor+"/delete", nil)
	if rec.Header().Get("HX-Redirect") != "/admin/members" {
		t.Errorf("remove member redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	if got := svc.TenantMembers(ctx, identitymodel.NewTenantID("acme")); len(got) != 0 {
		t.Errorf("expected 0 members, got %d", len(got))
	}
}

func TestPanel_TenantAdminUpdateMemberRole(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "ops@acme.com")
	h, svc := newTenantAdminPanel(t, user, "acme")

	mustCreateTenant(t, svc, "acme")
	member := mustRegister(t, svc, "u-up1", "up1@acme.com")
	mustAddMember(t, svc, member.User.ID, identitymodel.NewTenantID("acme"), identitymodel.RoleViewer)

	actor := identitymodel.ActorIDFromUser(member.User.ID).PrefixedString()
	rec := postForm(h, "/admin/members/"+actor, url.Values{"roles": {"admin", "viewer"}})
	if rec.Header().Get("HX-Redirect") != "/admin/members" {
		t.Fatalf("update role redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	members := svc.TenantMembers(ctx, identitymodel.NewTenantID("acme"))
	if len(members) != 1 || !members[0].HasRole(identitymodel.RoleAdmin) ||
		!members[0].HasRole(identitymodel.RoleViewer) {
		t.Errorf("expected viewer+admin roles, got %+v", members)
	}
}
