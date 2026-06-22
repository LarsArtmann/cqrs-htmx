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
