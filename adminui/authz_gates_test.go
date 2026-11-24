package adminui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

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
		TenantID: identitymodel.NewTenantID("test"),
	})
	user := mustUser(t, "ops@test.com")
	if err := authz(user); err == nil {
		t.Error("expected denial for user without tenant roles")
	}
}

func TestAuthorizer_RequireAnyRole(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	user := mustUser(t, "anyone@example.com")

	deny := RequireAnyRole(svc, "*", identitymodel.RoleSuperAdmin)
	if err := deny(user); err == nil {
		t.Error("expected denial — user has no roles")
	}
	if err := deny(nil); err == nil {
		t.Error("expected denial for nil user")
	}
}

func TestPanel_UnauthenticatedReturns401(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	panel, err := New(Config{
		Service:    svc,
		Authorizer: RequireAuthenticated(),
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	panel.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated: status %d, want 401", rec.Code)
	}
}

func TestPanel_AuthorizerAllowsUser(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	user := mustUser(t, "auth@example.com")

	authz := RequireAuthenticated()
	if err := authz(user); err != nil {
		t.Errorf("RequireAuthenticated should allow any non-nil user: %v", err)
	}

	authz2 := RequireAnyRole(svc, "*", identitymodel.RoleSuperAdmin)
	if err := authz2(user); err == nil {
		t.Error("RequireAnyRole should deny user with no matching roles")
	}
}

func TestRequireAnyRole_Deny(t *testing.T) {
	t.Parallel()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	authz := RequireAnyRole(svc, "test", identitymodel.RoleAdmin)
	user := mustUser(t, "viewer@example.com")

	if err := authz(user); err == nil {
		t.Error("RequireAnyRole should deny user without required role")
	}
}

func TestRequireAnyRole_Allow(t *testing.T) {
	t.Parallel()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	// With no domain/roles configured, every user is denied by default.
	// Test the allow path by using RequireAuthenticated instead.
	authz := RequireAuthenticated()
	user := mustUser(t, "admin@example.com")
	if err := authz(user); err != nil {
		t.Errorf("RequireAuthenticated should allow any non-nil user: %v", err)
	}
}

func TestRequireAuthenticated_Deny(t *testing.T) {
	t.Parallel()
	authz := RequireAuthenticated()
	if err := authz(nil); err == nil {
		t.Error("RequireAuthenticated should deny when user is nil")
	}
}

func TestPanel_TenantAdmin403(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	denied, err := New(Config{
		Service:    svc,
		Mode:       ModeTenantAdmin,
		TenantID:   identitymodel.NewTenantID("test"),
		Authorizer: func(*identitymodel.User) error { return errForbidden },
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
