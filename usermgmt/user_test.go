package usermgmt

import (
	"errors"
	"testing"
	"time"
)

func TestNewUser(t *testing.T) {
	u := NewUser("user-1", "test@example.com", "Test User")
	if u.ID != "user-1" {
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
	u := NewUser("user-1", "test@example.com", "Test User")
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

func TestUser_Roles(t *testing.T) {
	u := NewUser("user-1", "test@example.com", "Test User")
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

func TestUser_MarshalJSON(t *testing.T) {
	u := NewUser("user-1", "test@example.com", "Test User")
	_ = u.SetPasswordWithCost("secret", minBcryptCost)
	data, err := u.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) == "" {
		t.Error("expected non-empty JSON")
	}
}

func TestSession(t *testing.T) {
	session, err := NewSession("user-1", time.Hour)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	if session.UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", session.UserID)
	}
	if session.IsExpired() {
		t.Error("expected session not to be expired")
	}
	if !session.Valid(session.Token) {
		t.Error("expected session to be valid with correct token")
	}
	if session.Valid("wrong-token") {
		t.Error("expected session to be invalid with wrong token")
	}
}

func TestInMemoryUserStore(t *testing.T) {
	store := NewInMemoryUserStore()

	u := NewUser("user-1", "test@example.com", "Test User")
	_ = u.SetPasswordWithCost("pass", minBcryptCost)

	if err := store.Create(u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := store.FindByID("user-1")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", found.Email)
	}

	byEmail, err := store.FindByEmail("test@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if byEmail.ID != "user-1" {
		t.Errorf("expected ID user-1, got %s", byEmail.ID)
	}

	_, err = store.FindByID("nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}

	if err := store.Delete("user-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.FindByID("user-1")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound after delete, got %v", err)
	}
}

func TestInMemoryUserStore_CreateDuplicate(t *testing.T) {
	store := NewInMemoryUserStore()
	u1 := NewUser("u1", "dup@test.com", "One")
	_ = store.Create(u1)

	u2 := NewUser("u2", "dup@test.com", "Two")
	if err := store.Create(u2); !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestInMemoryUserStore_SaveUpdatesEmailIndex(t *testing.T) {
	store := NewInMemoryUserStore()
	u := NewUser("u1", "old@test.com", "Test")
	_ = store.Create(u)

	updated := NewUser("u1", "new@test.com", "Test")
	_ = store.Save(updated)

	if _, err := store.FindByEmail("old@test.com"); !errors.Is(err, ErrUserNotFound) {
		t.Error("expected old email to be gone from index")
	}
	if found, err := store.FindByEmail("new@test.com"); err != nil {
		t.Fatalf("FindByEmail new: %v", err)
	} else if found.ID != "u1" {
		t.Errorf("expected u1, got %s", found.ID)
	}
}

func TestInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()

	session, err := store.Create("user-1", time.Hour)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := store.Find(session.Token)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if found.UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", found.UserID)
	}

	_, err = store.Find("nonexistent")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}

	if err := store.Delete(session.Token); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.Find(session.Token)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestInMemorySessionStore_DeleteByUserID(t *testing.T) {
	store := NewInMemorySessionStore()
	s1, _ := store.Create("user-1", time.Hour)
	s2, _ := store.Create("user-1", time.Hour)
	_, _ = store.Create("user-2", time.Hour)

	if err := store.DeleteByUserID("user-1"); err != nil {
		t.Fatalf("DeleteByUserID: %v", err)
	}

	if _, err := store.Find(s1.Token); !errors.Is(err, ErrSessionNotFound) {
		t.Error("expected s1 deleted")
	}
	if _, err := store.Find(s2.Token); !errors.Is(err, ErrSessionNotFound) {
		t.Error("expected s2 deleted")
	}
}
