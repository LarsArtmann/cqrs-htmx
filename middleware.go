package cqrshtmx

import (
	"net/http"
	"slices"
)

// ContextEnrichmentMiddleware extracts the user ID from the request using
// the provided extractor and stores it in the context for downstream CQRS handlers.
func ContextEnrichmentMiddleware(extractor UserIDExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Propagate correlation ID from request header if present.
			if cid := r.Header.Get("X-Correlation-ID"); cid != "" {
				ctx = WithCorrelationID(ctx, cid)
			}

			if extractor != nil {
				userIDStr := extractor(r)
				if userIDStr != "" {
					userID, err := ParseUserID(userIDStr)
					if err == nil {
						ctx = WithUserID(ctx, userID)
					}
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
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
		for _, v := range slices.Backward(middlewares) {
			final = v(final)
		}

		return final
	}
}
