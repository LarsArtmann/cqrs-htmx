package auditlog

import (
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
