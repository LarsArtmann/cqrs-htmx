package main

import (
	"log"
	"net/http"

	cqrsauditlog "github.com/larsartmann/cqrs-htmx/auditlog/v4"
	cqrshealth "github.com/larsartmann/cqrs-htmx/health/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/samber/do/v2"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// hostStatusProvider adapts *projectionhost.Host to the health module's
// ProjectionStatusProvider (the same field mapping usermgmt uses internally;
// kept here because the e2e server has no usermgmt.Service).
type hostStatusProvider struct {
	host *projectionhost.Host
}

func (p hostStatusProvider) ProjectionStatuses() []cqrshtmx.ProjectionStatusEntry {
	if p.host == nil {
		return nil
	}

	workers := p.host.Status()
	entries := make([]cqrshtmx.ProjectionStatusEntry, len(workers))
	for i, w := range workers {
		entries[i] = cqrshtmx.ProjectionStatusEntry{
			Name:       w.Name,
			Status:     string(w.Status),
			Checkpoint: w.Checkpoint,
			Processed:  w.Processed,
			Errors:     w.Errors,
			Restarts:   w.Restarts,
			LagMillis:  w.Lag.Milliseconds(),
			LastError:  w.LastError,
		}
	}

	return entries
}

// bridgeProbeService is the single do-registered service behind
// POST /api/debug/invoke: invoking it through the injector makes the audit
// plugin record a lifecycle event, which the live viewer streams to
// connected browsers — the deterministic driver for the bridges smoke spec.
type bridgeProbeService struct{}

// mountBridges mounts the two bridge modules' browser surfaces onto the e2e
// server: the health module's go-health dashboard (per-projection checks
// over the projection host) and the auditlog module's live viewer (do
// lifecycle events over SSE). A mount failure logs and skips that surface
// only — the battery's other specs must never be hostage to a bridge mount
// (the compile stays the real API contract).
func mountBridges(mux *http.ServeMux, host *projectionhost.Host) {
	probe, err := cqrshealth.NewProbe(hostStatusProvider{host: host})
	if err != nil {
		log.Printf("bridges: health probe unavailable, skipping health dashboard: %v", err)

		return
	}

	mux.Handle("GET /bridges/health/ui", cqrshealth.NewDashboard(probe).Handler())

	auditSetup, err := cqrsauditlog.WithAuditLog(
		auditlog.Config{MaxEvents: 1_000},
		live.Config{Prefix: "/bridges/auditlog"},
	)
	if err != nil {
		log.Printf("bridges: auditlog setup failed, skipping viewer: %v", err)

		return
	}

	injector := do.NewWithOpts(auditSetup.Opts)
	do.Provide(injector, func(_ do.Injector) (*bridgeProbeService, error) {
		return &bridgeProbeService{}, nil
	})

	mux.Handle("/bridges/auditlog/", auditSetup.Viewer)

	mux.HandleFunc("POST /api/debug/invoke", func(w http.ResponseWriter, _ *http.Request) {
		if _, iErr := do.Invoke[*bridgeProbeService](injector); iErr != nil {
			http.Error(w, iErr.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusAccepted)
	})
}
