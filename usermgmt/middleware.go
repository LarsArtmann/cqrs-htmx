package usermgmt

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

const bearerPrefix = "Bearer "

type userContextKeyType struct{}

var userContextKey userContextKeyType //nolint:gochecknoglobals // sentinel type for context keys, standard Go pattern

type sessionOriginKeyType struct{}

var sessionOriginKey sessionOriginKeyType //nolint:gochecknoglobals // stores impersonator info from session
// sessionOriginInfo stores the actor and impersonator extracted from a session.
// Stored in context by authenticateRequest for audit-trail purposes.
type sessionOriginInfo struct {
	ActorID        string
	ImpersonatorID string // empty if not impersonating
}

// WithUser stores the authenticated User in the context.
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext retrieves the authenticated User from the context.
// Returns the user and true if found, or nil and false otherwise.
func UserFromContext(ctx context.Context) (*User, bool) {
	if ctx == nil {
		return nil, false
	}
	user, ok := ctx.Value(userContextKey).(*User)
	return user, ok
}

// UserFromContextOr returns the user from context, or the provided fallback.
func UserFromContextOr(ctx context.Context, fallback *User) *User {
	if user, ok := UserFromContext(ctx); ok {
		return user
	}
	return fallback
}

// NewSessionMiddleware returns HTTP middleware that authenticates requests
// via session cookie or Bearer token and stores the User in context.
// When the session has an Impersonation origin, the impersonator info is also
// stored for retrieval via ImpersonatorIDFromRequest.
//
// To bridge impersonation info to cqrs-htmx's context chain:
//
//	if id := usermgmt.ImpersonatorIDFromRequest(r); id != "" {
//	    ctx = cqrshtmx.WithImpersonatorID(ctx, id)
//	}
func NewSessionMiddleware(service *Service, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = authenticateRequest(service, cookieName, w, r)
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFromRequest extracts the authenticated user's ID as a string from the
// request context. Returns empty string if no user is authenticated.
//
// To bridge with cqrs-htmx's ContextEnrichmentMiddleware, wrap it:
//
//	cqrshtmx.ContextEnrichmentMiddleware(func(r *http.Request) (cqrshtmx.UserID, error) {
//	    return cqrshtmx.MustParseUserID(usermgmt.UserIDFromRequest(r)), nil
//	})
func UserIDFromRequest(r *http.Request) string {
	if user, ok := UserFromContext(r.Context()); ok && user != nil {
		return user.ID.Get().String()
	}
	return ""
}

// ImpersonatorIDFromRequest extracts the impersonator's user ID from the
// request context. Returns empty string if the request is not an impersonation
// session. Bridge to cqrs-htmx's context chain as shown in NewSessionMiddleware docs.
func ImpersonatorIDFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	info, ok := r.Context().Value(sessionOriginKey).(*sessionOriginInfo)
	if !ok || info == nil {
		return ""
	}
	return info.ImpersonatorID
}

func extractToken(r *http.Request, cookieName string) string {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}
	auth := r.Header.Get("Authorization")
	if token, ok := strings.CutPrefix(auth, bearerPrefix); ok {
		return token
	}
	return ""
}

// authenticateRequest authenticates the request via session token and enriches
// the context with User and session origin info (actor + impersonator).
func authenticateRequest(service *Service, cookieName string, _ http.ResponseWriter, r *http.Request) *http.Request {
	token := extractToken(r, cookieName)
	if token == "" {
		return r
	}
	user, err := service.Authenticate(r.Context(), token)
	if err != nil {
		slog.DebugContext(r.Context(), "session authentication failed",
			"error", err, "cookie_name", cookieName)
		return r
	}
	ctx := WithUser(r.Context(), user)
	if session, err := service.sessions.Find(r.Context(), token); err == nil {
		info := &sessionOriginInfo{ //nolint:exhaustruct // ImpersonatorID set conditionally below
			ActorID: session.ActorID.String(),
		}
		if imp, ok := session.Origin.(Impersonation); ok {
			info.ImpersonatorID = imp.By.String()
		}
		ctx = context.WithValue(ctx, sessionOriginKey, info)
	}
	return r.WithContext(ctx)
}
