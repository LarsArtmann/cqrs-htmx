package cqrshtmx

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gorilla/csrf"
)

const (
	defaultCSRFCookieName = "csrf_token"
	defaultCSRFHeaderName = "X-CSRF-Token"
	defaultCSRFFieldName  = "csrf_token"
	defaultCSRFMaxAge     = 24 * time.Hour
)

// ErrCSRFInvalid is returned when a CSRF token is missing, malformed, or does not match.
// Uses gorilla/csrf under the hood for token generation and validation.
var ErrCSRFInvalid = fmt.Errorf("%w: invalid or missing CSRF token", ErrForbidden)

// CSRFConfig configures CSRF protection.
//
// All fields are optional; zero values use secure defaults.
// Uses gorilla/csrf internally for token generation, masking (BREACH mitigation),
// cookie management via securecookie, and validation.
type CSRFConfig struct {
	// Secret is the 32-byte authentication key for HMAC token signing.
	// If empty, a random key is generated (not persisted across restarts).
	// For production, always provide a stable secret.
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
	// Default: true (auto-detected from request scheme if false)
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

	// TrustedOrigins configures origins allowed for cross-domain CSRF.
	// Default: nil (same-origin only)
	TrustedOrigins []string

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

func (c *CSRFConfig) sameSite() csrf.SameSiteMode {
	switch c.SameSite {
	case http.SameSiteDefaultMode:
		return csrf.SameSiteDefaultMode
	case http.SameSiteLaxMode:
		return csrf.SameSiteLaxMode
	case http.SameSiteStrictMode:
		return csrf.SameSiteStrictMode
	case http.SameSiteNoneMode:
		return csrf.SameSiteNoneMode
	default:
		return csrf.SameSiteLaxMode
	}
}

// Validate checks the CSRF configuration for common misconfigurations.
// Returns a non-nil error if the config would produce insecure or broken behavior.
// Call this in production startup code to fail fast on misconfiguration.
func (c *CSRFConfig) Validate() error {
	if len(c.Secret) == 0 {
		return fmt.Errorf(
			"%w: CSRFConfig.Secret is empty: tokens will not persist across server restarts",
			ErrCSRFConfig,
		)
	}

	if c.SameSite == http.SameSiteNoneMode && !c.Secure {
		return fmt.Errorf("%w: SameSite=None requires Secure=true", ErrCSRFConfig)
	}

	return nil
}

func (c *CSRFConfig) secret() []byte {
	if len(c.Secret) >= 32 {
		return c.Secret
	}
	if len(c.Secret) > 0 {
		slog.Warn("cqrs-htmx: CSRF secret is shorter than 32 bytes — padding with zeros",
			slog.Int("provided", len(c.Secret)),
			slog.String("hint", "use a 32-byte secret for production"),
		)
	}
	secret := make([]byte, 32)
	copy(secret, c.Secret)
	return secret
}

// buildGorillaOptions maps our CSRFConfig to gorilla/csrf options.
func buildGorillaOptions(cfg CSRFConfig) []csrf.Option {
	opts := []csrf.Option{
		csrf.CookieName(cfg.cookieName()),
		csrf.RequestHeader(cfg.headerName()),
		csrf.FieldName(cfg.fieldName()),
		csrf.MaxAge(int(cfg.maxAge().Seconds())),
		csrf.Path(cfg.path()),
		csrf.Secure(cfg.Secure),
		csrf.HttpOnly(false), // Must be readable by JavaScript for double-submit pattern
		csrf.SameSite(cfg.sameSite()),
	}

	if cfg.Domain != "" {
		opts = append(opts, csrf.Domain(cfg.Domain))
	}

	if len(cfg.TrustedOrigins) > 0 {
		opts = append(opts, csrf.TrustedOrigins(cfg.TrustedOrigins))
	}

	if cfg.ErrorHandler != nil {
		opts = append(
			opts,
			csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cfg.ErrorHandler(w, r, ErrCSRFInvalid)
			})),
		)
	}

	return opts
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

func csrfTokenFromRequest(r *http.Request) string {
	if token := csrf.Token(r); token != "" {
		return token
	}
	return CSRFTokenFromContext(r.Context())
}

// RotateCSRFToken invalidates the current CSRF cookie, forcing a new token
// to be generated on the next request. Call this on login/logout to prevent
// CSRF fixation attacks.
func RotateCSRFToken(w http.ResponseWriter, cfg CSRFConfig) {
	//nolint:gosec,exhaustruct // HttpOnly=false required for double-submit; http.Cookie has many optional fields
	cookie := &http.Cookie{
		Name:     cfg.cookieName(),
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Now().Add(-time.Hour),
		Path:     cfg.path(),
		Domain:   cfg.Domain,
		Secure:   cfg.Secure,
		HttpOnly: false,
		SameSite: cfg.SameSite,
	}
	http.SetCookie(w, cookie)
}

