package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	errorfamily "github.com/larsartmann/go-error-family"
)

//nolint:dupl // mirrors decideChangeDisplayName; single-field deciders are structurally identical by design
func decideChangeEmail(
	aggID id.AggregateID,
	email string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "change_email"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection("usermgmt.change_email.deleted",
				"cannot change email of deleted user")
		}
		if state.Email == email {
			return nil, nil
		}
		payload, err := marshalPayload(EmailChangedPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         email,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.change_email.marshal_failed",
				"marshal EmailChanged payload",
			)
		}
		evt, err := event.NewEvent(
			eventEmailChanged, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.change_email.event_failed",
				"create EmailChanged event",
			)
		}
		return []event.Event{evt}, nil
	}
}

//nolint:dupl // mirrors decideChangeEmail; single-field deciders are structurally identical by design
func decideChangeDisplayName(
	aggID id.AggregateID,
	displayName string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "change_display_name"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection("usermgmt.change_display_name.deleted",
				"cannot change display name of deleted user")
		}
		if state.DisplayName == displayName {
			return nil, nil
		}
		payload, err := marshalPayload(DisplayNameChangedPayload{
			SchemaVersion: currentSchemaVersion,
			DisplayName:   displayName,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.change_display_name.marshal_failed",
				"marshal DisplayNameChanged payload",
			)
		}
		evt, err := event.NewEvent(
			eventDisplayNameChanged, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.change_display_name.event_failed",
				"create DisplayNameChanged event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideVerifyEmail(
	aggID id.AggregateID,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "verify_email"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection("usermgmt.verify_email.deleted",
				"cannot verify email of deleted user")
		}
		if state.EmailVerified {
			return nil, nil
		}
		payload, err := marshalPayload(EmailVerifiedPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         state.Email,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.verify_email.marshal_failed",
				"marshal EmailVerified payload",
			)
		}
		evt, err := event.NewEvent(
			eventEmailVerified, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.verify_email.event_failed",
				"create EmailVerified event",
			)
		}
		return []event.Event{evt}, nil
	}
}
