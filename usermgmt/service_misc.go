package usermgmt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// GetUser retrieves a user by ID. Returns ErrUserNotFound if not found.
func (s *Service) GetUser(ctx context.Context, id UserID) (*User, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, fmt.Errorf("get user %q: %w", id, err)
		}
		return nil, withUserIDContext(event.NewTransient("internal", "get user").WithCause(err), id)
	}
	return u, nil
}

// transientErr creates a transient error with a user ID context. It is
// a thin wrapper over event.NewTransient + WithCause + withUserIDContext
// used to keep the public service methods concise.
func transientErr(userID UserID, msg string, cause error) error {
	return withUserIDContext(
		event.NewTransient("internal", msg).WithCause(cause),
		userID,
	)
}

// UpdateRoles replaces the user's roles in both the Casbin policy and the user store.
func (s *Service) UpdateRoles(
	ctx context.Context,
	userID UserID,
	roles []Role,
	domain string,
) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return fmt.Errorf("update roles: find user %q in domain %q: %w", userID, domain, err)
		}
		return transientErr(userID, fmt.Sprintf("find user %q", userID), err)
	}

	currentRoles, err := s.authz.RolesForUser(userID, domain)
	if err != nil {
		return transientErr(
			userID,
			fmt.Sprintf("get roles for user %q in domain %q", userID, domain),
			err,
		)
	}

	remove := make([]GroupPolicy, 0, len(currentRoles))
	for _, role := range currentRoles {
		remove = append(remove, GroupPolicy{
			Subject: userID.Get(), Role: role, Domain: domain,
		})
	}

	add := make([]GroupPolicy, 0, len(roles))
	for _, role := range roles {
		add = append(add, GroupPolicy{
			Subject: userID.Get(), Role: role, Domain: domain,
		})
	}

	user.SetRoles(roles)

	if err := s.authz.Apply(PolicyUpdate{
		RemoveGroups: remove,
		AddGroups:    add,
	}); err != nil {
		return transientErr(userID, fmt.Sprintf("apply role update for user %q", userID), err)
	}

	if err := s.saveUser(ctx, user, "after role update", userID); err != nil {
		return err
	}

	s.logAuth("roles_updated", userID, "roles", formatRoles(roles), "domain", domain)

	s.emit(userID, RolesUpdatedEvent{
		Roles:      append([]Role(nil), roles...),
		Domain:     domain,
		OccurredAt: time.Now().UTC(),
	})
	return nil
}

func formatRoles(roles []Role) string {
	strs := make([]string, len(roles))
	for i, r := range roles {
		strs[i] = string(r)
	}
	return strings.Join(strs, ",")
}

func (s *Service) saveUser(ctx context.Context, user *User, context string, userID UserID) error {
	if err := s.users.Save(ctx, user); err != nil {
		return withUserIDContext(
			event.NewTransient("internal", fmt.Sprintf("save user %q %s", userID, context)).
				WithCause(err),
			userID,
		)
	}
	return nil
}

func classifyLoginError(err error) error {
	if errors.Is(err, ErrUserNotFound) {
		return ErrInvalidCredentials
	}
	return event.NewTransient("internal", "find user by email").WithCause(err)
}

// ChangePassword verifies the old password, validates the new password length,
// and updates the stored hash.
func (s *Service) ChangePassword(
	ctx context.Context,
	userID UserID,
	oldPassword, newPassword string,
) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return withUserIDContext(
			event.NewTransient("internal", fmt.Sprintf("find user %q", userID)).WithCause(err),
			userID,
		)
	}

	matched, err := user.ChangePassword(oldPassword, newPassword, s.bcryptCost)
	if err != nil {
		return err
	}
	if !matched {
		return ErrInvalidCredentials
	}

	if err := s.saveUser(ctx, user, "after password change", userID); err != nil {
		return err
	}
	s.emit(userID, PasswordChangedEvent{OccurredAt: time.Now().UTC()})
	return nil
}
