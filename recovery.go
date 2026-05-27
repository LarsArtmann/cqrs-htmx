package cqrshtmx

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/larsartmann/go-cqrs-lite/core/event"
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
func writePanicResponse(
	w http.ResponseWriter,
	r *http.Request,
	rec any,
	handler ErrorHandler,
) {
	slog.ErrorContext(
		r.Context(), "panic recovered",
		slog.Any("panic", rec),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("stack", string(debug.Stack())),
	)

	handler(w, r, event.NewInfrastructure("panic", fmt.Sprintf("panic: %v", rec)))
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
					panic(rec)
				}

				writePanicResponse(w, r, rec, DefaultErrorHandler)
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
						panic(rec)
					}

					writePanicResponse(w, r, rec, a.errorHandler)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
