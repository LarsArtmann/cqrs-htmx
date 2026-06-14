package cqrshtmx

import "net/http"

func applyHTMXResponse(w http.ResponseWriter, r *http.Request, cfg *handlerConfig) bool {
	if cfg.redirect == "" && cfg.trigger == "" && cfg.pushURL == "" && len(cfg.triggerDetail) == 0 {
		return false
	}

	resp := NewResponse(w, r)

	if cfg.redirect != "" {
		resp.Redirect(cfg.redirect)
	}

	if cfg.trigger != "" {
		resp.Trigger(cfg.trigger)
	}

	for name, detail := range cfg.triggerDetail {
		resp.TriggerWithDetail(name, detail)
	}

	if cfg.pushURL != "" {
		resp.PushURL(cfg.pushURL)
	}

	return resp.Apply()
}

// OnError returns a HandlerOption that registers a per-handler error callback.
// The callback is invoked after the App-level error handler, allowing handlers
// to add custom logging, metrics, or cleanup for specific routes.
func OnError(fn func(r *http.Request, err error)) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.onError = fn
	}
}
