package cqrshtmx

import (
	"github.com/larsartmann/httputil"
)

// Server-Timing now lives in httputil. These aliases preserve backward
// compatibility for cqrs-htmx consumers.
//
// Deprecated: import github.com/larsartmann/httputil directly. These aliases
// will be removed in cqrs-htmx v5.

// ServerTiming is an alias for httputil.ServerTiming.
//
// Deprecated: use httputil.ServerTiming.
type ServerTiming = httputil.ServerTiming

var (
	// ServerTimingMiddleware is an alias for httputil.ServerTimingMiddleware.
	//
	// Deprecated: use httputil.ServerTimingMiddleware.
	ServerTimingMiddleware = httputil.ServerTimingMiddleware
	// ServerTimingMiddlewareWhen is an alias for httputil.ServerTimingMiddlewareWhen.
	//
	// Deprecated: use httputil.ServerTimingMiddlewareWhen.
	ServerTimingMiddlewareWhen = httputil.ServerTimingMiddlewareWhen
	// ServerTimingFromContext is an alias for httputil.ServerTimingFromContext.
	//
	// Deprecated: use httputil.ServerTimingFromContext.
	ServerTimingFromContext = httputil.ServerTimingFromContext
	// WithServerTiming is an alias for httputil.WithServerTiming.
	//
	// Deprecated: use httputil.WithServerTiming.
	WithServerTiming = httputil.WithServerTiming
	// RecordServerTiming is an alias for httputil.RecordServerTiming.
	//
	// Deprecated: use httputil.RecordServerTiming.
	RecordServerTiming = httputil.RecordServerTiming
	// MeasureServerTiming is an alias for httputil.MeasureServerTiming.
	//
	// Deprecated: use httputil.MeasureServerTiming.
	MeasureServerTiming = httputil.MeasureServerTiming
)

// headerServerTiming mirrors httputil.HeaderServerTiming.
//
// Deprecated: use httputil.HeaderServerTiming.
const headerServerTiming = httputil.HeaderServerTiming
