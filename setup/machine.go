package setup

import (
	"net/http"
	"runtime"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Machine endpoints (M6 of the 2026-09-17 gap-bundle plan): session-gated
// JSON surfaces for monitors, scripts, and runbooks, mounted at the
// opt-in Config paths. The root module ships all three handlers; this file
// owns their construction (fail-fast) and their session-gated mounting.

// attachMachineEndpoints constructs the opt-in machine-endpoint handlers and
// stores them on the bundle. On failure the caller must call [Bundle.cleanup]
// to release everything created so far. All handlers are built even when
// their Config path is unset — construction is cheap and keeps Mount trivial
// — except that nothing is served without a path.
func (b *Bundle) attachMachineEndpoints() error {
	catalog, err := cqrshtmx.EventCatalogHandler(usermgmt.DefaultEventCatalog())
	if err != nil {
		return errorfamily.WrapRejection(
			err,
			"setup.event_catalog_creation_failed",
			"failed to serialize the event catalog",
		)
	}

	b.eventCatalog = catalog
	b.projectionStatus = cqrshtmx.ProjectionStatusHandler(b.serviceStatusProvider())
	b.debug = cqrshtmx.DebugHandler(map[string]any{
		"version":   b.config.Version,
		"goVersion": runtime.Version(),
		"title":     b.config.Title,
	})

	return nil
}

// serviceStatusProvider adapts the bundle's service onto the root module's
// ProjectionStatusProvider, staying nil-safe the way healthHandler is.
func (b *Bundle) serviceStatusProvider() cqrshtmx.ProjectionStatusProvider {
	if b.Service == nil {
		return nil
	}

	return b.Service
}

// mountMachineEndpoints registers the opt-in machine endpoints, each behind
// the session middleware (context enrichment) and the requireSession gate
// (401 without an authenticated user) — event metadata is not public data.
func (b *Bundle) mountMachineEndpoints(mux *http.ServeMux) {
	cfg := b.config
	sessionMW := b.SessionMiddleware()

	if cfg.EventCatalogPath != "" {
		mux.Handle(cfg.EventCatalogPath, sessionMW(requireSession(b.eventCatalog)))
	}

	if cfg.ProjectionStatusPath != "" {
		mux.Handle(cfg.ProjectionStatusPath, sessionMW(requireSession(b.projectionStatus)))
	}

	if cfg.DebugPath != "" {
		mux.Handle(cfg.DebugPath, sessionMW(requireSession(b.debug)))
	}
}
