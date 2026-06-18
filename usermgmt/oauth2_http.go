package usermgmt

import (
	"net/http"
)

// RegisterOAuth2Routes registers the OAuth2 login endpoints on the given ServeMux:
//
//	GET  /auth/oauth/{provider}/begin    — redirect to provider's authorization page
//	GET  /auth/oauth/{provider}/callback — handle the OAuth2 callback redirect
//	POST /auth/oauth/{provider}/unlink   — unlink an external account (requires session)
func (h *AuthHandler) RegisterOAuth2Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/oauth/{provider}/begin", h.handleOAuth2Begin)
	mux.HandleFunc("GET /auth/oauth/{provider}/callback", h.handleOAuth2Callback)
	mux.HandleFunc("POST /auth/oauth/{provider}/unlink", h.handleOAuth2Unlink)
}

func (h *AuthHandler) handleOAuth2Begin(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.oauthLimiter, "too many OAuth2 requests") {
		return
	}
	provider := r.PathValue("provider")
	if provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	ctx, cancel := h.withTimeout(r)
	defer cancel()

	resp, err := h.service.BeginOAuthLogin(ctx, provider)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	if !h.checkRateLimit(w, r, h.oauthLimiter, "too many OAuth2 requests") {
		return
	}
	provider := r.PathValue("provider")
	if provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "code and state query parameters are required")
		return
	}

	ctx, cancel := h.withTimeout(r)
	defer cancel()

	resp, err := h.service.FinishOAuthLogin(ctx, provider, code, state)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}

	h.setSessionCookie(w, resp.Session.Token)
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) handleOAuth2Unlink(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	provider := r.PathValue("provider")
	if provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	ctx, cancel := h.withTimeout(r)
	defer cancel()

	if err := h.service.UnlinkExternalAccount(ctx, user.ID, provider); err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{statusKey: statusUnlinked})
}
