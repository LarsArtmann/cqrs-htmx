package usermgmt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// AuthorizerFunc checks whether a user is authorized for a specific operation.
// Return an error (typically ErrForbidden) to deny access.
type AuthorizerFunc func(user *User) error

// RequireAdminRole is the default authorizer that requires the admin role.
func RequireAdminRole(user *User) error {
	if user == nil || !user.HasRole(RoleAdmin) {
		return ErrForbidden
	}
	return nil
}

// AuthHandler provides HTTP endpoints for user registration, login, logout, and identity.
type AuthHandler struct {
	service                *Service
	cookieName             string
	secure                 bool
	sessionMaxAge          int
	timeout                time.Duration
	regLimiter             *perIPRateLimiter
	importLimiter          *perIPRateLimiter
	totpLimiter            *perIPRateLimiter
	verificationLimiter    *perIPRateLimiter
	webauthnLimiter        *perIPRateLimiter
	oauthLimiter           *perIPRateLimiter
	importExportAuthorizer AuthorizerFunc
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
	// RegistrationRateLimit, if non-nil, limits the rate of POST /auth/register
	// requests. Use this to prevent credential stuffing and registration abuse.
	// The limiter is checked before the request body is decoded.
	RegistrationRateLimit RateLimitConfig
	// ImportRateLimit limits the rate of POST /auth/import requests per IP.
	ImportRateLimit RateLimitConfig
	// TOTPRateLimit limits the rate of /auth/totp/* requests per IP.
	TOTPRateLimit RateLimitConfig
	// VerificationRateLimit limits the rate of /auth/email/verify* requests per IP.
	VerificationRateLimit RateLimitConfig
	// WebAuthnRateLimit limits the rate of /auth/webauthn/* requests per IP.
	// Use this to brute-force protection on the passwordless login/registration ceremonies.
	WebAuthnRateLimit RateLimitConfig
	// OAuthRateLimit limits the rate of /auth/oauth/* requests per IP.
	OAuthRateLimit RateLimitConfig
	// ImportExportAuthorizer controls who can call /auth/import and /auth/export.
	// Defaults to RequireAdminRole (only users with the admin role).
	// Set to nil to disable authorization, or provide a custom AuthorizerFunc.
	ImportExportAuthorizer AuthorizerFunc
}

// RateLimitConfig configures per-IP rate limiting for any endpoint group.
type RateLimitConfig struct {
	// Enabled controls whether rate limiting is active.
	Enabled bool
	// MaxRequests is the maximum number of requests per Window per IP.
	MaxRequests int
	// Window is the time window for rate counting.
	Window time.Duration
}

type rateLimitEntry struct {
	count     int
	windowEnd time.Time
}

// perIPRateLimiter is a simple in-memory per-IP rate limiter.
type perIPRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	max     int
	window  time.Duration
}

func newPerIPRateLimiter(maxReq int, window time.Duration) *perIPRateLimiter {
	//nolint:exhaustruct // mu is fine as zero value
	return &perIPRateLimiter{
		entries: make(map[string]*rateLimitEntry),
		max:     maxReq,
		window:  window,
	}
}

func (rl *perIPRateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[ip]
	if !exists || now.After(entry.windowEnd) {
		rl.entries[ip] = &rateLimitEntry{
			count:     1,
			windowEnd: now.Add(rl.window),
		}
		return true
	}
	if entry.count >= rl.max {
		return false
	}
	entry.count++
	return true
}

func newLimiterFromConfig(cfg RateLimitConfig) *perIPRateLimiter {
	if cfg.Enabled && cfg.MaxRequests > 0 {
		return newPerIPRateLimiter(cfg.MaxRequests, cfg.Window)
	}
	return nil
}

func applyConfigDefaults(cfg HandlerConfig) HandlerConfig {
	if cfg.CookieName == "" {
		cfg.CookieName = defaultCookieName
	}
	if cfg.Secure == nil {
		secure := true
		cfg.Secure = &secure
	}
	if cfg.ImportExportAuthorizer == nil {
		cfg.ImportExportAuthorizer = RequireAdminRole
	}
	return cfg
}

