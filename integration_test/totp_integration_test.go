package integration_test

import (
	"context"
	"testing"
	"time"

	otptotp "github.com/pquerna/otp/totp"

	usermgmttotp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// TestService_TOTP_EnableAndVerify_Integration verifies the full
// cross-module flow: Service.EnableTOTP → totp.Provider.GenerateSecret
// (pquerna/otp) → Service.VerifyTOTPSetup → event dispatch →
// Service.VerifyTOTP with a live code.
//
// This proves the TOTP provider interface contract works end-to-end:
// the Service calls GenerateSecret, stores the raw bytes, and later
// calls ValidateCode with those bytes. If the JSON/primitive-type
// contract drifts between core usermgmt and the totp sub-module,
// this test fails.
func TestService_TOTP_EnableAndVerify_Integration(t *testing.T) {
	t.Parallel()

	provider := usermgmttotp.New(usermgmttotp.Config{
		Issuer: "Integration Test",
		Window: 1,
	})

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		TOTP: provider,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	uid, err := usermgmt.ParseUserID("01HXKYGEG0QH8XJYQKZ3TOTP01")
	if err != nil {
		t.Fatalf("ParseUserID: %v", err)
	}

	_, err = svc.Register(context.Background(), usermgmt.RegisterRequest{
		ID:          uid,
		Email:       "totp-integration@test.com",
		DisplayName: "TOTP User",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Step 1: Enable TOTP — generates secret via provider
	setup, err := svc.EnableTOTP(context.Background(), uid)
	if err != nil {
		t.Fatalf("EnableTOTP: %v", err)
	}
	if setup.Secret == "" {
		t.Fatal("expected non-empty secret from EnableTOTP")
	}
	if setup.QRCodeURI == "" {
		t.Fatal("expected non-empty QR code URI from EnableTOTP")
	}

	// Step 2: Generate a valid TOTP code from the base32 secret
	code, err := otptotp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("totp.GenerateCode: %v", err)
	}

	// Step 3: Verify setup — dispatches TOTPEnabled event
	err = svc.VerifyTOTPSetup(context.Background(), uid, code)
	if err != nil {
		t.Fatalf("VerifyTOTPSetup: %v", err)
	}

	// Step 4: Verify a second code (new time step) against active TOTP
	code2, err := otptotp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("totp.GenerateCode (2nd): %v", err)
	}
	err = svc.VerifyTOTP(context.Background(), uid, code2)
	if err != nil {
		t.Fatalf("VerifyTOTP: %v", err)
	}

	// Step 5: Invalid code should fail
	err = svc.VerifyTOTP(context.Background(), uid, "000000")
	if err == nil {
		t.Error("expected error for invalid TOTP code")
	}
}

// TestService_TOTP_NilProvider_Guards verifies that a Service without
// a TOTP provider returns clear errors, not panics.
func TestService_TOTP_NilProvider_Guards(t *testing.T) {
	t.Parallel()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	uid, err := usermgmt.ParseUserID("01HXKYGEG0QH8XJYQKZ3TOTP02")
	if err != nil {
		t.Fatalf("ParseUserID: %v", err)
	}

	_, err = svc.EnableTOTP(context.Background(), uid)
	if err == nil {
		t.Error("expected error when TOTP is not configured (EnableTOTP)")
	}

	err = svc.VerifyTOTP(context.Background(), uid, "123456")
	if err == nil {
		t.Error("expected error when TOTP is not configured (VerifyTOTP)")
	}

	err = svc.DisableTOTP(context.Background(), uid, "123456")
	if err == nil {
		t.Error("expected error when TOTP is not configured (DisableTOTP)")
	}
}
