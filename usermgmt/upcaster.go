package usermgmt

import identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"

type (
	Upcaster         = identitymodel.Upcaster
	UpcasterRegistry = identitymodel.UpcasterRegistry
)

var (
	NewUpcasterRegistry = identitymodel.NewUpcasterRegistry
	SetUpcasterRegistry = identitymodel.SetUpcasterRegistry
)
