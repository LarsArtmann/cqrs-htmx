package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
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
	aggID id.StreamID,
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
		evt, err := event.New(
			eventMemberAdded, aggID, aggregateTypeMembership, version.Increment(),
			MemberAddedPayload{
				SchemaVersion: currentSchemaVersion,
				ActorKind:     actorID.Kind().String(),
				ActorID:       actorID.String(),
				TenantID:      tenantID.Get(),
				Roles:         rolesCopy,
			},
			//cqrs-lint:ignore(A027) one codec per repository is intentional; global DefaultCodec would couple unrelated aggregates
			event.WithCodec(codec.JSONCodec{}),
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
	aggID id.StreamID,
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
		evt, err := event.New(
			eventMemberRolesChanged, aggID, aggregateTypeMembership, version.Increment(),
			MemberRolesChangedPayload{
				SchemaVersion: currentSchemaVersion,
				ActorKind:     state.ActorID.Kind().String(),
				ActorID:       state.ActorID.String(),
				TenantID:      state.TenantID.Get(),
				Roles:         rolesCopy,
			},
			event.WithCodec(codec.JSONCodec{}),
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
	aggID id.StreamID,
) func(MembershipState, event.Version) ([]event.Event, error) {
	return func(state MembershipState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.membership_remove.not_found",
				"membership does not exist",
			)
		}
		evt, err := event.New(
			eventMemberRemoved, aggID, aggregateTypeMembership, version.Increment(),
			MemberRemovedPayload{
				SchemaVersion: currentSchemaVersion,
				ActorID:       state.ActorID.String(),
				TenantID:      state.TenantID.Get(),
			},
			event.WithCodec(codec.JSONCodec{}),
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
