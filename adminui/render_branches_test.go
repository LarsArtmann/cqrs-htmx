package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

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

// TestPanel_RenderDiverseUser exercises templ branches for users with
// display names and the user detail page structure.
func TestPanel_RenderDiverseUser(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	// Register a user with display name
	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:    identitymodel.SyntheticUserID("u-diverse"),
		Email: "diverse@test.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	userID := target.User.ID

	// Set display name
	if err := svc.ChangeDisplayName(ctx, userID, "Diverse User"); err != nil {
		t.Fatalf("ChangeDisplayName: %v", err)
	}

	// Render user detail page
	id := userID.Get().String()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/"+id, nil))
	body := rec.Body.String()

	// Check that display name renders
	if !strings.Contains(body, "Diverse User") {
		t.Errorf("user detail page missing display name %q", "Diverse User")
	}
	if !strings.Contains(body, "diverse@test.com") {
		t.Error("user detail page missing email")
	}
}

// TestPanel_RenderTenantWithMembers exercises templ branches for
// tenant detail with members list.
func TestPanel_RenderTenantWithMembers(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	// Create tenant
	mustCreateTenant(t, svc, "sigma")

	// Register and add members
	m1 := mustRegister(t, svc, "u-sig1", "sig1@sigma.com")
	mustAddMember(t, svc, m1.User.ID, identitymodel.NewTenantID("sigma"), identitymodel.RoleAdmin)
	m2 := mustRegister(t, svc, "u-sig2", "sig2@sigma.com")
	mustAddMember(t, svc, m2.User.ID, identitymodel.NewTenantID("sigma"), identitymodel.RoleViewer)

	// Render tenant detail
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/sigma", nil))
	body := rec.Body.String()

	actor1 := identitymodel.ActorIDFromUser(m1.User.ID).PrefixedString()
	for _, want := range []string{"sigma", actor1, "Members"} {
		if !strings.Contains(body, want) {
			t.Errorf("tenant detail page missing %q", want)
		}
	}
}

// TestPanel_RenderSuspendedTenant exercises the suspended-tenant templ branch.
func TestPanel_RenderSuspendedTenant(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	mustCreateTenant(t, svc, "suspended-tenant")
	if err := svc.SuspendTenant(
		context.Background(),
		identitymodel.NewTenantID("suspended-tenant"),
		"test suspension",
	); err != nil {
		t.Fatalf("SuspendTenant: %v", err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/tenants/suspended-tenant", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("suspended tenant detail: status %d, body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "suspended-tenant") {
		t.Error("suspended tenant detail should show tenant name")
	}
}

// TestPanel_UserDetailWithRoles exercises the ULID-format user-id path of the
// user detail page (ParseUserID requires ULIDs for synthetic detail lookups).
func TestPanel_UserDetailWithRoles(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	// Register a user with a ULID-format ID (required by ParseUserID)
	detailID := "01HXKYGEG0QH8XJYQKZ3TOTP99"
	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:    identitymodel.MustParseUserID(detailID),
		Email: "detail@example.com",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/"+detailID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("user detail: status %d, body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "detail@example.com") {
		t.Error("user detail page should show email")
	}
}
