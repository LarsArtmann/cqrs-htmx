package cqrshtmx

import (
	"github.com/larsartmann/httputil"
)

// CSRF core now lives in httputil. These aliases preserve backward
// compatibility for cqrs-htmx consumers.

type CSRFConfig = httputil.CSRFConfig
type ErrorHandler = httputil.CSRFErrorHandler

var (
	CSRFMiddleware               = httputil.CSRFMiddleware
	CSRFResponseHeaderMiddleware = httputil.CSRFResponseHeaderMiddleware
	CSRFTokenFromContext         = httputil.CSRFTokenFromContext
	WithCSRFToken                = httputil.WithCSRFToken
	CSRFTestToken                = httputil.CSRFTestToken
	InvalidateCSRFCookie         = httputil.InvalidateCSRFCookie
	CSRFTokenHTMLMeta            = httputil.CSRFTokenHTMLMeta
	CSRFTokenHXHeaders           = httputil.CSRFTokenHXHeaders
	CSRFTokenFormField           = httputil.CSRFTokenFormField
	ForbiddenErrorHandler        = httputil.ForbiddenCSRFHandler
	ErrCSRFInvalid               = httputil.ErrCSRFInvalid
)

const (
	defaultCSRFCookieName = httputil.DefaultCSRFCookieName
	defaultCSRFHeaderName = httputil.DefaultCSRFHeaderName
	defaultCSRFFieldName  = httputil.DefaultCSRFFieldName
)
