package cqrshtmx

import (
	"context"
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
			ctx := enrichRequestID(r.Context(), r)
			ctx = enrichCorrelationID(ctx, r)
			ctx = enrichUserAndActor(ctx, r, extractor)

			rid := RequestIDFromContext(ctx)
			if !rid.IsZero() {
				w.Header().Set(headerRequestID, rid.String())
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// enrichRequestID returns ctx with a RequestID set: the parsed X-Request-ID
// header when present and valid, otherwise a freshly generated one.
func enrichRequestID(ctx context.Context, r *http.Request) context.Context {
	if ridStr := r.Header.Get(headerRequestID); ridStr != "" {
		if rid, err := ParseRequestID(ridStr); err == nil {
			return WithRequestID(ctx, rid)
		}
	}

	return WithRequestID(ctx, NewRequestID())
}

// enrichCorrelationID returns ctx with the CorrelationID set from the
// X-Correlation-ID header when present and valid. Invalid values are logged
// at debug level and dropped.
func enrichCorrelationID(ctx context.Context, r *http.Request) context.Context {
	if cidStr := r.Header.Get(headerCorrelationID); cidStr != "" {
		cid, err := ParseCorrelationID(cidStr)
		if err != nil {
			slog.Debug(
				"cqrs-htmx: invalid correlation ID header",
				slog.String("header", headerCorrelationID),
				slog.String("value", cidStr),
				slog.String("error", err.Error()),
			)

			return ctx
		}

		return WithCorrelationID(ctx, cid)
	}

	return ctx
}

// enrichUserAndActor returns ctx with the UserID from the extractor (when it
// succeeds) and, unless a consumer already set one, an ActorID auto-derived
// from that UserID so event metadata carries the full audit trail.
func enrichUserAndActor(ctx context.Context, r *http.Request, extractor UserIDExtractor) context.Context {
	if extractor == nil {
		return ctx
	}

	userID, err := extractor(r)
	if err != nil || userID.IsZero() {
		return ctx
	}

	ctx = WithUserID(ctx, userID)

	// Skipped if a consumer already set an ActorID (e.g., impersonation
	// middleware targeting a different user).
	if ActorIDFromContext(ctx).IsZero() {
		ctx = WithActorID(ctx, ActorIDFromUser(userID))
	}

	return ctx
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
