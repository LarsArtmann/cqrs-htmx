package adminui

import (
	"context"
	"net/http"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// newTenantAdminPanel creates a tenant-scoped admin panel for testing
// the ModeTenantAdmin handler paths.
func newTenantAdminPanel(t *testing.T, user *identitymodel.User, tenantID string) (http.Handler, *usermgmt.Service) {
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
		TenantID:   identitymodel.NewTenantID(tenantID),
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
		ID: identitymodel.NewTenantID(id), Name: id, DisplayName: id,
	})
	if err != nil {
		t.Fatalf("CreateTenant %s: %v", id, err)
	}
	return tenant
}

func mustRegister(t *testing.T, svc *usermgmt.Service, id, email string) *usermgmt.RegisterResponse {
	t.Helper()
	res, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID(id), Email: email,
	})
	if err != nil {
		t.Fatalf("Register %s: %v", email, err)
	}
	return res
}

func mustAddMember(
	t *testing.T, svc *usermgmt.Service,
	uid identitymodel.UserID, tid identitymodel.TenantID, roles ...identitymodel.Role,
) {
	t.Helper()
	if err := svc.AddMember(context.Background(), identitymodel.ActorIDFromUser(uid), tid, roles); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
}
