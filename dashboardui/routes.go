package dashboardui

import "net/http"

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

// Routes returns the manifest of HTTP routes this instance registers: the
// exact set [Dashboard.Handler] serves, honoring the detected capabilities
// and ReadOnly. The catch-all styled-404 fallback is not listed; every other
// served route is.
//
// The manifest exists for consumer-side routing decisions: building an
// allowlist for a reverse proxy or policy engine, diffing the surface across
// versions in upgrade tests, or documenting what a mounted dashboard exposes.
// Registration order mirrors [Dashboard.routes].
func (d *Dashboard) Routes() []Route {
	routes := []Route{
		{Method: http.MethodGet, Pattern: "/-/dashboard.css", Panel: "Assets"},
		{Method: http.MethodGet, Pattern: "/-/dashboard-tw.css", Panel: "Assets"},
		{Method: http.MethodGet, Pattern: "/-/dashboard.js", Panel: "Assets"},
		{Method: http.MethodGet, Pattern: "/-/htmx.js", Panel: "Assets"},
		{Method: http.MethodGet, Pattern: "/-/healthz", Panel: "Observability"},
		{Method: http.MethodGet, Pattern: "/-/readyz", Panel: "Observability"},
		{Method: http.MethodGet, Pattern: "/-/versionz", Panel: "Observability"},
		{Method: http.MethodGet, Pattern: "/", Panel: "Overview"},
	}

	if d.caps.EventBus {
		routes = append(routes, Route{Method: http.MethodGet, Pattern: "/-/events/stream", Panel: "Live updates"})
	}

	if d.caps.HasEventRead() {
		routes = append(routes,
			Route{Method: http.MethodGet, Pattern: "/events", Panel: "Events"},
			Route{Method: http.MethodGet, Pattern: "/events/{id}", Panel: "Events"},
		)
	}

	if d.caps.StreamReader || d.caps.EventSource {
		routes = append(routes, Route{Method: http.MethodGet, Pattern: "/aggregates", Panel: "Aggregates"})

		if d.caps.EventSource {
			routes = append(routes,
				Route{Method: http.MethodGet, Pattern: "/aggregates/{type}/{id}", Panel: "Aggregates"},
			)
		}
	}

	if d.caps.ProjectionHost {
		routes = append(routes,
			Route{Method: http.MethodGet, Pattern: "/projections", Panel: "Projections"},
			Route{Method: http.MethodGet, Pattern: "/projections/{name}", Panel: "Projections"},
			Route{Method: http.MethodGet, Pattern: "/-/partials/projection-health", Panel: "Projections"},
		)

		if !d.config.ReadOnly {
			routes = append(routes,
				Route{Method: http.MethodPost, Pattern: "/projections/{name}/reset", Panel: "Projections", Write: true},
			)
		}
	}

	if d.caps.DeadLetterStore || d.caps.ProjectionHost {
		routes = append(routes,
			Route{Method: http.MethodGet, Pattern: "/dead-letters", Panel: "Dead letters"},
			Route{Method: http.MethodGet, Pattern: "/dead-letters/{projection}", Panel: "Dead letters"},
			Route{Method: http.MethodGet, Pattern: "/dead-letters/{projection}/{eventID}", Panel: "Dead letters"},
		)

		if !d.config.ReadOnly && d.caps.ProjectionHost {
			routes = append(routes,
				Route{Method: http.MethodPost, Pattern: "/dead-letters/{projection}/replay", Panel: "Dead letters", Write: true},
			)
		}

		if !d.config.ReadOnly && d.caps.DeadLetterStore {
			routes = append(routes,
				Route{Method: http.MethodPost, Pattern: "/dead-letters/{projection}/{eventID}/delete", Panel: "Dead letters", Write: true},
				Route{Method: http.MethodPost, Pattern: "/dead-letters/{projection}/purge", Panel: "Dead letters", Write: true},
			)
		}
	}

	if d.caps.CommandJournal {
		routes = append(routes,
			Route{Method: http.MethodGet, Pattern: "/commands", Panel: "Commands"},
			Route{Method: http.MethodGet, Pattern: "/commands/{id}", Panel: "Commands"},
		)
	}

	if d.caps.QueryJournal {
		routes = append(routes,
			Route{Method: http.MethodGet, Pattern: "/queries", Panel: "Queries"},
			Route{Method: http.MethodGet, Pattern: "/queries/{id}", Panel: "Queries"},
		)
	}

	if d.caps.EventSource {
		routes = append(routes,
			Route{Method: http.MethodGet, Pattern: "/time-travel", Panel: "Time travel"},
			Route{Method: http.MethodGet, Pattern: "/time-travel/{type}/{id}", Panel: "Time travel"},
		)
	}

	if d.caps.SnapshotStore {
		routes = append(routes,
			Route{Method: http.MethodGet, Pattern: "/snapshots", Panel: "Snapshots"},
			Route{Method: http.MethodGet, Pattern: "/snapshots/{type}/{id}", Panel: "Snapshots"},
		)

		if !d.config.ReadOnly {
			routes = append(routes,
				Route{Method: http.MethodPost, Pattern: "/snapshots/{type}/{id}/delete", Panel: "Snapshots", Write: true},
			)
		}
	}

	return routes
}
