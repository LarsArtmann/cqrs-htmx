package usermgmt

import (
	"net/http"
	"net/url"
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
		writeDispatchError(w, r, err)
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
		h.oauth2Error(w, http.StatusBadRequest, "provider is required")
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		h.oauth2Error(w, http.StatusBadRequest, "code and state query parameters are required")
		return
	}

	ctx, cancel := h.withTimeout(r)
	defer cancel()

	resp, err := h.service.FinishOAuthLogin(ctx, provider, code, state)
	if err != nil {
		h.oauth2Error(w, errorStatus(err), err.Error())
		return
	}

	h.setSessionCookie(w, resp.Session.Token)
	if h.oauth2SuccessURL != "" {
		http.Redirect(w, r, h.oauth2SuccessURL, http.StatusFound)
		return
	}
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
		writeDispatchError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{statusKey: statusUnlinked})
}

// oauth2Error writes an OAuth2 error response. When oauth2ErrorURL is configured,
// it redirects the browser with the error as a query parameter. Otherwise it
// returns a JSON error response (for API/HTMX consumers).
func (h *AuthHandler) oauth2Error(w http.ResponseWriter, status int, message string) {
	if h.oauth2ErrorURL != "" {
		redirectURL := h.oauth2ErrorURL + "?error=" + url.QueryEscape(message)
		http.Redirect(w, nil, redirectURL, http.StatusFound)
		return
	}
	writeError(w, status, message)
}
