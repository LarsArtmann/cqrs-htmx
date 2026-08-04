package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// newTenantAdminPanel creates a tenant-scoped admin panel for testing
// the ModeTenantAdmin handler paths.
func newTenantAdminPanel(t *testing.T, user *usermgmt.User, tenantID string) (http.Handler, *usermgmt.Service) {
	t.Helper()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		AuditLog: usermgmt.NewAuditLog(),
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	panel, err := New(Config{
		Service:    svc,
		Mode:       ModeTenantAdmin,
		TenantID:   usermgmt.NewTenantID(tenantID),
		Authorizer: RequireAuthenticated(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	mux := http.NewServeMux()
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if user != nil {
				r = r.WithContext(usermgmt.WithUser(r.Context(), user))
			}
			next.ServeHTTP(w, r)
		})
	}
	mux.Handle("/admin/", mw(http.StripPrefix("/admin", panel.Handler())))
	return mux, svc
}

func mustCreateTenant(t *testing.T, svc *usermgmt.Service, id string) *usermgmt.Tenant {
	t.Helper()
	tenant, err := svc.CreateTenant(context.Background(), usermgmt.CreateTenantRequest{
		ID: usermgmt.NewTenantID(id), Name: id, DisplayName: id,
	})
	if err != nil {
		t.Fatalf("CreateTenant %s: %v", id, err)
	}
	return tenant
}

func mustRegister(t *testing.T, svc *usermgmt.Service, id, email string) *usermgmt.RegisterResponse {
	t.Helper()
	res, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
		ID: usermgmt.SyntheticUserID(id), Email: email,
	})
	if err != nil {
		t.Fatalf("Register %s: %v", email, err)
	}
	return res
}

func mustAddMember(
	t *testing.T, svc *usermgmt.Service,
	uid usermgmt.UserID, tid usermgmt.TenantID, roles ...usermgmt.Role,
) {
	t.Helper()
	if err := svc.AddMember(context.Background(), usermgmt.ActorIDFromUser(uid), tid, roles); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
}

func TestPanel_TenantAdminMembersIndex(t *testing.T) {
	user := mustUser(t, "ops@acme.com")
	h, svc := newTenantAdminPanel(t, user, "acme")

	mustCreateTenant(t, svc, "acme")
	member := mustRegister(t, svc, "u-m1", "m1@acme.com")
	mustAddMember(t, svc, member.User.ID, usermgmt.NewTenantID("acme"), usermgmt.RoleUser)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("members index: status %d, body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "acme") {
		t.Error("members page should show tenant name")
	}
	actorPrefix := usermgmt.ActorIDFromUser(member.User.ID).PrefixedString()
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
		"email": {"add1@acme.com"}, "role": {"user"},
	})
	if rec.Header().Get("HX-Redirect") != "/admin/members" {
		t.Errorf("add member redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	members := svc.TenantMembers(ctx, usermgmt.NewTenantID("acme"))
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
}

func TestPanel_TenantAdminAddMemberUnknownEmail(t *testing.T) {
	user := mustUser(t, "ops@acme.com")
	h, svc := newTenantAdminPanel(t, user, "acme")

	mustCreateTenant(t, svc, "acme")

	rec := postForm(h, "/admin/members", url.Values{
		"email": {"nobody@nowhere.com"}, "role": {"user"},
	})
	if rec.Header().Get("HX-Redirect") != "/admin/members" {
		t.Errorf("add member redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	if !strings.Contains(rec.Header().Get("HX-Trigger"), "adminui:toast") {
		t.Error("expected error toast for unknown email")
	}
}

func TestPanel_TenantAdminRemoveMember(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "ops@acme.com")
	h, svc := newTenantAdminPanel(t, user, "acme")

	mustCreateTenant(t, svc, "acme")
	member := mustRegister(t, svc, "u-rm1", "rm1@acme.com")
	mustAddMember(t, svc, member.User.ID, usermgmt.NewTenantID("acme"), usermgmt.RoleUser)

	actor := usermgmt.ActorIDFromUser(member.User.ID).PrefixedString()
	rec := postForm(h, "/admin/members/"+actor+"/delete", nil)
	if rec.Header().Get("HX-Redirect") != "/admin/members" {
		t.Errorf("remove member redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	if got := svc.TenantMembers(ctx, usermgmt.NewTenantID("acme")); len(got) != 0 {
		t.Errorf("expected 0 members, got %d", len(got))
	}
}

func TestPanel_TenantAdminUpdateMemberRole(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "ops@acme.com")
	h, svc := newTenantAdminPanel(t, user, "acme")

	mustCreateTenant(t, svc, "acme")
	member := mustRegister(t, svc, "u-up1", "up1@acme.com")
	mustAddMember(t, svc, member.User.ID, usermgmt.NewTenantID("acme"), usermgmt.RoleViewer)

	actor := usermgmt.ActorIDFromUser(member.User.ID).PrefixedString()
	rec := postForm(h, "/admin/members/"+actor, url.Values{"role": {"admin"}})
	if rec.Header().Get("HX-Redirect") != "/admin/members" {
		t.Fatalf("update role redirect = %q", rec.Header().Get("HX-Redirect"))
	}
	members := svc.TenantMembers(ctx, usermgmt.NewTenantID("acme"))
	if len(members) != 1 || members[0].Roles[0] != usermgmt.RoleAdmin {
		t.Errorf("expected admin role, got %+v", members)
	}
}

func TestPanel_SuperAdminTenantNewPage(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/new", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant new: status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "New tenant") {
		t.Error("tenant new page should contain 'New tenant' heading")
	}
}

