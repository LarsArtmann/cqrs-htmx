package usermgmt

import (
	"context"
	"strconv"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

type RegisterRequest struct {
	ID          UserID `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func formatValidationErrors(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return event.NewRejection("validation", strings.Join(errs, "; ")).WithCause(ErrValidation)
}

func withUserIDContext(err *event.Error, userID UserID) *event.Error {
	if err == nil || userID.IsZero() {
		return err
	}
	return err.WithContext("user_id", userID.Get())
}

func (r *RegisterRequest) Validate() error {
	var errs []string
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	if r.ID.IsZero() {
		errs = append(errs, "id is required")
	}
	email, err := ParseEmail(r.Email)
	if err != nil {
		errs = append(errs, "invalid email")
	} else {
		r.Email = email.String()
	}
	if len(r.DisplayName) > maxDisplayNameLength {
		errs = append(errs,
			"display name must be under "+strconv.Itoa(maxDisplayNameLength)+" characters")
	}
	return formatValidationErrors(errs)
}

// AuthResult is the common result of any authentication or registration flow
// that establishes a session: the authenticated [User] and the [Session] the
// caller should use for subsequent requests.
//
// [Service.Register], [Service.FinishLogin], and [Service.FinishOAuthLogin]
// all return this type. The per-flow names below are aliases kept for
// call-site clarity and backwards compatibility.
type AuthResult struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

// RegisterResponse is an alias for [AuthResult], returned by [Service.Register].
type RegisterResponse = AuthResult

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if _, exists := s.readModel.FindByEmail(req.Email); exists {
		return nil, withUserIDContext(
			event.NewRejection("usermgmt.email_exists", "email already registered").
				WithCause(ErrEmailExists), req.ID,
		)
	}

	aggID, err := aggIDFromUser(req.ID)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "usermgmt.service.userid_conversion_failed", "convert userID")
	}
	err = s.dispatcher.Dispatch(ctx, NewRegisterUserCmd(
		aggID, req.Email, req.DisplayName, []Role{RoleViewer, RoleUser},
	))
	if err != nil {
		return nil, s.classifyDispatchError(err, req.ID)
	}

	user, ok := s.readModel.FindByID(aggID)
	if !ok {
		return nil, withUserIDContext(
			event.NewTransient("internal", "user not in read model after register"), req.ID,
		)
	}

	session, err := s.createSession(ctx, req.ID)
	if err != nil {
		return nil, withUserIDContext(
			event.NewTransient("internal", "create session").WithCause(err), req.ID,
		)
	}

	return &RegisterResponse{User: user, Session: session}, nil
}

func (s *Service) classifyDispatchError(err error, userID UserID) error {
	switch event.Classify(err) {
	case event.Conflict:
		return withUserIDContext(
			event.NewRejection("usermgmt.user_id_exists", "user ID already exists").
				WithCause(ErrUserIDExists), userID,
		)
	case event.Rejection:
		return err
	default:
		return withUserIDContext(
			event.NewTransient("internal", "dispatch command").WithCause(err), userID,
		)
	}
}

func (s *Service) logAuth(event string, userID UserID, attrs ...any) {
	args := make([]any, 0, 4+len(attrs))
	args = append(args, "event", event, "user_id", userID)
	args = append(args, attrs...)
	s.logger.Info("usermgmt: "+event, args...)
}

func (s *Service) revokeSessionsBestEffort(ctx context.Context, userID UserID, failureReason string) {
	if err := s.sessions.DeleteByUserID(ctx, userID); err != nil {
		s.logger.Warn("usermgmt: "+failureReason, "user_id", userID, "error", err)
	}
}

// createSession creates a new session for the user and persists it to the store.
func (s *Service) createSession(ctx context.Context, userID UserID) (*Session, error) {
	session, err := NewSession(userID, s.sessionTTL)
	if err != nil {
		return nil, event.NewTransient("internal", "create session").WithCause(err)
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, event.NewTransient("internal", "store session").WithCause(err)
	}
	return session, nil
}
