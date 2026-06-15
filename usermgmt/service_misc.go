package usermgmt

import (
	"context"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// GetUser retrieves a user by ID from the read model. Returns ErrUserNotFound if not found.
func (s *Service) GetUser(_ context.Context, id UserID) (*User, error) {
	user, ok := s.readModel.FindByUserID(id)
	if !ok {
		return nil, fmt.Errorf("get user %q: %w", id, ErrUserNotFound)
	}
	return user, nil
}

// UpdateRoles dispatches an UpdateRoles command. The Casbin projection updates
// policies automatically from the event.
func (s *Service) UpdateRoles(ctx context.Context, userID UserID, roles []Role, domain string) error {
	aggID := aggIDFromUser(userID)
	err := s.dispatcher.Dispatch(ctx, NewUpdateRolesCmd(aggID, roles, domain))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}
	s.logAuth("roles_updated", userID, "roles", formatRoles(roles), "domain", domain)
	s.emit(userID, RolesUpdatedEvent{
		Roles:      append([]Role(nil), roles...),
		Domain:     domain,
		OccurredAt: nowUTC(),
	})
	return nil
}

// ChangePassword verifies the old password, hashes the new one, and dispatches
// a ChangePassword command.
func (s *Service) ChangePassword(
	ctx context.Context,
	userID UserID,
	oldPassword, newPassword string,
) error {
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		return fmt.Errorf("change password: %w", ErrUserNotFound)
	}

	if !user.CheckPassword(oldPassword) {
		return ErrInvalidCredentials
	}

	if err := validatePassword(newPassword); err != nil {
		return err
	}

	hash, err := s.hashPassword(newPassword)
	if err != nil {
		return withUserIDContext(
			event.NewTransient("internal", "hash password").WithCause(err), userID,
		)
	}

	aggID := aggIDFromUser(userID)
	err = s.dispatcher.Dispatch(ctx, NewChangePasswordCmd(aggID, hash))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}

	s.emit(userID, PasswordChangedEvent{OccurredAt: nowUTC()})
	return nil
}

// ChangeEmail dispatches a ChangeEmail command. No event is emitted if the email is unchanged.
func (s *Service) ChangeEmail(ctx context.Context, userID UserID, newEmail string) error {
	aggID := aggIDFromUser(userID)
	err := s.dispatcher.Dispatch(ctx, NewChangeEmailCmd(aggID, newEmail))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}
	return nil
}

// ChangeDisplayName dispatches a ChangeDisplayName command.
func (s *Service) ChangeDisplayName(ctx context.Context, userID UserID, newName string) error {
	aggID := aggIDFromUser(userID)
	err := s.dispatcher.Dispatch(ctx, NewChangeDisplayNameCmd(aggID, newName))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}
	return nil
}

// DeleteUser dispatches a DeleteUser command (tombstone) and revokes all sessions.
func (s *Service) DeleteUser(ctx context.Context, userID UserID, reason string) error {
	aggID := aggIDFromUser(userID)
	err := s.dispatcher.Dispatch(ctx, NewDeleteUserCmd(aggID, reason))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}

	if err := s.sessions.DeleteByUserID(ctx, userID); err != nil {
		s.logger.Warn("usermgmt: failed to revoke sessions on delete",
			"user_id", userID, "error", err)
	}

	return nil
}

func formatRoles(roles []Role) string {
	strs := make([]string, len(roles))
	for i, r := range roles {
		strs[i] = string(r)
	}
	return strings.Join(strs, ",")
}
