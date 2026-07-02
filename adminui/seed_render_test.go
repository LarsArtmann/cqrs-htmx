package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// TestPanel_RendersSeededData boots a full panel over a real in-memory service
// with seeded users and tenants, then asserts the data appears in the HTML.
// This is the end-to-end rendering smoke test.
func TestPanel_RendersSeededData(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	// Seed users.
	for _, email := range []string{"alice@acme.dev", "bob@acme.dev", "carol@x.dev"} {
		if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
			ID:    usermgmt.NewUserID("u-" + email),
			Email: email,
		}); err != nil {
			t.Fatalf("register %s: %v", email, err)
		}
	}
	// Seed tenants.
	acme, err := svc.CreateTenant(ctx, usermgmt.CreateTenantRequest{
		ID: usermgmt.NewTenantID("acme"), Name: "acme", DisplayName: "Acme Corp",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if _, err := svc.CreateTenant(ctx, usermgmt.CreateTenantRequest{
		ID: usermgmt.NewTenantID("globex"), Name: "globex", DisplayName: "Globex",
	}); err != nil {
		t.Fatalf("create globex: %v", err)
	}

	get := func(path string) string {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d, body %s", path, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	// Dashboard shows counts.
	dash := get("/admin/")
	if !strings.Contains(dash, "Recent activity") {
		t.Error("dashboard missing recent activity")
	}

	// Users list shows seeded emails.
	usersHTML := get("/admin/users")
	for _, email := range []string{"alice@acme.dev", "bob@acme.dev", "carol@x.dev"} {
		if !strings.Contains(usersHTML, email) {
			t.Errorf("users list missing %q", email)
		}
	}

	// User detail page.
	aliceID := usermgmt.NewUserID("u-alice@acme.dev").Get().String()
	detail := get("/admin/users/" + aliceID)
	if !strings.Contains(detail, "alice@acme.dev") {
		t.Error("user detail missing email")
	}
	if !strings.Contains(detail, "Danger zone") {
		t.Error("user detail missing danger zone")
	}

	// Tenants list shows seeded tenants.
	tenantsHTML := get("/admin/tenants")
	if !strings.Contains(tenantsHTML, "Acme Corp") || !strings.Contains(tenantsHTML, "Globex") {
		t.Error("tenants list missing seeded tenants")
	}

	// Tenant detail shows name + members section.
	td := get("/admin/tenants/acme")
	if !strings.Contains(td, "Acme Corp") {
		t.Error("tenant detail missing name")
	}
	if !strings.Contains(td, "Members") {
		t.Error("tenant detail missing members section")
	}

	// Search filters: q=alice returns alice but not bob.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?q=alice", nil)
	req.Header.Set("HX-Request", "true")
	h.ServeHTTP(rec, req)
	partial := rec.Body.String()
	if !strings.Contains(partial, "alice@acme.dev") {
		t.Error("search partial missing alice")
	}
	if strings.Contains(partial, "bob@acme.dev") {
		t.Error("search partial should not contain bob")
	}

	// Tenant suspend via POST redirects (HTMX request → HX-Redirect).
	rec2 := httptest.NewRecorder()
	suspendReq := httptest.NewRequest(http.MethodPost, "/admin/tenants/acme/suspend", nil)
	suspendReq.Header.Set("HX-Request", "true")
	h.ServeHTTP(rec2, suspendReq)
	if rec2.Code != http.StatusOK {
		t.Errorf("suspend: status %d", rec2.Code)
	}
	if rec2.Header().Get("HX-Redirect") != "/admin/tenants/acme" {
		t.Errorf("suspend: HX-Redirect = %q", rec2.Header().Get("HX-Redirect"))
	}
	// Verify the tenant is now suspended.
	suspended, _ := svc.GetTenant(ctx, acme.ID)
	if !suspended.Suspended {
		t.Error("tenant should be suspended after POST")
	}

	// Audit page renders (audit log recorded registrations).
	auditHTML := get("/admin/audit")
	if !strings.Contains(auditHTML, "register") {
		t.Error("audit log should contain register events")
	}
}
