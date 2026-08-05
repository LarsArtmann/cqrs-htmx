package usermgmt

import (
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

type (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ExternalAccount = identitymodel.ExternalAccount
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ExternalAccountCore = identitymodel.ExternalAccountCore
)

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewExternalAccount(provider, subject, email, displayName string, linkedAt time.Time) ExternalAccount {
	return identitymodel.NewExternalAccount(provider, subject, email, displayName, linkedAt)
}
