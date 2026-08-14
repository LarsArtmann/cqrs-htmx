package health

import (
	gohealth "github.com/larsartmann/go-health"
	healthdashboard "github.com/larsartmann/go-health-dashboard"
)

// NewDashboard returns a go-health-dashboard for the probe: a live HTML view
// of every check (including the projection checks from [NewProbe]) with SSE
// push updates, plus JSON via the Accept header. Options are passed through
// to go-health-dashboard (WithTitle, WithBasePath, WithNonceExtractor for
// CSP setups, ...).
//
//	dash := health.NewDashboard(probe, healthdashboard.WithTitle("My App"))
//	mux.Handle("GET /health/ui", dash.Handler())
func NewDashboard(probe *gohealth.Probe, opts ...healthdashboard.Option) *healthdashboard.Dashboard {
	return healthdashboard.New(probe, opts...)
}
