package dashboardui

import (
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// Panel names used across the route table and the [Dashboard.Routes] manifest.
const (
	panelAssets        = "Assets"
	panelObservability = "Observability"
	panelOverview      = "Overview"
	panelLiveUpdates   = "Live updates"
	panelEvents        = "Events"
	panelAggregates    = "Aggregates"
	panelProjections   = "Projections"
	panelDeadLetters   = "Dead letters"
	panelCommands      = "Commands"
	panelQueries       = "Queries"
	panelTimeTravel    = "Time travel"
	panelSnapshots     = "Snapshots"
	panelTelemetry     = "Telemetry"
)

// Route describes one HTTP route a [Dashboard] serves — the manifest entry
// behind [Dashboard.Routes].
type Route struct {
	// Method is the HTTP method the route answers ("GET" or "POST").
	Method string

	// Pattern is the consumer-facing path relative to the mount point;
	// wildcards appear in braces, e.g. "/events/{id}".
	Pattern string

	// Panel names the dashboard section the route belongs to ("Events",
	// "Dead letters", "Observability", ...).
	Panel string

	// Write marks a mutating route. Write routes are absent from the manifest
	// — and from the mux — whenever Config.ReadOnly is set.
	Write bool
}

// routeGuard selects how a route relates to the dashboard's authorizer.
type routeGuard uint8

const (
	// guardAlways wraps the handler in d.guard. It is the zero value, so a
	// row that forgets its guard mode fails closed.
	guardAlways routeGuard = iota

	// guardNever serves the handler unguarded: observability probes that
	// load balancers need, and the vendored HTMX script.
	guardNever

	// guardVersionz serves versionz publicly unless
	// Config.VersionzRequireAuth opts it into the guard.
	guardVersionz
)

// routeSpec is one row of the dashboard's route table — the single source of
// truth the mux registration ([Dashboard.routes]) and the manifest
// ([Dashboard.Routes]) are both generated from. Rows omit `when` for
// unconditional routes and `write` for read-only ones.
type routeSpec struct {
	method    string
	pattern   string
	panel     string
	handler   http.HandlerFunc
	guardMode routeGuard
	write     bool
	when      func(caps Capabilities, readOnly bool) bool
}

// whenCaps lifts a pure capability predicate into a routeSpec condition.
func whenCaps(predicate func(Capabilities) bool) func(Capabilities, bool) bool {
	return func(caps Capabilities, _ bool) bool {
		return predicate(caps)
	}
}

// serveHTMXScript adapts the vendored HTMX script handler once, so the route
// table can carry it like any dashboard handler.
func serveHTMXScript() http.HandlerFunc {
	script := cqrshtmx.HTMXScriptHandler()

	return script.ServeHTTP
}

// routeTable is the dashboard's complete route surface in registration order:
// the unconditional rows first, then the capability-gated ones. Capabilities
// gate the conditional rows; write rows additionally require !ReadOnly.
func (d *Dashboard) routeTable() []routeSpec {
	routes := append(d.unconditionalRoutes(), d.capabilityRoutes()...)

	return append(routes, d.inspectionRoutes()...)
}

// unconditionalRoutes are always registered, whatever the store provides.
func (d *Dashboard) unconditionalRoutes() []routeSpec {
	return []routeSpec{
		// Static assets.
		{method: http.MethodGet, pattern: "/-/dashboard.css", panel: panelAssets, handler: d.serveCSS()},
		{method: http.MethodGet, pattern: "/-/dashboard-tw.css", panel: panelAssets, handler: d.twCSS.ServeHTTP},
		{method: http.MethodGet, pattern: "/-/dashboard.js", panel: panelAssets, handler: d.serveJS()},
		{
			method:    http.MethodGet,
			pattern:   "/-/htmx.js",
			panel:     panelAssets,
			handler:   serveHTMXScript(),
			guardMode: guardNever,
		},

		// Observability probes (unguarded: load balancers and k8s need
		// access; versionz is config-revealing and opts in via
		// VersionzRequireAuth).
		{
			method:    http.MethodGet,
			pattern:   "/-/healthz",
			panel:     panelObservability,
			handler:   d.healthzHandler,
			guardMode: guardNever,
		},
		{
			method:    http.MethodGet,
			pattern:   "/-/readyz",
			panel:     panelObservability,
			handler:   d.readyzHandler,
			guardMode: guardNever,
		},
		{
			method:    http.MethodGet,
			pattern:   "/-/versionz",
			panel:     panelObservability,
			handler:   d.versionzHandler,
			guardMode: guardVersionz,
		},

		// Overview (always available).
		{method: http.MethodGet, pattern: "/", panel: panelOverview, handler: d.overviewHandler},
	}
}

// capabilityRoutes light up panel by panel as the configured store implements
// the matching introspection interface.
func (d *Dashboard) capabilityRoutes() []routeSpec {
	return []routeSpec{
		// SSE live updates.
		{
			method:  http.MethodGet,
			pattern: "/-/events/stream",
			panel:   panelLiveUpdates,
			handler: d.sseHandler(),
			when:    whenCaps(func(caps Capabilities) bool { return caps.EventBus }),
		},

		// Event Stream Browser.
		{
			method:  http.MethodGet,
			pattern: "/events",
			panel:   panelEvents,
			handler: d.eventsIndexHandler,
			when:    whenCaps(Capabilities.HasEventRead),
		},
		{
			method:  http.MethodGet,
			pattern: "/events/{id}",
			panel:   panelEvents,
			handler: d.eventDetailHandler,
			when:    whenCaps(Capabilities.HasEventRead),
		},

		// Aggregate Browser.
		{
			method:  http.MethodGet,
			pattern: "/aggregates",
			panel:   panelAggregates,
			handler: d.aggregatesIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.StreamReader || caps.EventSource }),
		},
		{
			method:  http.MethodGet,
			pattern: "/aggregates/{type}/{id}",
			panel:   panelAggregates,
			handler: d.aggregateDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.EventSource }),
		},

		// Projection Dashboard.
		{
			method:  http.MethodGet,
			pattern: "/projections",
			panel:   panelProjections,
			handler: d.projectionsIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.ProjectionHost }),
		},
		{
			method:  http.MethodGet,
			pattern: "/projections/{name}",
			panel:   panelProjections,
			handler: d.projectionDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.ProjectionHost }),
		},
		{
			method:  http.MethodGet,
			pattern: "/-/partials/projection-health",
			panel:   panelProjections,
			handler: d.projectionHealthPartialHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.ProjectionHost }),
		},
		{
			method:  http.MethodPost,
			pattern: "/projections/{name}/reset",
			panel:   panelProjections,
			handler: d.projectionResetHandler,
			write:   true,
			when: func(caps Capabilities, readOnly bool) bool {
				return caps.ProjectionHost && !readOnly
			},
		},

		// Dead-Letter Queue.
		{
			method:  http.MethodGet,
			pattern: "/dead-letters",
			panel:   panelDeadLetters,
			handler: d.dlqIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.DeadLetterStore || caps.ProjectionHost }),
		},
		{
			method:  http.MethodGet,
			pattern: "/dead-letters/{projection}",
			panel:   panelDeadLetters,
			handler: d.dlqDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.DeadLetterStore || caps.ProjectionHost }),
		},
		{
			method:  http.MethodGet,
			pattern: "/dead-letters/{projection}/{eventID}",
			panel:   panelDeadLetters,
			handler: d.dlqEntryDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.DeadLetterStore || caps.ProjectionHost }),
		},
		{
			method:  http.MethodPost,
			pattern: "/dead-letters/{projection}/replay",
			panel:   panelDeadLetters,
			handler: d.dlqReplayHandler,
			write:   true,
			when: func(caps Capabilities, readOnly bool) bool {
				return caps.ProjectionHost && !readOnly
			},
		},
		{
			method:  http.MethodPost,
			pattern: "/dead-letters/{projection}/{eventID}/delete",
			panel:   panelDeadLetters,
			handler: d.dlqDeleteHandler,
			write:   true,
			when: func(caps Capabilities, readOnly bool) bool {
				return caps.DeadLetterStore && !readOnly
			},
		},
		{
			method:  http.MethodPost,
			pattern: "/dead-letters/{projection}/purge",
			panel:   panelDeadLetters,
			handler: d.dlqPurgeHandler,
			write:   true,
			when: func(caps Capabilities, readOnly bool) bool {
				return caps.DeadLetterStore && !readOnly
			},
		},

		// Command Audit.
		{
			method:  http.MethodGet,
			pattern: "/commands",
			panel:   panelCommands,
			handler: d.commandsIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.CommandJournal }),
		},
		{
			method:  http.MethodGet,
			pattern: "/commands/{id}",
			panel:   panelCommands,
			handler: d.commandDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.CommandJournal }),
		},

		// Query Audit.
		{
			method:  http.MethodGet,
			pattern: "/queries",
			panel:   panelQueries,
			handler: d.queriesIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.QueryJournal }),
		},
		{
			method:  http.MethodGet,
			pattern: "/queries/{id}",
			panel:   panelQueries,
			handler: d.queryDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.QueryJournal }),
		},
	}
}

