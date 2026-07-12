package usermgmt

import (
	"bytes"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// UserState is the aggregate state for the User, reconstructed by folding events.
// Roles are NOT tracked here — they are managed by the Membership aggregate
// and CasbinProjection. Use MembershipReadModel or Authz.RolesForActor to query roles.
type UserState struct {
	Email            string
	DisplayName      string
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

// unmarshalPayload decodes an event's payload into a typed value using the
// codec that matches the event's declared encoding (JSON or CBOR). This makes
// the library transparently compatible with consumers who set
// event.DefaultCodec = codec.CBORCodec{} for more compact storage.
//
// Upcasters run first: they may transform legacy payload shapes before decoding.
// The codec is resolved per-event via codec.ForEncoding(evt.Encoding()), so
// mixed JSON+CBOR event streams decode correctly.
func unmarshalPayload[T any](evt event.Event) (T, error) {
	raw, err := applyUpcasters(evt.Type(), evt.Payload())
	if err != nil {
		return *new(T), errorfamily.WrapCorruption(err, "usermgmt.payload.upcast_failed", "upcast payload")
	}
	c, err := codec.ForEncoding(evt.Encoding())
	if err != nil {
		return *new(T), errorfamily.WrapCorruption(err,
			"usermgmt.payload_decode_failed",
			"resolve codec for encoding "+string(evt.Encoding())+
				" (event "+string(evt.Type())+")")
	}
	var target T
	if err := c.Decode(raw, &target); err != nil {
		return target, errorfamily.WrapCorruption(err,
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
		next = UserState{
			Email:       p.Email,
			DisplayName: p.DisplayName,
		}

	case eventRolesUpdated:
		// Roles are no longer part of UserState. They are managed by the
		// Membership aggregate and derived by CasbinProjection. We still
		// decode the payload to detect corruption.
		_, err := unmarshalPayload[RolesUpdatedPayload](evt)
		if err != nil {
			return state, err
		}

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
			externalAccountCore: externalAccountCore{
				Provider:    p.Provider,
				Subject:     p.Subject,
				Email:       p.Email,
				DisplayName: p.DisplayName,
			},
			LinkedAt: evt.OccurredAt(),
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
		return state, errorfamily.NewRejection(
			"usermgmt.user.unknown_event",
			"foldUser received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}
