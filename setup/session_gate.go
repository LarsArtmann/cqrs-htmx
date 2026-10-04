package setup

import (
	"net/http"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// RequireSession blocks requests that carry no authenticated user in the
// request context, responding 401 Unauthorized (the JSON/API convention used
// by the dashboard, event feeds, and machine endpoints). Mount it after
// [Bundle.SessionMiddleware] — the session middleware only enriches the
// context, it never blocks, so this gate is what actually enforces the
// session.
func RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := usermgmt.UserFromContext(r.Context()); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireSessionRedirect returns middleware that blocks requests carrying no
// authenticated user by redirecting the browser to loginURL with
// 303 See Other — the HTML-page convention; pair it with the login page or
// your own login route. Mount it after [Bundle.SessionMiddleware].
// Already-authenticated requests pass through unchanged.
func RequireSessionRedirect(loginURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := usermgmt.UserFromContext(r.Context()); !ok {
				http.Redirect(w, r, loginURL, http.StatusSeeOther)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
