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

// TestUsersIndex_AccessColumn pins the directory's access visibility: platform
// roles render as badges and tenant memberships as a count, so an admin can
// spot who holds power WITHOUT opening every profile.
func TestUsersIndex_AccessColumn(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	granted, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-granted"), Email: "granted@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-plain"), Email: "plain@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.GrantRoles(ctx,
		identitymodel.ActorIDFromUser(granted.User.ID),
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
		[]identitymodel.Role{identitymodel.RoleSuperAdmin},
	); err != nil {
		t.Fatal(err)
	}
	tenant := mustCreateTenant(t, svc, "acme")
	mustAddMember(t, svc, granted.User.ID, tenant.ID, identitymodel.RoleOwner)

	body := renderUsersIndex(t, h, "/admin/users?q=granted")

	if !strings.Contains(body, "Access") {
		t.Error("users table should carry an Access column")
	}
	if !strings.Contains(body, "super_admin") {
		t.Error("platform role badge missing from the access column")
	}
	if !strings.Contains(body, "1 tenant") {
		t.Error("tenant membership count missing from the access column")
	}

	// A user with no platform role and no tenant reads as a quiet dash, not a
	// blank cell that could mean "still loading".
	plain := renderUsersIndex(t, h, "/admin/users?q=plain")
	if strings.Contains(plain, "super_admin") {
		t.Error("plain user must not inherit another user's role badge")
	}
	if !strings.Contains(plain, "—") {
		t.Error("access-less user should render the dash placeholder")
	}
}

// TestUsersIndex_AdministratorsChip pins the privileged-user shortcut: the chip
// narrows the directory to platform-role holders only, and composes with the
// search.
func TestUsersIndex_AdministratorsChip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	granted, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-boss"), Email: "boss@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-nobody"), Email: "nobody@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.GrantRoles(ctx,
		identitymodel.ActorIDFromUser(granted.User.ID),
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
		[]identitymodel.Role{identitymodel.RoleAdmin},
	); err != nil {
		t.Fatal(err)
	}

	body := renderUsersIndex(t, h, "/admin/users?filter=admin")
	if !strings.Contains(body, "boss@example.com") {
		t.Error("administrators chip should keep the granted user")
	}
	if strings.Contains(body, "nobody@example.com") {
		t.Error("administrators chip should drop users without platform roles")
	}
	if !strings.Contains(body, "Administrators") {
		t.Error("the chip itself should be rendered")
	}

	// Composes with search: a search that excludes the admin renders the
	// honest empty state instead of leaking them back in.
	empty := renderUsersIndex(t, h, "/admin/users?filter=admin&q=nobody")
	if strings.Contains(empty, "boss@example.com") {
		t.Error("search must still narrow a filtered list")
	}
}

func renderUsersIndex(t *testing.T, h http.Handler, target string) string {
	t.Helper()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d; body=%s", target, rec.Code, rec.Body.String())
	}

	return rec.Body.String()
}
