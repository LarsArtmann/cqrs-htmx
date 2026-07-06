package cqrshtmx

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
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

func (a *App) handleCommandDispatch(
	w http.ResponseWriter,
	r *http.Request,
	cmdType command.Type,
	cfg *handlerConfig,
) {
	ctx, err := a.dispatchContext(w, r, cfg)
	if err != nil {
		return
	}

	if cfg.commandDecoder == nil {
		a.handleErr(w, r, ctx, cfg, errDecoderMissing)
		return
	}

	cmd, err := cfg.commandDecoder(r)
	if err != nil {
		wrappedErr := errorfamily.Wrapf(err, event.Rejection,
			"cqrshtmx.decode.command_failed", "decode command %s", cmdType)
		a.handleErr(w, r, ctx, cfg, wrappedErr)
		return
	}

	if cmd == nil {
		a.handleErr(w, r, ctx, cfg, errDecoderMissing)
		return
	}

	if cfg.requestGuard != nil {
		if guardErr := cfg.requestGuard(r, cmd); guardErr != nil {
			a.handleErr(w, r, ctx, cfg, guardErr)
			return
		}
	}

	ctx, cancel := a.timeoutCtx(ctx, cfg)
	defer cancel()

	if err = a.commands.Dispatch(ctx, cmd); err != nil {
		a.handleErr(w, r, ctx, cfg, errorfamily.Wrapf(err, errorfamily.Classify(err),
			"cqrshtmx.dispatch.command_failed", "dispatch command %s", cmdType))
		return
	}

	a.applyCommandResponse(w, r.WithContext(ctx), cfg)
	a.afterDispatchHook(ctx, r, nil)
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
	ctx, err := a.dispatchContext(w, r, cfg)
	if err != nil {
		return
	}

	if cfg.queryDecoder == nil {
		a.handleErr(w, r, ctx, cfg, errDecoderMissing)
		return
	}

	qry, err := cfg.queryDecoder(r)
	if err != nil {
		wrappedErr := errorfamily.Wrapf(err, event.Rejection,
			"cqrshtmx.decode.query_failed", "decode query %s", qryType)
		a.handleErr(w, r, ctx, cfg, wrappedErr)
		return
	}

	if qry == nil {
		a.handleErr(w, r, ctx, cfg, errDecoderMissing)
		return
	}

	if cfg.requestGuard != nil {
		if guardErr := cfg.requestGuard(r, qry); guardErr != nil {
			a.handleErr(w, r, ctx, cfg, guardErr)
			return
		}
	}

	ctx, cancel := a.timeoutCtx(ctx, cfg)
	defer cancel()

	result, err := a.queries.Dispatch(ctx, qry)
	if err != nil {
		a.handleErr(w, r, ctx, cfg, errorfamily.Wrapf(err, errorfamily.Classify(err),
			"cqrshtmx.dispatch.query_failed", "dispatch query %s", qryType))
		return
	}

	a.applyQueryResponse(w, r.WithContext(ctx), cfg, result)
	a.afterDispatchHook(ctx, r, nil)
}
