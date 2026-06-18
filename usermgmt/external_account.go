package usermgmt

import "time"

// ExternalAccount represents a third-party identity provider account linked
// to a User (e.g., Google, GitHub, Microsoft via OAuth2/OIDC).
// Stored as part of the User aggregate state, updated via events.
//
// The Provider+Subject pair is the unique key for deduplication — a given
// provider subject can only be linked to one user at a time.
type ExternalAccount struct {
	Provider    string    `json:"provider"`        // e.g., "google", "github"
	Subject     string    `json:"subject"`         // provider's unique user ID (OIDC "sub" claim)
	Email       string    `json:"email,omitempty"` // email from provider (may differ from User.Email)
	DisplayName string    `json:"display_name,omitempty"`
	LinkedAt    time.Time `json:"linked_at"`
}

// Clone returns a deep copy of the external account.
// All fields are value types so this is effectively a copy, but the method
// is provided for API consistency with WebAuthnCredential.Clone().
func (e ExternalAccount) Clone() ExternalAccount {
	return e
}
