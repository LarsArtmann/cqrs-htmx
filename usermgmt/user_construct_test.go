package usermgmt

import (
	"testing"
)

func TestNewUser(t *testing.T) {
	u := NewUser(NewUserID("user-1"), "test@example.com", "Test User")
	if u.ID != NewUserID("user-1") {
		t.Errorf("expected ID user-1, got %s", u.ID)
	}
	if u.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", u.Email)
	}
	if len(u.Roles) != 1 || u.Roles[0] != RoleViewer {
		t.Errorf("expected default role [viewer], got %v", u.Roles)
	}
	if u.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if u.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestUser_Password(t *testing.T) {
	u := NewUser(NewUserID("user-1"), "test@example.com", "Test User")
	if err := u.SetPasswordWithCost("secret123", minBcryptCost); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}
	if u.PasswordHash == "" {
		t.Error("expected non-empty password hash")
	}
	if !u.CheckPassword("secret123") {
		t.Error("expected password to match")
	}
	if u.CheckPassword("wrong") {
		t.Error("expected wrong password not to match")
	}
}

func TestUser_SetPassword(t *testing.T) {
	u := NewUser(NewUserID("u1"), "a@b.com", "Test")
	if err := u.SetPassword("secret123"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	if !u.CheckPassword("secret123") {
		t.Error("expected password to match after SetPassword")
	}
}
