package adminui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// newTestPanel builds a panel backed by an in-memory service and wrapped by
// middleware that injects the given user into the request context. This mirrors
// how a consumer mounts the panel behind usermgmt.NewSessionMiddleware.
func newTestPanel(t *testing.T, user *usermgmt.User, config ...Config) (http.Handler, *usermgmt.Service) {
	t.Helper()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		AuditLog: usermgmt.NewAuditLog(),
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	c := Config{Service: svc, Authorizer: RequireAuthenticated()}
	if len(config) > 0 {
		c = config[0]
		if c.Authorizer == nil {
			c.Authorizer = RequireAuthenticated()
		}
	}
	panel, err := New(c)
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

func mustUser(t *testing.T, email string) *usermgmt.User {
	t.Helper()
	return usermgmt.NewUser(usermgmt.SyntheticUserID("01HXTEST"+pad(email)), email, "")
}

func pad(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s + strings.Repeat("0", 10-len(s))
}

func TestPanel_DashboardRenders(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Dashboard", "Recent activity", "admin-tw.css", "htmx.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
}

func TestPanel_UsersIndexEmptyAndSearch(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	// Empty list.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No users found") {
		t.Error("expected empty state")
	}

	// HTMX partial request returns just the table fragment.
	rec2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?q=alice", nil)
	req.Header.Set("HX-Request", "true")
	h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("partial status = %d", rec2.Code)
	}
	if strings.Contains(rec2.Body.String(), "<!DOCTYPE html>") {
		t.Error("HTMX partial should not be a full document")
	}
}

func TestPanel_TenantsAndAuditRender(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	for _, path := range []string{"/admin/tenants", "/admin/audit"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d", path, rec.Code)
		}
	}
}

func TestPanel_UnauthorizedAndForbidden(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		AuditLog: usermgmt.NewAuditLog(),
	})
	if err != nil {
		t.Fatal(err)
	}
	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/admin/", http.StripPrefix("/admin", panel.Handler()))

	// No user in context -> 401.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no user: status = %d, want 401", rec.Code)
	}

	// User present but denied authorizer -> 403.
	deny := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(usermgmt.WithUser(r.Context(), mustUser(t, "x@example.com")))
			next.ServeHTTP(w, r)
		})
	}
	strict, _ := New(Config{Service: svc, Authorizer: func(*usermgmt.User) error { return errForbidden }})
	mux2 := http.NewServeMux()
	mux2.Handle("/admin/", deny(http.StripPrefix("/admin", strict.Handler())))
	rec2 := httptest.NewRecorder()
	mux2.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if rec2.Code != http.StatusForbidden {
		t.Errorf("denied: status = %d, want 403", rec2.Code)
	}
}

func TestPanel_AssetsServed(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	for path, ct := range map[string]string{
		"/admin/-/admin-tw.css": "text/css",
		"/admin/-/admin.js":     "text/javascript",
		"/admin/-/htmx.js":      "text/javascript",
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status %d", path, rec.Code)
			continue
		}
		if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, ct) {
			t.Errorf("%s: content-type %q, want prefix %q", path, got, ct)
		}
		if _, err := io.ReadAll(rec.Result().Body); err != nil {
			t.Errorf("%s: read body: %v", path, err)
		}
	}
}

func TestPanel_TenantAdminMode(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	panel, err := New(Config{
		Service:    svc,
		Mode:       ModeTenantAdmin,
		TenantID:   usermgmt.NewTenantID("acme"),
		Authorizer: RequireAuthenticated(),
	})
	if err != nil {
		t.Fatal(err)
	}
	user := mustUser(t, "ops@acme.com")
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(usermgmt.WithUser(r.Context(), user)))
		})
	}
	mux := http.NewServeMux()
	mux.Handle("/admin/", mw(http.StripPrefix("/admin", panel.Handler())))

	// Tenant-admin nav has Members, not Users/Tenants.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "Members") {
		t.Error("tenant-admin dashboard should list Members nav")
	}
	if strings.Contains(body, ">Tenants<") {
		t.Error("tenant-admin dashboard should not list Tenants nav")
	}

	// /admin/users is not registered in tenant-admin mode -> 404.
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec2.Code != http.StatusNotFound {
		t.Errorf("tenant-admin /users: status = %d, want 404", rec2.Code)
	}
}

func TestNew_RequiresService(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Error("expected error for nil Service")
	}
}

func TestNew_TenantAdminRequiresTenantID(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	if _, err := New(Config{Service: svc, Mode: ModeTenantAdmin}); err == nil {
		t.Error("expected error for tenant-admin without TenantID")
	}
}

func TestMiddleware_PermissionsPolicyAndNonce(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatal(err)
	}

	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}

	mw := panel.Middleware()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	pp := rec.Header().Get("Permissions-Policy")
	if pp == "" {
		t.Fatal("expected Permissions-Policy header to be set")
	}

	for _, want := range []string{"geolocation=()", "microphone=()", "camera=()", "payment=()", "usb=()"} {
		if !strings.Contains(pp, want) {
			t.Errorf("Permissions-Policy missing %q, got %q", want, pp)
		}
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("expected Content-Security-Policy header to be set")
	}

	if !strings.Contains(csp, "nonce-") {
		t.Errorf("expected CSP to contain a nonce, got %q", csp)
	}
}
