package usermgmt

import (
	"context"
	"encoding/base32"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

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
		return nil, fmt.Errorf("enable totp: %w", ErrUserNotFound)
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
		s.logAuth("totp_setup_failed", userID, "reason", "secret_generation_error")
		return nil, event.NewTransient("internal", "generate totp key").WithCause(err)
	}
	rawSecret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(key.Secret())
	if err != nil {
		s.logAuth("totp_setup_failed", userID, "reason", "secret_decode_error")
		return nil, event.NewTransient("internal", "decode totp secret").WithCause(err)
	}
	// Store the pending secret temporarily
	s.pendingTOTP.mu.Lock()
	s.pendingTOTP.secrets[userID.Get()] = pendingTOTPSecret{
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
	pending, ok := s.pendingTOTP.secrets[userID.Get()]
	if ok {
		delete(s.pendingTOTP.secrets, userID.Get())
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
		return fmt.Errorf("convert userID: %w", err)
	}
	if err := s.dispatcher.Dispatch(ctx, NewEnableTOTPCmd(aggID, pending.secret)); err != nil {
		s.logAuth("totp_setup_verify_failed", userID, "reason", "dispatch_error")
		return fmt.Errorf("enable totp dispatch: %w", err)
	}
	s.logAuth(statusTOTPEnabled, userID)
	return nil
}

// VerifyTOTP validates a TOTP code against the user's active TOTP secret.
// This is used as a second factor during login.
func (s *Service) VerifyTOTP(ctx context.Context, userID UserID, code string) error {
	if s.totpConfig == nil {
		return ErrTOTPNotConfigured
	}
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		s.logAuth("totp_verify_failed", userID, "reason", "user_not_found")
		return fmt.Errorf("verify totp: %w", ErrUserNotFound)
	}
	if !user.TOTPEnabled || len(user.TOTPSecret) == 0 {
		s.logAuth("totp_verify_failed", userID, "reason", "totp_not_enabled")
		return ErrTOTPNotEnabled
	}
	if !validateTOTP(user.TOTPSecret, code, s.totpConfig.Window) {
		s.logAuth("totp_verify_failed", userID, "reason", "invalid_code")
		return ErrInvalidTOTPCode
	}
	s.logAuth(statusTOTPVerified, userID)
	return nil
}

// DisableTOTP removes the TOTP configuration for a user.
// A valid TOTP code is required to prevent MFA stripping via session hijack.
func (s *Service) DisableTOTP(ctx context.Context, userID UserID, code string) error {
	if s.totpConfig == nil {
		return ErrTOTPNotConfigured
	}
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		s.logAuth("totp_disable_failed", userID, "reason", "user_not_found")
		return fmt.Errorf("disable totp: %w", ErrUserNotFound)
	}
	if !user.TOTPEnabled {
		s.logAuth("totp_disable_failed", userID, "reason", "totp_not_enabled")
		return ErrTOTPNotEnabled
	}
	if !validateTOTP(user.TOTPSecret, code, s.totpConfig.Window) {
		s.logAuth("totp_disable_failed", userID, "reason", "invalid_code")
		return ErrInvalidTOTPCode
	}
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		s.logAuth("totp_disable_failed", userID, "reason", "invalid_user_id")
		return fmt.Errorf("convert userID: %w", err)
	}
	if err := s.dispatcher.Dispatch(ctx, NewDisableTOTPCmd(aggID)); err != nil {
		s.logAuth("totp_disable_failed", userID, "reason", "dispatch_error")
		return fmt.Errorf("disable totp dispatch: %w", err)
	}
	s.logAuth(statusTOTPDisabled, userID)
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
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for key, entry := range s.secrets {
		if now.After(entry.expiresAt) {
			delete(s.secrets, key)
			count++
		}
	}
	return count
}

// startEviction launches a background goroutine that periodically removes
// expired pending TOTP secrets. Returns a stop function that must be called
// to terminate the goroutine (e.g. on shutdown or in tests).
func (s *pendingTOTPStore) startEviction() (stop func()) {
	ticker := time.NewTicker(pendingTTOTPEvictionInterval)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				s.EvictExpired()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() { close(done) }
}
