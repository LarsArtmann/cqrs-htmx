package usermgmt

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestService_Authenticate_InvalidToken(t *testing.T) {
	svc, _ := NewService(newTestServiceConfig())
	_, err := svc.Authenticate(context.Background(), "nonexistent-token")
	assertErrorIs(t, err, ErrUnauthorized, "ErrUnauthorized")
}

func TestService_Authenticate_SessionExpired(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "se1", "se@test.com", "secret12")

	reg.Session.ExpiresAt = time.Now().Add(-time.Hour)
	store, ok := svc.sessions.(*InMemorySessionStore)
	if !ok {
		t.Fatal("expected InMemorySessionStore")
	}
	store.mu.Lock()
	store.sessions[reg.Session.Token] = reg.Session
	store.mu.Unlock()

	_, err := svc.Authenticate(ctx, reg.Session.Token)
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}
}

func TestService_Authenticate_UserGone(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "ud1", "ud@test.com", "secret12")

	if err := svc.DeleteUser(ctx, reg.User.ID, "test cleanup"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	_, err := svc.Authenticate(ctx, reg.Session.Token)
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized (session revoked), got %v", err)
	}
}

func TestService_Register_DisplayNameTooLong(t *testing.T) {
	svc, _ := NewService(newTestServiceConfig())
	longName := strings.Repeat("x", 101)
	_, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "long@test.com", Password: "secret12",
		DisplayName: longName,
	})
	assertErrorIs(t, err, ErrValidation, "ErrValidation for long display name")
}

func TestService_Register_DuplicateUserID(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	registerTestUser(t, svc, "u1", "first@test.com", "secret12")

	_, err := svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u1"), Email: "second@test.com", Password: "secret12",
	})
	assertErrorIs(t, err, ErrUserIDExists, "ErrUserIDExists for duplicate user ID")
}

func TestService_Login_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Login(
		context.Background(),
		LoginRequest{Email: "nobody@test.com", Password: "secret12"},
	)
	assertErrorIs(t, err, ErrInvalidCredentials, "ErrInvalidCredentials")
}

func TestService_Logout_StoreError(t *testing.T) {
	sessions := &mockSessionStore{
		DeleteFn: func(_ context.Context, _ string) error {
			return errors.New("db connection lost")
		},
	}
	svc, _ := NewService(ServiceConfig{
		SessionStore: sessions,
		BcryptCost:   minBcryptCost,
	})

	err := svc.Logout(context.Background(), "some-token")
	if err == nil {
		t.Fatal("expected error from Logout when store fails")
	}
}

func TestService_GetUser_NotFound(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.GetUser(context.Background(), NewUserID("nonexistent"))
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestService_ChangePassword_WrongOld(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "cp1", "cp@test.com", "secret12")
	err := svc.ChangePassword(ctx, NewUserID("cp1"), "wrong-old", "newpass123")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestService_ChangePassword_NewTooShort(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "cp2", "cp2@test.com", "secret12")
	err := svc.ChangePassword(ctx, NewUserID("cp2"), "secret12", "short")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestService_ChangePassword_NewTooLong(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "cp3", "cp3@test.com", "secret12")
	longPass := strings.Repeat("a", 129)
	err := svc.ChangePassword(ctx, NewUserID("cp3"), "secret12", longPass)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestNewService_WithLogger(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		Lockout:    NewAccountLockout(),
	})
	if err != nil {
		t.Fatalf("NewService with lockout: %v", err)
	}
	if svc == nil {
		t.Error("expected non-nil service")
	}
}

func TestNewService_ZeroBcryptCost(t *testing.T) {
	svc, err := NewService(ServiceConfig{BcryptCost: 0})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if svc.bcryptCost != defaultBcryptCost {
		t.Errorf("expected default cost %d, got %d", defaultBcryptCost, svc.bcryptCost)
	}
}

func TestNewService_CustomSessionTTL(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		SessionTTL: 2 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if svc.sessionTTL != 2*time.Hour {
		t.Errorf("expected 2h TTL, got %v", svc.sessionTTL)
	}
}

func TestNewService_NilAuthz(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if svc.authz == nil {
		t.Error("expected default authz to be created")
	}
}
