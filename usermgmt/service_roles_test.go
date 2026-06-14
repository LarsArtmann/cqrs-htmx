package usermgmt

import (
	"context"
	"errors"
	"testing"
)

func TestService_UpdateRoles(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "user-1", "a@b.com", "secret12")

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

func TestService_UpdateRoles_AuthzFailurePreservesUser(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "user-1", "a@b.com", "secret12")

	// Force authz failure by nil'ing the enforcer after user creation.
	svc.authz = &Authz{enforcer: nil}

	userBefore, _ := svc.GetUser(ctx, NewUserID("user-1"))
	beforeRoles := append([]Role(nil), userBefore.Roles...)

	err := svc.UpdateRoles(ctx, NewUserID("user-1"), []Role{RoleAdmin}, "user-1")
	if err == nil {
		t.Fatal("expected UpdateRoles to fail when authz Apply fails")
	}

	userAfter, _ := svc.GetUser(ctx, NewUserID("user-1"))
	if len(userAfter.Roles) != len(beforeRoles) {
		t.Fatalf("expected roles unchanged after failed update, got %v, want %v",
			userAfter.Roles, beforeRoles)
	}
	for i := range beforeRoles {
		if userAfter.Roles[i] != beforeRoles[i] {
			t.Fatalf("expected roles unchanged after failed update, got %v, want %v",
				userAfter.Roles, beforeRoles)
		}
	}
}

func TestService_UpdateRoles_AuthzApplyError(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "user-1", "a@b.com", "secret12")

	// Force authz Apply to fail by nil'ing the enforcer.
	svc.authz = &Authz{enforcer: nil}

	err := svc.UpdateRoles(ctx, NewUserID("user-1"), []Role{RoleAdmin}, "user-1")
	if err == nil {
		t.Fatal("expected UpdateRoles to fail when authz.Apply fails")
	}
}

func TestService_UpdateRoles_SaveError(t *testing.T) {
	users := &mockUserStore{
		FindByIDFn: func(_ context.Context, id UserID) (*User, error) {
			return NewUser(id, "a@b.com", ""), nil
		},
		SaveFn: func(_ context.Context, _ *User) error {
			return errors.New("db write failed")
		},
	}
	authz, aErr := NewAuthz()
	if aErr != nil {
		t.Fatalf("NewAuthz: %v", aErr)
	}
	svc, _ := NewService(ServiceConfig{
		UserStore:  users,
		BcryptCost: minBcryptCost,
		Authz:      authz,
	})

	ctx := context.Background()
	err := svc.UpdateRoles(ctx, NewUserID("user-1"), []Role{RoleAdmin}, "user-1")
	if err == nil {
		t.Fatal("expected UpdateRoles to fail when users.Save fails")
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
