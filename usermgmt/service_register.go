package usermgmt

import (
	"context"
	"fmt"
	"net/mail"
	"strconv"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"golang.org/x/crypto/bcrypt"
)

// RegisterRequest contains the fields required to create a new user account.
type RegisterRequest struct {
	ID          UserID `json:"id"`
	Email       string `json:"email"`
	Password    string `json:"password"`
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

// Validate checks the RegisterRequest fields and returns ErrValidation with
// a joined list of problems if any field is invalid.
func (r *RegisterRequest) Validate() error {
	var errs []string
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	if r.ID.IsZero() {
		errs = append(errs, "id is required")
	}
	if _, err := mail.ParseAddress(r.Email); err != nil {
		errs = append(errs, "invalid email")
	}
	if err := validatePassword(r.Password); err != nil {
		errs = append(errs, err.Error())
	}
	if len(r.DisplayName) > maxDisplayNameLength {
		errs = append(errs,
			"display name must be under "+strconv.Itoa(maxDisplayNameLength)+" characters")
	}
	return formatValidationErrors(errs)
}

// RegisterResponse contains the newly created User and active Session.
type RegisterResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

// Register validates the request, hashes the password, dispatches a RegisterUser command,
// waits for the read model to update (read-your-writes consistency via MemoryBus),
// and creates a session.
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

	hash, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, withUserIDContext(
			event.NewTransient("internal", "hash password").WithCause(err), req.ID,
		)
	}

	aggID := aggIDFromUser(req.ID)
	err = s.dispatcher.Dispatch(ctx, NewRegisterUserCmd(
		aggID, req.Email, req.DisplayName, hash, []Role{RoleViewer, RoleUser},
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

	session, err := s.sessions.Create(ctx, req.ID, s.sessionTTL)
	if err != nil {
		return nil, withUserIDContext(
			event.NewTransient("internal", "create session").WithCause(err), req.ID,
		)
	}

	s.emit(req.ID, UserRegisteredEvent{
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Roles:       append([]Role(nil), user.Roles...),
		OccurredAt:  nowUTC(),
	})

	return &RegisterResponse{User: user, Session: session}, nil
}

func (s *Service) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return "", fmt.Errorf("cost=%d: %w", s.bcryptCost, err)
	}
	return string(hash), nil
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

func (s *Service) emit(userID UserID, evt any) {
	if s.eventHandler == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.Warn("usermgmt: event handler panicked", "user_id", userID, "recover", r)
		}
	}()
	s.eventHandler(userID, evt)
}
