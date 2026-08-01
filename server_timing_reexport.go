package cqrshtmx

import (
	"github.com/larsartmann/httputil"
)

// Server-Timing now lives in httputil. These aliases preserve backward
// compatibility for cqrs-htmx consumers.

type ServerTiming = httputil.ServerTiming

var (
	ServerTimingMiddleware     = httputil.ServerTimingMiddleware
	ServerTimingMiddlewareWhen = httputil.ServerTimingMiddlewareWhen
	ServerTimingFromContext    = httputil.ServerTimingFromContext
	WithServerTiming           = httputil.WithServerTiming
	RecordServerTiming         = httputil.RecordServerTiming
	MeasureServerTiming        = httputil.MeasureServerTiming
)

const headerServerTiming = httputil.HeaderServerTiming
