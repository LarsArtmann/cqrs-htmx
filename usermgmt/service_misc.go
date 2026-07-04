package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// GetUser retrieves a user by ID from the read model.
func (s *Service) GetUser(_ context.Context, id UserID) (*User, error) {
	user, ok := s.readModel.FindByUserID(id)
	if !ok {
		return nil, event.Wrapf(ErrUserNotFound, event.Rejection, "usermgmt.service.user_not_found", "get user %q", id)
	}
	return user, nil
}

// ChangeEmail dispatches a ChangeEmail command.
func (s *Service) ChangeEmail(ctx context.Context, userID UserID, newEmail string) error {
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return event.WrapInfrastructure(err, "usermgmt.service.userid_conversion_failed", "convert userID")
	}
	err = s.dispatcher.Dispatch(ctx, NewChangeEmailCmd(aggID, newEmail))
	if err != nil {
		return s.classifyDispatchError(err, userID, "new_email", newEmail)
	}
	return nil
}

// ChangeDisplayName dispatches a ChangeDisplayName command.
func (s *Service) ChangeDisplayName(ctx context.Context, userID UserID, newName string) error {
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return event.WrapInfrastructure(err, "usermgmt.service.userid_conversion_failed", "convert userID")
	}
	err = s.dispatcher.Dispatch(ctx, NewChangeDisplayNameCmd(aggID, newName))
	if err != nil {
		return s.classifyDispatchError(err, userID, "new_name", newName)
	}
	return nil
}

// DeleteUser dispatches a DeleteUser command (tombstone) and revokes all sessions.
func (s *Service) DeleteUser(ctx context.Context, userID UserID, reason string) error {
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return event.WrapInfrastructure(err, "usermgmt.service.userid_conversion_failed", "convert userID")
	}
	err = s.dispatcher.Dispatch(ctx, NewDeleteUserCmd(aggID, reason))
	if err != nil {
		return s.classifyDispatchError(err, userID, "reason", reason)
	}
	s.revokeSessionsBestEffort(ctx, userID, "failed to revoke sessions on delete")
	return nil
}

// AddCredential dispatches an AddCredential command.
func (s *Service) AddCredential(ctx context.Context, userID UserID, cred WebAuthnCredential) error {
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return event.WrapInfrastructure(err, "usermgmt.service.userid_conversion_failed", "convert userID")
	}
	err = s.dispatcher.Dispatch(ctx, NewAddCredentialCmd(aggID, cred))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}
	s.logAuth("credential_added", userID, "credential_name", cred.Name)
	return nil
}

// RemoveCredential dispatches a RemoveCredential command.
func (s *Service) RemoveCredential(ctx context.Context, userID UserID, credentialID []byte) error {
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return event.WrapInfrastructure(err, "usermgmt.service.userid_conversion_failed", "convert userID")
	}
	err = s.dispatcher.Dispatch(ctx, NewRemoveCredentialCmd(aggID, credentialID))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}
	s.logAuth("credential_removed", userID)
	return nil
}
