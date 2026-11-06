package cqrshtmx

import (
	"github.com/larsartmann/cqrs-htmx/v4/eventcatalog"
)

// The Published Language surface — the event catalog and its immutable JSON
// handler — lives in the eventcatalog subpackage. It has no dependency on the
// CQRS handler machinery, so consumers building projections can import it
// without the rest of the library.
//
// Every symbol in this file is a deprecated alias. Import the subpackage
// directly:
//
//	import "github.com/larsartmann/cqrs-htmx/v4/eventcatalog"
//
// then write eventcatalog.Catalog, eventcatalog.New, eventcatalog.Handler.
// These aliases are identical types and function values and are deleted in
// cqrs-htmx v5; see ADR-0062 and docs/migrations/v4-to-v5.md.

type (
	// EventCatalog is a mutable registry of event types and their metadata.
	//
	// Deprecated: use eventcatalog.Catalog from github.com/larsartmann/cqrs-htmx/v4/eventcatalog
	// directly. Removed in cqrs-htmx v5.
	EventCatalog = eventcatalog.Catalog

	// EventMetadata describes a single event type in the catalog.
	//
	// Deprecated: use eventcatalog.EventMetadata from github.com/larsartmann/cqrs-htmx/v4/eventcatalog
	// directly. Removed in cqrs-htmx v5.
	EventMetadata = eventcatalog.EventMetadata

	// PayloadField describes a single field in an event payload.
	//
	// Deprecated: use eventcatalog.PayloadField from github.com/larsartmann/cqrs-htmx/v4/eventcatalog
	// directly. Removed in cqrs-htmx v5.
	PayloadField = eventcatalog.PayloadField
)

var (
	// NewEventCatalog returns an empty EventCatalog ready for registration.
	//
	// Deprecated: use eventcatalog.New from github.com/larsartmann/cqrs-htmx/v4/eventcatalog
	// directly. Removed in cqrs-htmx v5.
	NewEventCatalog = eventcatalog.New

	// EventCatalogHandler returns an http.HandlerFunc that serves the given
	// event catalog as indented JSON with an ETag-driven
	// `Cache-Control: no-cache`.
	//
	// Deprecated: use eventcatalog.Handler from github.com/larsartmann/cqrs-htmx/v4/eventcatalog
	// directly. Removed in cqrs-htmx v5.
	EventCatalogHandler = eventcatalog.Handler
)
