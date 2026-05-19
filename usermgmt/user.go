package usermgmt

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionTokenBytes = 32
	defaultBcryptCost = 12
	minBcryptCost     = 4
)

type User struct {
	ID           UserID    `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name,omitempty"`
	PasswordHash string    `json:"-"`
	Roles        []string  `json:"roles"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewUser(id UserID, email, displayName string) *User {
	now := time.Now().UTC()
	return &User{
		ID:          id,
		Email:       email,
		DisplayName: displayName,
		Roles:       []string{RoleViewer},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (u *User) SetPassword(password string) error {
	return u.SetPasswordWithCost(password, defaultBcryptCost)
}

func (u *User) SetPasswordWithCost(password string, cost int) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	u.PasswordHash = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}

func (u *User) HasRole(role string) bool {
	return slices.Contains(u.Roles, role)
}

func (u *User) AddRole(role string) {
	if slices.Contains(u.Roles, role) {
		return
	}
	u.Roles = append(u.Roles, role)
	u.UpdatedAt = time.Now().UTC()
}

func (u *User) RemoveRole(role string) {
	for i, r := range u.Roles {
		if r == role {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
			u.UpdatedAt = time.Now().UTC()
			return
		}
	}
}

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

type Session struct {
	Token     string    `json:"token"`
	UserID    UserID    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewSession(userID UserID, ttl time.Duration) (*Session, error) {
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token for user %q: %w", userID, err)
	}
	now := time.Now().UTC()
	return &Session{
		Token:     token,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}, nil
}

func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

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