func TestAuthorizer_DefaultSuperAdmin(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	authz := defaultAuthorizer(Config{
		Mode:    ModeSuperAdmin,
		Service: svc,
	})
	user := mustUser(t, "admin@example.com")
	if err := authz(user); err == nil {
		t.Error("expected denial for user without super_admin role")
	}
	if err := authz(nil); err == nil {
		t.Error("expected denial for nil user")
	}
}

func TestAuthorizer_DefaultTenantAdmin(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	authz := defaultAuthorizer(Config{
		Mode:     ModeTenantAdmin,
		Service:  svc,
		TenantID: usermgmt.NewTenantID("test"),
	})
	user := mustUser(t, "ops@test.com")
	if err := authz(user); err == nil {
		t.Error("expected denial for user without tenant roles")
	}
}

func TestAuthorizer_RequireAnyRole(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	user := mustUser(t, "anyone@example.com")

	deny := RequireAnyRole(svc, "*", usermgmt.RoleSuperAdmin)
	if err := deny(user); err == nil {
		t.Error("expected denial — user has no roles")
	}
	if err := deny(nil); err == nil {
		t.Error("expected denial for nil user")
	}
}

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
	mustAddMember(t, svc, member.User.ID, usermgmt.NewTenantID("eps"), usermgmt.RoleViewer)

	actor := usermgmt.ActorIDFromUser(member.User.ID).PrefixedString()
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

func TestPanel_Mount(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	panel.Mount(mux, "/admin/")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec.Code == http.StatusNotFound {
		t.Error("Mount should register /admin/users route")
	}
}

func TestPanel_MountCoexistsWithRootIndex(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	panel.Mount(mux, "/admin/")
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET / status = %d, want 200", rec.Code)
	}
}

func TestPanel_TenantAdmin403(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	denied, err := New(Config{
		Service:    svc,
		Mode:       ModeTenantAdmin,
		TenantID:   usermgmt.NewTenantID("test"),
		Authorizer: func(*usermgmt.User) error { return errForbidden },
	})
	if err != nil {
		t.Fatal(err)
	}
	user := mustUser(t, "denied@example.com")
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(usermgmt.WithUser(r.Context(), user)))
		})
	}
	mux := http.NewServeMux()
	mux.Handle("/admin/", mw(http.StripPrefix("/admin", denied.Handler())))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/members", nil))
	if rec.Code != http.StatusForbidden {
		t.Errorf("tenant admin 403: status %d, want 403", rec.Code)
	}
}
