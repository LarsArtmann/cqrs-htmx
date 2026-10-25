package usermgmt

import (
	"bytes"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func decideAddCredential(
	aggID id.StreamID,
	cred WebAuthnCredential,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "add_credential"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection("usermgmt.add_credential.deleted",
				"cannot add credential to deleted user")
		}
		for _, existing := range state.Credentials {
			if bytes.Equal(existing.ID, cred.ID) {
				return nil, errorfamily.NewConflict("usermgmt.credential_already_exists",
					"credential with this ID already exists")
			}
		}
		evt, err := event.New(
			eventCredentialAdded, aggID, aggregateTypeUser, version.Increment(),
			CredentialAddedPayload{
				SchemaVersion:   currentSchemaVersion,
				ID:              cred.ID,
				PublicKey:       cred.PublicKey,
				AttestationType: cred.AttestationType,
				Transports:      cred.Transports,
				AAGUID:          cred.AAGUID,
				SignCount:       cred.SignCount,
				BackupEligible:  cred.BackupEligible,
				BackupState:     cred.BackupState,
				Name:            cred.Name,
			},
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.add_credential.event_failed",
				"create CredentialAdded event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideRemoveCredential(
	aggID id.StreamID,
	credentialID []byte,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "remove_credential"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection("usermgmt.remove_credential.deleted",
				"cannot remove credential from deleted user")
		}
		found := false
		for _, c := range state.Credentials {
			if bytes.Equal(c.ID, credentialID) {
				found = true
				break
			}
		}
		if !found {
			return nil, errorfamily.NewRejection("usermgmt.credential_not_found",
				"credential not found")
		}
		evt, err := event.New(
			eventCredentialRemoved, aggID, aggregateTypeUser, version.Increment(),
			CredentialRemovedPayload{
				SchemaVersion: currentSchemaVersion,
				ID:            credentialID,
			},
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.remove_credential.event_failed",
				"create CredentialRemoved event",
			)
		}
		return []event.Event{evt}, nil
	}
}

// decideRecordCredentialUse records the state a successful WebAuthn assertion
// observed on a registered credential. Two invariants live here:
//
//   - the signature counter must not regress (WebAuthn §7.2 step 17: a
//     non-advancing counter means the authenticator may be cloned);
//   - a no-op (counter 0→0, unchanged flags — authenticators without a counter)
//     emits nothing, so the journal does not fill with empty uses.
func decideRecordCredentialUse(
	aggID id.StreamID,
	use CredentialUse,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "record_credential_use"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection("usermgmt.record_credential_use.deleted",
				"cannot record credential use for deleted user")
		}
		existing := findCredential(state.Credentials, use.CredentialID)
		if existing == nil {
			return nil, errorfamily.NewRejection("usermgmt.credential_not_found",
				"credential not found")
		}
		if existing.CounterRegressed(use.SignCount) {
			return nil, errorfamily.NewConflict(
				"usermgmt.credential_counter_regressed",
				"signature counter did not advance: the authenticator may be cloned",
			).
				WithContextAny("stored_sign_count", existing.SignCount).
				WithContextAny("asserted_sign_count", use.SignCount)
		}
		if existing.SignCount == use.SignCount &&
			existing.UserPresent == use.UserPresent &&
			existing.UserVerified == use.UserVerified &&
			existing.BackupState == use.BackupState {
			return nil, nil
		}
		evt, err := event.New(
			eventCredentialUsed, aggID, aggregateTypeUser, version.Increment(),
			CredentialUsedPayload{
				SchemaVersion: currentSchemaVersion,
				CredentialID:  use.CredentialID,
				SignCount:     use.SignCount,
				UserPresent:   use.UserPresent,
				UserVerified:  use.UserVerified,
				BackupState:   use.BackupState,
			},
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.record_credential_use.event_failed",
				"create CredentialUsed event",
			)
		}
		return []event.Event{evt}, nil
	}
}

// findCredential returns the credential with the given ID, or nil.
func findCredential(credentials []WebAuthnCredential, credentialID []byte) *WebAuthnCredential {
	for i := range credentials {
		if bytes.Equal(credentials[i].ID, credentialID) {
			return &credentials[i]
		}
	}
	return nil
}
