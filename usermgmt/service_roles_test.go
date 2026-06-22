package usermgmt

import (
	"testing"
)

func TestService_RolesViaMembership(t *testing.T) {
	svc, _, _ := newTestServiceWithUser(t, "user-1", "a@b.com")

	// Roles are managed by the Membership aggregate now.
	// Grant admin via membership and verify via Authz.
	grantTestRole(t, svc, NewUserID("user-1"), RoleAdmin)

	// grantTestRole grants in the user's self-scoped domain (matching legacy Casbin)
	ok, _ := svc.Authz().Enforce("user-1", "user-1", "anything", ActionAll)
	if !ok {
		t.Error("expected admin user to have full access in test domain")
	}
}
