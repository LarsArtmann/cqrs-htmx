package usermgmt

import (
	"context"
	"strings"
	"testing"
)

func TestTOTP_Enable_UserNotFound(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()

	if _, err := svc.EnableTOTP(ctx, NewUserID("ghost")); err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestTOTP_Enable_AlreadyEnabled(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp-ae", "totpae@test.com")

	setup, _ := svc.EnableTOTP(ctx, reg.User.ID)
	_ = svc.VerifyTOTPSetup(ctx, reg.User.ID, currentTOTPCode(t, decodeSecret(t, setup.Secret)))

	if _, err := svc.EnableTOTP(ctx, reg.User.ID); err == nil {
		t.Fatal("expected error when TOTP already enabled")
	}
}

func TestTOTP_Enable_DefaultIssuer(t *testing.T) {
	svc, err := NewService(ServiceConfig{TOTPConfig: &TOTPConfig{Window: 1}}) // empty Issuer
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp-di", "totpdi@test.com")

	setup, err := svc.EnableTOTP(ctx, reg.User.ID)
	if err != nil {
		t.Fatalf("EnableTOTP: %v", err)
	}
	if !strings.HasPrefix(setup.QRCodeURI, "otpauth://totp/cqrs-htmx") {
		t.Errorf("expected default issuer prefix, got %q", setup.QRCodeURI)
	}
}

func TestTOTP_VerifySetup_NoPendingSecret(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp-np", "totpnp@test.com")

	if err := svc.VerifyTOTPSetup(ctx, reg.User.ID, "123456"); err == nil {
		t.Fatal("expected error when no pending secret")
	}
}

func TestTOTP_VerifySetup_InvalidCode(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp-ic", "totpic@test.com")

	_, _ = svc.EnableTOTP(ctx, reg.User.ID)
	if err := svc.VerifyTOTPSetup(ctx, reg.User.ID, "000000"); err == nil {
		t.Fatal("expected error for invalid setup code")
	}
}

func TestTOTP_Verify_UserNotFound(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()

	if err := svc.VerifyTOTP(ctx, NewUserID("ghost"), "123456"); err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestTOTP_Verify_NotEnabled(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp-ne", "totpne@test.com")

	if err := svc.VerifyTOTP(ctx, reg.User.ID, "123456"); err == nil {
		t.Fatal("expected error when TOTP not enabled")
	}
}

func TestTOTP_Disable_UserNotFound(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()

	if err := svc.DisableTOTP(ctx, NewUserID("ghost"), "123456"); err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestTOTP_Disable_NotEnabled(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp-dne", "totpdne@test.com")

	if err := svc.DisableTOTP(ctx, reg.User.ID, "123456"); err == nil {
		t.Fatal("expected error when disabling TOTP that is not enabled")
	}
}

func TestTOTP_Disable_InvalidCode(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp-dic", "totpdic@test.com")

	setup, _ := svc.EnableTOTP(ctx, reg.User.ID)
	_ = svc.VerifyTOTPSetup(ctx, reg.User.ID, currentTOTPCode(t, decodeSecret(t, setup.Secret)))

	if err := svc.DisableTOTP(ctx, reg.User.ID, "000000"); err == nil {
		t.Fatal("expected error for invalid disable code")
	}
}

func TestTOTP_VerifyAndDisable_NotConfigured(t *testing.T) {
	svc := newTestService(t) // no TOTP config
	ctx := context.Background()
	uid := NewUserID("x")

	if err := svc.VerifyTOTP(ctx, uid, "123456"); err == nil {
		t.Error("VerifyTOTP: expected error when not configured")
	}
	if err := svc.VerifyTOTPSetup(ctx, uid, "123456"); err == nil {
		t.Error("VerifyTOTPSetup: expected error when not configured")
	}
	if err := svc.DisableTOTP(ctx, uid, "123456"); err == nil {
		t.Error("DisableTOTP: expected error when not configured")
	}
}
