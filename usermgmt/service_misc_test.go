package usermgmt

import (
	"context"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestNewService_Defaults(t *testing.T) {
	svc, err := NewService(newTestServiceConfig())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if svc.Authz() == nil {
		t.Error("expected non-nil Authz")
	}
}

func TestService_Logout(t *testing.T) {
	svc, ctx, reg := newTestServiceWithUser(t, "user-1", "a@b.com")

	if err := svc.Logout(ctx, reg.Session.Token); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	_, err := svc.Authenticate(ctx, reg.Session.Token)
	if err == nil {
		t.Error("expected error after logout")
	}
}

func TestService_Authorize(t *testing.T) {
	svc, _ := NewService(ServiceConfig{
		Authz: newTestAuthz(
			t,
			Policy{RoleOwner, "*", "game.play_round", ActionExecute, EffectAllow},
		),
	})
	ctx := context.Background()
	registerTestUser(t, svc, "user-1", "a@b.com")
	_ = svc.Authz().
		AddGroupPolicy(GroupPolicy{Subject: "user-1", Role: RoleOwner, Domain: "game-1"})

	err := svc.Authorize(ctx, "user-1", "game-1", "game.play_round", ActionExecute)
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	err = svc.Authorize(ctx, "user-1", "other-game", "game.play_round", ActionExecute)
	if err == nil {
		t.Error("expected forbidden for wrong domain")
	}
}

func TestUser_Clone(t *testing.T) {
	original := &User{
		ID:          NewUserID("u1"),
		Email:       "clone@test.com",
		DisplayName: "Original",
		Credentials: []WebAuthnCredential{{CredentialCore: CredentialCore{ID: []byte{1}}}},
	}

	cloned := original.Clone()
	if cloned == original {
		t.Error("Clone returned same pointer")
	}
	if cloned.Email != original.Email {
		t.Errorf("Clone email = %q, want %q", cloned.Email, original.Email)
	}

	cloned.Email = "modified@test.com"
	cloned.Credentials[0].ID = []byte{2}
	if original.Email != "clone@test.com" {
		t.Error("Clone mutation affected original Email")
	}
	if original.Credentials[0].ID[0] != 1 {
		t.Error("Clone mutation affected original Credentials")
	}
}

func TestUser_Clone_Nil(t *testing.T) {
	var u *User
	if u.Clone() != nil {
		t.Error("expected nil Clone for nil User")
	}
}

func TestWithUserIDContext(t *testing.T) {
	t.Run("annotates event.Error with user_id context", func(t *testing.T) {
		original := errorfamily.NewTransient("test code", "test message")
		uid := NewUserID("user-42")
		got := withUserIDContext(original, uid)

		if got == nil {
			t.Fatal("expected non-nil error")
		}
		if !got.HasContext("user_id") {
			t.Errorf("expected user_id context, got %#v", got.ErrorContext())
		}
		if got.ContextValue("user_id") != uid.Get().String() {
			t.Errorf("expected user_id=%s, got %q", uid.Get().String(), got.ContextValue("user_id"))
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		if got := withUserIDContext(nil, NewUserID("user-1")); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("returns error unchanged for zero user ID", func(t *testing.T) {
		original := errorfamily.NewTransient("code", "msg")
		got := withUserIDContext(original, UserID{})
		if got != original {
			t.Errorf("expected unchanged error pointer")
		}
		if got.HasContext("user_id") {
			t.Errorf("zero user ID should not add context")
		}
	})
}
