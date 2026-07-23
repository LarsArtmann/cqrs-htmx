package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

type MembershipState = identitymodel.MembershipState

var (
	actorKindUserStr = identitymodel.ActorKindUserStr
	actorKindBotStr  = identitymodel.ActorKindBotStr

	foldMembership      = identitymodel.FoldMembership
	actorKindFromString = identitymodel.ActorKindFromString
)
