package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func decideChangeEmail(
	aggID id.StreamID,
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
		evt, err := event.New(
			eventEmailChanged, aggID, aggregateTypeUser, version.Increment(),
			EmailChangedPayload{
				SchemaVersion: currentSchemaVersion,
				Email:         email,
			},
			//cqrs-lint:ignore(A027) one codec per repository is intentional; global DefaultCodec would couple unrelated aggregates
		event.WithCodec(codec.JSONCodec{}),
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

func decideChangeDisplayName(
	aggID id.StreamID,
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
		evt, err := event.New(
			eventDisplayNameChanged, aggID, aggregateTypeUser, version.Increment(),
			DisplayNameChangedPayload{
				SchemaVersion: currentSchemaVersion,
				DisplayName:   displayName,
			},
			event.WithCodec(codec.JSONCodec{}),
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
	aggID id.StreamID,
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
		evt, err := event.New(
			eventEmailVerified, aggID, aggregateTypeUser, version.Increment(),
			EmailVerifiedPayload{
				SchemaVersion: currentSchemaVersion,
				Email:         state.Email,
			},
			event.WithCodec(codec.JSONCodec{}),
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
