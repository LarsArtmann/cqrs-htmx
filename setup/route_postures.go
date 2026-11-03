package setup

import (
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// RoutePostures returns the bundle's aggregate route manifest: one row per
// mounted layer, honoring this instance's Config (Disable* flags, paths).
//
// The manifest describes the LAYERS [Bundle.Mount] wires — a panel mounted
// behind one uniform gate is one row, not a per-route copy of the panel's own
// manifest (which describes the panel's truth for a bare mount; get it from
// [adminui.Handler.Routes] or [dashboardui.Dashboard.Routes] directly). Both
// describe the same mount honestly at their own layer.
//
// Rows are ordered the way Mount registers them.
func (b *Bundle) RoutePostures() []cqrshtmx.RoutePosture {
	cfg := b.config
	rows := make([]cqrshtmx.RoutePosture, 0, 12)

	// Auth routes — usermgmt owns the per-route postures itself (public
	// ceremonies; session routes self-wrap with fail-closed 401s, ADR-0055),
	// so this layer declares no gate of its own.
	if b.Auth != nil {
		rows = append(rows, cqrshtmx.RoutePosture{
			Method:  "*",
			Pattern: "/auth/*",
			Posture: cqrshtmx.PostureUngated,
			Why:     "usermgmt enforces per-route postures itself (public ceremonies, self-wrapped 401 session routes — ADR-0055); this layer adds no gate",
		})
	}

	// Login page — public (it IS the login page).
	if b.Login != nil {
		rows = append(rows, cqrshtmx.RoutePosture{
			Method:  http.MethodGet,
			Pattern: "/",
			Posture: cqrshtmx.PosturePublic,
			Why:     "the login page is the anonymous entry point",
		})
	}

	// Admin panel — session + CSRF + the admin authorizer (role-based by
	// default; a custom authorizer's truth is the consumer's, and the
	// per-route manifest comes from adminui.Handler.Routes).
	if b.Admin != nil {
		rows = append(rows, cqrshtmx.RoutePosture{
			Method:  "*",
			Pattern: cfg.AdminPath,
			Posture: cqrshtmx.PostureRoleGated,
			Why:     "session + CSRF + the admin authorizer (role-based by default: super_admin/admin globally, admin/owner per tenant)",
		})
	}

	// Dashboard — session-gated; DashboardAuthorizer refines to role-gated.
	if b.Dashboard != nil {
		posture, why := cqrshtmx.PostureAuthenticated,
			"RequireSession: the dashboard renders event payloads and stream IDs, so it must never be public"
		if cfg.DashboardAuthorizer != nil {
			posture, why = cqrshtmx.PostureRoleGated,
				"RequireSession + the configured DashboardAuthorizer (403 for callers it refuses)"
		}

		rows = append(rows, cqrshtmx.RoutePosture{
			Method:  "*",
			Pattern: cfg.DashboardPath,
			Posture: posture,
			Why:     why,
		})
	}

	rows = append(rows, b.feedPostures()...)
	rows = append(rows, b.machineEndpointPostures()...)
	rows = append(rows, b.probePostures()...)

	return rows
}

// probePostures manifests the two orchestrator probes: readiness (public —
// orchestrators probe without sessions; "-" opts out) and liveness (public,
// always-200 while the process serves; opt-in via LivePath).
func (b *Bundle) probePostures() []cqrshtmx.RoutePosture {
	cfg := b.config
	var rows []cqrshtmx.RoutePosture

	if cfg.HealthPath != "" && cfg.HealthPath != "-" {
		rows = append(rows, cqrshtmx.RoutePosture{
			Method:  "*",
			Pattern: cfg.HealthPath,
			Posture: cqrshtmx.PosturePublic,
			Why:     "readiness probe: orchestrators must reach it without a session",
		})
	}

	if cfg.LivePath != "" {
		rows = append(rows, cqrshtmx.RoutePosture{
			Method:  http.MethodGet,
			Pattern: cfg.LivePath,
			Posture: cqrshtmx.PosturePublic,
			Why:     "liveness probe: orchestrators must reach it without a session",
		})
	}

	return rows
}

// feedPostures manifests the two real-time feeds (session-gated: event
// metadata is not public data) and their SDK scripts (static JS, public).
func (b *Bundle) feedPostures() []cqrshtmx.RoutePosture {
	cfg := b.config
	var rows []cqrshtmx.RoutePosture

	if b.Broadcaster != nil && cfg.SSEPath != "" {
		rows = append(rows, cqrshtmx.RoutePosture{
			Method:  http.MethodGet,
			Pattern: cfg.SSEPath,
			Posture: cqrshtmx.PostureAuthenticated,
			Why:     "the shared SSE feed streams event metadata — session-gated",
		})

		if cfg.SSEScriptPath != "" && cfg.SSEScriptPath != "-" {
			rows = append(rows, cqrshtmx.RoutePosture{
				Method:  http.MethodGet,
				Pattern: cfg.SSEScriptPath,
				Posture: cqrshtmx.PosturePublic,
				Why:     "static HTMX SSE extension script, no data",
			})
		}
	}

	if b.DataStarBroadcaster != nil {
		if cfg.DataStarPath != "" {
			rows = append(rows, cqrshtmx.RoutePosture{
				Method:  http.MethodGet,
				Pattern: cfg.DataStarPath,
				Posture: cqrshtmx.PostureAuthenticated,
				Why:     "the DataStar feed streams event metadata — session-gated",
			})
		}

		if cfg.DataStarScriptPath != "" && cfg.DataStarScriptPath != "-" {
			rows = append(rows, cqrshtmx.RoutePosture{
				Method:  http.MethodGet,
				Pattern: cfg.DataStarScriptPath,
				Posture: cqrshtmx.PosturePublic,
				Why:     "static DataStar SDK script, no data",
			})
		}
	}

	return rows
}

// machineEndpointPostures manifests the opt-in JSON endpoints — session-gated
// (event metadata is not public data).
func (b *Bundle) machineEndpointPostures() []cqrshtmx.RoutePosture {
	cfg := b.config
	var rows []cqrshtmx.RoutePosture

	machine := func(pattern, what string) cqrshtmx.RoutePosture {
		return cqrshtmx.RoutePosture{
			Method:  "*",
			Pattern: pattern,
			Posture: cqrshtmx.PostureAuthenticated,
			Why:     what + " carries event metadata — session-gated",
		}
	}

	if cfg.EventCatalogPath != "" {
		rows = append(rows, machine(cfg.EventCatalogPath, "the event catalog"))
	}

	if cfg.ProjectionStatusPath != "" {
		rows = append(rows, machine(cfg.ProjectionStatusPath, "projection status"))
	}

	if cfg.DebugPath != "" {
		rows = append(rows, machine(cfg.DebugPath, "the debug dump"))
	}

	if cfg.DocsPath != "" {
		rows = append(rows, machine(cfg.DocsPath, "the docs endpoint"))
	}

	return rows
}
