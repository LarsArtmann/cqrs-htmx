package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
type MembershipState = identitymodel.MembershipState

var (
	actorKindUserStr = identitymodel.ActorKindUserStr

	foldMembership      = identitymodel.FoldMembership
	actorKindFromString = identitymodel.ActorKindFromString
)
