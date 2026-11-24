package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func TestMergeRoutes_Gates(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatal(err)
	}
	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/admin/", http.StripPrefix("/admin", panel.Handler()))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/duplicates", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no user: status = %d, want 401", rec.Code)
	}

	strict, _ := New(Config{Service: svc, Authorizer: func(*identitymodel.User) error { return errForbidden }})
	deny := http.NewServeMux()
	deny.Handle("/admin/", http.StripPrefix("/admin", strict.Handler()))
	denied := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(usermgmt.WithUser(r.Context(), mustUser(t, "x@example.com")))
			next.ServeHTTP(w, r)
		})
	}
	for _, target := range []struct{ method, path string }{
		{http.MethodGet, "/admin/users/duplicates"},
		{http.MethodGet, "/admin/tenants/duplicates"},
		{http.MethodPost, "/admin/users/x/merge"},
		{http.MethodPost, "/admin/tenants/x/merge"},
		{http.MethodPost, "/admin/tenants/x/rename"},
	} {
		rec := httptest.NewRecorder()
		denied(deny).ServeHTTP(rec, httptest.NewRequest(target.method, target.path, nil))
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s: denied authorizer: status = %d, want 403", target.method, target.path, rec.Code)
		}
	}

	// Authorized: the report pages render.
	user := mustUser(t, "admin@example.com")
	h, svc2 := newTestPanel(t, user)
	_ = svc2
	for _, path := range []string{"/admin/users/duplicates", "/admin/tenants/duplicates"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", path, rec.Code)
		}
	}
}

func TestUserDuplicatesPage_EmptyState(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/duplicates", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Duplicate users") {
		t.Error("page does not carry its title")
	}
	if !strings.Contains(body, "No duplicate users") {
		t.Error("empty state missing")
	}
}

func TestUserMergeRoute_MergesAndRedirects(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	ctx := t.Context()

	survivor, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.NewUserID("01M4SURVIVOR00000000000001"), Email: "keep@example.com",
	})
	if err != nil {
		t.Fatalf("register survivor: %v", err)
	}
	loser, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.NewUserID("01M4LOSER0000000000000002"), Email: "loser@example.com",
	})
	if err != nil {
		t.Fatalf("register loser: %v", err)
	}

	form := "survivor=" + survivor.User.ID.Get().String()
	req := httptest.NewRequest(http.MethodPost,
		"/admin/users/"+loser.User.ID.Get().String()+"/merge", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("merge status = %d, body %s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/users/"+survivor.User.ID.Get().String() {
		t.Errorf("redirect Location = %q, want the survivor detail", loc)
	}
	if count := svc.ReadModel().Count(); count != 1 {
		t.Errorf("live users = %d, want 1 (the merge must retire the loser)", count)
	}
}

func TestTenantMergeAndRenameRoutes(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)
	ctx := t.Context()

	survivor, err := svc.CreateTenant(ctx, usermgmt.CreateTenantRequest{
		ID: identitymodel.NewTenantID("01M4TENSURVIVOR00000000001"), Name: "Acme Corp",
	})
	if err != nil {
		t.Fatalf("create survivor: %v", err)
	}
	if _, err := svc.CreateTenant(ctx, usermgmt.CreateTenantRequest{
		ID: identitymodel.NewTenantID("01M4TENLOSER0000000000002"), Name: " acme corp ",
	}); err != nil {
		t.Fatalf("create loser: %v", err)
	}

	// The duplicates report shows the cluster.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/duplicates", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("duplicates status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "acme corp") || !strings.Contains(body, "Duplicate tenants") {
		t.Error("the cluster is not rendered")
	}

	// Rename the survivor, then merge the loser into it.
	form := "display_name=Acme+Corporation"
	req := httptest.NewRequest(http.MethodPost,
		"/admin/tenants/"+survivor.ID.Get()+"/rename", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("rename status = %d, body %s", rec.Code, rec.Body.String())
	}
	got, err := svc.GetTenant(ctx, survivor.ID)
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if got.DisplayName != "Acme Corporation" {
		t.Errorf("DisplayName = %q, want the renamed label", got.DisplayName)
	}

	form = "survivor=" + survivor.ID.Get()
	req = httptest.NewRequest(http.MethodPost,
		"/admin/tenants/01M4TENLOSER0000000000002/merge", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("merge status = %d, body %s", rec.Code, rec.Body.String())
	}
	if count := len(svc.AllTenants()); count != 1 {
		t.Errorf("live tenants = %d, want 1", count)
	}

	// The report is empty after the merge.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/duplicates", nil))
	if !strings.Contains(rec.Body.String(), "No duplicate tenants") {
		t.Error("the report still shows the merged cluster")
	}
}
