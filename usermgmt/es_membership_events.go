package usermgmt

import identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"

type (
	MemberAddedPayload        = identitymodel.MemberAddedPayload
	MemberRolesChangedPayload = identitymodel.MemberRolesChangedPayload
	MemberRemovedPayload      = identitymodel.MemberRemovedPayload
)
