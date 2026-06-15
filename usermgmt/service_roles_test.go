package usermgmt

import (
	"context"
	"testing"
)

func TestService_UpdateRoles(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "user-1", "a@b.com")

	if err := svc.UpdateRoles(ctx, NewUserID("user-1"), []Role{RoleAdmin}, "user-1"); err != nil {
		t.Fatalf("UpdateRoles: %v", err)
	}

	ok, _ := svc.Authz().Enforce("user-1", "user-1", "anything", ActionAll)
	if !ok {
		t.Error("expected admin user to have full access in their domain")
	}

	user, _ := svc.GetUser(ctx, NewUserID("user-1"))
	if !user.HasRole(RoleAdmin) {
		t.Error("expected admin role in user object")
	}
}

func TestService_UpdateRoles_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	err := svc.UpdateRoles(ctx, NewUserID("nonexistent"), []Role{RoleAdmin}, "dom")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}
