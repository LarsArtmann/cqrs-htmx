package cqrshtmx

import (
	"fmt"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func (a *App) handleCommandDispatch(
	w http.ResponseWriter,
	r *http.Request,
	cmdType command.Type,
	cfg *handlerConfig,
) {
	ctx := r.Context()

	if a.beforeDispatch != nil {
		ctx = a.beforeDispatch(ctx, r)
	}

	if err := a.executePreDispatchChecks(w, r, cfg); err != nil {
		if a.afterDispatch != nil {
			a.afterDispatch(ctx, r, err)
		}
		return
	}

	if cfg.commandDecoder == nil {
		a.errorHandler(w, r, errDecoderMissing)
		if a.afterDispatch != nil {
			a.afterDispatch(ctx, r, errDecoderMissing)
		}
		return
	}

	cmd, err := cfg.commandDecoder(r)
	if err != nil {
		a.errorHandler(w, r, err)
		if a.afterDispatch != nil {
			a.afterDispatch(ctx, r, err)
		}
		return
	}

	ctx, cancel := a.timeoutCtx(ctx)
	defer cancel()

	if err = a.commands.Dispatch(ctx, cmd); err != nil {
		a.errorHandler(w, r, fmt.Errorf("%w: %s: %w", ErrDispatchFailed, cmdType, err))
		if a.afterDispatch != nil {
			a.afterDispatch(ctx, r, err)
		}
		return
	}

	a.applyCommandResponse(w, r, cfg)
	if a.afterDispatch != nil {
		a.afterDispatch(ctx, r, nil)
	}
}

func (a *App) executePreDispatchChecks(
	w http.ResponseWriter,
	r *http.Request,
	cfg *handlerConfig,
) error {
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
		w.WriteHeader(http.StatusNoContent)
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
			a.errorHandler(w, r, err)
			if a.afterDispatch != nil {
				a.afterDispatch(r.Context(), r, err)
			}
			return
		}
	}

	if cfg.hasNoExplicitBody() {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (a *App) handleQueryDispatch(
	w http.ResponseWriter,
	r *http.Request,
	qryType query.Type,
	cfg *handlerConfig,
) {
	ctx := r.Context()

	if a.beforeDispatch != nil {
		ctx = a.beforeDispatch(ctx, r)
	}

	if err := a.executePreDispatchChecks(w, r, cfg); err != nil {
		if a.afterDispatch != nil {
			a.afterDispatch(ctx, r, err)
		}
		return
	}

	if cfg.queryDecoder == nil {
		a.errorHandler(w, r, errDecoderMissing)
		if a.afterDispatch != nil {
			a.afterDispatch(ctx, r, errDecoderMissing)
		}
		return
	}

	qry, err := cfg.queryDecoder(r)
	if err != nil {
		err = fmt.Errorf("%w: %s: %w", ErrDecodeFailed, qryType, err)
		a.errorHandler(w, r, err)
		if a.afterDispatch != nil {
			a.afterDispatch(ctx, r, err)
		}
		return
	}

	ctx, cancel := a.timeoutCtx(ctx)
	defer cancel()

	result, err := a.queries.Dispatch(ctx, qry)
	if err != nil {
		qu := fmt.Errorf("%w: %s: %w", ErrDispatchFailed, qryType, err)
		a.errorHandler(w, r, qu)
		if a.afterDispatch != nil {
			a.afterDispatch(ctx, r, qu)
		}
		return
	}

	a.applyQueryResponse(w, r.WithContext(ctx), cfg, result)
	if a.afterDispatch != nil {
		a.afterDispatch(ctx, r, nil)
	}
}
