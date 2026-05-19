package usermgmt

import (
	"encoding/json"
	"errors"
	"net/http"
)

type AuthHandlers struct {
	service      *Service
	cookieName   string
	secure       bool
	sessionMaxAge int
}

type HandlerConfig struct {
	CookieName string
	Secure     bool
	// SessionMaxAge sets the Max-Age for session cookies in seconds.
	// Zero defaults to 86400 (24 hours).
	SessionMaxAge int
}

func NewAuthHandlers(service *Service, cfg ...HandlerConfig) *AuthHandlers {
	config := HandlerConfig{
		CookieName: "session_token",
		Secure:     true,
	}
	if len(cfg) > 0 {
		if cfg[0].CookieName != "" {
			config.CookieName = cfg[0].CookieName
		}
		config.Secure = cfg[0].Secure
	}
	return &AuthHandlers{
		service:      service,
		cookieName:   config.CookieName,
		secure:       config.Secure,
		sessionMaxAge: config.SessionMaxAge,
	}
}

func (h *AuthHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.handleRegister)
	mux.HandleFunc("POST /auth/login", h.handleLogin)
	mux.HandleFunc("POST /auth/logout", h.handleLogout)
	mux.HandleFunc("GET /auth/me", h.handleMe)
}

func (h *AuthHandlers) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}

	h.setSessionCookie(w, resp.Session.Token)
	writeJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}

	h.setSessionCookie(w, resp.Session.Token)
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r, h.cookieName)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "no session")
		return
	}

	if err := h.service.Logout(r.Context(), token); err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}

	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (h *AuthHandlers) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok || user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandlers) setSessionCookie(w http.ResponseWriter, token string) {
	maxAge := h.sessionMaxAge
	if maxAge <= 0 {
		maxAge = int(defaultSessionTTL.Seconds())
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	})
}

func (h *AuthHandlers) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func errorStatus(err error) int {
	switch {
	case errors.Is(err, ErrInvalidCredentials),
		errors.Is(err, ErrUnauthorized),
		errors.Is(err, ErrSessionExpired):
		return http.StatusUnauthorized
	case errors.Is(err, ErrEmailExists):
		return http.StatusConflict
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrSessionNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
