package cqrshtmx

import (
	"fmt"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// Enforcer checks authorization policy. *casbin.Enforcer satisfies this interface.
type Enforcer interface {
	Enforce(rvals ...any) (bool, error)
}

// UserIDExtractor extracts a validated user ID from an HTTP request.
// Return a zero-value UserID and a nil error if the user is not authenticated.
// Return a non-nil error if authentication data is present but invalid.
type UserIDExtractor func(r *http.Request) (UserID, error)

// Authorize returns a HandlerOption that checks Casbin permissions.
// The subject is extracted from the request context (via UserIDExtractor).
// If no user ID is found, the request is rejected as unauthorized.
func Authorize(resource, action string) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.resource = resource
		cfg.action = action
		cfg.authMode = authAuthorized
	}
}

// RequireAuth returns a HandlerOption that requires an authenticated user
// without checking specific Casbin permissions.
func RequireAuth() HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.authMode = authRequired
	}
}

// Enforce checks authorization policy for the given subject, resource, and action.
// Returns ErrForbidden if the policy denies the request.
// Returns ErrEnforcerNil if the enforcer is nil.
func Enforce(enforcer Enforcer, subject, resource, action string) error {
	if enforcer == nil {
		return event.NewInfrastructure("enforcer_nil",
			fmt.Sprintf("enforcer is nil for subject=%s resource=%s action=%s",
				subject, resource, action)).WithCause(ErrEnforcerNil)
	}

	ok, err := enforcer.Enforce(subject, resource, action)
	if err != nil {
		return event.NewTransient("enforce_failed",
			fmt.Sprintf("casbin enforce failed for subject=%s resource=%s action=%s",
				subject, resource, action)).WithCause(err)
	}

	if !ok {
		return event.NewRejection("forbidden",
			fmt.Sprintf("subject=%s resource=%s action=%s", subject, resource, action)).
			WithCause(ErrForbidden)
	}

	return nil
}

func (a *App) executeAuthorization(r *http.Request, cfg *handlerConfig) error {
	if cfg.authMode == authNone {
		return nil
	}

	userID := UserIDFromContext(r.Context())
	if userID.IsZero() {
		if cfg.authMode == authAuthorized {
			return event.NewRejection("unauthorized",
				fmt.Sprintf("%s/%s", cfg.resource, cfg.action)).WithCause(ErrUnauthorized)
		}
		return ErrUnauthorized
	}

	if cfg.authMode == authAuthorized && a.enforcer != nil {
		return Enforce(a.enforcer, userID.String(), cfg.resource, cfg.action)
	}

	return nil
}

func handleUnauthorized(w http.ResponseWriter, r *http.Request, resource, action, redirect string) {
	authErr := event.NewRejection("unauthorized",
		fmt.Sprintf("%s/%s", resource, action)).WithCause(ErrUnauthorized)
	DefaultErrorHandlerWithRedirect(w, r, authErr, redirect)
}

// AuthorizeMiddleware returns an HTTP middleware that checks Casbin authorization.
// The subject is extracted from the context using the configured UserIDExtractor.
// Auth errors are handled with HTMX awareness (HX-Redirect for auth failures).
func AuthorizeMiddleware(
	enforcer Enforcer,
	resource, action string,
	extractor UserIDExtractor,
	loginRedirect ...string,
) func(http.Handler) http.Handler {
	redirect := defaultLoginRedirect
	if len(loginRedirect) > 0 && loginRedirect[0] != "" {
		redirect = loginRedirect[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var subject string

			if userID := UserIDFromContext(r.Context()); !userID.IsZero() {
				subject = userID.String()
			} else if extractor != nil {
				uid, err := extractor(r)
				if err != nil || uid.IsZero() {
					handleUnauthorized(w, r, resource, action, redirect)
					return
				}
				subject = uid.String()
			} else {
				handleUnauthorized(w, r, resource, action, redirect)
				return
			}

			if err := Enforce(enforcer, subject, resource, action); err != nil {
				DefaultErrorHandlerWithRedirect(w, r, err, redirect)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IsAuthenticated returns true if the request has a non-zero UserID in context.
// This works with both ContextEnrichmentMiddleware and per-handler extraction.
func IsAuthenticated(r *http.Request) bool {
	return !UserIDFromContext(r.Context()).IsZero()
}
