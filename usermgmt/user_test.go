package usermgmt

import (
	"errors"
	"testing"
	"time"
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

func TestUser_MarshalJSON(t *testing.T) {
	u := NewUser(NewUserID("user-1"), "test@example.com", "Test User")
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
	session, err := NewSession(NewUserID("user-1"), time.Hour)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	if session.UserID != NewUserID("user-1") {
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

	u := NewUser(NewUserID("user-1"), "test@example.com", "Test User")
	_ = u.SetPasswordWithCost("pass", minBcryptCost)

	if err := store.Create(u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := store.FindByID(NewUserID("user-1"))
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
	if byEmail.ID != NewUserID("user-1") {
		t.Errorf("expected ID user-1, got %s", byEmail.ID)
	}

	_, err = store.FindByID(NewUserID("nonexistent"))
	assertErrorIs(t, err, ErrUserNotFound, "ErrUserNotFound")

	if err := store.Delete(NewUserID("user-1")); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.FindByID(NewUserID("user-1"))
	assertErrorIs(t, err, ErrUserNotFound, "ErrUserNotFound")
}

func TestInMemoryUserStore_CreateDuplicate(t *testing.T) {
	store := NewInMemoryUserStore()
	u1 := NewUser(NewUserID("u1"), "dup@test.com", "One")
	_ = store.Create(u1)

	u2 := NewUser(NewUserID("u2"), "dup@test.com", "Two")
	if err := store.Create(u2); !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestInMemoryUserStore_SaveUpdatesEmailIndex(t *testing.T) {
	store := NewInMemoryUserStore()
	u := NewUser(NewUserID("u1"), "old@test.com", "Test")
	_ = store.Create(u)

	updated := NewUser(NewUserID("u1"), "new@test.com", "Test")
	_ = store.Save(updated)

	if _, err := store.FindByEmail("old@test.com"); !errors.Is(err, ErrUserNotFound) {
		t.Error("expected old email to be gone from index")
	}
	if found, err := store.FindByEmail("new@test.com"); err != nil {
		t.Fatalf("FindByEmail new: %v", err)
	} else if found.ID != NewUserID("u1") {
		t.Errorf("expected u1, got %s", found.ID)
	}
}

func TestInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()

	session, err := store.Create(NewUserID("user-1"), time.Hour)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := store.Find(session.Token)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if found.UserID != NewUserID("user-1") {
		t.Errorf("expected UserID user-1, got %s", found.UserID)
	}

	_, err = store.Find("nonexistent")
	assertErrorIs(t, err, ErrSessionNotFound, "ErrSessionNotFound")

	if err := store.Delete(session.Token); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.Find(session.Token)
	assertErrorIs(t, err, ErrSessionNotFound, "ErrSessionNotFound")
}

func TestInMemorySessionStore_DeleteByUserID(t *testing.T) {
	store := NewInMemorySessionStore()
	s1, _ := store.Create(NewUserID("user-1"), time.Hour)
	s2, _ := store.Create(NewUserID("user-1"), time.Hour)
	_, _ = store.Create(NewUserID("user-2"), time.Hour)

	if err := store.DeleteByUserID(NewUserID("user-1")); err != nil {
		t.Fatalf("DeleteByUserID: %v", err)
	}

	if _, err := store.Find(s1.Token); !errors.Is(err, ErrSessionNotFound) {
		t.Error("expected s1 deleted")
	}
	if _, err := store.Find(s2.Token); !errors.Is(err, ErrSessionNotFound) {
		t.Error("expected s2 deleted")
	}
}

func TestInMemoryUserStore_CreateDuplicateID(t *testing.T) {
	store := NewInMemoryUserStore()
	u := NewUser(NewUserID("dup-id"), "a@b.com", "Test")
	_ = store.Create(u)

	u2 := NewUser(NewUserID("dup-id"), "other@b.com", "Other")
	if err := store.Create(u2); err == nil {
		t.Error("expected error for duplicate user ID")
	}
}

func TestInMemorySessionStore_WithTTL(t *testing.T) {
	store := NewInMemorySessionStore().WithTTL(5 * time.Minute)
	s, err := store.Create(NewUserID("u1"), time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil session")
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

func TestUserID_NewUserID(t *testing.T) {
	id := NewUserID("user-123")
	if id.IsZero() {
		t.Error("expected non-zero UserID")
	}
	if id.Get() != "user-123" {
		t.Errorf("expected 'user-123', got %q", id.Get())
	}
}

func TestUserID_IsZero(t *testing.T) {
	var id UserID
	if !id.IsZero() {
		t.Error("expected zero UserID to be IsZero")
	}
	id = NewUserID("nonempty")
	if id.IsZero() {
		t.Error("expected non-zero UserID to not be IsZero")
	}
}

func TestUserID_Equal(t *testing.T) {
	a := NewUserID("same")
	b := NewUserID("same")
	c := NewUserID("different")
	if !a.Equal(b) {
		t.Error("expected equal UserIDs")
	}
	if a.Equal(c) {
		t.Error("expected different UserIDs to not be equal")
	}
}
