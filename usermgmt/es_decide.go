package usermgmt

import (
	"bytes"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func decideRegisterUser(
	aggID id.AggregateID,
	email, displayName string,
	roles []Role,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if state.Exists() {
			return nil, event.NewConflict("usermgmt.user_already_exists",
				"user with this ID already exists")
		}
		if email == "" {
			return nil, event.NewRejection("usermgmt.register.email_required",
				"email is required")
		}
		rolesCopy := make([]Role, len(roles))
		copy(rolesCopy, roles)
		payload, err := marshalPayload(UserRegisteredPayload{
			Email:       email,
			DisplayName: displayName,
			Roles:       rolesCopy,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal UserRegistered payload: %w", err)
		}
		evt, err := event.NewEvent(
			eventUserRegistered, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("create UserRegistered event: %w", err)
		}
		return []event.Event{evt}, nil
	}
}

func decideUpdateRoles(
	aggID id.AggregateID,
	roles []Role,
	domain string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.update_roles.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.update_roles.deleted",
				"cannot update roles of deleted user")
		}
		rolesCopy := make([]Role, len(roles))
		copy(rolesCopy, roles)
		payload, err := marshalPayload(RolesUpdatedPayload{
			Roles:  rolesCopy,
			Domain: domain,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal RolesUpdated payload: %w", err)
		}
		evt, err := event.NewEvent(
			eventRolesUpdated, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("create RolesUpdated event: %w", err)
		}
		return []event.Event{evt}, nil
	}
}

func decideChangeEmail(
	aggID id.AggregateID,
	email string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.change_email.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.change_email.deleted",
				"cannot change email of deleted user")
		}
		if state.Email == email {
			return nil, nil
		}
		payload, err := marshalPayload(EmailChangedPayload{
			Email: email,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal EmailChanged payload: %w", err)
		}
		evt, err := event.NewEvent(
			eventEmailChanged, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("create EmailChanged event: %w", err)
		}
		return []event.Event{evt}, nil
	}
}

func decideChangeDisplayName(
	aggID id.AggregateID,
	displayName string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.change_display_name.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.change_display_name.deleted",
				"cannot change display name of deleted user")
		}
		if state.DisplayName == displayName {
			return nil, nil
		}
		payload, err := marshalPayload(DisplayNameChangedPayload{
			DisplayName: displayName,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal DisplayNameChanged payload: %w", err)
		}
		evt, err := event.NewEvent(
			eventDisplayNameChanged, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("create DisplayNameChanged event: %w", err)
		}
		return []event.Event{evt}, nil
	}
}

func decideDeleteUser(
	aggID id.AggregateID,
	reason string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.delete_user.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.delete_user.already_deleted",
				"user is already deleted")
		}
		payload, err := marshalPayload(UserDeletedPayload{
			Reason: reason,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal UserDeleted payload: %w", err)
		}
		evt, err := event.NewEvent(
			eventUserDeleted, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("create UserDeleted event: %w", err)
		}
		marked, markErr := event.MarkTombstone(evt)
		if markErr != nil {
			return nil, fmt.Errorf("mark tombstone: %w", markErr)
		}
		return []event.Event{marked}, nil
	}
}

func decideAddCredential(
	aggID id.AggregateID,
	cred WebAuthnCredential,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.add_credential.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.add_credential.deleted",
				"cannot add credential to deleted user")
		}
		for _, existing := range state.Credentials {
			if bytes.Equal(existing.ID, cred.ID) {
				return nil, event.NewConflict("usermgmt.credential_already_exists",
					"credential with this ID already exists")
			}
		}
		payload, err := marshalPayload(CredentialAddedPayload{
			ID:              cred.ID,
			PublicKey:       cred.PublicKey,
			AttestationType: cred.AttestationType,
			Transports:      cred.Transports,
			AAGUID:          cred.AAGUID,
			SignCount:       cred.SignCount,
			BackupEligible:  cred.BackupEligible,
			BackupState:     cred.BackupState,
			Name:            cred.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal CredentialAdded payload: %w", err)
		}
		evt, err := event.NewEvent(
			eventCredentialAdded, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("create CredentialAdded event: %w", err)
		}
		return []event.Event{evt}, nil
	}
}

func decideRemoveCredential(
	aggID id.AggregateID,
	credentialID []byte,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.remove_credential.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.remove_credential.deleted",
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
			return nil, event.NewRejection("usermgmt.credential_not_found",
				"credential not found")
		}
		payload, err := marshalPayload(CredentialRemovedPayload{
			ID: credentialID,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal CredentialRemoved payload: %w", err)
		}
		evt, err := event.NewEvent(
			eventCredentialRemoved, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("create CredentialRemoved event: %w", err)
		}
		return []event.Event{evt}, nil
	}
}

func decideVerifyEmail(
	aggID id.AggregateID,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, event.NewRejection("usermgmt.verify_email.not_found",
				"user does not exist")
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.verify_email.deleted",
				"cannot verify email of deleted user")
		}
		if state.EmailVerified {
			return nil, nil
		}
		payload, err := marshalPayload(EmailVerifiedPayload{
			Email: state.Email,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal EmailVerified payload: %w", err)
		}
		evt, err := event.NewEvent(
			eventEmailVerified, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("create EmailVerified event: %w", err)
		}
		return []event.Event{evt}, nil
	}
}
