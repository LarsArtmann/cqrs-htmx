package usermgmt

import (
	"context"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks that email and password are non-empty.
// It trims leading/trailing whitespace from Email in-place.
func (r *LoginRequest) Validate() error {
	var errs []string
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	if r.Email == "" {
		errs = append(errs, "email is required")
	}
	if r.Password == "" {
		errs = append(errs, "password is required")
	} else if len(r.Password) > maxPasswordLength {
		errs = append(errs, errMsgPasswordTooLong)
	}
	return formatValidationErrors(errs)
}

// LoginResponse contains the authenticated User and active Session.
type LoginResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

// Login validates credentials, enforces account lockout, and opens a session.
// Returns ErrInvalidCredentials, ErrAccountLocked, or ErrValidation on failure.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if s.lockout != nil && s.lockout.IsLocked(req.Email) {
		s.logger.Warn("usermgmt: login rejected — account locked", "email", req.Email)
		return nil, ErrAccountLocked
	}
	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, classifyLoginError(err)
	}

	if !user.CheckPassword(req.Password) {
		s.logger.Warn("usermgmt: login failed", "email", req.Email, "reason", "invalid_password")
		if s.lockout != nil && s.lockout.RecordFailure(req.Email) {
			s.logger.Warn("usermgmt: account locked", "email", req.Email)
		}
		return nil, ErrInvalidCredentials
	}

	if s.lockout != nil {
		s.lockout.Reset(req.Email)
	}

	session, err := s.sessions.Create(ctx, user.ID, s.sessionTTL)
	if err != nil {
		return nil, withUserIDContext(event.NewTransient("internal", "create session").WithCause(err), user.ID)
	}

	s.emit(user.ID, UserLoggedInEvent{
		Email:      user.Email,
		OccurredAt: time.Now().UTC(),
	})
	return &LoginResponse{User: user, Session: session}, nil
}

// Logout deletes the session associated with the given token.
func (s *Service) Logout(ctx context.Context, token string) error {
	if err := s.sessions.Delete(ctx, token); err != nil {
		return event.NewTransient("internal", "logout").WithCause(err)
	}
	return nil
}

// Authenticate validates a session token and returns the associated User.
// Expired or invalid tokens result in ErrSessionExpired or ErrUnauthorized.
func (s *Service) Authenticate(ctx context.Context, token string) (*User, error) {
	session, err := s.sessions.Find(ctx, token)
	if err != nil {
		return nil, ErrUnauthorized
	}

	// Proactively clean up expired sessions from the store.
	if session.IsExpired() {
		_ = s.sessions.Delete(ctx, token)
		return nil, ErrSessionExpired
	}

	if !session.Valid(token) {
		return nil, ErrUnauthorized
	}

	user, err := s.users.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// Authorize checks whether the subject can perform the action on the object
// within the given domain. Returns ErrForbidden on denial.
func (s *Service) Authorize(_ context.Context, sub, dom, obj string, act Action) error {
	return s.authz.Authorize(sub, dom, obj, act)
}
