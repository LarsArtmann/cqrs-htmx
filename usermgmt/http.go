package usermgmt

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// AuthorizerFunc checks whether a user is authorized for a specific operation.
// Return an error (typically ErrForbidden) to deny access.
type AuthorizerFunc func(user *User) error

// RequireAdminRole returns an AuthorizerFunc that checks the admin role
// via the membership/Authz system. Use this as the default ImportExportAuthorizer.
func RequireAdminRole(authz *Authz) AuthorizerFunc {
	return func(user *User) error {
		if user == nil {
			return ErrForbidden
		}
		roles, err := authz.RolesForUser(user.ID, NewTenantID(user.ID.Get().String()))
		if err != nil {
			return err
		}
		if slices.Contains(roles, RoleAdmin) {
			return nil
		}
		return ErrForbidden
	}
}

// AuthHandler provides HTTP endpoints for user registration, login, logout, and identity.
type AuthHandler struct {
	service                *Service
	cookieName             string
	secure                 bool
	sessionMaxAge          int
	timeout                time.Duration
	regLimiter             *cqrshtmx.RateLimiter
	importLimiter          *cqrshtmx.RateLimiter
	totpLimiter            *cqrshtmx.RateLimiter
	verificationLimiter    *cqrshtmx.RateLimiter
	webauthnLimiter        *cqrshtmx.RateLimiter
	oauthLimiter           *cqrshtmx.RateLimiter
	oauth2SuccessURL       string
	oauth2ErrorURL         string
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
	// OAuth2SuccessURL is the URL to redirect the browser to after successful
	// OAuth2 login. If empty, the callback returns JSON instead (useful for
	// HTMX/SPA flows where the consumer handles the redirect client-side).
	OAuth2SuccessURL string
	// OAuth2ErrorURL is the URL to redirect the browser to on OAuth2 login
	// failure. The error message is appended as a query parameter. If empty,
	// the callback returns a JSON error response.
	OAuth2ErrorURL string
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

func newLimiterFromConfig(cfg RateLimitConfig) *cqrshtmx.RateLimiter {
	if cfg.Enabled && cfg.MaxRequests > 0 {
		return cqrshtmx.NewRateLimiter(cqrshtmx.RateLimiterConfig{ //nolint:exhaustruct // consumer defaults
			Limit:        uint(cfg.MaxRequests),
			Window:       cfg.Window,
			KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
		})
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
		cfg.ImportExportAuthorizer = nil // set in NewAuthHandler where service is available
	}
	return cfg
}

const (
	defaultCookieName = "session_token"
	contentTypeJSON   = "application/json; charset=utf-8"
	statusKey         = "status"
	errorKey          = "error"
	codeKey           = "code"
	requestIDKey      = "request_id"

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
	if config.ImportExportAuthorizer == nil {
		config.ImportExportAuthorizer = RequireAdminRole(service.authz)
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
		oauth2SuccessURL:       config.OAuth2SuccessURL,
		oauth2ErrorURL:         config.OAuth2ErrorURL,
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
	if h.regLimiter != nil {
		if ok, _ := h.regLimiter.Check(r); !ok {
			writeError(w, http.StatusTooManyRequests, "too many registration requests")
			return
		}
	}

	ctx, cancel := h.withTimeout(r)
	defer cancel()

	var regReq RegisterRequest
	if err := json.UnmarshalRead(io.LimitReader(r.Body, maxAuthBodySize), &regReq); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf("%s: unmarshal register request: %s", ErrValidation, err),
		)
		return
	}
	resp, err := h.service.Register(ctx, regReq)
	if err != nil {
		writeDispatchError(w, r, err)
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
		writeDispatchError(w, r, err)
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
	if err := json.MarshalWrite(&buf, v); err != nil {
		http.Error(w, "json encode error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{errorKey: message})
}

// errorStatus derives the HTTP status for an error. Each usermgmt sentinel
// carries its own status (via cqrshtmx.WithHTTPStatus) where it differs from
// the error-family default, so this delegates entirely to cqrshtmx.MapError —
// no parallel switch to keep in sync with the root module.
func errorStatus(err error) int {
	return cqrshtmx.MapError(err)
}

// writeDispatchError writes an error response with the HTTP status derived from
// the error's family (via errorStatus) and includes the error's machine-readable
// code and request ID when available. Consolidates the
// writeDispatchError(w, r, err) pattern across all usermgmt HTTP handlers.
func writeDispatchError(w http.ResponseWriter, r *http.Request, err error) {
	status := errorStatus(err)
	body := map[string]string{errorKey: err.Error()}
	if code := cqrshtmx.ErrorCode(err); code != "" {
		body[codeKey] = code
	}
	if r != nil {
		if rid := cqrshtmx.RequestIDFromContext(r.Context()); !rid.IsZero() {
			body[requestIDKey] = rid.String()
		}
	}
	writeJSON(w, status, body)
}
