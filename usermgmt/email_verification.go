package usermgmt

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
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
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for token, entry := range s.tokens {
		if now.After(entry.expiresAt) {
			delete(s.tokens, token)
			count++
		}
	}
	return count
}

func generateVerificationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate verification token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SendVerificationEmail generates a token and either calls the configured
// SendEmail callback or returns the token directly (when no callback is set).
func (s *Service) SendVerificationEmail(ctx context.Context, userID UserID) (token string, err error) {
	if s.verificationTokens == nil {
		return "", ErrEmailVerificationNotConfigured
	}
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		return "", fmt.Errorf("send verification email: %w", ErrUserNotFound)
	}
	if user.EmailVerified {
		return "", ErrEmailAlreadyVerified
	}

	ttl := s.verificationTTL
	token, err = s.verificationTokens.Save(userID, user.Email, ttl)
	if err != nil {
		return "", event.NewTransient("internal", "generate verification token").WithCause(err)
	}

	if s.sendVerificationEmail != nil {
		if err := s.sendVerificationEmail(ctx, user.Email, token); err != nil {
			return "", event.NewTransient("internal", "send verification email").WithCause(err)
		}
	}
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
		return err
	}
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return fmt.Errorf("convert userID: %w", err)
	}
	if err := s.dispatcher.Dispatch(ctx, NewVerifyEmailCmd(aggID)); err != nil {
		return fmt.Errorf("verify email dispatch: %w", err)
	}
	return nil
}
