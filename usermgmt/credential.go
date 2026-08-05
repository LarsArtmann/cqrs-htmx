package usermgmt

import (
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

type (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	WebAuthnCredential = identitymodel.WebAuthnCredential
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	CredentialCore = identitymodel.CredentialCore
)

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewCredentialFromPayload(p CredentialAddedPayload, createdAt time.Time) WebAuthnCredential {
	return identitymodel.NewCredentialFromPayload(p, createdAt)
}