// inspectionRoutes cover the introspection panels: time-travel, read-only
// telemetry, and snapshots.
func (d *Dashboard) inspectionRoutes() []routeSpec {
	return []routeSpec{
		// Time-Travel.
		{
			method:  http.MethodGet,
			pattern: "/time-travel",
			panel:   panelTimeTravel,
			handler: d.timeTravelIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.EventSource }),
		},
		{
			method:  http.MethodGet,
			pattern: "/time-travel/{type}/{id}",
			panel:   panelTimeTravel,
			handler: d.timeTravelDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.EventSource }),
		},

		// Telemetry (read-only): topology, query placements, engine stats.
		{
			method:  http.MethodGet,
			pattern: "/telemetry",
			panel:   panelTelemetry,
			handler: d.telemetryHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.Telemetry }),
		},

		// Snapshot Inspector.
		{
			method:  http.MethodGet,
			pattern: "/snapshots",
			panel:   panelSnapshots,
			handler: d.snapshotsIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.SnapshotStore }),
		},
		{
			method:  http.MethodGet,
			pattern: "/snapshots/{type}/{id}",
			panel:   panelSnapshots,
			handler: d.snapshotDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.SnapshotStore }),
		},
		{
			method:  http.MethodPost,
			pattern: "/snapshots/{type}/{id}/delete",
			panel:   panelSnapshots,
			handler: d.snapshotDeleteHandler,
			write:   true,
			when: func(caps Capabilities, readOnly bool) bool {
				return caps.SnapshotStore && !readOnly
			},
		},
	}
}

