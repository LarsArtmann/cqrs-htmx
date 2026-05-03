package cqrshtmx

import (
	"net/http"
)

// ContextEnrichmentMiddleware extracts the user ID from the request using
// the provided extractor and stores it in the context for downstream CQRS handlers.
func ContextEnrichmentMiddleware(extractor UserIDExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if extractor != nil {
				userID := extractor(r)
				if userID != "" {
					r = r.WithContext(WithUserID(r.Context(), userID))
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// HTMXMiddleware parses HTMX request headers once and stores them in context.
// Apply this once to your router so downstream handlers can use
// HTMXFromContext, RenderPartial, and the accessor functions without
// repeated header parsing.
func HTMXMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := parseHTMXRequest(r)
		r = r.WithContext(WithHTMX(r.Context(), h))
		next.ServeHTTP(w, r)
	})
}

// Chain returns an HTTP middleware that applies multiple middleware in order.
// Middleware are applied left-to-right: Chain(a, b, c) wraps as a(b(c(handler))).
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}

		return final
	}
}
