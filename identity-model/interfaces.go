package identitymodel

import (
	"context"
	"time"
)

// TOTPProvider provides TOTP secret generation and code validation.
type TOTPProvider interface {
	GenerateSecret(accountName string) (rawSecret []byte, base32Secret, otpauthURI string, err error)
	ValidateCode(rawSecret []byte, code string) bool
}

// WebAuthnProvider defines the WebAuthn (passkey) authentication contract.
type WebAuthnProvider interface {
	BeginRegistration(ctx context.Context, userJSON []byte) (options, sessionData []byte, err error)
	FinishRegistration(ctx context.Context, userJSON, body, sessionData []byte) (credentialJSON []byte, err error)
	BeginLogin(ctx context.Context, userJSON []byte) (options, sessionData []byte, err error)
	FinishLogin(ctx context.Context, userJSON, body, sessionData []byte) error
}

// OAuth2Provider defines the OAuth2/OIDC authentication contract.
type OAuth2Provider interface {
	BeginLogin(ctx context.Context, providerName, state string) (redirectURL, pkceVerifier string, err error)
	FinishLogin(ctx context.Context, providerName, code, pkceVerifier string) (userInfoJSON []byte, err error)
}

// OAuth2UserInfo is the normalized user information returned by an OAuth2
// provider after token exchange.
type OAuth2UserInfo struct {
	Subject       string `json:"subject"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	DisplayName   string `json:"display_name"`
}

// WebAuthnSessionStore manages WebAuthn challenge session data as opaque bytes.
type WebAuthnSessionStore interface {
	Save(key string, data []byte)
	Get(key string) ([]byte, error)
	Delete(key string)
}

// VerificationTokenStore manages email verification tokens.
type VerificationTokenStore interface {
	Save(userID UserID, email string, ttl time.Duration) (string, error)
	Consume(token string) (UserID, error)
}

// LockoutStore tracks failed authentication attempts and enforces temporary account lockout.
type LockoutStore interface {
	IsLocked(email string) bool
	RecordFailure(email string) bool
	Reset(email string)
}

// PendingTOTPStore manages pending TOTP setup secrets during the enable-TOTP ceremony.
type PendingTOTPStore interface {
	Save(userID string, secret []byte, ttl time.Duration)
	Consume(userID string) ([]byte, bool)
}
