package usermgmt

import (
	"testing"
)

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

func TestUserID_NewUserID(t *testing.T) {
	id := NewUserID("user-123")
	if id.IsZero() {
		t.Error("expected non-zero UserID")
	}
	if id.Get() != "user-123" {
		t.Errorf("expected 'user-123', got %q", id.Get())
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
