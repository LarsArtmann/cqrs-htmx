package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

type MembershipState = identitymodel.MembershipState

// actorKind constants are used by foldMembership and the SQL session store.
const (
	actorKindUserStr = "user"
	actorKindBotStr  = "bot"
)

// foldMembership applies an event to the current MembershipState, returning the
// new state.
func foldMembership(state MembershipState, evt event.Event) (MembershipState, error) {
	next := state

	switch evt.Type() {
	case eventMemberAdded:
		p, err := unmarshalPayload[MemberAddedPayload](evt)
		if err != nil {
			return state, err
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		kind, err := actorKindFromString(p.ActorKind)
		if err != nil {
			return state, err
		}
		next.ActorID = NewActorID(kind, p.ActorID)
		next.TenantID = NewTenantID(p.TenantID)
		next.Roles = roles
		next.Removed = false

	case eventMemberRolesChanged:
		p, err := unmarshalPayload[MemberRolesChangedPayload](evt)
		if err != nil {
			return state, err
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		next.Roles = roles

	case eventMemberRemoved:
		next.Roles = nil
		next.Removed = true

	default:
		return state, errorfamily.NewRejection(
			"usermgmt.membership.unknown_event",
			"foldMembership received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}

func actorKindFromString(s string) (ActorKind, error) {
	switch s {
	case actorKindUserStr:
		return ActorUser, nil
	case actorKindBotStr:
		return ActorBot, nil
	default:
		return ActorUser, errorfamily.NewRejection(
			"usermgmt.membership.unknown_actor_kind",
			"unknown actor kind: "+s,
		)
	}
}
