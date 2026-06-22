package usermgmt

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

const (
	sessionTokenBytes = 32
)

// User represents a registered user with authentication credentials and authorization roles.
// In the event-sourced architecture, User is a read-only projection — all mutations
// happen through commands that produce events.
type User struct {
	ID               UserID               `json:"id"`
	Email            string               `json:"email"`
	DisplayName      string               `json:"display_name,omitempty"`
	Roles            []Role               `json:"roles"`
	Credentials      []WebAuthnCredential `json:"credentials,omitempty"`
	ExternalAccounts []ExternalAccount    `json:"external_accounts,omitempty"`
	EmailVerified    bool                 `json:"email_verified"`
	TOTPEnabled      bool                 `json:"totp_enabled"`
	TOTPSecret       []byte               `json:"-"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
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
	cp.ExternalAccounts = make([]ExternalAccount, len(u.ExternalAccounts))
	copy(cp.ExternalAccounts, u.ExternalAccounts)
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
		return nil, event.WrapInfrastructure(err, "usermgmt.user.marshal_failed", "marshal user")
	}
	return data, nil
}

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
// The ActorID is derived from the UserID, and the Origin is set to DirectLogin.
func NewSession(userID UserID, ttl time.Duration) (*Session, error) {
	actorID := ActorIDFromUser(userID)
	return newSession(actorID, DirectLogin{AuthenticatedAs: actorID}, ttl)
}

// NewImpersonationSession creates a Session for impersonation: the actor is the
// target being impersonated, but the Origin records who initiated it and why.
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
		return nil, event.NewTransient("token_gen_failed",
			fmt.Sprintf("generate token for actor %q", actorID.PrefixedString())).WithCause(err)
	}
	now := time.Now().UTC()
	return &Session{
		Token:     token,
		UserID:    NewUserID(actorID.String()),
		ActorID:   actorID,
		Origin:    origin,
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

// SessionOrigin describes how a session was established.
// Only DirectLogin and Impersonation can implement this interface
// (via the unexported isSessionOrigin method).
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

// Membership links an Actor to a Tenant with scoped roles.
// It replaces the flat Roles field on UserState, decoupling "who you are"
// from "what you can do" in a given tenant.
type Membership struct {
	ActorID  ActorID
	TenantID TenantID
	Roles    []Role
	AddedAt  time.Time
}

// HasRole reports whether the membership grants the given role.
func (m Membership) HasRole(role Role) bool {
	return slices.Contains(m.Roles, role)
}

// HasAnyRole reports whether the membership grants any of the given roles.
func (m Membership) HasAnyRole(roles ...Role) bool {
	for _, target := range roles {
		if m.HasRole(target) {
			return true
		}
	}
	return false
}

func generateToken() (string, error) {
	return randomBase64URLString(sessionTokenBytes, "session token")
}
