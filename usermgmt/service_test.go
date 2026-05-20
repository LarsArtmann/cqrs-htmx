package usermgmt

import (
	"context"
	"errors"
	"testing"
	"time"
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

func TestService_Register(t *testing.T) {
	svc, err := NewService(newTestServiceConfig())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	resp, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("user-1"), Email: "test@example.com",
		Password: "secret123", DisplayName: "Test User",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.User.ID != NewUserID("user-1") {
		t.Errorf("expected user ID user-1, got %s", resp.User.ID)
	}
	if resp.Session == nil {
		t.Error("expected non-nil session")
	}
	if !resp.User.HasRole(RoleUser) {
		t.Error("expected user role")
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)
	registerTestUser(t, svc, "u1", "a@b.com", "password")

	_, err := svc.Register(
		context.Background(),
		RegisterRequest{ID: NewUserID("u2"), Email: "a@b.com", Password: "password"},
	)
	assertErrorIs(t, err, ErrEmailExists, "ErrEmailExists")
}

func TestService_Login(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "user-1", "a@b.com", "secret12")

	resp, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "secret12"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.User.ID != NewUserID("user-1") {
		t.Errorf("expected user ID user-1, got %s", resp.User.ID)
	}
	if resp.Session == nil {
		t.Error("expected non-nil session")
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "user-1", "a@b.com", "secret12")

	_, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "wrong"})
	assertErrorIs(t, err, ErrInvalidCredentials, "ErrInvalidCredentials")
}

func TestService_Authenticate(t *testing.T) {
	svc, ctx, reg := newTestServiceWithUser(t, "user-1", "a@b.com", "secret12")

	user, err := svc.Authenticate(ctx, reg.Session.Token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if user.ID != NewUserID("user-1") {
		t.Errorf("expected user ID user-1, got %s", user.ID)
	}
}

func TestService_Logout(t *testing.T) {
	svc, ctx, reg := newTestServiceWithUser(t, "user-1", "a@b.com", "secret12")

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
		BcryptCost: minBcryptCost,
	})
	ctx := context.Background()
	registerTestUser(t, svc, "user-1", "a@b.com", "secret12")
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

func TestService_Register_Validation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(
		ctx,
		RegisterRequest{ID: NewUserID(""), Email: "a@b.com", Password: "secret12"},
	)
	assertErrorIs(t, err, ErrValidation, "empty ID")

	_, err = svc.Register(
		ctx,
		RegisterRequest{ID: NewUserID("u1"), Email: "invalid", Password: "secret12"},
	)
	assertErrorIs(t, err, ErrValidation, "bad email")

	_, err = svc.Register(
		ctx,
		RegisterRequest{ID: NewUserID("u1"), Email: "a@b.com", Password: "short"},
	)
	assertErrorIs(t, err, ErrValidation, "short password")
}

func TestService_Login_Validation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Login(ctx, LoginRequest{Email: "", Password: "secret12"})
	assertErrorIs(t, err, ErrValidation, "empty email")

	_, err = svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: ""})
	assertErrorIs(t, err, ErrValidation, "empty password")
}

func TestService_ChangePassword(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "user-1", "a@b.com", "secret12")

	if err := svc.ChangePassword(
		ctx,
		NewUserID("user-1"),
		"wrongold",
		"newsecret1",
	); !errors.Is(
		err,
		ErrInvalidCredentials,
	) {
		t.Errorf("expected ErrInvalidCredentials for wrong old password, got %v", err)
	}

	if err := svc.ChangePassword(
		ctx,
		NewUserID("user-1"),
		"secret12",
		"short",
	); !errors.Is(
		err,
		ErrValidation,
	) {
		t.Errorf("expected ErrValidation for short new password, got %v", err)
	}

	if err := svc.ChangePassword(ctx, NewUserID("user-1"), "secret12", "newsecret1"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	resp, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "newsecret1"})
	if err != nil {
		t.Fatalf("Login with new password: %v", err)
	}
	if resp.User.ID != NewUserID("user-1") {
		t.Errorf("expected user-1, got %s", resp.User.ID)
	}
}

func TestService_Authenticate_ExpiredSession(t *testing.T) {
	svc, ctx, reg := newTestServiceWithUser(t, "u1", "exp@test.com", "secret12")

	sessions := svc.sessions.(*InMemorySessionStore)
	sessions.mu.Lock()
	for _, s := range sessions.sessions {
		s.ExpiresAt = time.Now().Add(-time.Hour)
	}
	sessions.mu.Unlock()

	_, err := svc.Authenticate(ctx, reg.Session.Token)
	assertErrorIs(t, err, ErrSessionExpired, "ErrSessionExpired")
}

func TestService_Authenticate_UserDeleted(t *testing.T) {
	svc, ctx, reg := newTestServiceWithUser(t, "u1", "del@test.com", "secret12")

	svc.users.Delete(NewUserID("u1"))

	_, err := svc.Authenticate(ctx, reg.Session.Token)
	assertErrorIs(t, err, ErrUserNotFound, "ErrUserNotFound")
}

func TestService_ChangePassword_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	err := svc.ChangePassword(ctx, NewUserID("nonexistent"), "old", "newpass12")
	assertErrorIs(t, err, ErrUserNotFound, "ErrUserNotFound")
}

func TestService_UpdateRoles_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	err := svc.UpdateRoles(ctx, NewUserID("nonexistent"), []Role{RoleAdmin}, "dom")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestService_Register_NoDisplayName(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	resp, err := svc.Register(ctx, RegisterRequest{
		ID:       NewUserID("u1"),
		Email:    "nodisplay@test.com",
		Password: "secret12",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.User.DisplayName != "" {
		t.Errorf("expected empty display name, got %q", resp.User.DisplayName)
	}
}
