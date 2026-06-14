package usermgmt

import (
	"context"
	"strings"
	"testing"
)

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

func TestService_Register_TrimmedEmail(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	resp, err := svc.Register(ctx, RegisterRequest{
		ID:       NewUserID("u1"),
		Email:    "  spaced@test.com  ",
		Password: "secret12",
	})
	if err != nil {
		t.Fatalf("Register with trimmed email: %v", err)
	}
	if resp.User.Email != "spaced@test.com" {
		t.Errorf("expected trimmed email 'spaced@test.com', got %q", resp.User.Email)
	}
}

func TestService_Register_TrimmedDisplayName(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	resp, err := svc.Register(ctx, RegisterRequest{
		ID:          NewUserID("u1"),
		Email:       "trimdisplay@test.com",
		Password:    "secret12",
		DisplayName: "  Spaced Name  ",
	})
	if err != nil {
		t.Fatalf("Register with trimmed display name: %v", err)
	}
	if resp.User.DisplayName != "Spaced Name" {
		t.Errorf("expected trimmed display name 'Spaced Name', got %q", resp.User.DisplayName)
	}
}

func TestService_Register_PasswordTooLong(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterRequest{
		ID:       NewUserID("u1"),
		Email:    "longpw@test.com",
		Password: strings.Repeat("x", 129),
	})
	assertErrorIs(t, err, ErrValidation, "ErrValidation for too-long password")
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

func TestService_Register_DuplicateEmail_CaseInsensitive(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u1"), Email: "Case@Test.COM", Password: "secret12",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, err = svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u2"), Email: "case@test.com", Password: "secret12",
	})
	assertErrorIs(t, err, ErrEmailExists, "ErrEmailExists for case-insensitive duplicate")
}
