package cqrshtmx

import (
	"fmt"
	"net/http"

	"github.com/cockroachdb/errors"
)

// Enforcer checks authorization policy. *casbin.Enforcer satisfies this interface.
type Enforcer interface {
	Enforce(rvals ...any) (bool, error)
}

// UserIDExtractor extracts a user ID from an HTTP request.
// Return an empty string if the user is not authenticated.
type UserIDExtractor func(r *http.Request) string

// Authorize returns a HandlerOption that checks Casbin permissions.
// The subject is extracted from the request context (via UserIDExtractor).
// If no user ID is found, the request is rejected as unauthorized.
func Authorize(resource, action string) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.resource = resource
		cfg.action = action
		cfg.authorize = true
	}
}

// RequireAuth returns a HandlerOption that requires an authenticated user
// without checking specific Casbin permissions.
func RequireAuth() HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.requireAuth = true
	}
}

// Enforce checks authorization policy for the given subject, resource, and action.
// Returns ErrForbidden if the policy denies the request.
// Returns ErrEnforcerNil if the enforcer is nil.
func Enforce(enforcer Enforcer, subject, resource, action string) error {
	if enforcer == nil {
		return fmt.Errorf("%w: %s/%s", ErrEnforcerNil, resource, action)
	}

	ok, err := enforcer.Enforce(subject, resource, action)
	if err != nil {
		return errors.Wrapf(err, "casbin enforce failed for %s/%s/%s", subject, resource, action)
	}

	if !ok {
		return fmt.Errorf("%w: %s/%s/%s", ErrForbidden, subject, resource, action)
	}

	return nil
}

// AuthorizeMiddleware returns an HTTP middleware that checks Casbin authorization.
// The subject is extracted from the context using the configured UserIDExtractor.
func AuthorizeMiddleware(
	enforcer Enforcer,
	resource, action string,
	extractor UserIDExtractor,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := extractor(r)
			if userID == "" {
				http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
				return
			}

			if err := Enforce(enforcer, userID, resource, action); err != nil {
				status := MapError(err)
				http.Error(w, err.Error(), status)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
