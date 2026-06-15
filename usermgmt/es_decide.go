package usermgmt

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func decideRegisterUser(
	aggID id.AggregateID,
	email, displayName, passwordHash string,
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
		evt, err := event.NewEvent(
			eventUserRegistered, aggID, aggregateTypeUser, version.Increment(),
			marshalPayload(UserRegisteredPayload{
				Email:        email,
				DisplayName:  displayName,
				PasswordHash: passwordHash,
				Roles:        rolesCopy,
			}),
		)
		if err != nil {
			return nil, fmt.Errorf("create UserRegistered event: %w", err)
		}
		return []event.Event{evt}, nil
	}
}

func decideChangePassword(
	aggID id.AggregateID,
	passwordHash string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.change_password.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.change_password.deleted",
				"cannot change password of deleted user")
		}
		evt, err := event.NewEvent(
			eventPasswordChanged, aggID, aggregateTypeUser, version.Increment(),
			marshalPayload(PasswordChangedPayload{PasswordHash: passwordHash}),
		)
		if err != nil {
			return nil, fmt.Errorf("create PasswordChanged event: %w", err)
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
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.update_roles.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.update_roles.deleted",
				"cannot update roles of deleted user")
		}
		rolesCopy := make([]Role, len(roles))
		copy(rolesCopy, roles)
		evt, err := event.NewEvent(
			eventRolesUpdated, aggID, aggregateTypeUser, version.Increment(),
			marshalPayload(RolesUpdatedPayload{Roles: rolesCopy, Domain: domain}),
		)
		if err != nil {
			return nil, fmt.Errorf("create RolesUpdated event: %w", err)
		}
		return []event.Event{evt}, nil
	}
}

func decideChangeEmail(
	aggID id.AggregateID,
	email string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.change_email.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.change_email.deleted",
				"cannot change email of deleted user")
		}
		if state.Email == email {
			return nil, nil
		}
		evt, err := event.NewEvent(
			eventEmailChanged, aggID, aggregateTypeUser, version.Increment(),
			marshalPayload(EmailChangedPayload{Email: email}),
		)
		if err != nil {
			return nil, fmt.Errorf("create EmailChanged event: %w", err)
		}
		return []event.Event{evt}, nil
	}
}

func decideChangeDisplayName(
	aggID id.AggregateID,
	displayName string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.change_display_name.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.change_display_name.deleted",
				"cannot change display name of deleted user")
		}
		if state.DisplayName == displayName {
			return nil, nil
		}
		evt, err := event.NewEvent(
			eventDisplayNameChanged, aggID, aggregateTypeUser, version.Increment(),
			marshalPayload(DisplayNameChangedPayload{DisplayName: displayName}),
		)
		if err != nil {
			return nil, fmt.Errorf("create DisplayNameChanged event: %w", err)
		}
		return []event.Event{evt}, nil
	}
}

func decideDeleteUser(
	aggID id.AggregateID,
	reason string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.delete_user.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.delete_user.already_deleted",
				"user is already deleted")
		}
		evt, err := event.NewEvent(
			eventUserDeleted, aggID, aggregateTypeUser, version.Increment(),
			marshalPayload(UserDeletedPayload{Reason: reason}),
		)
		if err != nil {
			return nil, fmt.Errorf("create UserDeleted event: %w", err)
		}
		marked, markErr := event.MarkTombstone(evt)
		if markErr != nil {
			return nil, fmt.Errorf("mark tombstone: %w", markErr)
		}
		return []event.Event{marked}, nil
	}
}
