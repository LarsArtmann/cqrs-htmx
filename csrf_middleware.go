package cqrshtmx

import (
	"log/slog"
	"net/http"

	"github.com/justinas/nosurf"
)

// )(mux)
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
