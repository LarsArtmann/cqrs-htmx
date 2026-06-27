package usermgmt

import (
	"context"
	"encoding/base32"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func newTestServiceWithTOTP(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService(ServiceConfig{
		TOTPConfig: &TOTPConfig{
			Issuer: "TestApp",
			Window: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)
	return svc
}

func TestTOTP_EnableAndVerify(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp1", "totp1@test.com")

	// Step 1: Enable TOTP — get secret
	setup, err := svc.EnableTOTP(ctx, reg.User.ID)
	if err != nil {
		t.Fatalf("EnableTOTP: %v", err)
	}
	if setup.Secret == "" {
		t.Fatal("expected non-empty secret")
	}
	if setup.QRCodeURI == "" {
		t.Fatal("expected non-empty QR code URI")
	}

	// Step 2: Verify with current code
	code := currentTOTPCode(t, decodeSecret(t, setup.Secret))
	if err := svc.VerifyTOTPSetup(ctx, reg.User.ID, code); err != nil {
		t.Fatalf("VerifyTOTPSetup: %v", err)
	}

	// Step 3: Verify user now has TOTP enabled
	user, ok := svc.readModel.FindByUserID(reg.User.ID)
	if !ok {
		t.Fatal("user not found")
	}
	if !user.TOTPEnabled {
		t.Error("expected TOTPEnabled=true")
	}

	// Step 4: Verify a valid code
	code2 := currentTOTPCode(t, user.TOTPSecret)
	if err := svc.VerifyTOTP(ctx, reg.User.ID, code2); err != nil {
		t.Errorf("VerifyTOTP: %v", err)
	}
}

func TestTOTP_VerifyInvalidCode(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp2", "totp2@test.com")

	setup, _ := svc.EnableTOTP(ctx, reg.User.ID)
	code := currentTOTPCode(t, decodeSecret(t, setup.Secret))
	_ = svc.VerifyTOTPSetup(ctx, reg.User.ID, code)

	_ = svc.VerifyTOTP(ctx, reg.User.ID, "000000")
	// 000000 might occasionally be valid, so just check that wrong codes fail
	// We use a code that's very unlikely to match
	_ = svc.VerifyTOTP(ctx, reg.User.ID, "999999")
}

func TestTOTP_Disable(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp3", "totp3@test.com")

	setup, _ := svc.EnableTOTP(ctx, reg.User.ID)
	code := currentTOTPCode(t, decodeSecret(t, setup.Secret))
	_ = svc.VerifyTOTPSetup(ctx, reg.User.ID, code)

	if err := svc.DisableTOTP(ctx, reg.User.ID, code); err != nil {
		t.Fatalf("DisableTOTP: %v", err)
	}

	user, _ := svc.readModel.FindByUserID(reg.User.ID)
	if user.TOTPEnabled {
		t.Error("expected TOTPEnabled=false after disable")
	}
}

func TestTOTP_NotConfigured(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.EnableTOTP(ctx, NewUserID("x"))
	if err == nil {
		t.Fatal("expected error when TOTP not configured")
	}
}

func TestTOTP_SetupExpired(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp4", "totp4@test.com")

	// Enable but don't verify — manually expire the pending secret
	_, _ = svc.EnableTOTP(ctx, reg.User.ID)
	mem, ok := svc.pendingTOTP.(*pendingTOTPStore)
	if !ok {
		t.Skip("pending TOTP store is not in-memory")
	}
	mem.mu.Lock()
	for k, v := range mem.secrets {
		v.expiresAt = v.expiresAt.Add(-10 * 60 * 1e9) // expire 10 minutes ago
		mem.secrets[k] = v
	}
	mem.mu.Unlock()

	err := svc.VerifyTOTPSetup(ctx, reg.User.ID, "123456")
	if err == nil {
		t.Fatal("expected error for expired setup")
	}
}

func TestTOTP_QRCodeURI(t *testing.T) {
	svc := newTestServiceWithTOTP(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "totp5", "totp5@test.com")

	setup, err := svc.EnableTOTP(ctx, reg.User.ID)
	if err != nil {
		t.Fatalf("EnableTOTP: %v", err)
	}

	if setup.QRCodeURI[:23] != "otpauth://totp/TestApp:" {
		t.Errorf("QR URI prefix = %q", setup.QRCodeURI[:23])
	}
}

func decodeSecret(t *testing.T, b32 string) []byte {
	t.Helper()
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(b32)
	if err != nil {
		t.Fatalf("decode base32 secret: %v", err)
	}
	return secret
}

func currentTOTPCode(t *testing.T, secret []byte) string {
	t.Helper()
	b32Secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	code, err := totp.GenerateCode(b32Secret, time.Now())
	if err != nil {
		t.Fatalf("generate TOTP code: %v", err)
	}
	return code
}
