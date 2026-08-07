package cqrshtmx

import (
	"github.com/larsartmann/httputil/server_timing"
)

// Server-Timing now lives in httputil/server_timing. These aliases preserve
// backward compatibility for cqrs-htmx consumers.
//
// Deprecated: import github.com/larsartmann/httputil/server_timing directly.
// These aliases will be removed in cqrs-htmx v5.

// ServerTiming is an alias for servertiming.ServerTiming.
//
// Deprecated: use servertiming.ServerTiming.
type ServerTiming = servertiming.ServerTiming

var (
	// ServerTimingMiddleware is an alias for servertiming.ServerTimingMiddleware.
	//
	// Deprecated: use servertiming.ServerTimingMiddleware.
	ServerTimingMiddleware = servertiming.ServerTimingMiddleware
	// ServerTimingMiddlewareWhen is an alias for servertiming.ServerTimingMiddlewareWhen.
	//
	// Deprecated: use servertiming.ServerTimingMiddlewareWhen.
	ServerTimingMiddlewareWhen = servertiming.ServerTimingMiddlewareWhen
	// ServerTimingFromContext is an alias for servertiming.ServerTimingFromContext.
	//
	// Deprecated: use servertiming.ServerTimingFromContext.
	ServerTimingFromContext = servertiming.ServerTimingFromContext
	// WithServerTiming is an alias for servertiming.WithServerTiming.
	//
	// Deprecated: use servertiming.WithServerTiming.
	WithServerTiming = servertiming.WithServerTiming
	// RecordServerTiming is an alias for servertiming.RecordServerTiming.
	//
	// Deprecated: use servertiming.RecordServerTiming.
	RecordServerTiming = servertiming.RecordServerTiming
	// MeasureServerTiming is an alias for servertiming.MeasureServerTiming.
	//
	// Deprecated: use servertiming.MeasureServerTiming.
	MeasureServerTiming = servertiming.MeasureServerTiming
)

// headerServerTiming mirrors servertiming.HeaderServerTiming.
//
// Deprecated: use servertiming.HeaderServerTiming.
const headerServerTiming = servertiming.HeaderServerTiming
