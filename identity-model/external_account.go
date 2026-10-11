package identitymodel

import "time"

// ExternalAccountCore contains the shared fields for an external (OAuth2/OIDC)
// identity provider account.
type ExternalAccountCore struct {
	Provider    string `json:"provider"`
	Subject     string `json:"subject"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// ExternalAccount represents a third-party identity provider account linked
// to a User (e.g., Google, GitHub, Microsoft via OAuth2/OIDC).
type ExternalAccount struct {
	ExternalAccountCore
	LinkedAt time.Time `json:"linked_at"`
}

// NewExternalAccount constructs an ExternalAccount from its field values.
func NewExternalAccount(provider, subject, email, displayName string, linkedAt time.Time) ExternalAccount {
	return ExternalAccount{
		ExternalAccountCore: ExternalAccountCore{
			Provider:    provider,
			Subject:     subject,
			Email:       email,
			DisplayName: displayName,
		},
		LinkedAt: linkedAt,
	}
}
