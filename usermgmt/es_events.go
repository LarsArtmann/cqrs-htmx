package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

type (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	UserRegisteredPayload = identitymodel.UserRegisteredPayload
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	RolesUpdatedPayload = identitymodel.RolesUpdatedPayload
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	EmailChangedPayload = identitymodel.EmailChangedPayload
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	DisplayNameChangedPayload = identitymodel.DisplayNameChangedPayload
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	UserDeletedPayload = identitymodel.UserDeletedPayload
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	CredentialAddedPayload = identitymodel.CredentialAddedPayload
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	CredentialRemovedPayload = identitymodel.CredentialRemovedPayload
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	EmailVerifiedPayload = identitymodel.EmailVerifiedPayload
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	TOTPEnabledPayload = identitymodel.TOTPEnabledPayload
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	TOTPDisabledPayload = identitymodel.TOTPDisabledPayload
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ExternalAccountLinkedPayload = identitymodel.ExternalAccountLinkedPayload
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ExternalAccountUnlinkedPayload = identitymodel.ExternalAccountUnlinkedPayload
)

func marshalPayload(v any) ([]byte, error) {
	return identitymodel.MarshalPayload(v)
}
