package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func decideEnableTOTP(
	aggID id.StreamID,
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
		evt, err := event.New(
			eventTOTPEnabled, aggID, aggregateTypeUser, version.Increment(),
			TOTPEnabledPayload{
				SchemaVersion: currentSchemaVersion,
				Secret:        secret,
			},
			event.WithCodec(codec.JSONCodec{}),
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
	aggID id.StreamID,
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
		evt, err := event.New(
			eventTOTPDisabled, aggID, aggregateTypeUser, version.Increment(),
			TOTPDisabledPayload{
				SchemaVersion: currentSchemaVersion,
			},
			event.WithCodec(codec.JSONCodec{}),
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