const (
	defaultCookieName = "session_token"
	contentTypeJSON   = "application/json; charset=utf-8"
	statusKey         = "status"

	statusLoggedOut         = "logged_out"
	statusRegistered        = "registered"
	statusRemoved           = "removed"
	statusUnlinked          = "unlinked"
	statusVerified          = "verified"
	statusTOTPEnabled       = "totp_enabled"
	statusTOTPDisabled      = "totp_disabled"
	statusTOTPVerified      = "totp_verified"
	statusTOTPSetupVerified = "totp_setup_verified"
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
		service:                service,
		cookieName:             config.CookieName,
		secure:                 secure,
		sessionMaxAge:          config.SessionMaxAge,
		timeout:                config.Timeout,
		importExportAuthorizer: config.ImportExportAuthorizer,
		regLimiter:             newLimiterFromConfig(config.RegistrationRateLimit),
		importLimiter:          newLimiterFromConfig(config.ImportRateLimit),
		totpLimiter:            newLimiterFromConfig(config.TOTPRateLimit),
		verificationLimiter:    newLimiterFromConfig(config.VerificationRateLimit),
		webauthnLimiter:        newLimiterFromConfig(config.WebAuthnRateLimit),
		oauthLimiter:           newLimiterFromConfig(config.OAuthRateLimit),
	}
}

// RegisterRoutes registers the auth endpoints on the given ServeMux:
//
//	POST /auth/register              — create account (email only, no password)
//	POST /auth/webauthn/register/begin  — begin passkey registration
//	POST /auth/webauthn/register/finish — finish passkey registration (user_id via query param)
//	POST /auth/webauthn/login/begin     — begin passkey login
//	POST /auth/webauthn/login/finish    — finish passkey login (user_id via query param)
//	POST /auth/logout                    — clear session
//	GET  /auth/me                        — return current user
//	GET  /auth/credentials               — list current user's WebAuthn credentials
//	DELETE /auth/credentials/{id}        — remove a WebAuthn credential by base64url ID
//
// Verification, TOTP, and import/export routes are also registered; see
// RegisterVerificationTOTPRoutes for the full list.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.handleRegister)
	mux.HandleFunc("POST /auth/webauthn/register/begin", h.handleWebAuthnBeginRegistration)
	mux.HandleFunc("POST /auth/webauthn/register/finish", h.handleWebAuthnFinishRegistration)
	mux.HandleFunc("POST /auth/webauthn/login/begin", h.handleWebAuthnBeginLogin)
	mux.HandleFunc("POST /auth/webauthn/login/finish", h.handleWebAuthnFinishLogin)
	mux.HandleFunc("POST /auth/logout", h.handleLogout)
	mux.HandleFunc("GET /auth/me", h.handleMe)
	mux.HandleFunc("GET /auth/credentials", h.handleListCredentials)
	mux.HandleFunc("DELETE /auth/credentials/{id}", h.handleDeleteCredential)
	h.RegisterVerificationTOTPRoutes(mux)
	h.RegisterOAuth2Routes(mux)
}

const maxAuthBodySize = 1 << 20 // 1 MB

func (h *AuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if h.regLimiter != nil && !h.regLimiter.allow(r.RemoteAddr) {
		writeError(w, http.StatusTooManyRequests, "too many registration requests")
		return
	}

	ctx, cancel := h.withTimeout(r)
	defer cancel()

	var regReq RegisterRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxAuthBodySize)).Decode(&regReq); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			fmt.Errorf("%w: unmarshal register request: %w", ErrValidation, err).Error(),
		)
		return
	}
	resp, err := h.service.Register(ctx, regReq)
	if err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	h.setSessionCookie(w, resp.Session.Token)
	writeJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r, h.cookieName)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "no session")
		return
	}

	ctx, cancel := h.withTimeout(r)
	defer cancel()

	if err := h.service.Logout(ctx, token); err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}

	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{statusKey: statusLoggedOut})
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
		errors.Is(err, ErrSessionExpired),
		errors.Is(err, ErrSessionDataNotFound),
		errors.Is(err, ErrWebAuthnNotConfigured):
		return http.StatusUnauthorized
	case errors.Is(err, ErrEmailExists),
		errors.Is(err, ErrEmailAlreadyVerified),
		errors.Is(err, ErrTOTPAlreadyEnabled),
		errors.Is(err, ErrExternalAccountAlreadyLinked):
		return http.StatusConflict
	case errors.Is(err, ErrValidation),
		errors.Is(err, ErrTOTPNotEnabled),
		errors.Is(err, ErrTOTPSetupExpired),
		errors.Is(err, ErrInvalidVerificationToken):
		return http.StatusBadRequest
	case errors.Is(err, ErrAccountLocked):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrInvalidTOTPCode):
		return http.StatusUnauthorized
	case errors.Is(err, ErrTOTPNotConfigured),
		errors.Is(err, ErrEmailVerificationNotConfigured),
		errors.Is(err, ErrOAuthNotConfigured):
		return http.StatusServiceUnavailable
	case errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrSessionNotFound),
		errors.Is(err, ErrUserIDExists),
		errors.Is(err, ErrNoCredentials),
		errors.Is(err, ErrOAuthProviderNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrOAuthInvalidState),
		errors.Is(err, ErrOAuthTokenExchange):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
