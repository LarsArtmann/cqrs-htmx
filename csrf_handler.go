package cqrshtmx

import (
	"net/http"

	"github.com/larsartmann/httputil"
)

// CSRFProtect returns a HandlerOption that validates CSRF tokens for a specific
// command or query handler. Use this instead of global CSRFMiddleware when you
// want CSRF protection only on specific routes.
//
// When CSRFMiddleware is applied globally, CSRFProtect is redundant
// because nosurf already validates all state-changing requests.
// CSRFProtect is only needed when you want per-handler protection WITHOUT
// global middleware.
//
// Usage:
//
//	app.Command("CreateUser",
//	    cqrshtmx.CSRFProtect(httputil.CSRFConfig{}),
//	    cqrshtmx.DecodeJSON(...),
//	)
func CSRFProtect(config httputil.CSRFConfig) HandlerOption {
	return func(hc *handlerConfig) {
		hc.csrfConfig = &config
	}
}

// executeCSRFValidation checks CSRF for per-handler protection (CSRFProtect option).
// Returns nil if no CSRF config is set on the handler or validation passes.
func executeCSRFValidation(w http.ResponseWriter, r *http.Request, handlerCfg *handlerConfig) error {
	if handlerCfg.csrfConfig == nil {
		return nil
	}

	validated, rec := httputil.ValidateCSRF(r, *handlerCfg.csrfConfig)
	if validated {
		return nil
	}

	for k, vv := range rec.Header() {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(rec.Code)
	writeAll(w, rec.Body.Bytes())

	return httputil.ErrCSRFInvalid
}