// Routes returns the manifest of HTTP routes this instance registers: the
// exact set [Dashboard.Handler] serves, honoring the detected capabilities
// and ReadOnly — both are generated from the same route table. The catch-all
// styled-404 fallback is not listed; every other served route is.
//
// The manifest exists for consumer-side routing decisions: building an
// allowlist for a reverse proxy or policy engine, diffing the surface across
// versions in upgrade tests, or documenting what a mounted dashboard exposes.
func (d *Dashboard) Routes() []Route {
	specs := d.routeTable()
	routes := make([]Route, 0, len(specs))

	for _, spec := range specs {
		if !spec.enabled(d.caps, d.config.ReadOnly) {
			continue
		}

		routes = append(routes, Route{
			Method:  spec.method,
			Pattern: spec.pattern,
			Panel:   spec.panel,
			Write:   spec.write,
		})
	}

	return routes
}

func (s routeSpec) enabled(caps Capabilities, readOnly bool) bool {
	return s.when == nil || s.when(caps, readOnly)
}

// registrationPattern maps a consumer-facing pattern to its ServeMux
// registration form; only the overview root differs ("/" registers as
// "/{$}" so it does not shadow every route).
func registrationPattern(pattern string) string {
	if pattern == "/" {
		return "/{$}"
	}

	return pattern
}

// wrapped applies the spec's guard mode to its handler.
func (d *Dashboard) wrapped(spec routeSpec) http.Handler {
	switch spec.guardMode {
	case guardVersionz:
		return d.versionzRoute()
	case guardNever:
		return spec.handler
	case guardAlways:
		return d.guard(spec.handler)
	}

	// Unreachable — the switch covers every routeGuard — but fails closed
	// like the guardAlways case above if one is ever added.
	return d.guard(spec.handler)
}
