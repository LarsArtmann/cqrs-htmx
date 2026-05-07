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

	if err := a.executePreDispatchChecks(w, r, cfg); err != nil {
		return
	}

	cmd, err := cfg.commandDecoder(r)
	if err != nil {
		a.errorHandler(w, r, err)
		return
	}

	if err = a.commands.Dispatch(ctx, cmd); err != nil {
		a.errorHandler(w, r, fmt.Errorf("%w: %s: %w", ErrDispatchFailed, cmdType, err))
		return
	}

	a.applyCommandResponse(w, r, cfg)
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

	if cfg.commandDecoder == nil {
		a.errorHandler(w, r, ErrDecoderMissing)
		return ErrDecoderMissing
	}

	return nil
}

func (a *App) applyCommandResponse(w http.ResponseWriter, r *http.Request, cfg *handlerConfig) {
	applyHTMXResponse(w, r, cfg)

	if cfg.redirect == "" && cfg.trigger == "" && cfg.pushURL == "" && len(cfg.triggerDetail) == 0 {
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

	if err := a.executeAuthorization(r, cfg); err != nil {
		a.errorHandler(w, r, err)
		return
	}

	if cfg.queryDecoder == nil {
		a.errorHandler(w, r, ErrDecoderMissing)
		return
	}

	qry, err := cfg.queryDecoder(r)
	if err != nil {
		a.errorHandler(w, r, err)
		return
	}

	result, err := a.queries.Dispatch(ctx, qry)
	if err != nil {
		a.errorHandler(w, r, fmt.Errorf("%w: %s: %w", ErrDispatchFailed, qryType, err))
		return
	}

	applyHTMXResponse(w, r, cfg)

	if cfg.render != nil {
		if renderErr := cfg.render(w, r, result); renderErr != nil {
			a.errorHandler(w, r, renderErr)
		}

		return
	}

	if cfg.redirect == "" && cfg.trigger == "" && cfg.pushURL == "" {
		w.WriteHeader(http.StatusNoContent)
	}
}
