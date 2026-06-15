package usermgmt

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type webauthnBeginRegRequest struct {
	UserID string `json:"user_id"`
}

func (h *AuthHandler) handleWebAuthnBeginRegistration(w http.ResponseWriter, r *http.Request) {
	var req webauthnBeginRegRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.BeginRegistration(r.Context(), NewUserID(req.UserID))
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

type webauthnFinishRegRequest struct {
	UserID         string `json:"user_id"`
	CredentialName string `json:"credential_name"`
}

func (h *AuthHandler) handleWebAuthnFinishRegistration(w http.ResponseWriter, r *http.Request) {
	var meta webauthnFinishRegRequest
	body, err := io.ReadAll(io.LimitReader(r.Body, maxAuthBodySize))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	// Parse just the metadata (credential_name + user_id), the rest is the WebAuthn response
	// We re-create the request with the body for go-webauthn to parse
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err := json.Unmarshal(body, &meta); err != nil {
		// Non-JSON body is OK — go-webauthn reads from raw request
		meta.CredentialName = ""
	}

	err = h.service.FinishRegistration(r.Context(), NewUserID(meta.UserID), r, meta.CredentialName)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}

type webauthnBeginLoginRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) handleWebAuthnBeginLogin(w http.ResponseWriter, r *http.Request) {
	var req webauthnBeginLoginRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.BeginLogin(r.Context(), req.Email)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

type webauthnFinishLoginRequest struct {
	UserID string `json:"user_id"`
}

func (h *AuthHandler) handleWebAuthnFinishLogin(w http.ResponseWriter, r *http.Request) {
	var meta webauthnFinishLoginRequest
	body, err := io.ReadAll(io.LimitReader(r.Body, maxAuthBodySize))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	_ = json.Unmarshal(body, &meta)

	resp, err := h.service.FinishLogin(r.Context(), NewUserID(meta.UserID), r)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}

	h.setSessionCookie(w, resp.Session.Token)
	writeJSON(w, http.StatusOK, resp)
}
