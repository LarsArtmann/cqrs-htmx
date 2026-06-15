package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// Logout deletes the session associated with the given token.
func (s *Service) Logout(ctx context.Context, token string) error {
	if err := s.sessions.Delete(ctx, token); err != nil {
		return event.NewTransient("internal", "logout").WithCause(err)
	}
	return nil
}

// Authenticate validates a session token and returns the associated User.
func (s *Service) Authenticate(ctx context.Context, token string) (*User, error) {
	session, err := s.sessions.Find(ctx, token)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if session.IsExpired() {
		_ = s.sessions.Delete(ctx, token)
		return nil, ErrSessionExpired
	}

	if !session.Valid(token) {
		return nil, ErrUnauthorized
	}

	user, ok := s.readModel.FindByUserID(session.UserID)
	if !ok {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// Authorize checks whether the subject can perform the action on the object
// within the given domain. Returns ErrForbidden on denial.
func (s *Service) Authorize(_ context.Context, sub, dom, obj string, act Action) error {
	return s.authz.Authorize(sub, dom, obj, act)
}
