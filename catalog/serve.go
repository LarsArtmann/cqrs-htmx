package cataloghtmx

import (
	"encoding/json"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/d2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/openapi"
	errorfamily "github.com/larsartmann/go-error-family"
)

// ServeOption configures HTTP handler behavior.
type ServeOption func(*serveConfig)

type serveConfig struct {
	description string
	basePath    string
	format      ExportFormat
}

// ExportFormat selects the output format for multi-format handlers.
type ExportFormat int

const (
	// FormatJSON outputs JSON (default).
	FormatJSON ExportFormat = iota
	// FormatYAML outputs YAML.
	FormatYAML
)

// WithDescription sets the API description in the exported document.
func WithDescription(desc string) ServeOption {
	return func(c *serveConfig) {
		c.description = desc
	}
}

// WithBasePath sets the base path for OpenAPI endpoint generation.
// Defaults to "/api".
func WithBasePath(path string) ServeOption {
	return func(c *serveConfig) {
		c.basePath = path
	}
}

// WithFormat selects JSON or YAML output format.
func WithFormat(f ExportFormat) ServeOption {
	return func(c *serveConfig) {
		c.format = f
	}
}

// OpenAPIHandler returns an http.HandlerFunc that serves the catalog
// as an OpenAPI 3.0 document. Defaults to JSON; use WithFormat(FormatYAML)
// for YAML output.
func OpenAPIHandler(cat *catalog.Catalog, opts ...ServeOption) http.HandlerFunc {
	cfg := defaultServeConfig(opts...)

	exporter := openapi.NewExporter(
		string(cat.Title),
		string(cat.Version),
		openapi.WithDescription(cfg.description),
		openapi.WithBasePath(cfg.basePath),
	)

	doc := exporter.Export(cat)

	return docHandler(doc.MarshalJSON, doc.MarshalYAML, cfg.format)
}

// AsyncAPIHandler returns an http.HandlerFunc that serves the catalog
// as an AsyncAPI 3.0 document. Defaults to JSON.
func AsyncAPIHandler(cat *catalog.Catalog, opts ...ServeOption) http.HandlerFunc {
	cfg := defaultServeConfig(opts...)

	exporter := asyncapi.NewExporter(
		string(cat.Title),
		string(cat.Version),
		asyncapi.WithDescription(cfg.description),
	)

	doc := exporter.Export(cat)

	return docHandler(doc.MarshalJSON, doc.MarshalYAML, cfg.format)
}

// D2Handler returns an http.HandlerFunc that serves the catalog
// as a D2 architecture diagram. Content-Type is text/plain.
func D2Handler(cat *catalog.Catalog, opts ...ServeOption) http.HandlerFunc {
	cfg := defaultServeConfig(opts...)

	exporterOpts := []d2.Option{d2.WithDescription(cfg.description)}

	exporter := d2.NewExporter(
		string(cat.Title),
		string(cat.Version),
		exporterOpts...,
	)

	return func(w http.ResponseWriter, _ *http.Request) {
		text := exporter.Export(cat)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(text))
	}
}

// GenerateEventCatalog writes EventCatalog MDX files to the given outputDir.
// Call this at application startup to generate documentation files that
// can be served by the EventCatalog CLI or a static file server.
//
// This is a build-time/startup-time operation, not an HTTP handler.
// EventCatalog expects a directory of MDX files served statically — there
// is no meaningful way to serve it as a single HTTP response.
func GenerateEventCatalog(cat *catalog.Catalog, outputDir string) error {
	exporter := eventcatalog.NewExporter(outputDir)
	if err := exporter.Export(cat); err != nil {
		return errorfamily.Wrapf(err, errorfamily.Infrastructure,
			"cataloghtmx.event_catalog.generate_failed", "generate EventCatalog files in %q", outputDir)
	}

	return nil
}

// HealthCheckHandler returns a simple health check handler that verifies
// the catalog has at least one service. Returns 200 with JSON body if healthy,
// 503 if the catalog is empty.
func HealthCheckHandler(cat *catalog.Catalog) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if cat == nil || len(cat.Services) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":  "unhealthy",
				"message": "catalog has no services",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "healthy",
			"services": len(cat.Services),
		})
	}
}

func defaultServeConfig(opts ...ServeOption) serveConfig {
	cfg := serveConfig{ //nolint:exhaustruct // fields set below
		basePath: "/api",
		format:   FormatJSON,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

type marshaler func() ([]byte, error)

func docHandler(jsonFn, yamlFn marshaler, format ExportFormat) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeDoc(w, jsonFn, yamlFn, format)
	}
}

func writeDoc(w http.ResponseWriter, jsonFn, yamlFn marshaler, format ExportFormat) {
	var data []byte

	var err error

	switch format {
	case FormatYAML:
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		data, err = yamlFn()
	case FormatJSON:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		data, err = jsonFn()
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to marshal document")
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errchkjson // best-effort error response
}
