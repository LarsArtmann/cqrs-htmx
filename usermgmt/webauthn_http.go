package usermgmt

import (
	"net/http"
)

type webauthnBeginRegRequest struct {
	UserID string `json:"user_id"`
}

// requireUserIDFromQuery extracts the user_id query parameter and writes a
// 400 Bad Request if it is missing. Returns the parsed UserID and ok=true on
// success; on failure it has written the response and ok=false.
func (h *AuthHandler) requireUserIDFromQuery(w http.ResponseWriter, r *http.Request) (UserID, bool) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id query parameter is required")
		return UserID{}, false
	}
	return NewUserID(userID), true
}

// requireUserIDWithWebAuthnRateLimit rate-limits a WebAuthn ceremony and
// extracts the user_id query parameter. It writes the error response (429 or
// 400) on failure and returns ok=false; on success it returns the parsed
// UserID. Used by WebAuthn finish ceremonies that need a userID from the query.
func (h *AuthHandler) requireUserIDWithWebAuthnRateLimit(w http.ResponseWriter, r *http.Request) (UserID, bool) {
	if !h.checkRateLimit(w, r, h.webauthnLimiter, "too many WebAuthn requests") {
		return UserID{}, false
	}
	return h.requireUserIDFromQuery(w, r)
}

func (h *AuthHandler) handleWebAuthnBeginRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.webauthnLimiter, "too many WebAuthn requests") {
		return
	}
	var req webauthnBeginRegRequest
	if !h.decodeAuthJSON(w, r, &req) {
		return
	}

	resp, err := h.service.BeginRegistration(r.Context(), NewUserID(req.UserID))
	if err != nil {
		writeDispatchError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) handleWebAuthnFinishRegistration(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserIDWithWebAuthnRateLimit(w, r)
	if !ok {
		return
	}
	credentialName := r.URL.Query().Get("credential_name")

	err := h.service.FinishRegistration(r.Context(), userID, r, credentialName)
	if err != nil {
		writeDispatchError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{statusKey: statusRegistered})
}

type webauthnBeginLoginRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) handleWebAuthnBeginLogin(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.webauthnLimiter, "too many WebAuthn requests") {
		return
	}
	var req webauthnBeginLoginRequest
	if !h.decodeAuthJSON(w, r, &req) {
		return
	}

	resp, err := h.service.BeginLogin(r.Context(), req.Email)
	if err != nil {
		writeDispatchError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) handleWebAuthnFinishLogin(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserIDWithWebAuthnRateLimit(w, r)
	if !ok {
		return
	}

	resp, err := h.service.FinishLogin(r.Context(), userID, r)
	if err != nil {
		writeDispatchError(w, r, err)
		return
	}

	h.setSessionCookie(w, resp.Session.Token)
	writeJSON(w, http.StatusOK, resp)
}
