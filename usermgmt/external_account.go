package usermgmt

import (
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

type (
	ExternalAccount     = identitymodel.ExternalAccount
	ExternalAccountCore = identitymodel.ExternalAccountCore
)

func NewExternalAccount(provider, subject, email, displayName string, linkedAt time.Time) ExternalAccount {
	return identitymodel.NewExternalAccount(provider, subject, email, displayName, linkedAt)
}
