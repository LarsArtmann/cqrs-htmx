package usermgmt

import (
	"context"
	"testing"
)

func TestEmailVerification_SendAndVerify(t *testing.T) {
	svc := newTestServiceWithEmailVerification(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "ev1", "ev1@test.com")

	if reg.User.EmailVerified {
		t.Fatal("new user should not be verified")
	}

	token, err := svc.SendVerificationEmail(ctx, reg.User.ID)
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	if err := svc.VerifyEmail(ctx, token); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	user, ok := svc.readModel.FindByUserID(reg.User.ID)
	if !ok {
		t.Fatal("user not found")
	}
	if !user.EmailVerified {
		t.Error("expected EmailVerified=true after verification")
	}
}

func TestEmailVerification_TokenConsumedAfterUse(t *testing.T) {
	svc := newTestServiceWithEmailVerification(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "ev2", "ev2@test.com")

	token, err := svc.SendVerificationEmail(ctx, reg.User.ID)
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	if err := svc.VerifyEmail(ctx, token); err != nil {
		t.Fatalf("VerifyEmail first: %v", err)
	}

	err = svc.VerifyEmail(ctx, token)
	if err == nil {
		t.Fatal("expected error on reused token")
	}
}

func TestEmailVerification_AlreadyVerified(t *testing.T) {
	svc := newTestServiceWithEmailVerification(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "ev3", "ev3@test.com")

	token, _ := svc.SendVerificationEmail(ctx, reg.User.ID)
	_ = svc.VerifyEmail(ctx, token)

	_, err := svc.SendVerificationEmail(ctx, reg.User.ID)
	if err == nil {
		t.Fatal("expected error when sending verification for already-verified email")
	}
}

func TestEmailVerification_InvalidToken(t *testing.T) {
	svc := newTestServiceWithEmailVerification(t)
	ctx := context.Background()

	err := svc.VerifyEmail(ctx, "invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestEmailVerification_NotConfigured(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.SendVerificationEmail(ctx, NewUserID("ev4"))
	if err == nil {
		t.Fatal("expected error when email verification not configured")
	}

	err = svc.VerifyEmail(ctx, "any-token")
	if err == nil {
		t.Fatal("expected error when email verification not configured")
	}
}

func TestEmailVerification_EmailChangeResetsVerification(t *testing.T) {
	svc := newTestServiceWithEmailVerification(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "ev5", "ev5@test.com")

	// Verify email
	token, _ := svc.SendVerificationEmail(ctx, reg.User.ID)
	if err := svc.VerifyEmail(ctx, token); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	user, _ := svc.readModel.FindByUserID(reg.User.ID)
	if !user.EmailVerified {
		t.Fatal("expected verified before email change")
	}

	// Change email
	if err := svc.ChangeEmail(ctx, reg.User.ID, "ev5-new@test.com"); err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}

	user, _ = svc.readModel.FindByUserID(reg.User.ID)
	if user.EmailVerified {
		t.Error("expected EmailVerified=false after email change")
	}
}

func TestEmailVerification_SendEmailCallback(t *testing.T) {
	var capturedEmail, capturedToken string
	svc := newTestServiceWithEmailVerificationAndCallback(t, func(_ context.Context, email, token string) error {
		capturedEmail = email
		capturedToken = token
		return nil
	})
	ctx := context.Background()
	reg := registerTestUser(t, svc, "ev6", "ev6@test.com")

	token, err := svc.SendVerificationEmail(ctx, reg.User.ID)
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	if capturedEmail != "ev6@test.com" {
		t.Errorf("callback email = %q", capturedEmail)
	}
	if capturedToken != token {
		t.Errorf("callback token mismatch")
	}
}

// --- helpers ---

func newTestServiceWithEmailVerification(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService(ServiceConfig{
		EmailVerification: &EmailVerificationConfig{},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)
	return svc
}

func newTestServiceWithEmailVerificationAndCallback(
	t *testing.T,
	fn SendVerificationEmailFunc,
) *Service {
	t.Helper()
	svc, err := NewService(ServiceConfig{
		EmailVerification: &EmailVerificationConfig{
			SendEmail: fn,
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)
	return svc
}
