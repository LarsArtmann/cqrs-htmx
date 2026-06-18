package usermgmt

import (
	"context"
	"fmt"
	"strings"
)

// GetUser retrieves a user by ID from the read model.
func (s *Service) GetUser(_ context.Context, id UserID) (*User, error) {
	user, ok := s.readModel.FindByUserID(id)
	if !ok {
		return nil, fmt.Errorf("get user %q: %w", id, ErrUserNotFound)
	}
	return user, nil
}

// UpdateRoles dispatches an UpdateRoles command and revokes existing sessions
// to force re-authentication with the new privilege level.
func (s *Service) UpdateRoles(ctx context.Context, userID UserID, roles []Role, domain string) error {
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return fmt.Errorf("convert userID: %w", err)
	}
	err = s.dispatcher.Dispatch(ctx, NewUpdateRolesCmd(aggID, roles, domain))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}
	s.revokeSessionsBestEffort(ctx, userID, "failed to rotate sessions after role update")
	s.logAuth("roles_updated", userID, "roles", formatRoles(roles), "domain", domain)
	s.emit(userID, RolesUpdatedEvent{
		Roles:      append([]Role(nil), roles...),
		Domain:     domain,
		OccurredAt: nowUTC(),
	})
	return nil
}

// ChangeEmail dispatches a ChangeEmail command.
func (s *Service) ChangeEmail(ctx context.Context, userID UserID, newEmail string) error {
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return fmt.Errorf("convert userID: %w", err)
	}
	err = s.dispatcher.Dispatch(ctx, NewChangeEmailCmd(aggID, newEmail))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}
	return nil
}

// ChangeDisplayName dispatches a ChangeDisplayName command.
func (s *Service) ChangeDisplayName(ctx context.Context, userID UserID, newName string) error {
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return fmt.Errorf("convert userID: %w", err)
	}
	err = s.dispatcher.Dispatch(ctx, NewChangeDisplayNameCmd(aggID, newName))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}
	return nil
}

// DeleteUser dispatches a DeleteUser command (tombstone) and revokes all sessions.
func (s *Service) DeleteUser(ctx context.Context, userID UserID, reason string) error {
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return fmt.Errorf("convert userID: %w", err)
	}
	err = s.dispatcher.Dispatch(ctx, NewDeleteUserCmd(aggID, reason))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}
	s.revokeSessionsBestEffort(ctx, userID, "failed to revoke sessions on delete")
	return nil
}

// AddCredential dispatches an AddCredential command.
func (s *Service) AddCredential(ctx context.Context, userID UserID, cred WebAuthnCredential) error {
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return fmt.Errorf("convert userID: %w", err)
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
		return fmt.Errorf("convert userID: %w", err)
	}
	err = s.dispatcher.Dispatch(ctx, NewRemoveCredentialCmd(aggID, credentialID))
	if err != nil {
		return s.classifyDispatchError(err, userID)
	}
	s.logAuth("credential_removed", userID)
	return nil
}

func formatRoles(roles []Role) string {
	strs := make([]string, len(roles))
	for i, r := range roles {
		strs[i] = string(r)
	}
	return strings.Join(strs, ",")
}
