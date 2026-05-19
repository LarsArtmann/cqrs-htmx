package cqrshtmx

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultCSRFCookieName = "csrf_token"
	defaultCSRFHeaderName = "X-CSRF-Token"
	defaultCSRFFieldName  = "csrf_token"
	defaultCSRFMaxAge     = 24 * time.Hour
	csrfTokenLength       = 32
)

// ErrCSRFInvalid is returned when a CSRF token is missing, malformed, or does not match.
var ErrCSRFInvalid = fmt.Errorf("%w: invalid or missing CSRF token", ErrForbidden)

// CSRFConfig configures CSRF protection.
//
// All fields are optional; zero values use secure defaults.
type CSRFConfig struct {
	// Secret is an optional 32-byte key for HMAC-signed tokens.
	// If nil, tokens are random bytes (double-submit cookie pattern).
	// If set, tokens are HMAC(secret, random) for additional integrity.
	Secret []byte

	// CookieName is the name of the CSRF cookie.
	// Default: "csrf_token"
	CookieName string

	// HeaderName is the request header containing the CSRF token.
	// HTMX sends this header when configured with hx-headers.
	// Default: "X-CSRF-Token"
	HeaderName string

	// FieldName is the form field name containing the CSRF token.
	// Checked as fallback when the header is not present.
	// Default: "csrf_token"
	FieldName string

	// MaxAge is the cookie max age.
	// Default: 24 hours
	MaxAge time.Duration

	// Secure sets the Secure flag on the cookie.
	// Default: true (auto-detected from request scheme)
	Secure bool

	// SameSite sets the SameSite attribute on the cookie.
	// Default: http.SameSiteLaxMode
	SameSite http.SameSite

	// Domain sets the cookie domain.
	// Default: "" (host-only cookie)
	Domain string

	// Path sets the cookie path.
	// Default: "/"
	Path string

	// ErrorHandler is called when CSRF validation fails.
	// Default: writes 403 Forbidden with plain text
	ErrorHandler ErrorHandler
}

func (c *CSRFConfig) cookieName() string {
	if c.CookieName != "" {
		return c.CookieName
	}
	return defaultCSRFCookieName
}

func (c *CSRFConfig) headerName() string {
	if c.HeaderName != "" {
		return c.HeaderName
	}
	return defaultCSRFHeaderName
}

func (c *CSRFConfig) fieldName() string {
	if c.FieldName != "" {
		return c.FieldName
	}
	return defaultCSRFFieldName
}

func (c *CSRFConfig) maxAge() time.Duration {
	if c.MaxAge > 0 {
		return c.MaxAge
	}
	return defaultCSRFMaxAge
}

func (c *CSRFConfig) path() string {
	if c.Path != "" {
		return c.Path
	}
	return "/"
}

func (c *CSRFConfig) sameSite() http.SameSite {
	switch c.SameSite {
	case http.SameSiteDefaultMode,
		http.SameSiteLaxMode,
		http.SameSiteStrictMode,
		http.SameSiteNoneMode:
		return c.SameSite
	default:
		return http.SameSiteLaxMode
	}
}

func (c *CSRFConfig) errorHandler() ErrorHandler {
	if c.ErrorHandler != nil {
		return c.ErrorHandler
	}
	return func(w http.ResponseWriter, _ *http.Request, err error) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(err.Error())) //nolint:gosec // text/plain prevents HTML rendering
	}
}

type csrfKey struct{}

// WithCSRFToken stores a CSRF token in the context.
// Used by CSRFMiddleware to make the token available to templates and handlers.
func WithCSRFToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfKey{}, token)
}

// CSRFTokenFromContext retrieves the CSRF token stored by CSRFMiddleware.
// Returns an empty string if no token is present.
func CSRFTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(csrfKey{}).(string)
	return token
}

