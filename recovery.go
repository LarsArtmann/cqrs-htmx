package cqrshtmx

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// shouldRePanic reports whether a recovered value is http.ErrAbortHandler,
// which must be re-raised per net/http convention.
func shouldRePanic(rec any) bool {
	if err, ok := rec.(error); ok {
		return errors.Is(err, http.ErrAbortHandler)
	}

	return false
}

// writePanicResponse logs the panic and writes a 500 response via handler.
//
// The panic error is classified as Infrastructure for logging purposes, but
// MapError's explicit override returns 500 — a panic is a server bug, not a
// "service unavailable" condition (503 would imply the service is down when
// it is actually serving other requests fine).
const panicCode = "panic"

func isPanicError(err error) bool {
	e, ok := errors.AsType[*event.Error](err)

	return ok && e.Code() == panicCode
}

func writePanicResponse(
	w http.ResponseWriter,
	r *http.Request,
	rec any,
	handler ErrorHandler,
) {
	// RecoveryMiddleware/RecoverHandler run outside ContextEnrichmentMiddleware,
	// so the captured request may not have RequestID/CorrelationID in context.
	// ContextEnrichmentMiddleware writes the RequestID to the X-Request-ID
	// response header; recover it into the context so error handlers can echo it
	// — matching every non-panic error path. The correlation ID is only available
	// from the request header (there is no generated fallback), so recover it from
	// X-Correlation-ID when the client supplied it.
	if RequestIDFromContext(r.Context()).IsZero() {
		if ridStr := w.Header().Get(headerRequestID); ridStr != "" {
			if rid, err := ParseRequestID(ridStr); err == nil {
				r = r.WithContext(WithRequestID(r.Context(), rid))
			}
		}
	}

	if CorrelationIDFromContext(r.Context()).IsZero() {
		if cidStr := r.Header.Get(headerCorrelationID); cidStr != "" {
			if cid, err := ParseCorrelationID(cidStr); err == nil {
				r = r.WithContext(WithCorrelationID(r.Context(), cid))
			}
		}
	}

	slog.ErrorContext(
		r.Context(), "panic recovered",
		slog.Any("panic", rec),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("stack", string(debug.Stack())),
	)

	handler(w, r, errorfamily.Newf(event.Infrastructure, panicCode, "panic: %v", rec))
}

// RecoveryMiddleware returns HTTP middleware that recovers from panics in
// downstream handlers. Recovered panics are logged with a stack trace and
// written as 500 Internal Server Error using DefaultErrorHandler.
//
// http.ErrAbortHandler is re-raised without recovery, matching Go's
// net/http convention for request aborts (used by TimeoutHandler and
// ServeContent).
//
// Usage:
//
//	handler := cqrshtmx.Chain(
//	    cqrshtmx.SecurityHeadersMiddleware,
//	    cqrshtmx.RecoveryMiddleware,
//	    cqrshtmx.CSRFMiddleware(...),
//	)(mux)
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if shouldRePanic(rec) {
					panic(rec) //cqrs-lint:ignore(C009) re-panic http.ErrAbortHandler per net/http convention
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RecoverHandler returns HTTP middleware that recovers from panics using
// the App's configured ErrorHandler and LoginRedirect. Use this instead of
// the standalone RecoveryMiddleware when you want panics to be handled with
// the same error response format as dispatch errors (e.g., JSON error handler,
// custom login redirect, or request ID correlation).
//
// http.ErrAbortHandler is re-raised without recovery.
func (a *App) RecoverHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if shouldRePanic(rec) {
						panic(rec) //cqrs-lint:ignore(C009) re-panic http.ErrAbortHandler per net/http convention
					}

					writePanicResponse(w, r, rec, a.errorHandler)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
