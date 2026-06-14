package cqrshtmx

import (
	"context"
	"net/http"
	"time"

	"github.com/justinas/nosurf"
)

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
