package dashboardui

import (
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// Handler returns the root HTTP handler for the dashboard. All internal
// routes are relative (no prefix). Use [Dashboard.Mount] if you want
// prefix stripping.
func (d *Dashboard) Handler() http.Handler { return d.routes() }

// Mount registers the dashboard under the given pattern on a ServeMux.
// The pattern is stripped from internal route paths.
//
//	mux := http.NewServeMux()
//	d.Mount(mux, "/dashboard/")
func (d *Dashboard) Mount(mux *http.ServeMux, pattern string) {
	mux.Handle(pattern, http.StripPrefix(pattern, d.routes()))
}

// Middleware returns the recommended middleware chain for the dashboard.
func (d *Dashboard) Middleware() func(http.Handler) http.Handler {
	return cqrshtmx.Chain(
		cqrshtmx.SecurityHeadersMiddleware,
		cqrshtmx.RecoveryMiddleware,
	)
}

func (d *Dashboard) routes() http.Handler {
	mux := http.NewServeMux()

	// Static assets
	mux.Handle("GET /-/dashboard.css", d.guard(d.serveCSS()))
	mux.Handle("GET /-/dashboard.js", d.guard(d.serveJS()))
	mux.Handle("GET /-/htmx.js", cqrshtmx.HTMXScriptHandler())

	// Overview (always available)
	mux.HandleFunc("GET /{$}", d.guard(d.overviewHandler))

	// Event Stream Browser
	if d.caps.hasEventRead() {
		mux.HandleFunc("GET /events", d.guard(d.eventsIndexHandler))
		mux.HandleFunc("GET /events/{id}", d.guard(d.eventDetailHandler))
	}

	// Aggregate Browser
	if d.caps.StreamReader || d.caps.EventSource {
		mux.HandleFunc("GET /aggregates", d.guard(d.aggregatesIndexHandler))
		if d.caps.EventSource {
			mux.HandleFunc("GET /aggregates/{type}/{id}", d.guard(d.aggregateDetailHandler))
		}
	}

	// Projection Dashboard
	if d.caps.ProjectionHost {
		mux.HandleFunc("GET /projections", d.guard(d.projectionsIndexHandler))
		if !d.cfg.ReadOnly {
			mux.HandleFunc("POST /projections/{name}/reset", d.guard(d.projectionResetHandler))
		}
	}

	// Dead-Letter Queue
	if d.caps.DeadLetterStore || d.caps.ProjectionHost {
		mux.HandleFunc("GET /dead-letters", d.guard(d.dlqIndexHandler))
		mux.HandleFunc("GET /dead-letters/{projection}", d.guard(d.dlqDetailHandler))
		if !d.cfg.ReadOnly && d.caps.ProjectionHost {
			mux.HandleFunc("POST /dead-letters/{projection}/replay", d.guard(d.dlqReplayHandler))
		}
		if !d.cfg.ReadOnly && d.caps.DeadLetterStore {
			mux.HandleFunc("POST /dead-letters/{projection}/{eventID}/delete", d.guard(d.dlqDeleteHandler))
			mux.HandleFunc("POST /dead-letters/{projection}/purge", d.guard(d.dlqPurgeHandler))
		}
	}

	// Command Audit
	if d.caps.CommandJournal {
		mux.HandleFunc("GET /commands", d.guard(d.commandsIndexHandler))
	}

	// Query Audit
	if d.caps.QueryJournal {
		mux.HandleFunc("GET /queries", d.guard(d.queriesIndexHandler))
	}

	// Time-Travel
	if d.caps.EventSource {
		mux.HandleFunc("GET /time-travel", d.guard(d.timeTravelIndexHandler))
		mux.HandleFunc("GET /time-travel/{type}/{id}", d.guard(d.timeTravelDetailHandler))
	}

	// Snapshot Inspector
	if d.caps.SnapshotStore {
		mux.HandleFunc("GET /snapshots", d.guard(d.snapshotsIndexHandler))
		mux.HandleFunc("GET /snapshots/{type}/{id}", d.guard(d.snapshotDetailHandler))
		if !d.cfg.ReadOnly {
			mux.HandleFunc("POST /snapshots/{type}/{id}/delete", d.guard(d.snapshotDeleteHandler))
		}
	}

	return mux
}
