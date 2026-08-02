package usermgmt

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
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
	return errorfamily.NewRejection("validation", strings.Join(errs, "; ")).WithCause(ErrValidation)
}

func withUserIDContext(err *event.Error, userID UserID) *event.Error {
	if err == nil || userID.IsZero() {
		return err
	}
	return err.WithContext("user_id", userID.Get().String())
}

func (r *RegisterRequest) Validate() error {
	var errs []string
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	// ID is optional — when empty, the server auto-generates a ULID in Register().
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
	if req.ID.IsZero() {
		req.ID = id.NewUserID()
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if _, exists := s.readModel.FindByEmail(req.Email); exists {
		return nil, withUserIDContext(
			errorfamily.NewRejection("usermgmt.email_exists", "email already registered").
				WithCause(ErrEmailExists), req.ID,
		)
	}

	aggID, err := aggIDFromUser(req.ID)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "usermgmt.service.userid_conversion_failed", "convert userID")
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
			errorfamily.NewTransient("usermgmt.user.read_model_missing", "user not in read model after register"), req.ID,
		)
	}

	session, err := s.createSession(ctx, req.ID)
	if err != nil {
		return nil, withUserIDContext(
			errorfamily.NewTransient("usermgmt.session.create", "create session").WithCause(err), req.ID,
		)
	}

	return &RegisterResponse{User: user, Session: session}, nil
}

func (s *Service) classifyDispatchError(err error, userID UserID, kv ...string) error {
	var classified *event.Error
	switch errorfamily.Classify(err) {
	case event.Conflict:
		classified = withUserIDContext(
			errorfamily.NewRejection("usermgmt.user_id_exists", "user ID already exists").
				WithCause(ErrUserIDExists), userID,
		)
	case event.Rejection:
		if len(kv) == 0 {
			return err
		}
		ee, ok := errors.AsType[*event.Error](err)
		if !ok {
			return err
		}
		classified = withUserIDContext(ee, userID)
	case event.Transient, event.Corruption, event.Infrastructure, errorfamily.Orchestration:
		classified = withUserIDContext(
			errorfamily.NewTransient("usermgmt.command.dispatch", "dispatch command").WithCause(err), userID,
		)
	default:
		// Safety net for future errorfamily classifications or unclassified errors.
		// Without this, classified would be nil and .WithContext below would panic.
		classified = withUserIDContext(
			errorfamily.NewTransient("usermgmt.command.dispatch", "dispatch command").WithCause(err), userID,
		)
	}
	for i := 0; i+1 < len(kv); i += 2 {
		classified = classified.WithContext(kv[i], kv[i+1])
	}
	return classified
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
		return nil, errorfamily.NewTransient("usermgmt.session.create", "create session").WithCause(err)
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, errorfamily.NewTransient("usermgmt.session.store", "store session").WithCause(err)
	}
	return session, nil
}
