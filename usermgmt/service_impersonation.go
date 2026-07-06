package usermgmt

import (
	"context"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	errorfamily "github.com/larsartmann/go-error-family"
)

// BeginImpersonation creates a session that allows the caller to act as the
// target user. The session's Origin records who initiated the impersonation
// and why, enabling full audit trails.
//
// Security guards:
//   - Caller must exist and have the super_admin role
//   - Caller cannot impersonate themselves
//   - Target must exist
//   - Reason must not be empty
//
// The returned session token should be used by the admin for subsequent
// requests. The admin's original session remains active.
//
//nolint:funlen // security guards are inherently sequential; splitting harms readability
func (s *Service) BeginImpersonation(
	ctx context.Context, callerID, targetID UserID, reason string,
) (*Session, error) {
	if reason == "" {
		return nil, errorfamily.NewRejection(
			"usermgmt.impersonation.reason_required",
			"impersonation reason is required for audit trail",
		)
	}

	if callerID == targetID {
		return nil, errorfamily.NewRejection(
			"usermgmt.impersonation.self_impersonation",
			"cannot impersonate yourself",
		)
	}

	callerAggID, err := aggIDFromUser(callerID)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "usermgmt.impersonation.caller_id_invalid",
			"convert caller UserID",
		)
	}

	// Verify caller has super_admin role.
	roles, err := s.authz.RolesForUser(callerID, NewTenantID(callerAggID.String()))
	if err != nil {
		return nil, errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.impersonation.role_check_failed",
			"check roles for caller %s", callerID,
		)
	}
	if !slices.Contains(roles, RoleSuperAdmin) {
		return nil, errorfamily.NewRejection(
			"usermgmt.impersonation.insufficient_privileges",
			"caller must have super_admin role to impersonate",
		)
	}

	// Verify target exists.
	targetAggID, err := aggIDFromUser(targetID)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "usermgmt.impersonation.target_id_invalid",
			"convert target UserID",
		)
	}
	if _, ok := s.readModel.FindByID(targetAggID); !ok {
		return nil, errorfamily.NewRejection(
			"usermgmt.impersonation.target_not_found",
			"target user does not exist",
		)
	}

	callerActor := ActorIDFromUser(callerID)
	targetActor := ActorIDFromUser(targetID)

	session, err := NewImpersonationSession(targetActor, callerActor, reason, s.sessionTTL)
	if err != nil {
		return nil, errorfamily.NewTransient(
			"internal", "create impersonation session",
		).WithCause(err)
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, errorfamily.NewTransient(
			"internal", "store impersonation session",
		).WithCause(err)
	}

	s.logAuth(
		"impersonation_begun", callerID,
		"target", targetID.Get(),
		"reason", reason,
	)

	return session, nil
}

// EndImpersonation deletes an impersonation session by its token.
// The caller's original session is not affected.
func (s *Service) EndImpersonation(ctx context.Context, token string) error {
	session, err := s.sessions.Find(ctx, token)
	if err != nil {
		return err //nolint:wrapcheck // SessionStore returns typed errors (ErrSessionNotFound)
	}

	if _, isImpersonation := session.Origin.(Impersonation); !isImpersonation {
		return errorfamily.NewRejection(
			"usermgmt.impersonation.not_impersonation",
			"session is not an impersonation session",
		)
	}

	if err := s.sessions.Delete(ctx, token); err != nil {
		return errorfamily.WrapTransient(
			err, "usermgmt.impersonation.delete_failed",
			"delete impersonation session",
		)
	}

	s.logger.Info(
		"usermgmt: impersonation ended",
		"impersonated_user", session.UserID.Get(),
	)

	return nil
}
