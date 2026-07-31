package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// dispatchUserCommand resolves a userID to its aggregate stream ID, dispatches
// the command produced by build, and routes the dispatch error through
// classifyDispatchError so per-handler methods can stay one line. kv is
// appended as key/value context to classified errors, mirroring the existing
// classifier contract.
//
// build receives the resolved stream ID and returns the command to dispatch.
// Using a builder keeps the helper compatible with constructors whose first
// argument is the aggregate ID.
func (s *Service) dispatchUserCommand(
	ctx context.Context,
	userID UserID,
	build func(id.StreamID) command.Command,
	kv ...string,
) error {
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "usermgmt.service.userid_conversion_failed", "convert userID")
	}
	if err := s.dispatcher.Dispatch(ctx, build(aggID)); err != nil {
		return s.classifyDispatchError(err, userID, kv...)
	}
	return nil
}

// GetUser retrieves a user by ID from the read model.
func (s *Service) GetUser(_ context.Context, id UserID) (*User, error) {
	user, ok := s.readModel.FindByUserID(id)
	if !ok {
		return nil, errorfamily.WrapRejection(ErrUserNotFound, "usermgmt.service.user_not_found", "get user")
	}
	return user, nil
}

// ChangeEmail dispatches a ChangeEmail command.
func (s *Service) ChangeEmail(ctx context.Context, userID UserID, newEmail string) error {
	return s.dispatchUserCommand(ctx, userID,
		func(aggID id.StreamID) command.Command { return NewChangeEmailCmd(aggID, newEmail) },
		"new_email", newEmail,
	)
}

// ChangeDisplayName dispatches a ChangeDisplayName command.
func (s *Service) ChangeDisplayName(ctx context.Context, userID UserID, newName string) error {
	return s.dispatchUserCommand(ctx, userID,
		func(aggID id.StreamID) command.Command { return NewChangeDisplayNameCmd(aggID, newName) },
		"new_name", newName,
	)
}

// DeleteUser dispatches a DeleteUser command (tombstone), revokes all sessions,
// and best-effort removes all memberships and bots owned by the user. The
// CasbinProjection handles authorization policy removal automatically via the
// UserDeleted event.
func (s *Service) DeleteUser(ctx context.Context, userID UserID, reason string) error {
	if err := s.dispatchUserCommand(ctx, userID,
		func(aggID id.StreamID) command.Command { return NewDeleteUserCmd(aggID, reason) },
		"reason", reason,
	); err != nil {
		return err
	}
	s.revokeSessionsBestEffort(ctx, userID, "failed to revoke sessions on delete")
	s.removeMembershipsForUserBestEffort(ctx, userID)
	s.deleteBotsForUserBestEffort(ctx, userID, reason)
	return nil
}

// removeMembershipsForUserBestEffort removes all memberships for a user.
// Errors are logged but not returned — the user is already deleted.
func (s *Service) removeMembershipsForUserBestEffort(ctx context.Context, userID UserID) {
	memberships := s.membershipReadModel.FindByActor(userID.Get().String())
	for _, mem := range memberships {
		removalCmd := NewRemoveMemberCmd(mem.ActorID, mem.TenantID)
		if err := s.dispatcher.Dispatch(ctx, removalCmd); err != nil {
			s.logger.Warn(
				"usermgmt: failed to remove membership on user deletion",
				"user_id", userID.Get(),
				"tenant_id", mem.TenantID.Get(),
				"error", err,
			)
		}
	}
}

// deleteBotsForUserBestEffort deletes all bots owned by a user.
// Errors are logged but not returned — the user is already deleted.
func (s *Service) deleteBotsForUserBestEffort(ctx context.Context, userID UserID, reason string) {
	bots := s.botReadModel.FindByOwner(userID)
	for _, bot := range bots {
		aggID, err := aggIDFromBot(bot.ID)
		if err != nil {
			s.logger.Warn(
				"usermgmt: failed to convert bot ID on user deletion",
				"user_id", userID.Get(),
				"bot_id", bot.ID.Get(),
				"error", err,
			)
			continue
		}
		if err := s.dispatcher.Dispatch(ctx, NewDeleteBotCmd(aggID, reason)); err != nil {
			s.logger.Warn(
				"usermgmt: failed to delete bot on user deletion",
				"user_id", userID.Get(),
				"bot_id", bot.ID.Get(),
				"error", err,
			)
		}
	}
}

// AddCredential dispatches an AddCredential command.
func (s *Service) AddCredential(ctx context.Context, userID UserID, cred WebAuthnCredential) error {
	if err := s.dispatchUserCommand(ctx, userID,
		func(aggID id.StreamID) command.Command { return NewAddCredentialCmd(aggID, cred) },
	); err != nil {
		return err
	}
	s.logAuth("credential_added", userID, "credential_name", cred.Name)
	return nil
}

// RemoveCredential dispatches a RemoveCredential command.
func (s *Service) RemoveCredential(ctx context.Context, userID UserID, credentialID []byte) error {
	if err := s.dispatchUserCommand(ctx, userID,
		func(aggID id.StreamID) command.Command { return NewRemoveCredentialCmd(aggID, credentialID) },
	); err != nil {
		return err
	}
	s.logAuth("credential_removed", userID)
	return nil
}
