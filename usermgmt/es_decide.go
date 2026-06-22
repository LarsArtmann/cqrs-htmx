package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// requireExists returns a "user does not exist" rejection if the user has not
// been registered yet. Used as the first guard in every decider that operates
// on an existing user. The domain argument becomes the rejection code suffix,
// e.g. domain="update_roles" → "usermgmt.update_roles.not_found".
func requireExists(state UserState, domain string) error {
	if !state.Exists() {
		return event.NewRejection(
			"usermgmt."+domain+".not_found",
			"user does not exist",
		)
	}
	return nil
}

func decideRegisterUser(
	aggID id.AggregateID,
	email, displayName string,
	roles []Role,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if state.Exists() {
			return nil, event.NewConflict("usermgmt.user_already_exists",
				"user with this ID already exists")
		}
		if email == "" {
			return nil, event.NewRejection("usermgmt.register.email_required",
				"email is required")
		}
		rolesCopy := make([]Role, len(roles))
		copy(rolesCopy, roles)
		payload, err := marshalPayload(UserRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         email,
			DisplayName:   displayName,
			Roles:         rolesCopy,
		})
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.register.marshal_failed",
				"marshal UserRegistered payload",
			)
		}
		evt, err := event.NewEvent(
			eventUserRegistered, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(err, "usermgmt.register.event_failed", "create UserRegistered event")
		}
		return []event.Event{evt}, nil
	}
}

func decideUpdateRoles(
	aggID id.AggregateID,
	roles []Role,
	domain string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "update_roles"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.update_roles.deleted",
				"cannot update roles of deleted user")
		}
		rolesCopy := make([]Role, len(roles))
		copy(rolesCopy, roles)
		payload, err := marshalPayload(RolesUpdatedPayload{
			SchemaVersion: currentSchemaVersion,
			Roles:         rolesCopy,
			Domain:        domain,
		})
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.update_roles.marshal_failed",
				"marshal RolesUpdated payload",
			)
		}
		evt, err := event.NewEvent(
			eventRolesUpdated, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(err, "usermgmt.update_roles.event_failed", "create RolesUpdated event")
		}
		return []event.Event{evt}, nil
	}
}

func decideDeleteUser(
	aggID id.AggregateID,
	reason string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "delete_user"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.delete_user.already_deleted",
				"user is already deleted")
		}
		payload, err := marshalPayload(UserDeletedPayload{
			SchemaVersion: currentSchemaVersion,
			Reason:        reason,
		})
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.delete_user.marshal_failed",
				"marshal UserDeleted payload",
			)
		}
		evt, err := event.NewEvent(
			eventUserDeleted, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(err, "usermgmt.delete_user.event_failed", "create UserDeleted event")
		}
		marked, markErr := event.MarkTombstone(evt)
		if markErr != nil {
			return nil, event.WrapInfrastructure(markErr, "usermgmt.delete_user.tombstone_failed", "mark tombstone")
		}
		return []event.Event{marked}, nil
	}
}
