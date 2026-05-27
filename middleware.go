package cqrshtmx

import (
	"log/slog"
	"net/http"
	"slices"
)

const (
	headerCorrelationID = "X-Correlation-ID"
	headerRequestID     = "X-Request-ID"
)

// ContextEnrichmentMiddleware extracts the user ID from the request using
// the provided extractor and stores it in the context for downstream CQRS handlers.
// It also auto-generates a RequestID (or extracts it from X-Request-ID header)
// and extracts CorrelationID from X-Correlation-ID header.
func ContextEnrichmentMiddleware(extractor UserIDExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			if ridStr := r.Header.Get(headerRequestID); ridStr != "" {
				rid, err := ParseRequestID(ridStr)
				if err == nil {
					ctx = WithRequestID(ctx, rid)
				} else {
					ctx = WithRequestID(ctx, NewRequestID())
				}
			} else {
				ctx = WithRequestID(ctx, NewRequestID())
			}

			if cidStr := r.Header.Get(headerCorrelationID); cidStr != "" {
				cid, err := ParseCorrelationID(cidStr)
				if err == nil {
					ctx = WithCorrelationID(ctx, cid)
				} else {
					slog.Debug("cqrs-htmx: invalid correlation ID header",
						slog.String("header", headerCorrelationID),
						slog.String("value", cidStr),
						slog.String("error", err.Error()),
					)
				}
			}

			if extractor != nil {
				userID, err := extractor(r)
				if err == nil && !userID.IsZero() {
					ctx = WithUserID(ctx, userID)
				}
			}

			rid := RequestIDFromContext(ctx)
			if !rid.IsZero() {
				w.Header().Set(headerRequestID, rid.String())
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
