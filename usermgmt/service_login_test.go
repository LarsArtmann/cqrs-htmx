package usermgmt

import (
	"testing"
)

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

func TestService_Login_TrimmedEmail(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "user-1", "trim@test.com", "secret12")

	_, err := svc.Login(ctx, LoginRequest{Email: "  trim@test.com  ", Password: "secret12"})
	if err != nil {
		t.Fatalf("Login with trimmed email: %v", err)
	}
}

func TestService_Login_CaseInsensitive(t *testing.T) {
	svc, ctx, _ := newTestServiceWithUser(t, "user-1", "CASE@TEST.COM", "secret12")

	resp, err := svc.Login(ctx, LoginRequest{Email: "case@test.com", Password: "secret12"})
	if err != nil {
		t.Fatalf("Login with case-insensitive email: %v", err)
	}
	if resp.User.ID != NewUserID("user-1") {
		t.Errorf("expected user ID user-1, got %s", resp.User.ID)
	}
}
