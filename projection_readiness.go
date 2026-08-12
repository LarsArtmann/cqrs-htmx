package cqrshtmx

import (
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// projectionDrainReady statuses mean the worker has finished its initial journal
// drain and registered its live handler (or gracefully stopped after doing so).
// These are the same terminal states waitForDrain treats as drain-complete.
//
//nolint:gochecknoglobals // immutable lookup table
var projectionDrainReady = map[string]struct{}{
	"live":    {},
	"stopped": {},
}

// ProjectionReadinessCheck returns a [NamedCheck] for [ReadinessHandler] that
// fails while any projection is still draining its initial journal backlog or
// has entered a terminal failure state.
//
// This is the readiness gate that makes async startup (usermgmt
// ServiceConfig.AsyncStartup) safe: the HTTP server binds immediately, but the
// reverse proxy's health check returns 503 until every projection worker
// reaches "live" state, then flips to 200. Point your load balancer at the
// endpoint serving this check (setup.Bundle mounts it at /health by default).
//
// A projection is considered ready when its status is "live" or "stopped"
// (caught up / live handler registered). It is not-ready while "idle",
// "running", "backoff", or "draining" (still catching up), and failed when
// "failed" (exhausted its restart budget).
//
//	mux.Handle("/ready", cqrshtmx.ReadinessHandler(
//	    cqrshtmx.ProjectionReadinessCheck(svc),
//	))
func ProjectionReadinessCheck(provider ProjectionStatusProvider) NamedCheck {
	return NewNamedCheck("projections", func() error {
		if provider == nil {
			return nil
		}

		var draining []string

		for _, s := range provider.ProjectionStatuses() {
			if _, ready := projectionDrainReady[s.Status]; ready {
				continue
			}

			if s.Status == "failed" {
				return errorfamily.NewInfrastructure("projection.failed",
					fmt.Sprintf("projection %q has failed: %s", s.Name, s.LastError))
			}

			// idle, running, backoff, draining — still catching up.
			draining = append(draining, s.Name)
		}

		if len(draining) > 0 {
			return errorfamily.NewTransient("projection.draining",
				"projections still draining: "+strings.Join(draining, ", "))
		}

		return nil
	})
}
