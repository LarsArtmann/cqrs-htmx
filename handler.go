package cqrshtmx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
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
		a.handleErr(w, r, ctx, cfg, err)
		return
	}

	if cmd == nil {
		a.handleErr(w, r, ctx, cfg, errDecoderMissing)
		return
	}

	ctx, cancel := a.timeoutCtx(ctx, cfg)
	defer cancel()

	if err = a.commands.Dispatch(ctx, cmd); err != nil {
		a.handleErr(w, r, ctx, cfg, fmt.Errorf("%w: %s: %w", ErrDispatchFailed, cmdType, err))
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
		a.errorHandler(w, r, fmt.Errorf("%w: got %s, want %s",
			ErrMethodNotAllowed, r.Method, cfg.requireMethod))
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

func (a *App) applyCommandResponse(w http.ResponseWriter, r *http.Request, cfg *handlerConfig) {
	if applyHTMXResponse(w, r, cfg) {
		return
	}

	if cfg.hasNoExplicitBody() {
		status := cfg.successStatus
		if status == 0 {
			status = http.StatusNoContent
		}
		w.WriteHeader(status)
	}
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

	if cfg.hasNoExplicitBody() {
		status := cfg.successStatus
		if status == 0 {
			status = http.StatusNoContent
		}
		w.WriteHeader(status)
	}
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
		wrappedErr := fmt.Errorf("%w: %s: %w", ErrDecodeFailed, qryType, err)
		a.handleErr(w, r, ctx, cfg, wrappedErr)
		return
	}

	ctx, cancel := a.timeoutCtx(ctx, cfg)
	defer cancel()

	result, err := a.queries.Dispatch(ctx, qry)
	if err != nil {
		a.handleErr(w, r, ctx, cfg, fmt.Errorf("%w: %s: %w", ErrDispatchFailed, qryType, err))
		return
	}

	a.applyQueryResponse(w, r.WithContext(ctx), cfg, result)
	a.afterDispatchHook(ctx, r, nil)
}
