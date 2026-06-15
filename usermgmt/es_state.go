package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// UserState is the aggregate state for the User, reconstructed by folding events.
// It is immutable — foldUser returns a new copy for each event.
type UserState struct {
	Email        string
	DisplayName  string
	PasswordHash string
	Roles        []Role
	Deleted      bool
	DeleteReason string
}

// Exists reports whether the user has been registered (has at least one event).
func (s UserState) Exists() bool {
	return s.Email != ""
}

func unmarshalPayload[T any](evt event.Event) (T, error) {
	return event.DecodePayload[T](evt, codec.JSONCodec{})
}

func foldUser(state UserState, evt event.Event) (UserState, error) {
	switch evt.Type() {
	case eventUserRegistered:
		p, err := unmarshalPayload[UserRegisteredPayload](evt)
		if err != nil {
			return state, err
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		return UserState{
			Email:        p.Email,
			DisplayName:  p.DisplayName,
			PasswordHash: p.PasswordHash,
			Roles:        roles,
			Deleted:      false,
			DeleteReason: "",
		}, nil

	case eventPasswordChanged:
		p, err := unmarshalPayload[PasswordChangedPayload](evt)
		if err != nil {
			return state, err
		}
		return UserState{
			Email:        state.Email,
			DisplayName:  state.DisplayName,
			PasswordHash: p.PasswordHash,
			Roles:        state.Roles,
			Deleted:      state.Deleted,
			DeleteReason: state.DeleteReason,
		}, nil

	case eventRolesUpdated:
		p, err := unmarshalPayload[RolesUpdatedPayload](evt)
		if err != nil {
			return state, err
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		return UserState{
			Email:        state.Email,
			DisplayName:  state.DisplayName,
			PasswordHash: state.PasswordHash,
			Roles:        roles,
			Deleted:      state.Deleted,
			DeleteReason: state.DeleteReason,
		}, nil

	case eventEmailChanged:
		p, err := unmarshalPayload[EmailChangedPayload](evt)
		if err != nil {
			return state, err
		}
		return UserState{
			Email:        p.Email,
			DisplayName:  state.DisplayName,
			PasswordHash: state.PasswordHash,
			Roles:        state.Roles,
			Deleted:      state.Deleted,
			DeleteReason: state.DeleteReason,
		}, nil

	case eventDisplayNameChanged:
		p, err := unmarshalPayload[DisplayNameChangedPayload](evt)
		if err != nil {
			return state, err
		}
		return UserState{
			Email:        state.Email,
			DisplayName:  p.DisplayName,
			PasswordHash: state.PasswordHash,
			Roles:        state.Roles,
			Deleted:      state.Deleted,
			DeleteReason: state.DeleteReason,
		}, nil

	case eventUserDeleted:
		p, err := unmarshalPayload[UserDeletedPayload](evt)
		if err != nil {
			return state, err
		}
		return UserState{
			Email:        state.Email,
			DisplayName:  state.DisplayName,
			PasswordHash: state.PasswordHash,
			Roles:        state.Roles,
			Deleted:      true,
			DeleteReason: p.Reason,
		}, nil

	default:
		return state, nil
	}
}
