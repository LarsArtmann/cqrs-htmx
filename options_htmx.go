package cqrshtmx

import "net/http"

func applyHTMXResponse(w http.ResponseWriter, r *http.Request, config *handlerConfig) bool {
	if config.redirect == "" && config.trigger == "" && config.pushURL == "" && len(config.triggerDetail) == 0 {
		return false
	}

	resp := NewResponse(w, r)

	if config.redirect != "" {
		resp.Redirect(config.redirect)
	}

	if config.trigger != "" {
		resp.Trigger(config.trigger)
	}

	for name, detail := range config.triggerDetail {
		resp.TriggerWithDetail(name, detail)
	}

	if config.pushURL != "" {
		resp.PushURL(config.pushURL)
	}

	return resp.Apply()
}

// OnError returns a HandlerOption that registers a per-handler error callback.
// The callback is invoked after the App-level error handler, allowing handlers
// to add custom logging, metrics, or cleanup for specific routes.
func OnError(fn func(r *http.Request, err error)) HandlerOption {
	return func(config *handlerConfig) {
		config.onError = fn
	}
}
