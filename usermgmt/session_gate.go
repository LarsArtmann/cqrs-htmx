package usermgmt

import "net/http"

// sessionUserID returns the authenticated user from the request context,
// writing a 401 Unauthorized response when no session is present.
// ok=false means the response has been written and the caller must return.
func (h *AuthHandler) sessionUserID(w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := UserFromContext(r.Context())
	if !ok || user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return nil, false
	}
	return user, true
}

// requireSessionUserTarget enforces the owner-match rule for credential
// ceremonies: the request must carry a session AND the requested target user
// must be the session user themselves.
//
// Responses: 401 without a session, 400 when target is empty, 403 when the
// target is a different user. On success it returns the session user's ID,
// which by construction equals the requested target.
//
// There is no HTTP-level opt-out: first-user bootstrap works because
// POST /auth/register issues the session cookie before any ceremony, and
// headless or administrative enrollment uses the service-level API
// (Service.BeginRegistration / FinishRegistration) directly.
func (h *AuthHandler) requireSessionUserTarget(
	w http.ResponseWriter, r *http.Request, target string,
) (UserID, bool) {
	user, ok := h.sessionUserID(w, r)
	if !ok {
		return UserID{}, false
	}
	if target == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return UserID{}, false
	}
	if target != user.ID.Get().String() {
		writeError(w, http.StatusForbidden, "cannot enroll credentials for another user")
		return UserID{}, false
	}
	return user.ID, true
}

// withSession wraps a handler with an enrich-only session pass so handlers
// that read the authenticated user from the request context (currentUser,
// UserFromContext) work even when RegisterRoutes is mounted without an
// external session middleware. Requests without a valid session pass through
// unauthenticated — the wrapped handlers keep their fail-closed 401 behavior.
// When an external NewSessionMiddleware already wrapped the mux, the second
// pass is redundant but harmless (it resolves the same user).
func (h *AuthHandler) withSession(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.sessionMiddleware(handler).ServeHTTP(w, r)
	}
}
