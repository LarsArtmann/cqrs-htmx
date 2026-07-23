package usermgmt

import (
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

type WebAuthnCredential = identitymodel.WebAuthnCredential

func NewCredentialFromPayload(p CredentialAddedPayload, createdAt time.Time) WebAuthnCredential {
	return identitymodel.NewCredentialFromPayload(p, createdAt)
}
