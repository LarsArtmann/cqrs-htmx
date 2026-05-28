package usermgmt

import (
	"context"
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

func TestUser_ChangePassword(t *testing.T) {
	u := NewUser(NewUserID("user-1"), "test@example.com", "Test User")
	_ = u.SetPasswordWithCost("oldpass12", minBcryptCost)
	before := u.UpdatedAt

	t.Run("success", func(t *testing.T) {
		matched, err := u.ChangePassword("oldpass12", "newpass12", minBcryptCost)
		if err != nil {
			t.Fatalf("ChangePassword: %v", err)
		}
		if !matched {
			t.Error("expected matched=true")
		}
		if !u.CheckPassword("newpass12") {
			t.Error("expected new password to work")
		}
		if !u.UpdatedAt.After(before) {
			t.Error("expected UpdatedAt to be updated")
		}
	})

	t.Run("wrong old password", func(t *testing.T) {
		matched, err := u.ChangePassword("wrongpass", "another1", minBcryptCost)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if matched {
			t.Error("expected matched=false for wrong old password")
		}
	})

	t.Run("new password too short", func(t *testing.T) {
		_, err := u.ChangePassword("newpass12", "short", minBcryptCost)
		if err == nil {
			t.Error("expected validation error for short password")
		}
	})
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

func TestUser_SetEmail(t *testing.T) {
	u := NewUser(NewUserID("u1"), "old@test.com", "Test")
	before := u.UpdatedAt

	u.SetEmail("new@test.com")

	if u.Email != "new@test.com" {
		t.Errorf("expected new@test.com, got %s", u.Email)
	}
	if !u.UpdatedAt.After(before) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestUser_SetDisplayName(t *testing.T) {
	u := NewUser(NewUserID("u1"), "a@b.com", "Old Name")
	before := u.UpdatedAt

	u.SetDisplayName("New Name")

	if u.DisplayName != "New Name" {
		t.Errorf("expected 'New Name', got %s", u.DisplayName)
	}
	if !u.UpdatedAt.After(before) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestUser_IsPasswordSet(t *testing.T) {
	u := NewUser(NewUserID("u1"), "a@b.com", "Test")
	if u.IsPasswordSet() {
		t.Error("expected no password initially")
	}
	_ = u.SetPasswordWithCost("secret123", minBcryptCost)
	if !u.IsPasswordSet() {
		t.Error("expected password to be set")
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
	if !session.TokenMatches(session.Token) {
		t.Error("expected TokenMatches to return true for correct token")
	}
	if session.TokenMatches("wrong-token") {
		t.Error("expected TokenMatches to return false for wrong token")
	}
}

func TestInMemoryUserStore(t *testing.T) {
	store := NewInMemoryUserStore()

	u := NewUser(NewUserID("user-1"), "test@example.com", "Test User")
	_ = u.SetPasswordWithCost("pass", minBcryptCost)

	if err := store.Create(context.Background(), u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := store.FindByID(context.Background(), NewUserID("user-1"))
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", found.Email)
	}

	byEmail, err := store.FindByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if byEmail.ID != NewUserID("user-1") {
		t.Errorf("expected ID user-1, got %s", byEmail.ID)
	}

	_, err = store.FindByID(context.Background(), NewUserID("nonexistent"))
	assertErrorIs(t, err, ErrUserNotFound, "ErrUserNotFound")

	if err := store.Delete(context.Background(), NewUserID("user-1")); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.FindByID(context.Background(), NewUserID("user-1"))
	assertErrorIs(t, err, ErrUserNotFound, "ErrUserNotFound")
}

func TestInMemoryUserStore_CreateDuplicate(t *testing.T) {
	store := NewInMemoryUserStore()
	u1 := NewUser(NewUserID("u1"), "dup@test.com", "One")
	_ = store.Create(context.Background(), u1)

	u2 := NewUser(NewUserID("u2"), "dup@test.com", "Two")
	if err := store.Create(context.Background(), u2); !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestInMemoryUserStore_SaveUpdatesEmailIndex(t *testing.T) {
	store := NewInMemoryUserStore()
	ctx := context.Background()
	u := NewUser(NewUserID("u1"), "old@test.com", "Test")
	_ = store.Create(ctx, u)

	updated := NewUser(NewUserID("u1"), "new@test.com", "Test")
	_ = store.Save(ctx, updated)

	_, err := store.FindByEmail(ctx, "old@test.com")
	if !errors.Is(err, ErrUserNotFound) {
		t.Error("expected old email to be gone from index")
	}
	found, err := store.FindByEmail(ctx, "new@test.com")
	if err != nil {
		t.Fatalf("FindByEmail new: %v", err)
	}
	if found.ID != NewUserID("u1") {
		t.Errorf("expected u1, got %s", found.ID)
	}
}

func TestInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()

	session, err := store.Create(context.Background(), NewUserID("user-1"), time.Hour)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := store.Find(context.Background(), session.Token)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if found.UserID != NewUserID("user-1") {
		t.Errorf("expected UserID user-1, got %s", found.UserID)
	}

	_, err = store.Find(context.Background(), "nonexistent")
	assertErrorIs(t, err, ErrSessionNotFound, "ErrSessionNotFound")

	if err := store.Delete(context.Background(), session.Token); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = store.Find(context.Background(), session.Token)
	assertErrorIs(t, err, ErrSessionNotFound, "ErrSessionNotFound")
}

func TestInMemorySessionStore_DeleteByUserID(t *testing.T) {
	store := NewInMemorySessionStore()
	s1, _ := store.Create(context.Background(), NewUserID("user-1"), time.Hour)
	s2, _ := store.Create(context.Background(), NewUserID("user-1"), time.Hour)
	_, _ = store.Create(context.Background(), NewUserID("user-2"), time.Hour)

	if err := store.DeleteByUserID(context.Background(), NewUserID("user-1")); err != nil {
		t.Fatalf("DeleteByUserID: %v", err)
	}

	if _, err := store.Find(context.Background(), s1.Token); !errors.Is(err, ErrSessionNotFound) {
		t.Error("expected s1 deleted")
	}
	if _, err := store.Find(context.Background(), s2.Token); !errors.Is(err, ErrSessionNotFound) {
		t.Error("expected s2 deleted")
	}
}

func TestInMemoryUserStore_CreateDuplicateID(t *testing.T) {
	store := NewInMemoryUserStore()
	u := NewUser(NewUserID("dup-id"), "a@b.com", "Test")
	_ = store.Create(context.Background(), u)

	u2 := NewUser(NewUserID("dup-id"), "other@b.com", "Other")
	if err := store.Create(context.Background(), u2); !errors.Is(err, ErrUserIDExists) {
		t.Errorf("expected ErrUserIDExists, got %v", err)
	}
}

func TestInMemorySessionStore_EvictExpired(t *testing.T) {
	store := NewInMemorySessionStore()
	s1, _ := store.Create(context.Background(), NewUserID("u1"), -time.Hour)
	s2, _ := store.Create(context.Background(), NewUserID("u2"), time.Hour)

	evicted := store.EvictExpired()
	if evicted != 1 {
		t.Errorf("expected 1 eviction, got %d", evicted)
	}
	if _, err := store.Find(context.Background(), s1.Token); !errors.Is(err, ErrSessionNotFound) {
		t.Error("expected expired session to be evicted")
	}
	if _, err := store.Find(context.Background(), s2.Token); err != nil {
		t.Errorf("expected valid session to remain: %v", err)
	}

	if store.EvictExpired() != 0 {
		t.Error("expected 0 evictions on clean store")
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
