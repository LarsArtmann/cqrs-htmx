package usermgmt

import "time"

// externalAccountCore contains the shared fields for an external (OAuth2/OIDC)
// identity provider account. Embedded by both ExternalAccount (domain model)
// and ExternalAccountLinkedPayload (event payload).
type externalAccountCore struct {
	Provider    string `json:"provider"`
	Subject     string `json:"subject"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// ExternalAccount represents a third-party identity provider account linked
// to a User (e.g., Google, GitHub, Microsoft via OAuth2/OIDC).
// Stored as part of the User aggregate state, updated via events.
//
// The Provider+Subject pair is the unique key for deduplication — a given
// provider subject can only be linked to one user at a time.
type ExternalAccount struct {
	externalAccountCore
	LinkedAt time.Time `json:"linked_at"`
}

// NewExternalAccount constructs an ExternalAccount from its field values.
// Provided because the embedded externalAccountCore is unexported, making a
// struct literal impossible from outside the package. Use this to build
// fixtures, read-model seeds, or test data; the live User aggregate is still
// populated exclusively by folding ExternalAccountLinked events.
func NewExternalAccount(provider, subject, email, displayName string, linkedAt time.Time) ExternalAccount {
	return ExternalAccount{
		externalAccountCore: externalAccountCore{
			Provider:    provider,
			Subject:     subject,
			Email:       email,
			DisplayName: displayName,
		},
		LinkedAt: linkedAt,
	}
}
