package cqrshtmx

import (
	"net/http"
	"net/http/httptest"

	"github.com/justinas/nosurf"
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
//	    cqrshtmx.CSRFProtect(cqrshtmx.CSRFConfig{}),
//	    cqrshtmx.DecodeJSON(...),
//	)
func CSRFProtect(cfg CSRFConfig) HandlerOption {
	return func(hc *handlerConfig) {
		hc.csrfConfig = &cfg
	}
}

// executeCSRFValidation checks CSRF for per-handler protection (CSRFProtect option).
// Returns nil if no CSRF config is set on the handler or validation passes.
func executeCSRFValidation(w http.ResponseWriter, r *http.Request, hc *handlerConfig) error {
	if hc.csrfConfig == nil {
		return nil
	}

	if nosurf.Token(r) != "" {
		return nil
	}

	var validated bool

	dummy := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		validated = true
	})

	handler := nosurf.New(dummy)
	configureNosurfHandler(handler, *hc.csrfConfig)

	rec := httptest.NewRecorder()

	// For plain HTTP requests without origin headers, set Sec-Fetch-Site
	// to allow nosurf to skip origin validation.
	setPlaintextHTTPOrigin(r, *hc.csrfConfig)

	needsTranslation := hc.csrfConfig.headerName() != defaultCSRFHeaderName ||
		hc.csrfConfig.fieldName() != defaultCSRFFieldName
	if needsTranslation {
		translateCSRFHeaders(r, *hc.csrfConfig)
	}

	handler.ServeHTTP(rec, r)

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
