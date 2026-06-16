package usermgmt

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // G505: SHA1 required by TOTP RFC 6238
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
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

// totpState tracks per-user TOTP configuration in the aggregate state.
// EnableTOTP generates a new TOTP secret for the user and returns the
// setup information (secret + QR code URI). The secret is NOT yet active
// until VerifyTOTPSetup is called with a valid code from the authenticator app.
//

func (s *Service) EnableTOTP(ctx context.Context, userID UserID) (*TOTPSetupResponse, error) {
	if s.totpConfig == nil {
		return nil, ErrTOTPNotConfigured
	}
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		return nil, fmt.Errorf("enable totp: %w", ErrUserNotFound)
	}
	if user.TOTPEnabled {
		return nil, ErrTOTPAlreadyEnabled
	}
	secret, err := generateTOTPSecret()
	if err != nil {
		return nil, event.NewTransient("internal", "generate totp secret").WithCause(err)
	}
	// Store the pending secret temporarily
	s.pendingTOTP.mu.Lock()
	s.pendingTOTP.secrets[userID.Get()] = pendingTOTPSecret{
		secret:    secret,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	s.pendingTOTP.mu.Unlock()
	account := user.Email
	issuer := s.totpConfig.Issuer
	if issuer == "" {
		issuer = "cqrs-htmx"
	}
	secretB32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	qrURI := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		issuer, account, secretB32, issuer, TOTPDigits, int(TOTPTimeStep.Seconds()))
	return &TOTPSetupResponse{
		Secret:    secretB32,
		QRCodeURI: qrURI,
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
		return ErrTOTPSetupExpired
	}
	if !validateTOTP(pending.secret, code, s.totpConfig.Window) {
		return ErrInvalidTOTPCode
	}
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return fmt.Errorf("convert userID: %w", err)
	}
	if err := s.dispatcher.Dispatch(ctx, NewEnableTOTPCmd(aggID, pending.secret)); err != nil {
		return fmt.Errorf("enable totp dispatch: %w", err)
	}
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
		return fmt.Errorf("verify totp: %w", ErrUserNotFound)
	}
	if !user.TOTPEnabled || len(user.TOTPSecret) == 0 {
		return ErrTOTPNotEnabled
	}
	if !validateTOTP(user.TOTPSecret, code, s.totpConfig.Window) {
		return ErrInvalidTOTPCode
	}
	return nil
}

// DisableTOTP removes the TOTP configuration for a user.
func (s *Service) DisableTOTP(ctx context.Context, userID UserID) error {
	if s.totpConfig == nil {
		return ErrTOTPNotConfigured
	}
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		return fmt.Errorf("disable totp: %w", ErrUserNotFound)
	}
	if !user.TOTPEnabled {
		return ErrTOTPNotEnabled
	}
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return fmt.Errorf("convert userID: %w", err)
	}
	if err := s.dispatcher.Dispatch(ctx, NewDisableTOTPCmd(aggID)); err != nil {
		return fmt.Errorf("disable totp dispatch: %w", err)
	}
	return nil
}

// --- TOTP algorithm (RFC 6238) ---
func generateTOTPSecret() ([]byte, error) {
	secret := make([]byte, TOTPSecretLength)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate totp secret: %w", err)
	}
	return secret, nil
}

func validateTOTP(secret []byte, code string, window int) bool {
	code = strings.TrimSpace(code)
	if len(code) != TOTPDigits {
		return false
	}
	now := time.Now().Unix()
	for i := -window; i <= window; i++ {
		counter := (now + int64(i*int(TOTPTimeStep.Seconds()))) / int64(TOTPTimeStep.Seconds())
		if generateTOTPCode(secret, counter) == code {
			return true
		}
	}
	return false
}

func generateTOTPCode(secret []byte, counter int64) string {
	var counterBytes [8]byte
	binary.BigEndian.PutUint64(counterBytes[:], uint64(counter)) //nolint:gosec // G115: time-derived counter

	mac := hmac.New(sha1.New, secret)
	mac.Write(counterBytes[:])
	hash := mac.Sum(nil)
	offset := hash[len(hash)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff
	code := truncated % 1000000
	return fmt.Sprintf("%06d", code)
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

func newPendingTOTPStore() pendingTOTPStore {
	return pendingTOTPStore{ //nolint:exhaustruct // mu is zero-value
		secrets: make(map[string]pendingTOTPSecret)}
}
