package cqrshtmx

import (
	"fmt"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
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
	return func(config *handlerConfig) {
		config.resource = resource
		config.action = action
		config.authMode = authAuthorized
	}
}

// RequireAuth returns a HandlerOption that requires an authenticated user
// without checking specific Casbin permissions.
func RequireAuth() HandlerOption {
	return func(config *handlerConfig) {
		config.authMode = authRequired
	}
}

// RequestGuardFunc is a custom authorization guard that runs after decode but
// before dispatch. It receives the request and the decoded command/query.
// Return an error to abort dispatch (the error is passed to the error handler).
// Return nil to proceed.
//
// Unlike Authorize (which requires Casbin), RequestGuard supports any auth model:
// cookie-based ownership checks, JWT claims, API keys, etc.
type RequestGuardFunc func(r *http.Request, cmdOrQuery any) error

// RequestGuard returns a HandlerOption that runs a custom authorization guard
// after the body is decoded but before dispatch. The guard receives the decoded
// command or query (as `any` — type-assert in your function).
//
// Use this for auth models that don't fit the Casbin Enforcer pattern (e.g.,
// "does this player own this game?"):
//
//	app.Command("DeleteGame",
//	    cqrshtmx.DecodeJSONWithRequest(decodeGameCmd),
//	    cqrshtmx.RequestGuard(func(r *http.Request, cmd any) error {
//	        gc := cmd.(*deleteGameCmd)
//	        if !ownsGame(extractPlayerID(r), gc.GameID) {
//	            return ErrForbidden
//	        }
//	        return nil
//	    }),
//	)
func RequestGuard(guard RequestGuardFunc) HandlerOption {
	return func(config *handlerConfig) {
		config.requestGuard = guard
	}
}

// Enforce checks authorization policy for the given subject, resource, and action.
// Returns ErrForbidden if the policy denies the request.
// Returns ErrEnforcerNil if the enforcer is nil.
func Enforce(enforcer Enforcer, subject, resource, action string) error {
	if enforcer == nil {
		return errorfamily.NewInfrastructure("enforcer_nil",
			fmt.Sprintf("enforcer is nil for subject=%s resource=%s action=%s",
				subject, resource, action)).WithCause(ErrEnforcerNil)
	}

	ok, err := enforcer.Enforce(subject, resource, action)
	if err != nil {
		return errorfamily.NewTransient("enforce_failed",
			fmt.Sprintf("casbin enforce failed for subject=%s resource=%s action=%s",
				subject, resource, action)).WithCause(err)
	}

	if !ok {
		return errorfamily.NewRejection("forbidden",
			fmt.Sprintf("subject=%s resource=%s action=%s", subject, resource, action)).
			WithCause(ErrForbidden)
	}

	return nil
}

func (a *App) executeAuthorization(r *http.Request, config *handlerConfig) error {
	if config.authMode == authNone {
		return nil
	}

	userID := UserIDFromContext(r.Context())
	if userID.IsZero() {
		if config.authMode == authAuthorized {
			return errorfamily.NewRejection("unauthorized",
				fmt.Sprintf("%s/%s", config.resource, config.action)).WithCause(ErrUnauthorized)
		}

		return ErrUnauthorized
	}

	if config.authMode == authAuthorized {
		return Enforce(a.enforcer, userID.String(), config.resource, config.action)
	}

	return nil
}

func handleUnauthorized(w http.ResponseWriter, r *http.Request, resource, action, redirect string) {
	authErr := errorfamily.NewRejection("unauthorized",
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
