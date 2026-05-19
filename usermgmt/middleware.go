package usermgmt

import (
	"context"
	"net/http"
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

func SessionMiddleware(service *Service, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r, cookieName)
			if token != "" {
				if user, err := service.Authenticate(token); err == nil {
					r = r.WithContext(WithUser(r.Context(), user))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request, cookieName string) string {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}