// CSRFMiddleware returns HTTP middleware that implements double-submit cookie
// CSRF protection with HTMX awareness.
//
// Uses gorilla/csrf internally for:
//   - Cryptographically secure token generation
//   - Per-request token masking (BREACH attack mitigation)
//   - Cookie integrity via securecookie
//   - Referer validation for HTTPS requests
//   - Trusted origins support for cross-domain use cases
//
// For GET/HEAD/OPTIONS/TRACE requests, the middleware ensures a CSRF token
// cookie exists and stores the masked token in context for use in templates.
//
// For state-changing methods (POST/PUT/PATCH/DELETE), it validates that the
// request includes a matching token in either:
//   - The X-CSRF-Token header (HTMX default)
//   - A form field named "csrf_token"
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
	opts := buildGorillaOptions(cfg)
	protect := csrf.Protect(cfg.secret(), opts...)

	return func(next http.Handler) http.Handler {
		inner := protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := csrf.Token(r); token != "" {
				r = r.WithContext(WithCSRFToken(r.Context(), token))
			}
			next.ServeHTTP(w, r)
		}))

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS == nil {
				r = csrf.PlaintextHTTPRequest(r)
			}
			inner.ServeHTTP(w, r)
		})
	}
}

// CSRFResponseHeaderMiddleware returns HTTP middleware that automatically sets
// the X-CSRF-Token response header on every request. This eliminates the need
// for individual handlers to manually call resp.CSRFToken(token).
//
// Place this AFTER CSRFMiddleware in the chain so the token is already in context:
//
//	handler := cqrshtmx.Chain(
//	    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
//	    cqrshtmx.CSRFResponseHeaderMiddleware,
//	    cqrshtmx.HTMXMiddleware,
//	    app.Middleware(),
//	)(mux)
//
// The header is only set when a token exists in the request context.
// For HTMX requests, the client reads this header and includes it in
// subsequent requests via hx-headers. For regular requests, the token
// is still available to server-side rendering.
func CSRFResponseHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token := csrfTokenFromRequest(r); token != "" {
			w.Header().Set(defaultCSRFHeaderName, token)
		}
		next.ServeHTTP(w, r)
	})
}

// CSRFProtect returns a HandlerOption that validates CSRF tokens for a specific
// command or query handler. Use this instead of global CSRFMiddleware when you
// want CSRF protection only on specific routes.
//
// When CSRFMiddleware is applied globally, CSRFProtect is redundant
// because gorilla/csrf already validates all state-changing requests.
// CSRFProtect is only needed when you want per-handler protection WITHOUT
// global middleware.
//
// Usage:
//
//	app.Command("CreateUser",
//	    cqrshtmx.CSRFProtect(cqrshtmx.CSRFConfig{}),
//	    cqrshtmx.DecodeJSON(...),
//	)
func CSRFProtect(cfg CSRFConfig) HandlerOption {
	opts := buildGorillaOptions(cfg)
	protect := csrf.Protect(cfg.secret(), opts...)
	return func(hc *handlerConfig) {
		hc.csrfConfig = &cfg
		hc.csrfProtect = protect
	}
}

// executeCSRFValidation checks CSRF for per-handler protection (CSRFProtect option).
// Returns nil if no CSRF config is set on the handler or validation passes.
func executeCSRFValidation(w http.ResponseWriter, r *http.Request, cfg *handlerConfig) error {
	if cfg.csrfConfig == nil {
		return nil
	}

	// If gorilla/csrf middleware already ran (token in its context), validation passed
	if csrf.Token(r) != "" {
		return nil
	}

	// gorilla/csrf hasn't run; validate using the cached middleware instance.
	// We capture its response via httptest.ResponseRecorder to avoid writing
	// directly to w, which would conflict with the caller's error handling.
	protect := cfg.csrfProtect
	var validated bool
	dummy := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		validated = true
	})

	rec := httptest.NewRecorder()
	if r.TLS == nil {
		r = csrf.PlaintextHTTPRequest(r)
	}
	protect(dummy).ServeHTTP(rec, r)

	if !validated {
		// gorilla/csrf rejected the request; copy its status and headers
		// to the real response so the client sees the same result.
		for k, vv := range rec.Header() {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(rec.Code)
		_, _ = w.Write(rec.Body.Bytes())
		return ErrCSRFInvalid
	}

	return nil
}
