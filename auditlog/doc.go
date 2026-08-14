// Package auditlog wires samber-do-auditlog into a cqrs-htmx application's
// samber/do v2 container in one call: the DI audit plugin plus the live HTML
// viewer with SSE updates.
//
//	setup, err := auditlog.WithAuditLog(
//		auditlog.Config{MaxEvents: 10_000},
//		live.Config{Prefix: "/auditlog"},
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	injector := do.NewWithOpts(setup.Opts)
//	defer injector.Shutdown()
//
//	mux.Handle("/auditlog/", setup.Viewer) // live dashboard + JSON/SSE API
//
// Every do lifecycle event (service invoked, shutdown, ...) is recorded by
// the plugin and streamed to connected viewers. The plugin also satisfies
// go-health's HealthRecorder implicitly, so it composes with the
// cqrs-htmx/health module.
//
// Consumers who do not import this module pay zero dependency cost.
package auditlog
