package cqrshtmx

import (
	"github.com/larsartmann/cqrs-htmx/v4/openapi"
)

// OpenAPIRoute is the collected metadata from one [WithOpenAPI] option,
// captured when the command or query handler was registered.
type OpenAPIRoute struct {
	// Kind is "command" or "query" — which dispatch surface the handler serves.
	Kind string
	// Method is the lowercase HTTP method declared by the openapi builder
	// ("post", "get", ...). Empty when the Operation was constructed literally.
	Method string
	// Operation is the attached OpenAPI operation metadata.
	Operation openapi.Operation
}

// OpenAPIRoutes returns the OpenAPI metadata collected from every
// [WithOpenAPI] handler option used with this App, in registration order.
//
// The App never learns the concrete route paths (routes are wired by the
// consumer on their own mux), so this returns operations — not paths. Merge
// them into your own [openapi.Spec] at the paths you control and serve it
// with [OpenAPISpecHandler]. The returned slice is a copy; mutating it does
// not affect the App.
func (a *App) OpenAPIRoutes() []OpenAPIRoute {
	a.openapiMu.Lock()
	defer a.openapiMu.Unlock()

	out := make([]OpenAPIRoute, len(a.openapiRoutes))
	copy(out, a.openapiRoutes)
	return out
}

// collectOpenAPI records the operation attached to a freshly built handler
// config, if any. Called from buildHandlerConfigChecked where the dispatch
// kind is known.
func (a *App) collectOpenAPI(kind string, config *handlerConfig) {
	if config == nil || config.openapiMeta == nil {
		return
	}

	a.openapiMu.Lock()
	defer a.openapiMu.Unlock()
	a.openapiRoutes = append(a.openapiRoutes, OpenAPIRoute{
		Kind:      kind,
		Method:    config.openapiMeta.HTTPMethod,
		Operation: *config.openapiMeta,
	})
}
