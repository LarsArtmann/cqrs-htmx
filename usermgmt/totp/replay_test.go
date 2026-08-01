package totp

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// TestProvider_ValidateCode_ReplayWithinWindow documents the stateless nature
// of TOTP validation: a valid code can be validated multiple times within the
// acceptance window. This is by design — the provider does not track used
// codes. Consumers that need replay protection should wrap the provider with
// a used-code store (see provider.go:78-86).
func TestProvider_ValidateCode_ReplayWithinWindow(t *testing.T) {
	t.Parallel()

	p := New(Config{Issuer: "TestApp"})

	rawSecret, b32Secret, _, err := p.GenerateSecret("replay@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}

	code, err := totp.GenerateCode(b32Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	if !p.ValidateCode(rawSecret, code) {
		t.Fatal("first validation should succeed")
	}

	if !p.ValidateCode(rawSecret, code) {
		t.Fatal("second validation should also succeed — provider is stateless by design")
	}
}

// TestProvider_WindowEffect shows that with Window=1, the current-time code
// is always accepted.
func TestProvider_WindowEffect(t *testing.T) {
	t.Parallel()

	p := New(Config{Issuer: "TestApp", Window: 1})

	rawSecret, b32Secret, _, err := p.GenerateSecret("window@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}

	codeNow, err := totp.GenerateCode(b32Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	if !p.ValidateCode(rawSecret, codeNow) {
		t.Fatal("current-time code should validate with Window=1")
	}
}
