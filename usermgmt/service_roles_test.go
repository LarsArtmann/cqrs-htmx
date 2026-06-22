package usermgmt

import (
	"testing"
)

func TestService_RolesViaMembership(t *testing.T) {
	svc, _, _ := newTestServiceWithUser(t, "user-1", "a@b.com")

	uid := NewUserID("user-1")
	grantTestRole(t, svc, uid, RoleAdmin)

	uidStr := uid.Get().String()
	ok, _ := svc.Authz().Enforce(uidStr, uidStr, "anything", ActionAll)
	if !ok {
		t.Error("expected admin user to have full access in test domain")
	}
}
