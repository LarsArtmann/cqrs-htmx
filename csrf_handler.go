package cqrshtmx

import (
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/csrf"
)

// CSRFProtect returns a HandlerOption that validates CSRF tokens for a specific
// command or query handler. Use this instead of global CSRFMiddleware when you
// want CSRF protection only on specific routes.
//
// When CSRFMiddleware is applied globally, CSRFProtect is redundant
// because gorilla/csrf already validates all state-changing requests.
// CSRFProtect is only needed when you want per-handler protection WITHOUT
// global middleware.
//
// Usage:
//
//	app.Command("CreateUser",
//	    cqrshtmx.CSRFProtect(cqrshtmx.CSRFConfig{}),
//	    cqrshtmx.DecodeJSON(...),
//	)
func CSRFProtect(cfg CSRFConfig) HandlerOption {
	opts := buildGorillaOptions(cfg)
	protect := csrf.Protect(cfg.secret(), opts...)
	return func(hc *handlerConfig) {
		hc.csrfConfig = &cfg
		hc.csrfProtect = protect
	}
}

// executeCSRFValidation checks CSRF for per-handler protection (CSRFProtect option).
// Returns nil if no CSRF config is set on the handler or validation passes.
func executeCSRFValidation(w http.ResponseWriter, r *http.Request, cfg *handlerConfig) error {
	if cfg.csrfConfig == nil {
		return nil
	}

	if csrf.Token(r) != "" {
		return nil
	}

	protect := cfg.csrfProtect
	var validated bool
	dummy := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		validated = true
	})

	rec := httptest.NewRecorder()
	if r.TLS == nil {
		r = csrf.PlaintextHTTPRequest(r)
	}
	protect(dummy).ServeHTTP(rec, r)

	if !validated {
		for k, vv := range rec.Header() {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(rec.Code)
		_, _ = w.Write(rec.Body.Bytes())
		return ErrCSRFInvalid
	}

	return nil
}
