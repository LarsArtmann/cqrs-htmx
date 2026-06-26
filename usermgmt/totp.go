package usermgmt

import (
	"context"
	"encoding/base32"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// totpTransient logs an auth failure and returns a transient error wrapping
// the underlying cause. Used by EnableTOTP when the totp library or base32
// decoder returns an error that is not the caller's fault.
func (s *Service) totpTransient(evt string, userID UserID, reason, msg string, err error) error {
	s.logAuth(evt, userID, "reason", reason)
	return event.NewTransient("internal", msg).WithCause(err)
}

// TOTPTimeStep is the standard TOTP time step (30 seconds, per RFC 6238).
const (
	TOTPTimeStep = 30 * time.Second
	// TOTPDigits is the number of digits in the TOTP code (6, per RFC 6238).
	TOTPDigits = 6
	// TOTPSecretLength is the length of the TOTP secret in bytes (20, per RFC 6238).
	TOTPSecretLength = 20
)

// TOTPConfig configures the TOTP multi-factor authentication flow.
type TOTPConfig struct {
	// Issuer is the name shown in the authenticator app (e.g., "My App").
	Issuer string
	// Window is the number of time steps to accept before and after the
	// current time. Default is 1 (allows ±30 seconds clock drift).
	Window int
}

// TOTPSetupResponse is returned when enabling TOTP for a user.
type TOTPSetupResponse struct {
	// Secret is the base32-encoded shared secret.
	Secret string `json:"secret"`
	// QRCodeURI is the otpauth:// URI for QR code generation.
	QRCodeURI string `json:"qr_code_uri"`
}

// EnableTOTP generates a new TOTP secret for the user and returns the
// setup information (secret + QR code URI). The secret is NOT yet active
// until VerifyTOTPSetup is called with a valid code from the authenticator app.
func (s *Service) EnableTOTP(ctx context.Context, userID UserID) (*TOTPSetupResponse, error) {
	if s.totpConfig == nil {
		return nil, ErrTOTPNotConfigured
	}
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		s.logAuth("totp_setup_failed", userID, "reason", "user_not_found")
		return nil, event.WrapRejection(ErrUserNotFound, "usermgmt.totp.user_not_found", "enable totp")
	}
	if user.TOTPEnabled {
		s.logAuth("totp_setup_failed", userID, "reason", "already_enabled")
		return nil, ErrTOTPAlreadyEnabled
	}
	issuer := s.totpConfig.Issuer
	if issuer == "" {
		issuer = "cqrs-htmx"
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: user.Email,
		Algorithm:   otp.AlgorithmSHA1, // RFC 6238 default
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return nil, s.totpTransient("totp_setup_failed", userID, "secret_generation_error",
			"generate totp key", err)
	}
	rawSecret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(key.Secret())
	if err != nil {
		return nil, s.totpTransient("totp_setup_failed", userID, "secret_decode_error",
			"decode totp secret", err)
	}
	// Store the pending secret temporarily
	s.pendingTOTP.mu.Lock()
	s.pendingTOTP.secrets[userID.Get().String()] = pendingTOTPSecret{
		secret:    rawSecret,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.pendingTOTP.mu.Unlock()
	s.logAuth("totp_setup_initiated", userID)
	return &TOTPSetupResponse{
		Secret:    key.Secret(),
		QRCodeURI: key.URL(),
	}, nil
}

// VerifyTOTPSetup confirms the TOTP setup by verifying a code from the
// user's authenticator app. On success, dispatches the TOTPEnabled event.
func (s *Service) VerifyTOTPSetup(ctx context.Context, userID UserID, code string) error {
	if s.totpConfig == nil {
		return ErrTOTPNotConfigured
	}
	s.pendingTOTP.mu.Lock()
	pending, ok := s.pendingTOTP.secrets[userID.Get().String()]
	if ok {
		delete(s.pendingTOTP.secrets, userID.Get().String())
	}
	s.pendingTOTP.mu.Unlock()
	if !ok || time.Now().After(pending.expiresAt) {
		s.logAuth("totp_setup_verify_failed", userID, "reason", "setup_expired")
		return ErrTOTPSetupExpired
	}
	if !validateTOTP(pending.secret, code, s.totpConfig.Window) {
		s.logAuth("totp_setup_verify_failed", userID, "reason", "invalid_code")
		return ErrInvalidTOTPCode
	}
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		s.logAuth("totp_setup_verify_failed", userID, "reason", "invalid_user_id")
		return event.WrapInfrastructure(err, "usermgmt.totp.userid_conversion_failed", "convert userID")
	}
	if err := s.dispatcher.Dispatch(ctx, NewEnableTOTPCmd(aggID, pending.secret)); err != nil {
		s.logAuth("totp_setup_verify_failed", userID, "reason", "dispatch_error")
		return event.Wrapf(err, event.Classify(err),
			"usermgmt.totp.dispatch_failed", "enable totp dispatch")
	}
	s.logAuth(statusTOTPEnabled, userID)
	return nil
}

