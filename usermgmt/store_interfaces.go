package usermgmt

import (
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnSessionStore manages WebAuthn challenge session data.
// The default in-memory implementation is suitable for single-process deployments.
// Implement this interface with Redis or SQL for multi-instance deployments.
type WebAuthnSessionStore interface {
	Save(key string, data *webauthn.SessionData)
	Get(key string) (*webauthn.SessionData, error)
	Delete(key string)
}

// VerificationTokenStore manages email verification tokens.
// The default in-memory implementation is suitable for single-process deployments.
// Implement this interface with Redis or SQL for multi-instance deployments.
type VerificationTokenStore interface {
	Save(userID UserID, email string, ttl time.Duration) (string, error)
	Consume(token string) (UserID, error)
}

// LockoutStore tracks failed authentication attempts and enforces temporary account lockout.
// The default in-memory implementation is suitable for single-process deployments.
// Implement this interface with Redis or SQL for distributed lockout across instances.
type LockoutStore interface {
	IsLocked(email string) bool
	RecordFailure(email string) bool
	Reset(email string)
}

// PendingTOTPStore manages pending TOTP setup secrets during the enable-TOTP ceremony.
// The default in-memory implementation is suitable for single-process deployments.
// Implement this interface with Redis or SQL for multi-instance deployments.
type PendingTOTPStore interface {
	Save(userID string, secret []byte, ttl time.Duration)
	Consume(userID string) ([]byte, bool)
}
