package usermgmt

import (
	"context"
	"errors"
	"testing"
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
		ID: "user-1", Email: "test@example.com",
		Password: "secret123", DisplayName: "Test User",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.User.ID != "user-1" {
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
	svc, _ := NewService(newTestServiceConfig())
	ctx := context.Background()
	_, _ = svc.Register(ctx, RegisterRequest{ID: "u1", Email: "a@b.com", Password: "password"})

	_, err := svc.Register(ctx, RegisterRequest{ID: "u2", Email: "a@b.com", Password: "password"})
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestService_Login(t *testing.T) {
	svc, _ := NewService(newTestServiceConfig())
	ctx := context.Background()
	_, _ = svc.Register(ctx, RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret12"})

	resp, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "secret12"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.User.ID != "user-1" {
		t.Errorf("expected user ID user-1, got %s", resp.User.ID)
	}
	if resp.Session == nil {
		t.Error("expected non-nil session")
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	svc, _ := NewService(newTestServiceConfig())
	ctx := context.Background()
	_, _ = svc.Register(ctx, RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret12"})

	_, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "wrong"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestService_Authenticate(t *testing.T) {
	svc, _ := NewService(newTestServiceConfig())
	ctx := context.Background()
	reg, _ := svc.Register(ctx, RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret12"})

	user, err := svc.Authenticate(ctx, reg.Session.Token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if user.ID != "user-1" {
		t.Errorf("expected user ID user-1, got %s", user.ID)
	}
}

func TestService_Logout(t *testing.T) {
	svc, _ := NewService(newTestServiceConfig())
	ctx := context.Background()
	reg, _ := svc.Register(ctx, RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret12"})

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
		Authz: newTestAuthz(t,
			Policy{RoleOwner, "*", "game.play_round", ActionExecute, EffectAllow},
		),
		BcryptCost: minBcryptCost,
	})
	ctx := context.Background()
	_, _ = svc.Register(ctx, RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret12"})
	_ = svc.Authz().AddGroupPolicy(GroupPolicy{User: "user-1", Role: RoleOwner, Domain: "game-1"})

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
	svc, _ := NewService(newTestServiceConfig())
	ctx := context.Background()
	_, _ = svc.Register(ctx, RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret12"})

	if err := svc.UpdateRoles(ctx, "user-1", []string{RoleAdmin}, "user-1"); err != nil {
		t.Fatalf("UpdateRoles: %v", err)
	}

	ok, _ := svc.Authz().Enforce("user-1", "user-1", "anything", ActionAll)
	if !ok {
		t.Error("expected admin user to have full access in their domain")
	}

	user, _ := svc.GetUser(ctx, "user-1")
	if !user.HasRole(RoleAdmin) {
		t.Error("expected admin role in user object")
	}
}

func TestService_Register_Validation(t *testing.T) {
	svc, _ := NewService(newTestServiceConfig())
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterRequest{ID: "", Email: "a@b.com", Password: "secret12"})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for empty ID, got %v", err)
	}

	_, err = svc.Register(ctx, RegisterRequest{ID: "u1", Email: "invalid", Password: "secret12"})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for bad email, got %v", err)
	}

	_, err = svc.Register(ctx, RegisterRequest{ID: "u1", Email: "a@b.com", Password: "short"})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for short password, got %v", err)
	}
}

func TestService_Login_Validation(t *testing.T) {
	svc, _ := NewService(newTestServiceConfig())
	ctx := context.Background()

	_, err := svc.Login(ctx, LoginRequest{Email: "", Password: "secret12"})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for empty email, got %v", err)
	}

	_, err = svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: ""})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for empty password, got %v", err)
	}
}

func TestService_ChangePassword(t *testing.T) {
	svc, _ := NewService(newTestServiceConfig())
	ctx := context.Background()
	_, _ = svc.Register(ctx, RegisterRequest{ID: "user-1", Email: "a@b.com", Password: "secret12"})

	if err := svc.ChangePassword(ctx, "user-1", "wrongold", "newsecret1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for wrong old password, got %v", err)
	}

	if err := svc.ChangePassword(ctx, "user-1", "secret12", "short"); !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for short new password, got %v", err)
	}

	if err := svc.ChangePassword(ctx, "user-1", "secret12", "newsecret1"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	resp, err := svc.Login(ctx, LoginRequest{Email: "a@b.com", Password: "newsecret1"})
	if err != nil {
		t.Fatalf("Login with new password: %v", err)
	}
	if resp.User.ID != "user-1" {
		t.Errorf("expected user-1, got %s", resp.User.ID)
	}
}
