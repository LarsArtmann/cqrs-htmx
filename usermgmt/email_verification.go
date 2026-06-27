package usermgmt

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// VerificationTokenTTL is the default lifetime of an email verification token.
const VerificationTokenTTL = 24 * time.Hour

// SendVerificationEmailFunc is called when a verification token is generated.
// The implementation should send an email containing the verification link
// (e.g., https://example.com/verify-email?token=<token>).
type SendVerificationEmailFunc func(ctx context.Context, email, token string) error

// EmailVerificationConfig configures the email verification flow.
// All fields are optional — sensible defaults are applied.
type EmailVerificationConfig struct {
	// TokenTTL is how long a verification token remains valid.
	// Defaults to 24 hours.
	TokenTTL time.Duration

	// SendEmail is called when a token is generated.
	// If nil, the token is only returned from the API (useful for development/testing).
	SendEmail SendVerificationEmailFunc
}

// verificationTokenStore is an in-memory store mapping tokens to user IDs.
// Like the WebAuthn session store, tokens are ephemeral and auto-expire.
type verificationTokenStore struct {
	mu     sync.Mutex
	tokens map[string]verificationEntry
}

type verificationEntry struct {
	userID    UserID
	email     string
	expiresAt time.Time
}

func newVerificationTokenStore() *verificationTokenStore {
	return &verificationTokenStore{ //nolint:exhaustruct // mu is zero-value (sync.Mutex)
		tokens: make(map[string]verificationEntry),
	}
}

func (s *verificationTokenStore) Save(userID UserID, email string, ttl time.Duration) (string, error) {
	token, err := generateVerificationToken()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.tokens[token] = verificationEntry{
		userID:    userID,
		email:     email,
		expiresAt: time.Now().Add(ttl),
	}
	s.mu.Unlock()
	return token, nil
}

func (s *verificationTokenStore) Consume(token string) (UserID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.tokens[token]
	if !ok {
		return UserID{}, ErrInvalidVerificationToken
	}
	delete(s.tokens, token)
	if time.Now().After(entry.expiresAt) {
		return UserID{}, ErrInvalidVerificationToken
	}
	return entry.userID, nil
}

func (s *verificationTokenStore) EvictExpired() int {
	return evictExpired(&s.mu, s.tokens, func(_ string, e verificationEntry) bool {
		return time.Now().After(e.expiresAt)
	})
}

// verificationEvictionInterval is how often expired verification tokens are
// cleaned up by the background goroutine.
const verificationEvictionInterval = 5 * time.Minute

// startEviction launches a background goroutine that periodically removes
// expired verification tokens. Returns a stop function that must be called
// to terminate the goroutine (e.g. on shutdown or in tests).
func (s *verificationTokenStore) startEviction() (stop func()) {
	return startPeriodicEviction(s.EvictExpired, verificationEvictionInterval)
}

func generateVerificationToken() (string, error) {
	return randomBase64URLString(32, "verification token")
}

// SendVerificationEmail generates a token and either calls the configured
// SendEmail callback or returns the token directly (when no callback is set).
func (s *Service) SendVerificationEmail(ctx context.Context, userID UserID) (token string, err error) {
	if s.verificationTokens == nil {
		return "", ErrEmailVerificationNotConfigured
	}
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		s.logAuth("verification_email_failed", userID, "reason", "user_not_found")
		return "", event.WrapRejection(
			ErrUserNotFound,
			"usermgmt.verification.user_not_found",
			"send verification email",
		)
	}
	if user.EmailVerified {
		s.logAuth("verification_email_failed", userID, "reason", "already_verified")
		return "", ErrEmailAlreadyVerified
	}

	ttl := s.verificationTTL
	token, err = s.verificationTokens.Save(userID, user.Email, ttl)
	if err != nil {
		s.logAuth("verification_email_failed", userID, "reason", "token_generation_error")
		return "", event.NewTransient("internal", "generate verification token").WithCause(err)
	}

	if s.sendVerificationEmail != nil {
		if err := s.sendVerificationEmail(ctx, user.Email, token); err != nil {
			s.logAuth("verification_email_failed", userID, "reason", "send_callback_error")
			return "", event.NewTransient("internal", "send verification email").WithCause(err)
		}
	}
	s.logAuth("verification_email_sent", userID, "email", user.Email)
	return token, nil
}

// VerifyEmail consumes a verification token and dispatches the VerifyEmail
// command, marking the user's email as verified.
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	if s.verificationTokens == nil {
		return ErrEmailVerificationNotConfigured
	}
	userID, err := s.verificationTokens.Consume(token)
	if err != nil {
		s.logAuth("email_verify_failed", userID, "reason", "invalid_or_expired_token")
		return err //nolint:wrapcheck // domain sentinel error
	}
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		s.logAuth("email_verify_failed", userID, "reason", "invalid_user_id")
		return event.WrapInfrastructure(err, "usermgmt.service.userid_conversion_failed", "convert userID")
	}
	if err := s.dispatcher.Dispatch(ctx, NewVerifyEmailCmd(aggID)); err != nil {
		s.logAuth("email_verify_failed", userID, "reason", "dispatch_error")
		return event.Wrapf(err, event.Classify(err),
			"usermgmt.verification.dispatch_failed", "verify email dispatch")
	}
	s.logAuth(statusVerified, userID)
	return nil
}
