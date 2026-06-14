package usermgmt

import (
	"context"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

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

// withUserIDContext annotates an *event.Error with the affected user ID.
// The user ID is the only identifying context added: it is already known
// to the service and is required for log correlation. Email, display name,
// and other PII are deliberately NOT included to avoid leaking data through
// error chains. The returned error is the same pointer if err is nil or
// is not an *event.Error.
func withUserIDContext(err *event.Error, userID UserID) *event.Error {
	if err == nil || userID.IsZero() {
		return err
	}
	return err.WithContext("user_id", userID.Get())
}

// Validate checks the RegisterRequest fields and returns ErrValidation with
// a joined list of problems if any field is invalid.
// It trims leading/trailing whitespace from Email and DisplayName in-place.
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
		errStr := err.Error()
		errs = append(errs, errStr)
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

// Register validates the request, creates the user, assigns the "user" role,
// and opens a session. Partial failures are compensated: if role assignment or
// session creation fails after the user is created, the user and role are rolled back.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	user := NewUser(req.ID, req.Email, req.DisplayName)
	if err := user.SetPasswordWithCost(req.Password, s.bcryptCost); err != nil {
		return nil, withUserIDContext(event.NewTransient("internal", "set password").WithCause(err), user.ID)
	}

	user.AddRole(RoleUser)

	if err := s.users.Create(ctx, user); err != nil {
		return nil, withUserIDContext(event.NewTransient("internal", "create user").WithCause(err), user.ID)
	}

	policy := GroupPolicy{
		Subject: user.ID.Get(), Role: RoleUser, Domain: user.ID.Get(),
	}
	if err := s.authz.AddGroupPolicy(policy); err != nil {
		if delErr := s.users.Delete(ctx, user.ID); delErr != nil {
			s.logAuth("register_rollback_delete_failed", user.ID, "rollback_error", delErr)
		}
		return nil, withUserIDContext(event.NewTransient("internal", "assign role").WithCause(err), user.ID)
	}

	session, err := s.sessions.Create(ctx, user.ID, s.sessionTTL)
	if err != nil {
		if rmErr := s.authz.RemoveGroupPolicy(policy); rmErr != nil {
			s.logAuth("register_rollback_policy_failed", user.ID, "rollback_error", rmErr)
		}
		if delErr := s.users.Delete(ctx, user.ID); delErr != nil {
			s.logAuth("register_rollback_delete_failed", user.ID, "rollback_error", delErr)
		}
		return nil, withUserIDContext(event.NewTransient("internal", "create session").WithCause(err), user.ID)
	}

	s.emit(user.ID, UserRegisteredEvent{
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Roles:       append([]Role(nil), user.Roles...),
		OccurredAt:  time.Now().UTC(),
	})
	return &RegisterResponse{User: user, Session: session}, nil
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
