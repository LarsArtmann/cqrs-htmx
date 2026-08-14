package identitymodel

import (
	"crypto/subtle"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

const sessionTokenBytes = 32

// Session represents an authenticated session tied to an actor.
type Session struct {
	Token     string        `json:"token"`
	UserID    UserID        `json:"user_id"`
	ActorID   ActorID       `json:"actor_id"`
	Origin    SessionOrigin `json:"-"`
	CreatedAt time.Time     `json:"created_at"`
	ExpiresAt time.Time     `json:"expires_at"`
}

// NewSession creates a Session with a cryptographically random token for the given user.
func NewSession(userID UserID, ttl time.Duration) (*Session, error) {
	actorID := ActorIDFromUser(userID)
	return newSession(actorID, DirectLogin{AuthenticatedAs: actorID}, ttl)
}

// NewImpersonationSession creates a Session for impersonation.
func NewImpersonationSession(
	target, impersonator ActorID, reason string, ttl time.Duration,
) (*Session, error) {
	return newSession(target, Impersonation{
		By:     impersonator,
		Reason: reason,
		At:     time.Now().UTC(),
	}, ttl)
}

func newSession(actorID ActorID, origin SessionOrigin, ttl time.Duration) (*Session, error) {
	token, err := generateToken()
	if err != nil {
		return nil, errorfamily.NewTransient("token_gen_failed",
			fmt.Sprintf("generate token for actor %q", actorID.PrefixedString())).WithCause(err)
	}
	now := time.Now().UTC()
	//nolint:exhaustruct // intentional: UserID is conditionally set below
	sess := &Session{
		Token:     token,
		ActorID:   actorID,
		Origin:    origin,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
	// Only populate UserID when the actor is a human user. Bot, system,
	// and service actors have non-UserID identifiers.
	if uid, ok := ActorIDAsUserID(actorID); ok {
		sess.UserID = uid
	}
	return sess, nil
}

// IsExpired reports whether the session has passed its expiration time.
func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// TokenMatches performs a constant-time comparison of the provided token
// against the session token. It does NOT check expiration.
func (s *Session) TokenMatches(token string) bool {
	return subtle.ConstantTimeCompare([]byte(s.Token), []byte(token)) == 1
}

// Valid performs a constant-time comparison of the token and checks expiration.
func (s *Session) Valid(token string) bool {
	return !s.IsExpired() && s.TokenMatches(token)
}

// SessionOrigin describes how a session was established.
type SessionOrigin interface {
	isSessionOrigin()
}

// DirectLogin indicates the user authenticated directly (no impersonation).
type DirectLogin struct {
	AuthenticatedAs ActorID
}

func (DirectLogin) isSessionOrigin() {}

// Impersonation indicates an admin is acting on behalf of another actor.
type Impersonation struct {
	By     ActorID   // the real admin who initiated the impersonation
	Reason string    // mandatory justification for audit
	At     time.Time // when the impersonation began
}

func (Impersonation) isSessionOrigin() {}

func generateToken() (string, error) {
	return RandomBase64URLString(sessionTokenBytes, "session token")
}
