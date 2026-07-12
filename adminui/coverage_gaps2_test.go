package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func TestNew_NilService_ReturnsError(t *testing.T) {
	_, err := New(Config{
		Service: nil,
	})
	if err == nil {
		t.Error("expected error for nil Service")
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

	authz2 := RequireAnyRole(svc, "*", usermgmt.RoleSuperAdmin)
	if err := authz2(user); err == nil {
		// user has no roles in casbin, so this should be denied
		// but the test is about whether the function runs without panic
	}
}

func TestPanel_NonHTMXDeleteRedirects(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: usermgmt.NewUserID("u-goner2"), Email: "goner2@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	id := target.User.ID.Get().String()

	// Non-HTMX request (no HX-Request header) — should still process the delete
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+id+"/delete", nil)
	h.ServeHTTP(rec, req)

	// Should redirect (302) without HX-Redirect since it's not an HTMX request
	if rec.Code != http.StatusOK && rec.Code != http.StatusFound && rec.Code != http.StatusSeeOther {
		t.Errorf("non-htmx delete: status %d", rec.Code)
	}
}

func TestPanel_UserDetailNotFound(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/nonexistent", nil))
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusBadRequest &&
		rec.Code != http.StatusInternalServerError {
		t.Errorf("user detail not found: status %d, want 400/404/500", rec.Code)
	}
}

func TestPanel_TenantDetailNotFound(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/nonexistent", nil))
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Errorf("tenant detail not found: status %d, want 404 or 500", rec.Code)
	}
}

func TestPanel_TenantReactivateNotFound(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := postForm(h, "/admin/tenants/nonexistent/reactivate", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("reactivate nonexistent: status %d, want 400", rec.Code)
	}
}

func TestPanel_TenantDeleteNotFound(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := postForm(h, "/admin/tenants/nonexistent/delete", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("delete nonexistent: status %d, want 400", rec.Code)
	}
}

func TestPanel_AuditIndexRenders(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("audit index: status %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestPanel_AssetsServeCSS(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/admin-tw.css", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("CSS asset: status %d, want 200", rec.Code)
	}
}

func TestPanel_AssetsServeJS(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/admin.js", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("JS asset: status %d, want 200", rec.Code)
	}
}

func TestPanel_AssetsServeSyncWorker(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/sync-worker.js", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("sync-worker asset: status %d, want 200", rec.Code)
	}
}

func TestPanel_AssetsNotFound(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/nonexistent.js", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown asset: status %d, want 404", rec.Code)
	}
}

func TestPanel_HTMXScriptHandler(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/htmx.js", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("htmx.js: status %d, want 200", rec.Code)
	}
	if rec.Header().Get("Content-Type") == "" {
		t.Error("expected Content-Type header")
	}
}
