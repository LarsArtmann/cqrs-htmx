package usermgmt

import (
	"context"
	"errors"
	"strings"
	"testing"
)

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

func TestService_ChangePassword_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	err := svc.ChangePassword(ctx, NewUserID("nonexistent"), "old", "newpass12")
	assertErrorIs(t, err, ErrUserNotFound, "ErrUserNotFound")
}

func TestService_ChangePassword_NewPasswordTooLong(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "user-1", "longchange@test.com", "secret12")

	err := svc.ChangePassword(ctx, NewUserID("user-1"), "secret12", strings.Repeat("y", 129))
	assertErrorIs(t, err, ErrValidation, "ErrValidation for too-long new password")
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		pw      string
		wantErr bool
	}{
		{"empty", "", true},
		{"too short", "abc", true},
		{"boundary short", "1234567", true},
		{"boundary valid", "12345678", false},
		{"normal", "secret12", false},
		{"at max", strings.Repeat("x", 128), false},
		{"over max", strings.Repeat("x", 129), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.pw)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePassword(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}
