package cqrshtmx_test

import (
	"context"
	"net/http"
	"time"

	"github.com/larsartmann/cqrs-htmx/v4"
)

// This file is a compile-time contract test for the root package's public
// surface. It pins every symbol that the htmx, eventcatalog, and httperr
// subpackages now own, asserting the exact name and signature a consumer
// compiles against.
//
// It exists because those three packages were extracted from the root package
// and re-exported through *_reexport.go aliases: without this file, a forgotten
// alias would still pass every consumer build in this repo (nothing here
// imports the moved symbols under their old shape) and only surface when an
// external consumer upgraded. Breaking any line below is a breaking API change
// and belongs behind a major version, not in a refactor.

// HTMX request vocabulary.
var (
	_ func(context.Context, *cqrshtmx.HTMXRequest) context.Context = cqrshtmx.WithHTMX
	_ func(context.Context) *cqrshtmx.HTMXRequest                  = cqrshtmx.HTMXFromContext
	_ func(*http.Request) bool                                     = cqrshtmx.IsHTMXRequest
	_ func(*http.Request) bool                                     = cqrshtmx.IsBoosted
	_ func(*http.Request) bool                                     = cqrshtmx.IsHistoryRestore
	_ func(*http.Request) bool                                     = cqrshtmx.RenderPartial
	_ func(*http.Request) string                                   = cqrshtmx.HTMXTarget
	_ func(*http.Request) string                                   = cqrshtmx.HTMXTrigger
	_ func(*http.Request) string                                   = cqrshtmx.HTMXTriggerName
	_ func(*http.Request) string                                   = cqrshtmx.HTMXPrompt
	_ func(*http.Request) string                                   = cqrshtmx.HTMXCurrentURL
)

// HTMX script and extension serving.
var (
	_ func() string                     = cqrshtmx.HTMXVersion
	_ func() http.Handler               = cqrshtmx.HTMXScriptHandler
	_ func([]byte, string) http.Handler = cqrshtmx.HTMXScriptHandlerWith
	_ func(string) string               = cqrshtmx.HTMXScriptTag
	_ func(string) string               = cqrshtmx.HTMXCDNScriptTag
	_ func(string) http.Handler         = cqrshtmx.HTMXExtensionHandler
	_ func(...string) http.Handler      = cqrshtmx.HTMXExtensionsHandler
	_ func(string) string               = cqrshtmx.HTMXExtensionVersion
	_ func() []string                   = cqrshtmx.HTMXExtensionNames
	_ func(string) string               = cqrshtmx.HTMXExtensionCDNScriptTag
)

// Event catalog (Published Language).
var (
	_ func() *cqrshtmx.EventCatalog                                  = cqrshtmx.NewEventCatalog
	_ func(*cqrshtmx.EventCatalog) (http.HandlerFunc, error)         = cqrshtmx.EventCatalogHandler
	_ = (*cqrshtmx.EventCatalog)(nil)
	_ = (*cqrshtmx.EventMetadata)(nil)
	_ = (*cqrshtmx.PayloadField)(nil)
)

// Error mapping and the RFC 9457 payload.
var (
	_ func(error) int                                            = cqrshtmx.MapError
	_ func(error, int, bool) string                              = cqrshtmx.SafeDetail
	_ func(error) string                                         = cqrshtmx.ErrorCode
	_ func(error, int) error                                     = cqrshtmx.WithHTTPStatus
	_ func(error, time.Duration) error                           = cqrshtmx.WithRetryAfter
	_ func(error, *http.Request) cqrshtmx.StructuredError        = cqrshtmx.NewStructuredError
	_ func(error, context.Context) cqrshtmx.StructuredError      = cqrshtmx.NewStructuredErrorWithContext
	_ = (*cqrshtmx.StructuredError)(nil)
	_ func(cqrshtmx.HTTPStatusCarrier)                           = func(cqrshtmx.HTTPStatusCarrier) {}
	_ func(cqrshtmx.RetryAfterCarrier)                           = func(cqrshtmx.RetryAfterCarrier) {}
)

// Ready-made error handlers.
var (
	_ func(http.ResponseWriter, *http.Request, error)                = cqrshtmx.DefaultErrorHandler
	_ func(http.ResponseWriter, *http.Request, error, string)        = cqrshtmx.DefaultErrorHandlerWithRedirect
	_ func(http.ResponseWriter, *http.Request, error)                = cqrshtmx.DefaultErrorHandlerWithRequestID
	_ func(http.ResponseWriter, *http.Request, error, string)        = cqrshtmx.DefaultErrorHandlerWithRedirectAndRequestID
	_ func(http.ResponseWriter, *http.Request, error)                = cqrshtmx.JSONErrorHandler
	_ func(http.ResponseWriter, *http.Request, error, string)        = cqrshtmx.JSONErrorHandlerWithRedirect
	_ func(http.ResponseWriter, *http.Request, error)                = cqrshtmx.ProblemDetailsErrorHandler
	_ func(http.ResponseWriter, *http.Request, error, string)        = cqrshtmx.ProblemDetailsErrorHandlerWithRedirect
)

// Constants and sentinel errors.
var (
	_ cqrshtmx.SwapStrategy = cqrshtmx.SwapInnerHTML
	_ cqrshtmx.SwapStrategy = cqrshtmx.SwapOuterHTML
	_ cqrshtmx.SwapStrategy = cqrshtmx.SwapBeforeBegin
	_ cqrshtmx.SwapStrategy = cqrshtmx.SwapAfterBegin
	_ cqrshtmx.SwapStrategy = cqrshtmx.SwapBeforeEnd
	_ cqrshtmx.SwapStrategy = cqrshtmx.SwapAfterEnd
	_ cqrshtmx.SwapStrategy = cqrshtmx.SwapDelete
	_ cqrshtmx.SwapStrategy = cqrshtmx.SwapNone

	_ string        = cqrshtmx.HeaderTrue
	_ string        = cqrshtmx.HTMXCDNBaseURL
	_ string        = cqrshtmx.HTMXExtSSE
	_ string        = cqrshtmx.HTMXExtIdiomorph
	_ string        = cqrshtmx.DefaultLoginRedirect
	_ string        = cqrshtmx.CodeUnauthorized
	_ string        = cqrshtmx.CodeForbidden
	_ time.Duration = cqrshtmx.DefaultRetryAfter

	_ error = cqrshtmx.ErrUnauthorized
	_ error = cqrshtmx.ErrForbidden
	_ error = cqrshtmx.ErrDecodeFailed
	_ error = cqrshtmx.ErrDispatchFailed
	_ error = cqrshtmx.ErrEnforcerNil
	_ error = cqrshtmx.ErrValidationFailed
	_ error = cqrshtmx.ErrRequestTooLarge
	_ error = cqrshtmx.ErrMethodNotAllowed
)
