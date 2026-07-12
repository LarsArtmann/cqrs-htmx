package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// MembershipDecider returns the Decider for the Membership aggregate.
// It pairs MembershipState as the initial state with foldMembership.
func MembershipDecider() decider.Decider[MembershipState] {
	return decider.Decider[MembershipState]{
		Initial: MembershipState{}, //nolint:exhaustruct // zero-value is correct for aggregate initial state
		Apply:   foldMembership,
	}
}

func decideAddMember(
	aggID id.AggregateID,
	actorID ActorID,
	tenantID TenantID,
	roles []Role,
) func(MembershipState, event.Version) ([]event.Event, error) {
	return func(state MembershipState, version event.Version) ([]event.Event, error) {
		if state.Exists() {
			return nil, errorfamily.NewConflict(
				"usermgmt.membership.already_exists",
				"membership already exists for this actor+tenant pair",
			)
		}
		if actorID.IsZero() {
			return nil, errorfamily.NewRejection(
				"usermgmt.membership.actor_required",
				"actor ID is required",
			)
		}
		if tenantID.IsZero() {
			return nil, errorfamily.NewRejection(
				"usermgmt.membership.tenant_required",
				"tenant ID is required",
			)
		}
		rolesCopy := make([]Role, len(roles))
		copy(rolesCopy, roles)
		payload, err := marshalPayload(MemberAddedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorKind:     actorID.Kind().String(),
			ActorID:       actorID.String(),
			TenantID:      tenantID.Get(),
			Roles:         rolesCopy,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.membership.marshal_failed",
				"marshal MemberAdded payload",
			)
		}
		evt, err := event.NewEvent(
			eventMemberAdded, aggID, aggregateTypeMembership, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.membership.event_failed",
				"create MemberAdded event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideUpdateMemberRoles(
	aggID id.AggregateID,
	roles []Role,
) func(MembershipState, event.Version) ([]event.Event, error) {
	return func(state MembershipState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.membership_roles.not_found",
				"membership does not exist",
			)
		}
		rolesCopy := make([]Role, len(roles))
		copy(rolesCopy, roles)
		payload, err := marshalPayload(MemberRolesChangedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorKind:     state.ActorID.Kind().String(),
			ActorID:       state.ActorID.String(),
			TenantID:      state.TenantID.Get(),
			Roles:         rolesCopy,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.membership_roles.marshal_failed",
				"marshal MemberRolesChanged payload",
			)
		}
		evt, err := event.NewEvent(
			eventMemberRolesChanged, aggID, aggregateTypeMembership, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.membership_roles.event_failed",
				"create MemberRolesChanged event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideRemoveMember(
	aggID id.AggregateID,
) func(MembershipState, event.Version) ([]event.Event, error) {
	return func(state MembershipState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.membership_remove.not_found",
				"membership does not exist",
			)
		}
		payload, err := marshalPayload(MemberRemovedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorID:       state.ActorID.String(),
			TenantID:      state.TenantID.Get(),
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.membership_remove.marshal_failed",
				"marshal MemberRemoved payload",
			)
		}
		evt, err := event.NewEvent(
			eventMemberRemoved, aggID, aggregateTypeMembership, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.membership_remove.event_failed",
				"create MemberRemoved event",
			)
		}
		return []event.Event{evt}, nil
	}
}
