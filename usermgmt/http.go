package usermgmt

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cockroachdb/errors"
)

// AuthHandler provides HTTP endpoints for user registration, login, logout, and identity.
type AuthHandler struct {
	service       *Service
	cookieName    string
	secure        bool
	sessionMaxAge int
}

// HandlerConfig controls cookie and session settings for AuthHandler.
type HandlerConfig struct {
	// CookieName is the session cookie name. Defaults to "session_token".
	CookieName string
	// Secure sets the cookie Secure flag. Defaults to true.
	Secure bool
	// SessionMaxAge sets the Max-Age for session cookies in seconds.
	// Zero defaults to 86400 (24 hours).
	SessionMaxAge int
}

// NewAuthHandler creates an AuthHandler for the given Service with optional config.
func NewAuthHandler(service *Service, cfg ...HandlerConfig) *AuthHandler {
	config := HandlerConfig{
		CookieName: "session_token",
		Secure:     true,
	}
	if len(cfg) > 0 {
		if cfg[0].CookieName != "" {
			config.CookieName = cfg[0].CookieName
		}
		config.Secure = cfg[0].Secure
		config.SessionMaxAge = cfg[0].SessionMaxAge
	}
	return &AuthHandler{
		service:       service,
		cookieName:    config.CookieName,
		secure:        config.Secure,
		sessionMaxAge: config.SessionMaxAge,
	}
}

// RegisterRoutes registers the auth endpoints on the given ServeMux:
//
//	POST /auth/register — create account
//	POST /auth/login    — authenticate
//	POST /auth/logout   — clear session
//	GET  /auth/me       — return current user
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.handleRegister)
	mux.HandleFunc("POST /auth/login", h.handleLogin)
	mux.HandleFunc("POST /auth/logout", h.handleLogout)
	mux.HandleFunc("GET /auth/me", h.handleMe)
}

func (h *AuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	h.handleAuthEndpoint(
		w,
		r,
		func(ctx context.Context, req json.RawMessage) (*LoginResponse, error) {
			var regReq RegisterRequest
			if err := json.Unmarshal(req, &regReq); err != nil {
				return nil, err
			}
			resp, err := h.service.Register(ctx, regReq)
			if err != nil {
				return nil, err
			}
			return &LoginResponse{User: resp.User, Session: resp.Session}, nil
		},
		http.StatusCreated,
	)
}

func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	h.handleAuthEndpoint(
		w,
		r,
		func(ctx context.Context, req json.RawMessage) (*LoginResponse, error) {
			var loginReq LoginRequest
			if err := json.Unmarshal(req, &loginReq); err != nil {
				return nil, err
			}
			return h.service.Login(ctx, loginReq)
		},
		http.StatusOK,
	)
}

func (h *AuthHandler) handleAuthEndpoint(
	w http.ResponseWriter,
	r *http.Request,
	process func(context.Context, json.RawMessage) (*LoginResponse, error),
	successStatus int,
) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := process(r.Context(), raw)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}

	h.setSessionCookie(w, resp.Session.Token)
	writeJSON(w, successStatus, resp)
}

func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
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

func (h *AuthHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok || user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
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

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
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
	case errors.Is(err, ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, ErrAccountLocked):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrSessionNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
