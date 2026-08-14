package usermgmt

import (
	"context"
	"testing"
)

func TestService_Register(t *testing.T) {
	svc, err := NewService(newTestServiceConfig())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	resp, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("user-1"), Email: "test@example.com",
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
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)
	registerTestUser(t, svc, "u1", "a@b.com")

	_, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u2"), Email: "a@b.com",
	})
	assertErrorIs(t, err, ErrEmailExists, "ErrEmailExists")
}

func TestService_Register_Validation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Empty ID is auto-generated — registration should succeed.
	resp, err := svc.Register(ctx, RegisterRequest{
		ID: NewUserID(""), Email: "autoid@test.com",
	})
	if err != nil {
		t.Fatalf("Register with empty ID should auto-generate: %v", err)
	}
	if resp.User.ID.IsZero() {
		t.Error("expected auto-generated user ID, got zero")
	}

	_, err = svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u1"), Email: "invalid",
	})
	assertErrorIs(t, err, ErrValidation, "bad email")
}

func TestService_Register_TrimmedEmail(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	resp, err := svc.Register(ctx, RegisterRequest{
		ID:    NewUserID("u1"),
		Email: "  spaced@test.com  ",
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
		DisplayName: "  Spaced Name  ",
	})
	if err != nil {
		t.Fatalf("Register with trimmed display name: %v", err)
	}
	if resp.User.DisplayName != "Spaced Name" {
		t.Errorf("expected trimmed display name 'Spaced Name', got %q", resp.User.DisplayName)
	}
}

func TestService_Register_NoDisplayName(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	resp, err := svc.Register(ctx, RegisterRequest{
		ID:    NewUserID("u1"),
		Email: "nodisplay@test.com",
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
		ID: NewUserID("u1"), Email: "Case@Test.COM",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, err = svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u2"), Email: "case@test.com",
	})
	assertErrorIs(t, err, ErrEmailExists, "ErrEmailExists for case-insensitive duplicate")
}

func TestService_Register_MaxUsersReached(t *testing.T) {
	svc, err := NewService(ServiceConfig{MaxUsers: 1})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx := context.Background()

	registerTestUser(t, svc, "u1", "first@example.com")

	_, err = svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u2"), Email: "second@example.com",
	})
	assertErrorIs(t, err, ErrRegistrationClosed, "registration closed after max users reached")
}

func TestService_Register_MaxUsersZero_Unlimited(t *testing.T) {
	svc, err := NewService(ServiceConfig{MaxUsers: 0})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	for i := range 5 {
		registerTestUser(t, svc, "u"+string(rune('1'+i)), "user"+string(rune('1'+i))+"@example.com")
	}
}

func TestService_Register_MaxUsersTwo_AllowsThird(t *testing.T) {
	svc, err := NewService(ServiceConfig{MaxUsers: 2})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx := context.Background()

	registerTestUser(t, svc, "u1", "first@example.com")
	registerTestUser(t, svc, "u2", "second@example.com")

	_, err = svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u3"), Email: "third@example.com",
	})
	assertErrorIs(t, err, ErrRegistrationClosed, "registration closed after 2 users with MaxUsers=2")
}
