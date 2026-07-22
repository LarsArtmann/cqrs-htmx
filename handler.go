package cqrshtmx

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// dispatchContext runs the beforeDispatch hook and pre-dispatch checks.
// Returns the (possibly modified) context, or skips with an error written to the response.
// The second return value indicates whether pre-dispatch checks passed (nil = continue).
func (a *App) dispatchContext(
	w http.ResponseWriter,
	r *http.Request,
	cfg *handlerConfig,
) (context.Context, error) {
	ctx := r.Context()

	if a.beforeDispatch != nil {
		ctx = a.beforeDispatch(ctx, r)
	}

	if err := a.executePreDispatchChecks(w, r, cfg); err != nil {
		a.afterDispatchHook(ctx, r, err)

		return ctx, err
	}

	return ctx, nil
}

// handleErr calls the error handler and afterDispatch hook, then returns.
// Centralizes the repeated error handling pattern used across all dispatch paths.
func (a *App) handleErr(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	cfg *handlerConfig,
	err error,
) {
	captureDispatchError(w, err)
	slog.WarnContext(
		ctx, "cqrs-htmx: dispatch error",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("error", err.Error()),
	)
	a.errorHandler(w, r, err)

	if cfg.onError != nil {
		cfg.onError(r, err)
	}

	a.afterDispatchHook(ctx, r, err)
}

// dispatchRequest runs the shared command/query dispatch pipeline and is the
// single source of truth for its ordering:
//
//	dispatchContext (authz + CSRF + method guard) → decode → nil-check →
//	requestGuard → timeout → dispatch → respond.
//
// kind is "command" or "query" and is interpolated into error tags/messages so
// the wire output is byte-identical to the pre-extraction handlers.
// decoderNil gates the unconfigured-decoder branch; decode runs the decoder;
// dispatch performs the actual CQRS call (returning a result, nil for commands);
// respond renders the success response. All four are per-kind adapters.
func dispatchRequest[Q, R any](
	a *App,
	w http.ResponseWriter,
	r *http.Request,
	cfg *handlerConfig,
	typeName, kind string,
	decoderNil func() bool,
	decode func(*http.Request) (Q, error),
	dispatch func(ctx context.Context, q Q) (R, error),
	respond func(http.ResponseWriter, *http.Request, *handlerConfig, R),
) {
	ctx, err := a.dispatchContext(w, r, cfg)
	if err != nil {
		return
	}

	if decoderNil() {
		a.handleErr(w, r, ctx, cfg, errDecoderMissing)

		return
	}

	v, err := decode(r)
	if err != nil {
		wrappedErr := errorfamily.Wrapf(err, event.Rejection,
			"cqrshtmx.decode."+kind+"_failed", "decode %s %s", kind, typeName)
		a.handleErr(w, r, ctx, cfg, wrappedErr)

		return
	}

	if v == nil {
		// The decoder was configured but returned (nil, nil): a server-side
		// wiring bug, not a transient infrastructure problem. Classify as
		// Corruption (500) so it is not retried as 503.
		a.handleErr(w, r, ctx, cfg, errDecoderReturnedNil)

		return
	}

	if cfg.requestGuard != nil {
		if guardErr := cfg.requestGuard(r, v); guardErr != nil {
			a.handleErr(w, r, ctx, cfg, guardErr)

			return
		}
	}

	ctx, cancel := a.timeoutCtx(ctx, cfg)
	defer cancel()

	result, dispatchErr := dispatch(ctx, v)
	if dispatchErr != nil {
		a.handleErr(w, r, ctx, cfg, errorfamily.Wrapf(dispatchErr, errorfamily.Classify(dispatchErr),
			"cqrshtmx.dispatch."+kind+"_failed", "dispatch %s %s", kind, typeName))

		return
	}

	respond(w, r.WithContext(ctx), cfg, result)
	a.afterDispatchHook(ctx, r, nil)
}

func (a *App) handleCommandDispatch(
	w http.ResponseWriter,
	r *http.Request,
	cmdType command.Type,
	cfg *handlerConfig,
) {
	dispatchRequest[any, any](a, w, r, cfg, string(cmdType), "command",
		func() bool { return cfg.commandDecoder == nil },
		func(r *http.Request) (any, error) { return cfg.commandDecoder(r) },
		func(ctx context.Context, v any) (any, error) {
			cmd, _ := v.(command.Command)

			return nil, a.commands.Dispatch(ctx, cmd)
		},
		func(w http.ResponseWriter, r *http.Request, cfg *handlerConfig, _ any) {
			a.applyCommandResponse(w, r, cfg)
		},
	)
}

func (a *App) executePreDispatchChecks(
	w http.ResponseWriter,
	r *http.Request,
	cfg *handlerConfig,
) error {
	if cfg.requireMethod != "" && r.Method != cfg.requireMethod {
		a.errorHandler(w, r, errorfamily.Wrapf(ErrMethodNotAllowed, event.Rejection,
			"cqrshtmx.handler.method_not_allowed", "got %s, want %s", r.Method, cfg.requireMethod))

		return ErrMethodNotAllowed
	}

	if err := a.executeAuthorization(r, cfg); err != nil {
		a.errorHandler(w, r, err)

		return err
	}

	if err := executeCSRFValidation(w, r, cfg); err != nil {
		return err
	}

	return nil
}

