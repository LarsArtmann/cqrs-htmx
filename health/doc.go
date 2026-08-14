// Package health bridges cqrs-htmx projection health into the go-health
// ecosystem: a [gohealth.Probe] with one named check per projection, and a
// go-health-dashboard for visualizing it.
//
// A projection is healthy when its worker is "live" or "stopped" (the same
// semantics as cqrshtmx.ProjectionReadinessCheck). While draining
// ("idle", "running", "backoff", "draining") it reports a transient error;
// once failed it reports an infrastructure error including the last error.
//
// Basic wiring:
//
//	probe, err := health.NewProbe(svc,
//		gohealth.WithVersion("1.2.3"),
//		gohealth.WithCriticalServices("user-read-model", "casbin-projection"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	probe.RegisterRoutes(mux, gohealth.DefaultRoutes())
//
// The optional dashboard (separate import cost only when used):
//
//	dash := health.NewDashboard(probe, healthdashboard.WithTitle("My App"))
//	mux.Handle("GET /health/ui", dash.Handler())
//
// Applications already using samber/do v2 should keep their injector and add
// projection checks via [Recorder] instead of [NewProbe], so their own
// services and the projections are checked in one batch:
//
//	probe := gohealth.New(injector, gohealth.WithHealthRecorder(health.Recorder(svc)))
//
// *usermgmt.Service and *usermgmt.EventSourcedSetup satisfy
// [ProjectionStatusProvider] directly.
package health
