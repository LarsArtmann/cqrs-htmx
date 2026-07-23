package usermgmt

import (
	"bytes"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func decideAddCredential(
	aggID id.AggregateID,
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
		payload, err := marshalPayload(CredentialAddedPayload{
			SchemaVersion: currentSchemaVersion,
			CredentialCore: CredentialCore{
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
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.add_credential.marshal_failed",
				"marshal CredentialAdded payload",
			)
		}
		evt, err := event.NewEvent(
			eventCredentialAdded, aggID, aggregateTypeUser, version.Increment(),
			payload,
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
	aggID id.AggregateID,
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
		payload, err := marshalPayload(CredentialRemovedPayload{
			SchemaVersion: currentSchemaVersion,
			ID:            credentialID,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.remove_credential.marshal_failed",
				"marshal CredentialRemoved payload",
			)
		}
		evt, err := event.NewEvent(
			eventCredentialRemoved, aggID, aggregateTypeUser, version.Increment(),
			payload,
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