// VerifyTOTP validates a TOTP code against the user's active TOTP secret.
// This is used as a second factor during login.
func (s *Service) VerifyTOTP(ctx context.Context, userID UserID, code string) error {
	if err := s.requireValidTOTP(userID, code, "totp_verify_failed"); err != nil {
		return err
	}
	s.logAuth(statusTOTPVerified, userID)
	return nil
}

// DisableTOTP removes the TOTP configuration for a user.
// A valid TOTP code is required to prevent MFA stripping via session hijack.
func (s *Service) DisableTOTP(ctx context.Context, userID UserID, code string) error {
	if err := s.requireValidTOTP(userID, code, "totp_disable_failed"); err != nil {
		return err
	}
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		s.logAuth("totp_disable_failed", userID, "reason", "invalid_user_id")
		return event.WrapInfrastructure(err, "usermgmt.totp.userid_conversion_failed", "convert userID")
	}
	if err := s.dispatcher.Dispatch(ctx, NewDisableTOTPCmd(aggID)); err != nil {
		s.logAuth("totp_disable_failed", userID, "reason", "dispatch_error")
		return event.Wrapf(err, event.Classify(err),
			"usermgmt.totp.dispatch_failed", "disable totp dispatch")
	}
	s.logAuth(statusTOTPDisabled, userID)
	return nil
}

func (s *Service) requireValidTOTP(userID UserID, code, failEvent string) error {
	if s.totpConfig == nil {
		return ErrTOTPNotConfigured
	}
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		s.logAuth(failEvent, userID, "reason", "user_not_found")
		return event.WrapRejection(ErrUserNotFound, "usermgmt.totp.user_not_found", "totp")
	}
	if !user.TOTPEnabled || len(user.TOTPSecret) == 0 {
		s.logAuth(failEvent, userID, "reason", "totp_not_enabled")
		return ErrTOTPNotEnabled
	}
	if !validateTOTP(user.TOTPSecret, code, s.totpConfig.Window) {
		s.logAuth(failEvent, userID, "reason", "invalid_code")
		return ErrInvalidTOTPCode
	}
	return nil
}

// --- TOTP validation (using pquerna/otp library) ---

func validateTOTP(secret []byte, code string, window int) bool {
	b32Secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	valid, err := totp.ValidateCustom(code, b32Secret, time.Now(), totp.ValidateOpts{
		Skew:      uint(window),
		Period:    30,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return err == nil && valid
}

// pendingTOTPSecret is a temporary secret stored between EnableTOTP and VerifyTOTPSetup.
type pendingTOTPSecret struct {
	secret    []byte
	expiresAt time.Time
}

// pendingTOTPStore holds in-flight TOTP secrets awaiting verification.
type pendingTOTPStore struct {
	mu      sync.Mutex
	secrets map[string]pendingTOTPSecret
}

// pendingTTOTPEvictionInterval is how often expired pending TOTP secrets are
// cleaned up by the background goroutine.
const pendingTTOTPEvictionInterval = 1 * time.Minute

func newPendingTOTPStore() pendingTOTPStore {
	return pendingTOTPStore{ //nolint:exhaustruct // mu is zero-value
		secrets: make(map[string]pendingTOTPSecret)}
}

// EvictExpired removes all pending TOTP secrets whose expiry time has passed.
// Returns the number of secrets evicted.
func (s *pendingTOTPStore) EvictExpired() int {
	return evictExpired(&s.mu, s.secrets, func(_ string, e pendingTOTPSecret) bool {
		return time.Now().After(e.expiresAt)
	})
}

// startEviction launches a background goroutine that periodically removes
// expired pending TOTP secrets. Returns a stop function that must be called
// to terminate the goroutine (e.g. on shutdown or in tests).
func (s *pendingTOTPStore) startEviction() (stop func()) {
	return startPeriodicEviction(s.EvictExpired, pendingTTOTPEvictionInterval)
}
