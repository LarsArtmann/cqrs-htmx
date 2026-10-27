package totp

import (
	"encoding/base32"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestProvider_GenerateSecret(t *testing.T) {
	p := New(Config{Issuer: "TestApp", Window: 1})

	rawSecret, b32Secret, uri, err := p.GenerateSecret("user@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	if len(rawSecret) == 0 {
		t.Error("expected non-empty raw secret")
	}
	if b32Secret == "" {
		t.Error("expected non-empty base32 secret")
	}
	if uri == "" {
		t.Error("expected non-empty otpauth URI")
	}

	// Verify the base32 secret matches the raw secret
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(b32Secret)
	if err != nil {
		t.Fatalf("decode base32: %v", err)
	}
	if string(decoded) != string(rawSecret) {
		t.Error("base32 secret does not match raw secret")
	}

	// Verify URI prefix
	expected := "otpauth://totp/TestApp:user@example.com"
	if len(uri) < len(expected) || uri[:len(expected)] != expected {
		t.Errorf("URI prefix = %q, want %q", uri[:len(expected)], expected)
	}
}

func TestProvider_ValidateCode(t *testing.T) {
	p := New(Config{Issuer: "TestApp", Window: 1})

	rawSecret, _, _, err := p.GenerateSecret("user@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}

	// Generate a valid code using pquerna/otp
	b32Secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawSecret)
	validCode, err := totp.GenerateCode(b32Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	if !p.ValidateCode(rawSecret, validCode) {
		t.Error("expected valid code to pass")
	}
	if p.ValidateCode(rawSecret, "000000") {
		t.Error("expected invalid code to fail")
	}
}

func TestProvider_DefaultConfig(t *testing.T) {
	p := New(Config{})

	_, _, uri, err := p.GenerateSecret("user@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}

	// Default issuer should be "cqrs-htmx"
	expected := "otpauth://totp/cqrs-htmx:user@example.com"
	if len(uri) < len(expected) || uri[:len(expected)] != expected {
		t.Errorf("URI prefix = %q, want %q", uri[:len(expected)], expected)
	}
}

// TestNewNegativeWindowClampedToDefault pins S39/w03.7: a negative Window is
// clamped to the default (1). Pre-fix, ValidateCode widened the acceptance
// window via uint(Skew), so Window=-3 silently became a ~4-billion-step
// (multi-year) skew.
func TestNewNegativeWindowClampedToDefault(t *testing.T) {
	for _, window := range []int{-1, -3, -100} {
		p := New(Config{Issuer: "TestApp", Window: window})
		if p.window != 1 {
			t.Errorf("New(Window=%d).window = %d, want 1 (clamped)", window, p.window)
		}
	}

	// The clamped provider must still validate a current-time code normally.
	p := New(Config{Issuer: "TestApp", Window: -3})
	rawSecret, _, _, err := p.GenerateSecret("user@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	b32Secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawSecret)
	validCode, err := totp.GenerateCode(b32Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if !p.ValidateCode(rawSecret, validCode) {
		t.Error("clamped provider should validate a current-time code")
	}
}
