package usermgmt

import (
	"net/http"
)

type webauthnBeginRegRequest struct {
	UserID string `json:"user_id"`
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
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id query parameter is required")
		return
	}
	credentialName := r.URL.Query().Get("credential_name")

	err := h.service.FinishRegistration(r.Context(), NewUserID(userID), r, credentialName)
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
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id query parameter is required")
		return
	}

	resp, err := h.service.FinishLogin(r.Context(), NewUserID(userID), r)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}

	h.setSessionCookie(w, resp.Session.Token)
	writeJSON(w, http.StatusOK, resp)
}
