package usermgmt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AuthHandler provides HTTP endpoints for user registration, login, logout, and identity.
type AuthHandler struct {
	service       *Service
	cookieName    string
	secure        bool
	sessionMaxAge int
	timeout       time.Duration
}

// HandlerConfig controls cookie and session settings for AuthHandler.
// When passed to NewAuthHandler, all fields replace the defaults.
// Secure defaults to true. To set false, pass Secure: ptrBool(false).
type HandlerConfig struct {
	// CookieName is the session cookie name. Defaults to "session_token".
	CookieName string
	// Secure sets the cookie Secure flag. Defaults to true when nil.
	// Use ptrBool(false) to explicitly disable.
	Secure *bool
	// SessionMaxAge sets the Max-Age for session cookies in seconds.
	// Zero defaults to 86400 (24 hours).
	SessionMaxAge int
	// Timeout sets a maximum execution time for auth endpoint handlers.
	// Zero means no timeout (default).
	Timeout time.Duration
}

// PtrBool returns a pointer to the given bool value.
// Deprecated: Use new(bool) for false, or a local variable for true.
func PtrBool(v bool) *bool { return &v }

func applyConfigDefaults(cfg HandlerConfig) HandlerConfig {
	secure := true
	result := HandlerConfig{
		CookieName: defaultCookieName,
		Secure:     &secure,
	}
	if cfg.CookieName != "" {
		result.CookieName = cfg.CookieName
	}
	if cfg.Secure != nil {
		result.Secure = cfg.Secure
	}
	if cfg.SessionMaxAge != 0 {
		result.SessionMaxAge = cfg.SessionMaxAge
	}
	if cfg.Timeout != 0 {
		result.Timeout = cfg.Timeout
	}
	return result
}

const (
	defaultCookieName = "session_token"
	contentTypeJSON   = "application/json; charset=utf-8"
)

// NewAuthHandler creates an AuthHandler for the given Service with optional config.
func NewAuthHandler(service *Service, cfg ...HandlerConfig) *AuthHandler {
	config := applyConfigDefaults(HandlerConfig{})
	if len(cfg) > 0 {
		config = applyConfigDefaults(cfg[0])
	}
	secure := true
	if config.Secure != nil {
		secure = *config.Secure
	}
	return &AuthHandler{
		service:       service,
		cookieName:    config.CookieName,
		secure:        secure,
		sessionMaxAge: config.SessionMaxAge,
		timeout:       config.Timeout,
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
				return nil, fmt.Errorf("unmarshal register request: %w", err)
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
				return nil, fmt.Errorf("unmarshal login request: %w", err)
			}
			return h.service.Login(ctx, loginReq)
		},
		http.StatusOK,
	)
}

const maxAuthBodySize = 1 << 20 // 1 MB

func (h *AuthHandler) handleAuthEndpoint(
	w http.ResponseWriter,
	r *http.Request,
	process func(context.Context, json.RawMessage) (*LoginResponse, error),
	successStatus int,
) {
	ctx := r.Context()
	if h.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}

	var raw json.RawMessage
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodySize)).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := process(ctx, raw)
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

	ctx := r.Context()
	if h.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}

	if err := h.service.Logout(ctx, token); err != nil {
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
	//nolint:gosec,exhaustruct // G124: Secure configurable; zero-valued Cookie fields intentional
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
	//nolint:gosec,exhaustruct // clearing cookie; zero-valued fields intentional
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
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		http.Error(w, "json encode error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
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
		errors.Is(err, ErrSessionNotFound),
		errors.Is(err, ErrUserIDExists):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
