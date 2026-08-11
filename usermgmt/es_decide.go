package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// requireExists returns a "user does not exist" rejection if the user has not
// been registered yet. Used as the first guard in every decider that operates
// on an existing user. The domain argument becomes the rejection code suffix,
// e.g. domain="update_roles" → "usermgmt.update_roles.not_found".
func requireExists(state UserState, domain string) error {
	if !state.Exists() {
		return errorfamily.NewRejection(
			"usermgmt."+domain+".not_found",
			"user does not exist",
		)
	}
	return nil
}

func decideRegisterUser(
	aggID id.StreamID,
	email, displayName string,
	roles []Role,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if state.Exists() {
			return nil, errorfamily.NewConflict("usermgmt.user_already_exists",
				"user with this ID already exists")
		}
		if email == "" {
			return nil, errorfamily.NewRejection("usermgmt.register.email_required",
				"email is required")
		}
		rolesCopy := make([]Role, len(roles))
		copy(rolesCopy, roles)
		evt, err := event.New(
			eventUserRegistered, aggID, aggregateTypeUser, version.Increment(),
			UserRegisteredPayload{
				SchemaVersion: currentSchemaVersion,
				Email:         email,
				DisplayName:   displayName,
				Roles:         rolesCopy,
			},
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.register.event_failed",
				"create UserRegistered event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideDeleteUser(
	aggID id.StreamID,
	reason string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "delete_user"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection("usermgmt.delete_user.already_deleted",
				"user is already deleted")
		}
		evt, err := event.New(
			eventUserDeleted, aggID, aggregateTypeUser, version.Increment(),
			UserDeletedPayload{
				SchemaVersion: currentSchemaVersion,
				Reason:        reason,
			},
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.delete_user.event_failed",
				"create UserDeleted event",
			)
		}
		return []event.Event{evt}, nil
	}
}
