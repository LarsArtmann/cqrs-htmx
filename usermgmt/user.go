package usermgmt

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const (
	sessionTokenBytes = 32
)

// User represents a registered user with authentication credentials and authorization roles.
// In the event-sourced architecture, User is a read-only projection — all mutations
// happen through commands that produce events.
type User struct {
	ID            UserID               `json:"id"`
	Email         string               `json:"email"`
	DisplayName   string               `json:"display_name,omitempty"`
	Roles         []Role               `json:"roles"`
	Credentials   []WebAuthnCredential `json:"credentials,omitempty"`
	EmailVerified bool                 `json:"email_verified"`
	TOTPEnabled   bool                 `json:"totp_enabled"`
	TOTPSecret    []byte               `json:"-"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// NewUser creates a User with the given identity fields and a default "viewer" role.
func NewUser(id UserID, email, displayName string) *User {
	now := time.Now().UTC()
	return &User{
		ID:          id,
		Email:       email,
		DisplayName: displayName,
		Roles:       []Role{RoleViewer},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Clone returns a deep copy of the user.
// All slice and byte-array fields are independently allocated.
func (u *User) Clone() *User {
	if u == nil {
		return nil
	}
	cp := *u
	cp.Roles = make([]Role, len(u.Roles))
	copy(cp.Roles, u.Roles)
	cp.Credentials = make([]WebAuthnCredential, len(u.Credentials))
	for i, c := range u.Credentials {
		cp.Credentials[i] = c.Clone()
	}
	cp.TOTPSecret = append([]byte(nil), u.TOTPSecret...)
	return &cp
}

// HasRole reports whether the user has the specified role.
func (u *User) HasRole(role Role) bool {
	return slices.Contains(u.Roles, role)
}

// HasCredential reports whether the user has a credential with the given ID.
func (u *User) HasCredential(credID []byte) bool {
	return slices.ContainsFunc(u.Credentials, func(c WebAuthnCredential) bool {
		return slices.Equal(c.ID, credID)
	})
}

// MarshalJSON serializes the user. Credentials are included but public keys are not
// exposed in JSON (they are binary COSE format, not useful for API consumers).
func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	data, err := json.Marshal(&struct {
		*Alias
		CredentialCount int `json:"credential_count"`
	}{
		Alias:           (*Alias)(u),
		CredentialCount: len(u.Credentials),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal user: %w", err)
	}
	return data, nil
}

// Session represents an authenticated session tied to a user.
type Session struct {
	Token     string    `json:"token"`
	UserID    UserID    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewSession creates a Session with a cryptographically random token for the given user.
func NewSession(userID UserID, ttl time.Duration) (*Session, error) {
	token, err := generateToken()
	if err != nil {
		return nil, event.NewTransient("token_gen_failed",
			fmt.Sprintf("generate token for user %q", userID)).WithCause(err)
	}
	now := time.Now().UTC()
	return &Session{
		Token:     token,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}, nil
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

func generateToken() (string, error) {
	b := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
