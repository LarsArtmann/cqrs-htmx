package main

import (
	"context"
	"log"
	"net/http"

	cqrsauditlog "github.com/larsartmann/cqrs-htmx/auditlog/v4"
	cqrshealth "github.com/larsartmann/cqrs-htmx/health/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	healthdashboard "github.com/larsartmann/go-health-dashboard"
	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/samber/do/v2"
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

// bridgesAuditMaxEvents caps the in-memory auditlog event ring so a long
// e2e battery run cannot grow the viewer's memory without bound.
const bridgesAuditMaxEvents = 1_000

// bridgeProbeService is the single do-registered service behind
// POST /api/debug/invoke: invoking it through the injector makes the audit
// plugin record a lifecycle event, which the live viewer streams to
// connected browsers — the deterministic driver for the bridges smoke spec.
type bridgeProbeService struct{}

// mountHealthDashboard mounts the go-health dashboard with the FULL route
// set under /bridges/health/*. RegisterRoutes (not the bare Handler) is
// load-bearing: Handler() serves only the page, and the page's live updates
// need the /sse route beside it — a page-only mount renders a dashboard whose
// connection pill is stuck on "Reconnecting" forever (caught by the bridges
// spec, 2026-11-08). WithEmbeddedDatastarSDK swaps the CDN script tag for a
// same-origin one so the battery stays hermetic (no jsdelivr dependency in
// CI or offline runs).
func mountHealthDashboard(mux *http.ServeMux, host *projectionhost.Host) error {
	probe, err := cqrshealth.NewProbe(hostStatusProvider{host: host})
	if err != nil {
		return err
	}

	dash := cqrshealth.NewDashboard(
		probe,
		healthdashboard.WithRoutes(healthdashboard.Routes{
			Dashboard:  "/bridges/health/ui",
			SSE:        "/bridges/health/sse",
			Favicon:    "/bridges/health/favicon.svg",
			DatastarJS: "/bridges/health/datastar.js",
			Liveness:   "/bridges/health/livez",
			Readiness:  "/bridges/health/readyz",
			Startup:    "/bridges/health/startupz",
		}),
		healthdashboard.WithEmbeddedDatastarSDK(),
	)
	dash.RegisterRoutes(mux)

	// Start launches the SSE pusher goroutine — without it the /sse route
	// answers 503 "push is not active" and the page can never go live (the
	// second half of the page-only-mount trap; the server is disposable per
	// run, so no Shutdown wiring is needed).
	if err := dash.Start(context.Background()); err != nil {
		return err
	}

	return nil
}

// mountBridges mounts the two bridge modules' browser surfaces onto the e2e
// server: the health module's go-health dashboard (per-projection checks
// over the projection host) and the auditlog module's live viewer (do
// lifecycle events over SSE). A mount failure logs and skips that surface
// only — the battery's other specs must never be hostage to a bridge mount
// (the compile stays the real API contract).
func mountBridges(mux *http.ServeMux, host *projectionhost.Host) {
	if err := mountHealthDashboard(mux, host); err != nil {
		log.Printf("bridges: health dashboard unavailable, skipping: %v", err)
	}

	auditSetup, err := cqrsauditlog.WithAuditLog(
		auditlog.Config{
			Enabled:              true,
			MaxEvents:            bridgesAuditMaxEvents,
			ContainerID:          "e2e-bridges",
			RunID:                "",
			OnEvent:              nil,
			InitialEventCapacity: 0,
		},
		live.Config{
			Prefix:             "/bridges/auditlog",
			Addr:               "",
			ReadHeaderTimeout:  0,
			HeartbeatInterval:  0,
			CORSAllowedOrigins: "",
			ReplayBufferSize:   0,
		},
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
