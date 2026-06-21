package usermgmt

import (
	"bytes"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// requireExists returns a "user does not exist" rejection if the user has not
// been registered yet. Used as the first guard in every decider that operates
// on an existing user. The domain argument becomes the rejection code suffix,
// e.g. domain="update_roles" → "usermgmt.update_roles.not_found".
func requireExists(state UserState, domain string) error {
	if !state.Exists() {
		return event.NewRejection(
			"usermgmt."+domain+".not_found",
			"user does not exist",
		)
	}
	return nil
}

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
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.register.marshal_failed",
				"marshal UserRegistered payload",
			)
		}
		evt, err := event.NewEvent(
			eventUserRegistered, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(err, "usermgmt.register.event_failed", "create UserRegistered event")
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
		if err := requireExists(state, "update_roles"); err != nil {
			return nil, err
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
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.update_roles.marshal_failed",
				"marshal RolesUpdated payload",
			)
		}
		evt, err := event.NewEvent(
			eventRolesUpdated, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(err, "usermgmt.update_roles.event_failed", "create RolesUpdated event")
		}
		return []event.Event{evt}, nil
	}
}

func decideChangeEmail(
	aggID id.AggregateID,
	email string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "change_email"); err != nil {
			return nil, err
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
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.change_email.marshal_failed",
				"marshal EmailChanged payload",
			)
		}
		evt, err := event.NewEvent(
			eventEmailChanged, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(err, "usermgmt.change_email.event_failed", "create EmailChanged event")
		}
		return []event.Event{evt}, nil
	}
}

func decideChangeDisplayName(
	aggID id.AggregateID,
	displayName string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "change_display_name"); err != nil {
			return nil, err
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
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.change_display_name.marshal_failed",
				"marshal DisplayNameChanged payload",
			)
		}
		evt, err := event.NewEvent(
			eventDisplayNameChanged, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.change_display_name.event_failed",
				"create DisplayNameChanged event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideDeleteUser(
	aggID id.AggregateID,
	reason string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "delete_user"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.delete_user.already_deleted",
				"user is already deleted")
		}
		payload, err := marshalPayload(UserDeletedPayload{
			Reason: reason,
		})
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.delete_user.marshal_failed",
				"marshal UserDeleted payload",
			)
		}
		evt, err := event.NewEvent(
			eventUserDeleted, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(err, "usermgmt.delete_user.event_failed", "create UserDeleted event")
		}
		marked, markErr := event.MarkTombstone(evt)
		if markErr != nil {
			return nil, event.WrapInfrastructure(markErr, "usermgmt.delete_user.tombstone_failed", "mark tombstone")
		}
		return []event.Event{marked}, nil
	}
}

func decideAddCredential(
	aggID id.AggregateID,
	cred WebAuthnCredential,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "add_credential"); err != nil {
			return nil, err
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
			return nil, event.WrapInfrastructure(
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
			return nil, event.WrapInfrastructure(
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
			return nil, event.WrapInfrastructure(
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
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.remove_credential.event_failed",
				"create CredentialRemoved event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideVerifyEmail(
	aggID id.AggregateID,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "verify_email"); err != nil {
			return nil, err
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
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.verify_email.marshal_failed",
				"marshal EmailVerified payload",
			)
		}
		evt, err := event.NewEvent(
			eventEmailVerified, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.verify_email.event_failed",
				"create EmailVerified event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideEnableTOTP(
	aggID id.AggregateID,
	secret []byte,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "enable_totp"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.enable_totp.deleted",
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
			return nil, event.WrapInfrastructure(
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
			return nil, event.WrapInfrastructure(err, "usermgmt.enable_totp.event_failed", "create TOTPEnabled event")
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
			return nil, event.NewRejection("usermgmt.disable_totp.deleted",
				"cannot disable TOTP for deleted user")
		}
		if !state.TOTPEnabled {
			return nil, nil
		}
		payload, err := marshalPayload(TOTPDisabledPayload{
			SchemaVersion: currentSchemaVersion,
		})
		if err != nil {
			return nil, event.WrapInfrastructure(
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
			return nil, event.WrapInfrastructure(err, "usermgmt.disable_totp.event_failed", "create TOTPDisabled event")
		}
		return []event.Event{evt}, nil
	}
}

func decideLinkExternalAccount(
	aggID id.AggregateID,
	provider, subject, email, displayName string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "link_external_account"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.link_external_account.deleted",
				"cannot link external account to deleted user")
		}
		if provider == "" || subject == "" {
			return nil, event.NewRejection("usermgmt.link_external_account.invalid",
				"provider and subject are required")
		}
		for _, ea := range state.ExternalAccounts {
			if ea.Provider == provider && ea.Subject == subject {
				return nil, event.NewConflict("usermgmt.external_account_already_linked",
					"external account already linked to this user")
			}
		}
		payload, err := marshalPayload(ExternalAccountLinkedPayload{
			SchemaVersion: currentSchemaVersion,
			Provider:      provider,
			Subject:       subject,
			Email:         email,
			DisplayName:   displayName,
		})
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.link_external_account.marshal_failed",
				"marshal ExternalAccountLinked payload",
			)
		}
		evt, err := event.NewEvent(
			eventExternalAccountLinked, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.link_external_account.event_failed",
				"create ExternalAccountLinked event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideUnlinkExternalAccount(
	aggID id.AggregateID,
	provider, subject string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "unlink_external_account"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, event.NewRejection("usermgmt.unlink_external_account.deleted",
				"cannot unlink external account from deleted user")
		}
		found := false
		for _, ea := range state.ExternalAccounts {
			if ea.Provider == provider && ea.Subject == subject {
				found = true
				break
			}
		}
		if !found {
			return nil, event.NewRejection("usermgmt.external_account_not_found",
				"external account not linked to this user")
		}
		// Last-auth-method guard: reject if removing this would leave the user
		// with zero WebAuthn credentials and zero other external accounts.
		if len(state.Credentials) == 0 && len(state.ExternalAccounts) <= 1 {
			return nil, event.NewRejection("usermgmt.unlink_external_account.last_auth_method",
				"cannot remove the last authentication method")
		}
		payload, err := marshalPayload(ExternalAccountUnlinkedPayload{
			SchemaVersion: currentSchemaVersion,
			Provider:      provider,
			Subject:       subject,
		})
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.unlink_external_account.marshal_failed",
				"marshal ExternalAccountUnlinked payload",
			)
		}
		evt, err := event.NewEvent(
			eventExternalAccountUnlinked, aggID, aggregateTypeUser, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(
				err,
				"usermgmt.unlink_external_account.event_failed",
				"create ExternalAccountUnlinked event",
			)
		}
		return []event.Event{evt}, nil
	}
}
