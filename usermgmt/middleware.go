package usermgmt

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userContextKey contextKey = "usermgmt:user"

func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (*User, bool) {
	if ctx == nil {
		return nil, false
	}
	user, ok := ctx.Value(userContextKey).(*User)
	return user, ok
}

func UserFromContextOr(ctx context.Context, fallback *User) *User {
	if user, ok := UserFromContext(ctx); ok {
		return user
	}
	return fallback
}

func NewSessionMiddleware(service *Service, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r, cookieName)
			if token != "" {
				if user, err := service.Authenticate(r.Context(), token); err == nil {
					r = r.WithContext(WithUser(r.Context(), user))
				}
			}
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
		return user.ID.String()
	}
	return ""
}

func extractToken(r *http.Request, cookieName string) string {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}
	auth := r.Header.Get("Authorization")
	if token, ok := strings.CutPrefix(auth, "Bearer "); ok {
		return token
	}
	return ""
}
