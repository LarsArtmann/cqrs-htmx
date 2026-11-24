package adminui

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

func TestPanel_AddMemberMissingFields(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	mustCreateTenant(t, svc, "test")
	rec := postForm(h, "/admin/tenants/test/members", url.Values{})
	if rec.Header().Get("HX-Redirect") != "/admin/tenants/test" {
		t.Errorf("missing fields redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	if !strings.Contains(rec.Header().Get("HX-Trigger"), "adminui:toast") {
		t.Error("expected error toast for missing fields")
	}
}

func TestPanel_TenantCreateEmptyName(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := postForm(h, "/admin/tenants", url.Values{"name": {""}, "display_name": {""}})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty name: status %d, want 400", rec.Code)
	}
}

func TestPanel_UpdateRoleMissingRole(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	mustCreateTenant(t, svc, "eps")
	member := mustRegister(t, svc, "u-up2", "up2@eps.com")
	mustAddMember(t, svc, member.User.ID, identitymodel.NewTenantID("eps"), identitymodel.RoleViewer)

	actor := identitymodel.ActorIDFromUser(member.User.ID).PrefixedString()
	rec := postForm(h, "/admin/tenants/eps/members/"+actor, url.Values{})
	if !strings.Contains(rec.Header().Get("HX-Trigger"), "adminui:toast") {
		t.Error("expected error toast for missing role")
	}
}

func TestPanel_TenantSuspendNotFound(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := postForm(h, "/admin/tenants/nonexistent/suspend", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("suspend nonexistent: status %d, want 400", rec.Code)
	}
}
