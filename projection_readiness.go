package cqrshtmx

import (
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Projection worker states as reported by [ProjectionStatusEntry.Status].
// They mirror projectionhost/v4's WorkerStatus values one-to-one; the string
// form is the published wire vocabulary (ProjectionStatusHandler JSON, health
// bridges, readiness gates).
const (
	// ProjectionStatusIdle means the worker is registered but has not started.
	ProjectionStatusIdle = "idle"
	// ProjectionStatusRunning means the worker is actively processing events
	// (replay or catch-up, not yet live).
	ProjectionStatusRunning = "running"
	// ProjectionStatusBackoff means the worker is waiting before a restart.
	ProjectionStatusBackoff = "backoff"
	// ProjectionStatusDraining means the worker is shutting down, waiting for
	// in-flight events.
	ProjectionStatusDraining = "draining"
	// ProjectionStatusLive means the worker caught up on replay and is
	// processing live events.
	ProjectionStatusLive = "live"
	// ProjectionStatusStopped means the worker was gracefully stopped.
	ProjectionStatusStopped = "stopped"
	// ProjectionStatusFailed means the worker exhausted its restart budget.
	ProjectionStatusFailed = "failed"
)

// ProjectionDrainReady reports whether a projection worker state means the
// worker has finished catching up: "live" (replay done, live handler
// registered) and "stopped" (gracefully stopped after draining). Every other
// state (idle, running, backoff, draining, failed) reports false.
//
// This is the exact predicate [ProjectionReadinessCheck] applies per
// projection; health bridges can use it to classify check results the same
// way.
func ProjectionDrainReady(status string) bool {
	_, ready := projectionDrainReady[status]

	return ready
}

// projectionDrainReady statuses mean the worker has finished its initial journal
// drain and registered its live handler (or gracefully stopped after doing so).
// These are the same terminal states waitForDrain treats as drain-complete.
//
//nolint:gochecknoglobals // immutable lookup table
var projectionDrainReady = map[string]struct{}{
	ProjectionStatusLive:    {},
	ProjectionStatusStopped: {},
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
// A projection is considered ready when its status is [ProjectionStatusLive]
// or [ProjectionStatusStopped] (caught up / live handler registered). It is
// not-ready while [ProjectionStatusIdle], [ProjectionStatusRunning],
// [ProjectionStatusBackoff], or [ProjectionStatusDraining] (still catching
// up), and failed when [ProjectionStatusFailed] (exhausted its restart
// budget). See [ProjectionDrainReady] for the predicate as a function.
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
			if ProjectionDrainReady(s.Status) {
				continue
			}

			if s.Status == ProjectionStatusFailed {
				return errorfamily.NewInfrastructure("projection.failed",
					fmt.Sprintf("projection %q has failed: %s", s.Name, s.LastError))
			}

			// idle, running, backoff, draining: still catching up.
			draining = append(draining, s.Name)
		}

		if len(draining) > 0 {
			return errorfamily.NewTransient("projection.draining",
				"projections still draining: "+strings.Join(draining, ", "))
		}

		return nil
	})
}