// writeDefaultStatus writes cfg.successStatus (or 204 No Content) when the
// handler has no explicit body content to write.
func writeDefaultStatus(w http.ResponseWriter, cfg *handlerConfig) {
	if !cfg.hasNoExplicitBody() {
		return
	}

	status := cfg.successStatus
	if status == 0 {
		status = http.StatusNoContent
	}

	w.WriteHeader(status)
}

func (a *App) applyCommandResponse(w http.ResponseWriter, r *http.Request, cfg *handlerConfig) {
	if applyHTMXResponse(w, r, cfg) {
		return
	}

	writeDefaultStatus(w, cfg)
}

func (a *App) applyQueryResponse(
	w http.ResponseWriter,
	r *http.Request,
	cfg *handlerConfig,
	result any,
) {
	if applyHTMXResponse(w, r, cfg) {
		return
	}

	if cfg.render != nil {
		if err := cfg.render(w, r, result); err != nil {
			a.handleErr(w, r, r.Context(), cfg, err)

			return
		}
	}

	writeDefaultStatus(w, cfg)
}

func (a *App) handleQueryDispatch(
	w http.ResponseWriter,
	r *http.Request,
	qryType query.Type,
	cfg *handlerConfig,
) {
	dispatchRequest[any, any](a, w, r, cfg, string(qryType), "query",
		func() bool { return cfg.queryDecoder == nil },
		func(r *http.Request) (any, error) { return cfg.queryDecoder(r) },
		func(ctx context.Context, v any) (any, error) {
			qry, _ := v.(query.Query)

			return a.queries.Dispatch(ctx, qry)
		},
		func(w http.ResponseWriter, r *http.Request, cfg *handlerConfig, result any) {
			a.applyQueryResponse(w, r, cfg, result)
		},
	)
}

// handleCommandTypedDispatch runs the shared pipeline for a typed command handler.
// The decoder must return a value of type Q (which satisfies command.Command); if it
// returns a different concrete type, the handler rejects with ErrDecodeFailed.
func handleCommandTypedDispatch[Q command.Command](
	a *App,
	w http.ResponseWriter,
	r *http.Request,
	cmdType command.Type,
	cfg *handlerConfig,
) {
	dispatchRequest[Q, any](a, w, r, cfg, string(cmdType), "command",
		func() bool { return cfg.commandDecoder == nil },
		func(r *http.Request) (Q, error) {
			v, err := cfg.commandDecoder(r)
			if err != nil {
				var zero Q

				return zero, err
			}

			qry, ok := v.(Q)
			if !ok {
				var zero Q

				return zero, errorfamily.Wrapf(ErrDecodeFailed, event.Rejection,
					"cqrshtmx.handler.command_type_mismatch",
					"expected %T, got %T", zero, v)
			}

			return qry, nil
		},
		func(ctx context.Context, q Q) (any, error) {
			return nil, a.commands.Dispatch(ctx, q)
		},
		func(w http.ResponseWriter, r *http.Request, cfg *handlerConfig, _ any) {
			a.applyCommandResponse(w, r, cfg)
		},
	)
}

// handleQueryTypedDispatch runs the shared pipeline for a typed query handler.
// Q is the query type and R is the result type. The decoder must return a value of
// type Q (which satisfies query.Query); if it returns a different concrete type,
// the handler rejects with ErrDecodeFailed.
func handleQueryTypedDispatch[Q query.Query, R any](
	a *App,
	w http.ResponseWriter,
	r *http.Request,
	qryType query.Type,
	cfg *handlerConfig,
) {
	dispatchRequest[Q, R](a, w, r, cfg, string(qryType), "query",
		func() bool { return cfg.queryDecoder == nil },
		func(r *http.Request) (Q, error) {
			v, err := cfg.queryDecoder(r)
			if err != nil {
				var zero Q

				return zero, err
			}

			qry, ok := v.(Q)
			if !ok {
				var zero Q

				return zero, errorfamily.Wrapf(ErrDecodeFailed, event.Rejection,
					"cqrshtmx.handler.query_type_mismatch",
					"expected %T, got %T", zero, v)
			}

			return qry, nil
		},
		func(ctx context.Context, q Q) (R, error) {
			return query.DispatchTyped[R](ctx, a.queries, q)
		},
		func(w http.ResponseWriter, r *http.Request, cfg *handlerConfig, result R) {
			a.applyQueryResponse(w, r, cfg, result)
		},
	)
}

// captureDispatchError stores the dispatch error on the ResponseWriter chain
// so that logging middleware (RequestLoggingSlog) can include error context
// (code, family, context key-values) in the request log. Traverses the
// Unwrap chain to find a dispatchErrorRecorder (typically StatusRecorder).
func captureDispatchError(w http.ResponseWriter, err error) {
	for current := w; current != nil; {
		if rec, ok := current.(dispatchErrorRecorder); ok {
			rec.SetDispatchError(err)

			return
		}

		type unwrapper interface{ Unwrap() http.ResponseWriter }

		u, ok := current.(unwrapper)
		if !ok {
			return
		}

		current = u.Unwrap()
	}
}
