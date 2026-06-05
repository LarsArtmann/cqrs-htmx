package cqrshtmx

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/justinas/nosurf"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const (
	defaultCSRFCookieName = "csrf_token"
	defaultCSRFHeaderName = "X-CSRF-Token"
	defaultCSRFFieldName  = "csrf_token"
	defaultCSRFMaxAge     = 24 * time.Hour
)

// ErrCSRFInvalid is returned when a CSRF token is missing, malformed, or does not match.
// Uses justinas/nosurf under the hood for token generation and validation.
var ErrCSRFInvalid = event.NewRejection("csrf_invalid", "invalid or missing CSRF token").WithCause(ErrForbidden)

// CSRFConfig configures CSRF protection.
//
// All fields are optional; zero values use secure defaults.
// Uses justinas/nosurf internally for token generation, masking (BREACH mitigation),
// cookie management, and validation.
type CSRFConfig struct {
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
	// Default: false (auto-detected from request scheme)
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

// Validate checks the CSRF configuration for common misconfigurations.
// Returns a non-nil error if the config would produce insecure or broken behavior.
// Call this in production startup code to fail fast on misconfiguration.
func (c *CSRFConfig) Validate() error {
	if c.SameSite == http.SameSiteNoneMode && !c.Secure {
		return event.NewInfrastructure("csrf_samesite_insecure", "SameSite=None requires Secure=true").
			WithCause(ErrCSRFConfig)
	}

	for _, origin := range c.TrustedOrigins {
		if origin == "" || origin == "*" {
			return event.NewInfrastructure("csrf_unsafe_origin",
				fmt.Sprintf("TrustedOrigins contains unsafe entry %q — use specific domain names only",
					origin)).WithCause(ErrCSRFConfig)
		}
	}

	if !c.Secure {
		slog.Warn("cqrs-htmx: CSRFConfig.Validate: Secure is false — CSRF cookies will be sent over plain HTTP",
			slog.String("hint", "set Secure=true in production"))
	}

	return nil
}

// configureNosurfHandler applies CSRFConfig settings to a nosurf handler.
func configureNosurfHandler(handler *nosurf.CSRFHandler, cfg CSRFConfig) {
	//nolint:gosec,exhaustruct // HttpOnly=false required for double-submit; http.Cookie has many optional fields
	cookie := http.Cookie{
		Name:     cfg.cookieName(),
		Path:     cfg.path(),
		Secure:   cfg.Secure,
		HttpOnly: false,
		SameSite: cfg.SameSite,
		MaxAge:   int(cfg.maxAge().Seconds()),
	}
	if cfg.Domain != "" {
		cookie.Domain = cfg.Domain
	}
	handler.SetBaseCookie(cookie)

	handler.SetIsTLSFunc(func(r *http.Request) bool {
		return r.TLS != nil
	})

	if len(cfg.TrustedOrigins) > 0 {
		origins, err := nosurf.StaticOrigins(cfg.TrustedOrigins...)
		if err != nil {
			slog.Error(
				"cqrs-htmx: invalid TrustedOrigins",
				slog.String("error", err.Error()),
			)
		} else {
			handler.SetIsAllowedOriginFunc(origins)
		}
	}

	if cfg.ErrorHandler != nil {
		handler.SetFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg.ErrorHandler(w, r, ErrCSRFInvalid)
		}))
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

func csrfTokenFromRequest(r *http.Request) string {
	if token := nosurf.Token(r); token != "" {
		return token
	}
	return CSRFTokenFromContext(r.Context())
}

// InvalidateCSRFCookie invalidates the current CSRF cookie, forcing a new token
// to be generated on the next request. Call this on login/logout to prevent
// CSRF fixation attacks.
func InvalidateCSRFCookie(w http.ResponseWriter, cfg CSRFConfig) {
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
// Uses justinas/nosurf internally for:
//   - Cryptographically secure token generation (crypto/rand)
//   - Per-request token masking (BREACH attack mitigation)
//   - Same-origin validation via Origin/Referer/Sec-Fetch-Site headers
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
	if err := cfg.Validate(); err != nil {
		slog.Error("cqrs-htmx: CSRFConfig validation failed", slog.String("error", err.Error()))
	}

	return func(next http.Handler) http.Handler {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := nosurf.Token(r); token != "" {
				r = r.WithContext(WithCSRFToken(r.Context(), token))
			}
			next.ServeHTTP(w, r)
		})

		handler := nosurf.New(inner)
		configureNosurfHandler(handler, cfg)

		needsTranslation := cfg.headerName() != defaultCSRFHeaderName ||
			cfg.fieldName() != defaultCSRFFieldName

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setPlaintextHTTPOrigin(r)

			// Translate custom header/field names to nosurf defaults.
			if needsTranslation {
				translateCSRFHeaders(r, cfg)
			}

			handler.ServeHTTP(w, r)
		})
	}
}

// setPlaintextHTTPOrigin sets the Sec-Fetch-Site header to "same-origin" for
// plain HTTP requests without origin headers. This allows nosurf to skip
// origin validation, matching the behavior of gorilla/csrf's PlaintextHTTPRequest
// for HTTP deployments.
func setPlaintextHTTPOrigin(r *http.Request) {
	if r.TLS == nil &&
		r.Header.Get("Sec-Fetch-Site") == "" &&
		r.Header.Get("Origin") == "" &&
		r.Header.Get("Referer") == "" {
		r.Header.Set("Sec-Fetch-Site", "same-origin")
	}
}

// translateCSRFHeaders maps custom header/field names to nosurf's default
// "X-CSRF-Token" header. nosurf hardcodes its header and field names,
// so we translate before passing the request to nosurf.
func translateCSRFHeaders(r *http.Request, cfg CSRFConfig) {
	if cfg.headerName() != defaultCSRFHeaderName {
		if token := r.Header.Get(cfg.headerName()); token != "" {
			r.Header.Set(defaultCSRFHeaderName, token)
			return
		}
	}
	if cfg.fieldName() != defaultCSRFFieldName {
		if token := r.PostFormValue(cfg.fieldName()); token != "" {
			r.Header.Set(defaultCSRFHeaderName, token)
		}
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
