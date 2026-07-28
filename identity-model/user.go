package identitymodel

import (
	"slices"
	"time"
)

// User represents a registered user with authentication credentials.
// In the event-sourced architecture, User is a read-only projection — all mutations
// happen through commands that produce events.
type User struct {
	ID               UserID               `json:"id"`
	Email            string               `json:"email"`
	DisplayName      string               `json:"display_name,omitempty"`
	Credentials      []WebAuthnCredential `json:"credentials,omitempty"`
	ExternalAccounts []ExternalAccount    `json:"external_accounts,omitempty"`
	EmailVerified    bool                 `json:"email_verified"`
	TOTPEnabled      bool                 `json:"totp_enabled"`
	TOTPSecret       []byte               `json:"-"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// NewUser creates a User with the given identity fields.
func NewUser(id UserID, email, displayName string) *User {
	now := time.Now().UTC()
	return &User{
		ID:          id,
		Email:       email,
		DisplayName: displayName,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Clone returns a deep copy of the user.
func (u *User) Clone() *User {
	if u == nil {
		return nil
	}
	cp := *u
	cp.Credentials = make([]WebAuthnCredential, len(u.Credentials))
	for i, c := range u.Credentials {
		cp.Credentials[i] = c.Clone()
	}
	cp.ExternalAccounts = make([]ExternalAccount, len(u.ExternalAccounts))
	copy(cp.ExternalAccounts, u.ExternalAccounts)
	cp.TOTPSecret = append([]byte(nil), u.TOTPSecret...)
	return &cp
}

// HasCredential reports whether the user has a credential with the given ID.
func (u *User) HasCredential(credID []byte) bool {
	return slices.ContainsFunc(u.Credentials, func(c WebAuthnCredential) bool {
		return slices.Equal(c.ID, credID)
	})
}

// MarshalJSON serializes the user with a credential_count field.
func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	return marshalJSONOrWrap(&struct {
		*Alias
		CredentialCount int `json:"credential_count"`
	}{
		Alias:           (*Alias)(u),
		CredentialCount: len(u.Credentials),
	}, "usermgmt.user.marshal_failed", "marshal user")
}
