package dashboardui

import (
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
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

// routeTable is the dashboard's complete route surface in registration order.
// Capabilities gate the conditional rows; write rows additionally require
// !ReadOnly.
func (d *Dashboard) routeTable() []routeSpec {
	return []routeSpec{
		// Static assets.
		{method: http.MethodGet, pattern: "/-/dashboard.css", panel: "Assets", handler: d.serveCSS()},
		{method: http.MethodGet, pattern: "/-/dashboard-tw.css", panel: "Assets", handler: d.twCSS.ServeHTTP},
		{method: http.MethodGet, pattern: "/-/dashboard.js", panel: "Assets", handler: d.serveJS()},
		{
			method:    http.MethodGet,
			pattern:   "/-/htmx.js",
			panel:     "Assets",
			handler:   serveHTMXScript(),
			guardMode: guardNever,
		},

		// Observability probes (unguarded: load balancers and k8s need
		// access; versionz is config-revealing and opts in via
		// VersionzRequireAuth).
		{
			method:    http.MethodGet,
			pattern:   "/-/healthz",
			panel:     "Observability",
			handler:   d.healthzHandler,
			guardMode: guardNever,
		},
		{
			method:    http.MethodGet,
			pattern:   "/-/readyz",
			panel:     "Observability",
			handler:   d.readyzHandler,
			guardMode: guardNever,
		},
		{
			method:    http.MethodGet,
			pattern:   "/-/versionz",
			panel:     "Observability",
			handler:   d.versionzHandler,
			guardMode: guardVersionz,
		},

		// Overview (always available).
		{method: http.MethodGet, pattern: "/", panel: "Overview", handler: d.overviewHandler},

		// SSE live updates.
		{
			method:  http.MethodGet,
			pattern: "/-/events/stream",
			panel:   "Live updates",
			handler: d.sseHandler(),
			when:    whenCaps(func(caps Capabilities) bool { return caps.EventBus }),
		},

		// Event Stream Browser.
		{
			method:  http.MethodGet,
			pattern: "/events",
			panel:   "Events",
			handler: d.eventsIndexHandler,
			when:    whenCaps(Capabilities.HasEventRead),
		},
		{
			method:  http.MethodGet,
			pattern: "/events/{id}",
			panel:   "Events",
			handler: d.eventDetailHandler,
			when:    whenCaps(Capabilities.HasEventRead),
		},

		// Aggregate Browser.
		{
			method:  http.MethodGet,
			pattern: "/aggregates",
			panel:   "Aggregates",
			handler: d.aggregatesIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.StreamReader || caps.EventSource }),
		},
		{
			method:  http.MethodGet,
			pattern: "/aggregates/{type}/{id}",
			panel:   "Aggregates",
			handler: d.aggregateDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.EventSource }),
		},

		// Projection Dashboard.
		{
			method:  http.MethodGet,
			pattern: "/projections",
			panel:   "Projections",
			handler: d.projectionsIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.ProjectionHost }),
		},
		{
			method:  http.MethodGet,
			pattern: "/projections/{name}",
			panel:   "Projections",
			handler: d.projectionDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.ProjectionHost }),
		},
		{
			method:  http.MethodGet,
			pattern: "/-/partials/projection-health",
			panel:   "Projections",
			handler: d.projectionHealthPartialHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.ProjectionHost }),
		},
		{
			method:  http.MethodPost,
			pattern: "/projections/{name}/reset",
			panel:   "Projections",
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
			panel:   "Dead letters",
			handler: d.dlqIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.DeadLetterStore || caps.ProjectionHost }),
		},
		{
			method:  http.MethodGet,
			pattern: "/dead-letters/{projection}",
			panel:   "Dead letters",
			handler: d.dlqDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.DeadLetterStore || caps.ProjectionHost }),
		},
		{
			method:  http.MethodGet,
			pattern: "/dead-letters/{projection}/{eventID}",
			panel:   "Dead letters",
			handler: d.dlqEntryDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.DeadLetterStore || caps.ProjectionHost }),
		},
		{
			method:  http.MethodPost,
			pattern: "/dead-letters/{projection}/replay",
			panel:   "Dead letters",
			handler: d.dlqReplayHandler,
			write:   true,
			when: func(caps Capabilities, readOnly bool) bool {
				return caps.ProjectionHost && !readOnly
			},
		},
		{
			method:  http.MethodPost,
			pattern: "/dead-letters/{projection}/{eventID}/delete",
			panel:   "Dead letters",
			handler: d.dlqDeleteHandler,
			write:   true,
			when: func(caps Capabilities, readOnly bool) bool {
				return caps.DeadLetterStore && !readOnly
			},
		},
		{
			method:  http.MethodPost,
			pattern: "/dead-letters/{projection}/purge",
			panel:   "Dead letters",
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
			panel:   "Commands",
			handler: d.commandsIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.CommandJournal }),
		},
		{
			method:  http.MethodGet,
			pattern: "/commands/{id}",
			panel:   "Commands",
			handler: d.commandDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.CommandJournal }),
		},

		// Query Audit.
		{
			method:  http.MethodGet,
			pattern: "/queries",
			panel:   "Queries",
			handler: d.queriesIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.QueryJournal }),
		},
		{
			method:  http.MethodGet,
			pattern: "/queries/{id}",
			panel:   "Queries",
			handler: d.queryDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.QueryJournal }),
		},

		// Time-Travel.
		{
			method:  http.MethodGet,
			pattern: "/time-travel",
			panel:   "Time travel",
			handler: d.timeTravelIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.EventSource }),
		},
		{
			method:  http.MethodGet,
			pattern: "/time-travel/{type}/{id}",
			panel:   "Time travel",
			handler: d.timeTravelDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.EventSource }),
		},

		// Snapshot Inspector.
		{
			method:  http.MethodGet,
			pattern: "/snapshots",
			panel:   "Snapshots",
			handler: d.snapshotsIndexHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.SnapshotStore }),
		},
		{
			method:  http.MethodGet,
			pattern: "/snapshots/{type}/{id}",
			panel:   "Snapshots",
			handler: d.snapshotDetailHandler,
			when:    whenCaps(func(caps Capabilities) bool { return caps.SnapshotStore }),
		},
		{
			method:  http.MethodPost,
			pattern: "/snapshots/{type}/{id}/delete",
			panel:   "Snapshots",
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
	default: // guardAlways
		return d.guard(spec.handler)
	}
}
