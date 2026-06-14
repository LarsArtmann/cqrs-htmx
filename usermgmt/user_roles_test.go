package usermgmt

import (
	"testing"
)

func TestUser_Roles(t *testing.T) {
	u := NewUser(NewUserID("user-1"), "test@example.com", "Test User")
	u.AddRole(RoleAdmin)
	if !u.HasRole(RoleAdmin) {
		t.Error("expected admin role")
	}
	if !u.HasRole(RoleViewer) {
		t.Error("expected viewer role still present")
	}
	u.AddRole(RoleAdmin)
	if len(u.Roles) != 2 {
		t.Errorf("expected 2 roles after duplicate add, got %d", len(u.Roles))
	}
	u.RemoveRole(RoleAdmin)
	if u.HasRole(RoleAdmin) {
		t.Error("expected admin role removed")
	}
}

func TestUser_SetRoles(t *testing.T) {
	u := NewUser(NewUserID("user-1"), "test@example.com", "Test User")
	before := u.UpdatedAt

	u.SetRoles([]Role{RoleAdmin, RoleOwner})

	if len(u.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(u.Roles))
	}
	if u.Roles[0] != RoleAdmin || u.Roles[1] != RoleOwner {
		t.Errorf("expected [admin, owner], got %v", u.Roles)
	}
	if !u.UpdatedAt.After(before) {
		t.Error("expected UpdatedAt to be updated")
	}

	original := u.Roles
	u.SetRoles([]Role{RoleUser})
	if len(original) != 2 {
		t.Error("expected original slice to not be mutated by SetRoles")
	}
}

func TestUser_SetRoles_Nil(t *testing.T) {
	u := NewUser(NewUserID("u1"), "a@b.com", "Test")
	before := u.UpdatedAt

	u.SetRoles(nil)

	if len(u.Roles) != 0 {
		t.Errorf("expected empty roles, got %v", u.Roles)
	}
	if !u.UpdatedAt.After(before) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestUser_SetRoles_Empty(t *testing.T) {
	u := NewUser(NewUserID("u1"), "a@b.com", "Test")
	u.SetRoles([]Role{})
	if len(u.Roles) != 0 {
		t.Errorf("expected empty roles, got %v", u.Roles)
	}
}
