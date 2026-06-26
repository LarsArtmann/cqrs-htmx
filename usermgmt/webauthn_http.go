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
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) handleWebAuthnFinishRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.webauthnLimiter, "too many WebAuthn requests") {
		return
	}
	userID, ok := h.requireUserIDFromQuery(w, r)
	if !ok {
		return
	}
	credentialName := r.URL.Query().Get("credential_name")

	err := h.service.FinishRegistration(r.Context(), userID, r, credentialName)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
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
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) handleWebAuthnFinishLogin(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.webauthnLimiter, "too many WebAuthn requests") {
		return
	}
	userID, ok := h.requireUserIDFromQuery(w, r)
	if !ok {
		return
	}

	resp, err := h.service.FinishLogin(r.Context(), userID, r)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}

	h.setSessionCookie(w, resp.Session.Token)
	writeJSON(w, http.StatusOK, resp)
}
