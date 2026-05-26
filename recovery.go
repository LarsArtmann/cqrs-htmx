package cqrshtmx

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

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
		//nolint:contextcheck // defer after panic has no context param to pass
		defer func() {
			if rec := recover(); rec != nil {
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}

				slog.ErrorContext(r.Context(), "panic recovered",
					slog.Any("panic", rec),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("stack", string(debug.Stack())),
				)

				DefaultErrorHandler(w, r,
					event.NewInfrastructure("panic", fmt.Sprintf("panic: %v", rec)))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware returns HTTP middleware that recovers from panics using
// the App's configured ErrorHandler and LoginRedirect. Use this instead of
// the standalone RecoveryMiddleware when you want panics to be handled with
// the same error response format as dispatch errors (e.g., JSON error handler,
// custom login redirect, or request ID correlation).
//
// http.ErrAbortHandler is re-raised without recovery.
func (a *App) RecoveryMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//nolint:contextcheck // defer after panic has no context param to pass
			defer func() {
				if rec := recover(); rec != nil {
					if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
						panic(rec)
					}

					slog.ErrorContext(r.Context(), "panic recovered",
						slog.Any("panic", rec),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.String("stack", string(debug.Stack())),
					)

					a.errorHandler(w, r,
						event.NewInfrastructure("panic", fmt.Sprintf("panic: %v", rec)))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
