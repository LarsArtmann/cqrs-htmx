// Package totp provides TOTP (RFC 6238) secret generation and code validation
// using the pquerna/otp library.
//
// It implements the usermgmt.TOTPProvider interface via structural typing —
// no import of the core usermgmt package is required. Consumers inject a
// *Provider into usermgmt.ServiceConfig.TOTP to enable TOTP MFA.
package totp

import (
	"encoding/base32"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// Config configures the default TOTP provider.
type Config struct {
	// Issuer is the name shown in the authenticator app (e.g., "My App").
	// Defaults to "cqrs-htmx" when empty.
	Issuer string

	// Window is the number of time steps (±30s each) to accept before and
	// after the current time. Default is 1 (allows ±30 seconds clock drift).
	Window int
}

// Provider implements TOTP secret generation and code validation using
// pquerna/otp. It satisfies the usermgmt.TOTPProvider interface.
type Provider struct {
	issuer string
	window int
}

// New creates a TOTP provider with the given configuration.
func New(cfg Config) *Provider {
	issuer := cfg.Issuer
	if issuer == "" {
		issuer = "cqrs-htmx"
	}
	window := cfg.Window
	if window == 0 {
		window = 1
	}
	return &Provider{issuer: issuer, window: window}
}

// GenerateSecret creates a new TOTP secret for the given account name.
// Returns the raw secret bytes (for storage), the base32-encoded secret
// (for display), and the otpauth:// URI (for QR code generation).
func (p *Provider) GenerateSecret(accountName string) (rawSecret []byte, base32Secret, otpauthURI string, err error) {
	key, genErr := totp.Generate(totp.GenerateOpts{
		Issuer:      p.issuer,
		AccountName: accountName,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if genErr != nil {
		return nil, "", "", errorfamily.WrapInfrastructure(genErr, "totp.generate_key", "generate key").
			WithContext("account", accountName)
	}

	raw, decErr := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(key.Secret())
	if decErr != nil {
		return nil, "", "", errorfamily.WrapCorruption(decErr, "totp.decode_secret", "decode secret").
			WithContext("account", accountName)
	}

	return raw, key.Secret(), key.URL(), nil
}

// ValidateCode checks if the given TOTP code is valid for the provided
// raw secret bytes. Returns true if the code matches within the configured
// time window.
//
// # Replay protection
//
// ValidateCode is stateless: it does not track previously used codes. A valid
// code remains reusable within the acceptance window (±30s with the default
// Skew=1). RFC 6238 §5.2 recommends rejecting reused codes. Callers that need
// replay protection should wrap the provider with a used-code store (e.g.,
// a short-lived cache keyed by code hash that rejects duplicates within the
// current time step). The usermgmt.AccountLockout mechanism mitigates brute-force
// but does not prevent code reuse.
func (p *Provider) ValidateCode(rawSecret []byte, code string) bool {
	b32Secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawSecret)
	valid, err := totp.ValidateCustom(code, b32Secret, time.Now(), totp.ValidateOpts{
		Skew:      uint(p.window),
		Period:    30,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return err == nil && valid
}
