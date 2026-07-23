package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

type (
	UserRegisteredPayload          = identitymodel.UserRegisteredPayload
	RolesUpdatedPayload            = identitymodel.RolesUpdatedPayload
	EmailChangedPayload            = identitymodel.EmailChangedPayload
	DisplayNameChangedPayload      = identitymodel.DisplayNameChangedPayload
	UserDeletedPayload             = identitymodel.UserDeletedPayload
	CredentialAddedPayload         = identitymodel.CredentialAddedPayload
	CredentialRemovedPayload       = identitymodel.CredentialRemovedPayload
	EmailVerifiedPayload           = identitymodel.EmailVerifiedPayload
	TOTPEnabledPayload             = identitymodel.TOTPEnabledPayload
	TOTPDisabledPayload            = identitymodel.TOTPDisabledPayload
	ExternalAccountLinkedPayload   = identitymodel.ExternalAccountLinkedPayload
	ExternalAccountUnlinkedPayload = identitymodel.ExternalAccountUnlinkedPayload
)

func marshalPayload(v any) ([]byte, error) {
	return identitymodel.MarshalPayload(v)
}
