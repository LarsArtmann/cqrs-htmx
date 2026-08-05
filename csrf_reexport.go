package cqrshtmx

import (
	"github.com/larsartmann/httputil"
)

// CSRF core now lives in httputil. These aliases preserve backward
// compatibility for cqrs-htmx consumers.
//
// Deprecated: import github.com/larsartmann/httputil directly. These aliases
// will be removed in cqrs-htmx v5.

type (
	// CSRFConfig is an alias for httputil.CSRFConfig.
	//
	// Deprecated: use httputil.CSRFConfig.
	CSRFConfig   = httputil.CSRFConfig
	// ErrorHandler is an alias for httputil.ErrorHandler.
	//
	// Deprecated: use httputil.ErrorHandler.
	ErrorHandler = httputil.ErrorHandler
)

var (
	// CSRFMiddleware is an alias for httputil.CSRFMiddleware.
	//
	// Deprecated: use httputil.CSRFMiddleware.
	CSRFMiddleware = httputil.CSRFMiddleware
	// CSRFResponseHeaderMiddleware is an alias for httputil.CSRFResponseHeaderMiddleware.
	//
	// Deprecated: use httputil.CSRFResponseHeaderMiddleware.
	CSRFResponseHeaderMiddleware = httputil.CSRFResponseHeaderMiddleware
	// CSRFTokenFromContext is an alias for httputil.CSRFTokenFromContext.
	//
	// Deprecated: use httputil.CSRFTokenFromContext.
	CSRFTokenFromContext = httputil.CSRFTokenFromContext
	// WithCSRFToken is an alias for httputil.WithCSRFToken.
	//
	// Deprecated: use httputil.WithCSRFToken.
	WithCSRFToken = httputil.WithCSRFToken
	// CSRFTestToken is an alias for httputil.CSRFTestToken.
	//
	// Deprecated: use httputil.CSRFTestToken.
	CSRFTestToken = httputil.CSRFTestToken
	// InvalidateCSRFCookie is an alias for httputil.InvalidateCSRFCookie.
	//
	// Deprecated: use httputil.InvalidateCSRFCookie.
	InvalidateCSRFCookie = httputil.InvalidateCSRFCookie
	// CSRFTokenHTMLMeta is an alias for httputil.CSRFTokenHTMLMeta.
	//
	// Deprecated: use httputil.CSRFTokenHTMLMeta.
	CSRFTokenHTMLMeta = httputil.CSRFTokenHTMLMeta
	// CSRFTokenHXHeaders is an alias for httputil.CSRFTokenHXHeaders.
	//
	// Deprecated: use httputil.CSRFTokenHXHeaders.
	CSRFTokenHXHeaders = httputil.CSRFTokenHXHeaders
	// CSRFTokenFormField is an alias for httputil.CSRFTokenFormField.
	//
	// Deprecated: use httputil.CSRFTokenFormField.
	CSRFTokenFormField = httputil.CSRFTokenFormField
	// ForbiddenErrorHandler is an alias for httputil.ForbiddenHandler.
	//
	// Deprecated: use httputil.ForbiddenHandler.
	ForbiddenErrorHandler = httputil.ForbiddenHandler
	// ErrCSRFInvalid is an alias for httputil.ErrCSRFInvalid.
	//
	// Deprecated: use httputil.ErrCSRFInvalid.
	ErrCSRFInvalid = httputil.ErrCSRFInvalid
	// ErrCSRFConfig is an alias for httputil.ErrCSRFConfig.
	//
	// Deprecated: use httputil.ErrCSRFConfig.
	ErrCSRFConfig = httputil.ErrCSRFConfig
)

const (
	// defaultCSRFHeaderName mirrors httputil.DefaultCSRFHeaderName.
	//
	// Deprecated: use httputil.DefaultCSRFHeaderName.
	defaultCSRFHeaderName = httputil.DefaultCSRFHeaderName
)
