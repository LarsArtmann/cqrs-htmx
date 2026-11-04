package cqrshtmx

import (
	"github.com/larsartmann/cqrs-htmx/v4/eventcatalog"
)

// The Published Language surface — the event catalog and its immutable JSON
// handler — lives in the eventcatalog subpackage. It has no dependency on the
// CQRS handler machinery, so consumers building projections can import it
// without the rest of the library.
//
// The aliases below keep cqrshtmx.EventCatalog, cqrshtmx.NewEventCatalog and
// cqrshtmx.EventCatalogHandler compiling unchanged; they are the same types and
// the same function values as eventcatalog.Catalog, eventcatalog.New and
// eventcatalog.Handler.

type (
	// EventCatalog is a mutable registry of event types and their metadata.
	EventCatalog = eventcatalog.Catalog

	// EventMetadata describes a single event type in the catalog.
	EventMetadata = eventcatalog.EventMetadata

	// PayloadField describes a single field in an event payload.
	PayloadField = eventcatalog.PayloadField
)

var (
	// NewEventCatalog returns an empty EventCatalog ready for registration.
	NewEventCatalog = eventcatalog.New

	// EventCatalogHandler returns an http.HandlerFunc that serves the given
	// event catalog as indented JSON with an ETag-driven
	// `Cache-Control: no-cache`.
	EventCatalogHandler = eventcatalog.Handler
)
