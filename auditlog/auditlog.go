package auditlog

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"
	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/samber/do/v2"
)

// Setup is the result of [WithAuditLog]: the injector option, the plugin
// handle (for reports and exports), and the live viewer.
type Setup struct {
	// Opts wires the audit plugin into a samber/do container:
	// pass it to do.NewWithOpts.
	Opts *do.InjectorOpts

	// Plugin is the audit-log plugin handle: Report(), WriteReportJSON,
	// WriteMermaid, export helpers, ...
	Plugin *auditlog.Plugin

	// Viewer is the live dashboard (an http.Handler): HTML UI, JSON report
	// API, SSE event stream, and export endpoints under its configured
	// prefix. Mount it with mux.Handle("<prefix>/", setup.Viewer).
	Viewer *live.Server
}

// WithAuditLog builds the audit plugin wired to a live SSE viewer hub and
// returns everything a cqrs-htmx samber/do application needs in one call.
//
// auditCfg configures recording (MaxEvents, RunID, ...); liveCfg configures
// the viewer (Prefix, ReplayBufferSize, HeartbeatInterval, ...). The viewer
// is enabled and its hub attached automatically — OnEvent and Enabled on
// auditCfg are overwritten.
func WithAuditLog(auditCfg auditlog.Config, viewerCfg live.Config) (*Setup, error) {
	viewer, plugin, err := live.New(auditCfg, viewerCfg)
	if err != nil {
		return nil, errorfamily.Wrap(err, errorfamily.Orchestration,
			"cqrshtmx.auditlog.setup_failed", "build auditlog plugin and viewer")
	}

	return &Setup{
		Opts:   plugin.Opts(),
		Plugin: plugin,
		Viewer: viewer,
	}, nil
}

// Shutdown drains the live viewer for MOUNTED mode: it flushes the forwarder
// (when one is attached) and drains the SSE broadcaster, disconnecting every
// viewer with a clean end-of-stream. It works without ListenAndServe — the
// documented mount path (mux.Handle("<prefix>/", setup.Viewer)) owns the
// HTTP server, and only the viewer's own plumbing is drained here.
//
// Call it from the application's shutdown path:
//
//	setup.SignalComplete()      // mark the container lifecycle finished
//	_ = setup.Shutdown(ctx)     // flush + drain the viewer
//	_ = injector.Shutdown()
//
// Returns ctx.Err() when the context fires before the drain completes.
func (s *Setup) Shutdown(ctx context.Context) error {
	return s.Viewer.Shutdown(ctx) //nolint:wrapcheck // passthrough: caller checks ctx.Err() sentinels (context.Canceled)
}

// SignalComplete marks the container lifecycle as finished: SSE handlers
// waiting on the completion marker unblock and send the final report before
// [Setup.Shutdown] drains them. Optional but recommended before Shutdown.
func (s *Setup) SignalComplete() {
	s.Viewer.SignalComplete()
}
