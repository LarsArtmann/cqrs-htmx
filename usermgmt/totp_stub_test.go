package usermgmt

import (
	"encoding/base32"
	"fmt"
)

// testTOTPProvider is a deterministic TOTPProvider stub for testing the
// Service's TOTP domain logic without importing the real pquerna/otp-based
// totp module. It generates a fixed secret and validates against a fixed code.
type testTOTPProvider struct {
	issuer string
}

func newTestTOTPProvider(issuer string) testTOTPProvider {
	if issuer == "" {
		issuer = "cqrs-htmx" // matches totp.Provider default
	}
	return testTOTPProvider{issuer: issuer}
}

const testTOTPValidCode = "123456"

func (p testTOTPProvider) GenerateSecret(accountName string) ([]byte, string, string, error) {
	rawSecret := []byte("test-totp-secret-20b")
	b32Secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawSecret)
	uri := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", p.issuer, accountName, b32Secret, p.issuer)
	return rawSecret, b32Secret, uri, nil
}

func (testTOTPProvider) ValidateCode(_ []byte, code string) bool {
	return code == testTOTPValidCode
}
