package usermgmt

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"slices"
	"time"

	"github.com/cockroachdb/errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionTokenBytes = 32
	defaultBcryptCost = 12
	minBcryptCost     = 4
)

// User represents a registered user with authentication and authorization data.
type User struct {
	ID           UserID    `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name,omitempty"`
	PasswordHash string    `json:"-"`
	Roles        []Role    `json:"roles"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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

// SetPassword hashes the password with the default bcrypt cost (12).
func (u *User) SetPassword(password string) error {
	return u.SetPasswordWithCost(password, defaultBcryptCost)
}

// SetPasswordWithCost hashes the password with the given bcrypt cost.
// Use a lower cost (e.g. 4) in tests for speed.
func (u *User) SetPasswordWithCost(password string, cost int) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return errors.Wrapf(err, "hash password")
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword returns true if the plaintext password matches the stored hash.
func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}

// HasRole reports whether the user has the specified role.
func (u *User) HasRole(role Role) bool {
	return slices.Contains(u.Roles, role)
}

// AddRole appends the role if not already present and updates UpdatedAt.
func (u *User) AddRole(role Role) {
	if slices.Contains(u.Roles, role) {
		return
	}
	u.Roles = append(u.Roles, role)
	u.UpdatedAt = time.Now().UTC()
}

// RemoveRole removes the first occurrence of the role and updates UpdatedAt.
func (u *User) RemoveRole(role Role) {
	for i, r := range u.Roles {
		if r == role {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
			u.UpdatedAt = time.Now().UTC()
			return
		}
	}
}

// MarshalJSON adds a computed "has_password" field while omitting the raw hash.
func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		*Alias
		HasPassword bool `json:"has_password"`
	}{
		Alias:       (*Alias)(u),
		HasPassword: u.PasswordHash != "",
	})
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
		return nil, errors.Wrapf(err, "generate token for user %q", userID)
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

// Valid performs a constant-time comparison of the token and checks expiration.
func (s *Session) Valid(token string) bool {
	if s.IsExpired() {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(s.Token), []byte(token)) == 1
}

func generateToken() (string, error) {
	b := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
