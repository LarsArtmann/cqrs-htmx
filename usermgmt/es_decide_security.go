package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	errorfamily "github.com/larsartmann/go-error-family"
)

func decideEnableTOTP(
	aggID id.AggregateID,
	secret []byte,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "enable_totp"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection("usermgmt.enable_totp.deleted",
				"cannot enable TOTP for deleted user")
		}
		if state.TOTPEnabled {
			return nil, nil
		}
		payload, err := marshalPayload(TOTPEnabledPayload{
			SchemaVersion: currentSchemaVersion,
			Secret:        secret,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.enable_totp.marshal_failed",
				"marshal TOTPEnabled payload",
			)
		}
		evt, err := event.NewEvent(
			eventTOTPEnabled, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.enable_totp.event_failed",
				"create TOTPEnabled event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideDisableTOTP(
	aggID id.AggregateID,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "disable_totp"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection("usermgmt.disable_totp.deleted",
				"cannot disable TOTP for deleted user")
		}
		if !state.TOTPEnabled {
			return nil, nil
		}
		payload, err := marshalPayload(TOTPDisabledPayload{
			SchemaVersion: currentSchemaVersion,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.disable_totp.marshal_failed",
				"marshal TOTPDisabled payload",
			)
		}
		evt, err := event.NewEvent(
			eventTOTPDisabled, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.disable_totp.event_failed",
				"create TOTPDisabled event",
			)
		}
		return []event.Event{evt}, nil
	}
}
