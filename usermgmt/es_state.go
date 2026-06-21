package usermgmt

import (
	"bytes"
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// UserState is the aggregate state for the User, reconstructed by folding events.
type UserState struct {
	Email            string
	DisplayName      string
	Roles            []Role
	Credentials      []WebAuthnCredential
	ExternalAccounts []ExternalAccount
	Deleted          bool
	DeleteReason     string
	EmailVerified    bool
	TOTPEnabled      bool
	TOTPSecret       []byte
}

// Exists reports whether the user has been registered (has at least one event).
func (s UserState) Exists() bool {
	return s.Email != ""
}

func unmarshalPayload[T any](evt event.Event) (T, error) {
	raw, err := applyUpcasters(evt.Type(), evt.Payload())
	if err != nil {
		return *new(T), event.WrapCorruption(err, "usermgmt.payload.upcast_failed", "upcast payload")
	}
	var target T
	if err := json.Unmarshal(raw, &target); err != nil {
		return target, event.WrapCorruption(err,
			"usermgmt.payload_decode_failed",
			"decode payload for event "+string(evt.Type()))
	}
	return target, nil
}

// foldUser applies an event to the current UserState, returning the new state.
// Uses a shallow-copy-and-mutate pattern: next := state carries all existing
// fields forward, so each case only touches the fields it changes. This makes
// adding new aggregate fields O(1) instead of requiring every case to rebuild
// the entire struct.
//
//nolint:gocognit // inherent to 12-case event switch; each case is simple decode+mutate
func foldUser(state UserState, evt event.Event) (UserState, error) {
	next := state

	switch evt.Type() {
	case eventUserRegistered:
		p, err := unmarshalPayload[UserRegisteredPayload](evt)
		if err != nil {
			return state, err
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		next = UserState{
			Email:       p.Email,
			DisplayName: p.DisplayName,
			Roles:       roles,
		}

	case eventRolesUpdated:
		p, err := unmarshalPayload[RolesUpdatedPayload](evt)
		if err != nil {
			return state, err
		}
		roles := make([]Role, len(p.Roles))
		copy(roles, p.Roles)
		next.Roles = roles

	case eventEmailChanged:
		p, err := unmarshalPayload[EmailChangedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Email = p.Email
		next.EmailVerified = false // email change resets verification
		next.TOTPEnabled = false
		next.TOTPSecret = nil

	case eventDisplayNameChanged:
		p, err := unmarshalPayload[DisplayNameChangedPayload](evt)
		if err != nil {
			return state, err
		}
		next.DisplayName = p.DisplayName

	case eventUserDeleted:
		p, err := unmarshalPayload[UserDeletedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Deleted = true
		next.DeleteReason = p.Reason

	case eventCredentialAdded:
		p, err := unmarshalPayload[CredentialAddedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Credentials = append(next.Credentials, newCredentialFromPayload(p, evt.OccurredAt()))

	case eventCredentialRemoved:
		p, err := unmarshalPayload[CredentialRemovedPayload](evt)
		if err != nil {
			return state, err
		}
		filtered := make([]WebAuthnCredential, 0, len(next.Credentials))
		for _, c := range next.Credentials {
			if !bytes.Equal(c.ID, p.ID) {
				filtered = append(filtered, c)
			}
		}
		next.Credentials = filtered

	case eventEmailVerified:
		_, err := unmarshalPayload[EmailVerifiedPayload](evt)
		if err != nil {
			return state, err
		}
		next.EmailVerified = true

	case eventTOTPEnabled:
		p, err := unmarshalPayload[TOTPEnabledPayload](evt)
		if err != nil {
			return state, err
		}
		next.TOTPEnabled = true
		next.TOTPSecret = p.Secret

	case eventTOTPDisabled:
		next.TOTPEnabled = false
		next.TOTPSecret = nil

	case eventExternalAccountLinked:
		p, err := unmarshalPayload[ExternalAccountLinkedPayload](evt)
		if err != nil {
			return state, err
		}
		next.ExternalAccounts = append(next.ExternalAccounts, ExternalAccount{
			Provider:    p.Provider,
			Subject:     p.Subject,
			Email:       p.Email,
			DisplayName: p.DisplayName,
			LinkedAt:    evt.OccurredAt(),
		})

	case eventExternalAccountUnlinked:
		p, err := unmarshalPayload[ExternalAccountUnlinkedPayload](evt)
		if err != nil {
			return state, err
		}
		filtered := make([]ExternalAccount, 0, len(next.ExternalAccounts))
		for _, ea := range next.ExternalAccounts {
			if ea.Provider != p.Provider || ea.Subject != p.Subject {
				filtered = append(filtered, ea)
			}
		}
		next.ExternalAccounts = filtered

	default:
	}

	return next, nil
}