// CSRFMiddleware returns HTTP middleware that implements double-submit cookie
// CSRF protection with HTMX awareness.
//
// For GET/HEAD/OPTIONS/TRACE requests, the middleware ensures a CSRF token
// cookie exists and stores the token in context for use in templates.
//
// For state-changing methods (POST/PUT/PATCH/DELETE), it validates that the
// request includes a matching token in either:
//   - The X-CSRF-Token header (HTMX default)
//   - A form field named "csrf_token"
//
// The token from the cookie and the token from the request must match.
//
// Usage with HTMX:
//
//	// In your HTML template, set hx-headers on <body> or <html>:
//	<body hx-headers='{"X-CSRF-Token":"{{ .CSRFToken }}"}'>
//
//	// In your Go handler, pass the token to the template:
//	token := cqrshtmx.CSRFTokenFromContext(r.Context())
//	// ... render template with token
//
//	// Or use the Response builder:
//	resp := cqrshtmx.NewResponse(w, r)
//	resp.CSRFToken(token).Apply()
//
// Middleware ordering (important):
//
//	handler := cqrshtmx.Chain(
//	    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
//	    cqrshtmx.HTMXMiddleware,
//	    app.Middleware(),
//	)(mux)
func CSRFMiddleware(cfg CSRFConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine if this is a secure request for cookie settings
			secure := cfg.Secure
			if !secure {
				secure = isSecureRequest(r)
			}

			// Get or generate the CSRF token from the cookie
			token, cookieErr := csrfTokenFromCookie(r, cfg.cookieName())
			if cookieErr != nil || token == "" {
				token = generateCSRFToken(cfg.Secret)
				setCSRFCookie(w, cfg, token, secure)
			}

			// Store token in context for templates/handlers
			ctx := WithCSRFToken(r.Context(), token)
			r = r.WithContext(ctx)

			// Validate on state-changing methods
			if isStateChangingMethod(r.Method) {
				submitted := extractSubmittedToken(r, cfg.headerName(), cfg.fieldName())
				if submitted == "" || !constantTimeCompare(token, submitted) {
					cfg.errorHandler()(w, r, ErrCSRFInvalid)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CSRFProtect returns a HandlerOption that validates CSRF tokens for a specific
// command or query handler. Use this instead of global CSRFMiddleware when you
// want CSRF protection only on specific routes.
//
// The token must be present in the request context (set by CSRFMiddleware or
// manually via WithCSRFToken). For state-changing methods, the submitted token
// is validated against the cookie.
//
// Usage:
//
//	// Apply CSRFMiddleware without validation (just sets cookie + context):
//	handler := cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{ /* no error handler */ })(mux)
//
//	// Then protect specific handlers:
//	app.Command("CreateUser",
//	    cqrshtmx.CSRFProtect(cqrshtmx.CSRFConfig{}),
//	    cqrshtmx.DecodeJSON(...),
//	)
func CSRFProtect(cfg CSRFConfig) HandlerOption {
	return func(hc *handlerConfig) {
		hc.csrfConfig = &cfg
	}
}

// executeCSRFValidation checks CSRF for per-handler protection (CSRFProtect option).
// Returns nil if no CSRF config is set on the handler or validation passes.
func executeCSRFValidation(w http.ResponseWriter, r *http.Request, cfg *handlerConfig) error {
	if cfg.csrfConfig == nil {
		return nil
	}

	if !isStateChangingMethod(r.Method) {
		return nil
	}

	token := CSRFTokenFromContext(r.Context())
	if token == "" {
		// Try to get from cookie directly if not in context
		var err error

		token, err = csrfTokenFromCookie(r, cfg.csrfConfig.cookieName())
		if err != nil || token == "" {
			cfg.csrfConfig.errorHandler()(w, r, ErrCSRFInvalid)
			return ErrCSRFInvalid
		}
	}

	submitted := extractSubmittedToken(r, cfg.csrfConfig.headerName(), cfg.csrfConfig.fieldName())
	if submitted == "" || !constantTimeCompare(token, submitted) {
		cfg.csrfConfig.errorHandler()(w, r, ErrCSRFInvalid)
		return ErrCSRFInvalid
	}

	return nil
}

// isStateChangingMethod returns true for methods that modify server state.
func isStateChangingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// isSecureRequest returns true if the request uses HTTPS.
func isSecureRequest(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// generateCSRFToken creates a cryptographically secure random CSRF token.
// If secret is provided, the token is HMAC-signed for additional integrity.
func generateCSRFToken(secret []byte) string {
	b := make([]byte, csrfTokenLength)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use a simpler approach if crypto/rand fails
		// This should never happen in practice
		panic("csrf: failed to generate random token: " + err.Error())
	}

	if len(secret) > 0 {
		// HMAC the random bytes for integrity
		b = hmacSHA256(secret, b)
	}

	return base64.RawURLEncoding.EncodeToString(b)
}

// csrfTokenFromCookie retrieves the CSRF token from the request cookie.
func csrfTokenFromCookie(r *http.Request, name string) (string, error) {
		cookie, cookieErr := r.Cookie(name)
	if cookieErr != nil {
		return "", fmt.Errorf("csrf: get cookie: %w", cookieErr)
	}
	return cookie.Value, nil
}

// setCSRFCookie writes the CSRF token cookie to the response.
func setCSRFCookie(w http.ResponseWriter, cfg CSRFConfig, token string, secure bool) {
	cookie := &http.Cookie{
		Name:        cfg.cookieName(),
		Value:       token,
		Path:        cfg.path(),
		MaxAge:      int(cfg.maxAge().Seconds()),
		HttpOnly:    false, // Must be readable by JavaScript for double-submit pattern
		Secure:      secure,
		SameSite:    cfg.sameSite(),
		Domain:      cfg.Domain,
		Quoted:      false,
		Expires:     time.Time{},
		RawExpires:  "",
		Partitioned: false,
		Raw:         "",
		Unparsed:    nil,
	}

	http.SetCookie(w, cookie)
}

// extractSubmittedToken retrieves the submitted CSRF token from request header
// or form field.
func extractSubmittedToken(r *http.Request, headerName, fieldName string) string {
	// Check header first (HTMX sends X-CSRF-Token header)
	if token := r.Header.Get(headerName); token != "" {
		return token
	}

	// Fallback to form field
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		_ = r.ParseForm()
		if token := r.PostFormValue(fieldName); token != "" {
			return token
		}
	}

	// Fallback to query parameter (for GET-based state changes, though not recommended)
	if token := r.URL.Query().Get(fieldName); token != "" {
		return token
	}

	return ""
}

// constantTimeCompare compares two strings in constant time to prevent
// timing attacks.
func constantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// hmacSHA256 computes HMAC-SHA256 of data with the given key.
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
