package usermgmt

import (
	"bytes"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// UserState is the aggregate state for the User, reconstructed by folding events.
type UserState struct {
	Email         string
	DisplayName   string
	Roles         []Role
	Credentials   []WebAuthnCredential
	Deleted       bool
	DeleteReason  string
	EmailVerified bool
	TOTPEnabled   bool
	TOTPSecret    []byte
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
			Email:         p.Email,
			DisplayName:   p.DisplayName,
			Roles:         roles,
			Deleted:       false,
			DeleteReason:  "",
			EmailVerified: false,
			TOTPEnabled:   false,
			TOTPSecret:    nil,
		}, nil

	case eventRolesUpdated:
		p, err := unmarshalPayload[RolesUpdatedPayload](evt)
		if err != nil {
			return state, err
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		return UserState{
			Email:         state.Email,
			DisplayName:   state.DisplayName,
			Roles:         roles,
			Credentials:   state.Credentials,
			Deleted:       state.Deleted,
			DeleteReason:  state.DeleteReason,
			EmailVerified: state.EmailVerified,
			TOTPEnabled:   state.TOTPEnabled,
			TOTPSecret:    state.TOTPSecret,
		}, nil

	case eventEmailChanged:
		p, err := unmarshalPayload[EmailChangedPayload](evt)
		if err != nil {
			return state, err
		}
		return UserState{
			Email:         p.Email,
			DisplayName:   state.DisplayName,
			Roles:         state.Roles,
			Credentials:   state.Credentials,
			Deleted:       state.Deleted,
			DeleteReason:  state.DeleteReason,
			EmailVerified: false,
			TOTPEnabled:   false,
			TOTPSecret:    nil, // email change resets verification
		}, nil

	case eventDisplayNameChanged:
		p, err := unmarshalPayload[DisplayNameChangedPayload](evt)
		if err != nil {
			return state, err
		}
		return UserState{
			Email:         state.Email,
			DisplayName:   p.DisplayName,
			Roles:         state.Roles,
			Credentials:   state.Credentials,
			Deleted:       state.Deleted,
			DeleteReason:  state.DeleteReason,
			EmailVerified: state.EmailVerified,
			TOTPEnabled:   state.TOTPEnabled,
			TOTPSecret:    state.TOTPSecret,
		}, nil

	case eventUserDeleted:
		p, err := unmarshalPayload[UserDeletedPayload](evt)
		if err != nil {
			return state, err
		}
		return UserState{
			Email:         state.Email,
			DisplayName:   state.DisplayName,
			Roles:         state.Roles,
			Credentials:   state.Credentials,
			Deleted:       true,
			DeleteReason:  p.Reason,
			EmailVerified: state.EmailVerified,
			TOTPEnabled:   state.TOTPEnabled,
			TOTPSecret:    state.TOTPSecret,
		}, nil

	case eventCredentialAdded:
		p, err := unmarshalPayload[CredentialAddedPayload](evt)
		if err != nil {
			return state, err
		}
		cred := WebAuthnCredential{
			ID:              p.ID,
			PublicKey:       p.PublicKey,
			AttestationType: p.AttestationType,
			Transports:      append([]string(nil), p.Transports...),
			AAGUID:          append([]byte(nil), p.AAGUID...),
			SignCount:       p.SignCount,
			BackupEligible:  p.BackupEligible,
			BackupState:     p.BackupState,
			Name:            p.Name,
			CreatedAt:       evt.OccurredAt(),
		}
		return UserState{
			Email:         state.Email,
			DisplayName:   state.DisplayName,
			Roles:         state.Roles,
			Credentials:   append(state.Credentials, cred),
			Deleted:       state.Deleted,
			DeleteReason:  state.DeleteReason,
			EmailVerified: state.EmailVerified,
			TOTPEnabled:   state.TOTPEnabled,
			TOTPSecret:    state.TOTPSecret,
		}, nil

	case eventCredentialRemoved:
		p, err := unmarshalPayload[CredentialRemovedPayload](evt)
		if err != nil {
			return state, err
		}
		filtered := make([]WebAuthnCredential, 0, len(state.Credentials))
		for _, c := range state.Credentials {
			if !bytes.Equal(c.ID, p.ID) {
				filtered = append(filtered, c)
			}
		}
		return UserState{
			Email:         state.Email,
			DisplayName:   state.DisplayName,
			Roles:         state.Roles,
			Credentials:   filtered,
			Deleted:       state.Deleted,
			DeleteReason:  state.DeleteReason,
			EmailVerified: state.EmailVerified,
			TOTPEnabled:   state.TOTPEnabled,
			TOTPSecret:    state.TOTPSecret,
		}, nil

	case eventEmailVerified:
		p, err := unmarshalPayload[EmailVerifiedPayload](evt)
		if err != nil {
			return state, err
		}
		_ = p
		return UserState{
			Email:         state.Email,
			DisplayName:   state.DisplayName,
			Roles:         state.Roles,
			Credentials:   state.Credentials,
			Deleted:       state.Deleted,
			DeleteReason:  state.DeleteReason,
			EmailVerified: true,
		}, nil

	case eventTOTPEnabled:
		p, err := unmarshalPayload[TOTPEnabledPayload](evt)
		if err != nil {
			return state, err
		}
		return UserState{
			Email:         state.Email,
			DisplayName:   state.DisplayName,
			Roles:         state.Roles,
			Credentials:   state.Credentials,
			Deleted:       state.Deleted,
			DeleteReason:  state.DeleteReason,
			EmailVerified: state.EmailVerified,
			TOTPEnabled:   true,
			TOTPSecret:    p.Secret,
		}, nil

	case eventTOTPDisabled:
		return UserState{
			Email:         state.Email,
			DisplayName:   state.DisplayName,
			Roles:         state.Roles,
			Credentials:   state.Credentials,
			Deleted:       state.Deleted,
			DeleteReason:  state.DeleteReason,
			EmailVerified: state.EmailVerified,
			TOTPEnabled:   false,
			TOTPSecret:    nil,
		}, nil

	default:
		return state, nil
	}
}
