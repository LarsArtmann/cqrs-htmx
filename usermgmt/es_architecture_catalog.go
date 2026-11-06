package usermgmt

import (
	"strconv"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// architectureCatalogVersion is the catalog-level version stamp. The message
// inventory itself is drift-gated against the identity-model constants (see
// TestArchitectureCatalog_InventoryMatchesEventConstants), so this stamp only
// tracks the catalog document, not the domain.
const architectureCatalogVersion = "1.0.0"

// ArchitectureCatalog derives the identity domain's published event language
// as a go-cqrs-lite catalog.Catalog — the fleet-standard architecture format
// renderable by catalog/architecture and lintable by the EventCatalog
// governance tooling. It derives from the SAME registration table as
// [DefaultEventCatalog] (eventCatalogSpecs): every event appears as a sent
// message with its payload reflected into a JSON schema by the shared
// go-cqrs-lite catalog reflection, and the curated descriptions ride along as
// message summaries.
//
// Use DefaultEventCatalog for the cqrs-htmx HTTP discovery endpoint
// (EventCatalogHandler); use ArchitectureCatalog when the consumer's
// toolchain speaks the go-cqrs-lite catalog format. The two can never drift
// apart — one table feeds both.
func ArchitectureCatalog() *catalog.Catalog {
	builder := catalog.NewBuilder("cqrs-htmx identity", architectureCatalogVersion)

	messages := make([]catalog.MessageConfig, 0, len(eventCatalogSpecs))
	for _, spec := range eventCatalogSpecs {
		sch := spec.schema()
		if sch != nil {
			sch = sch.Clone() // SchemaFromType returns the shared cached schema
		}

		messages = append(messages, catalog.EventFromSchema(
			catalog.MessageID(spec.eventType),
			catalog.Sends,
			sch,
			catalog.WithSummary(spec.description),
			// Same stamp the served cqrshtmx catalog carries
			// (EventMetadata.SchemaVersion), as the fleet's derived-catalog
			// version vocabulary: the bare schema version integer.
			catalog.WithVersion(strconv.Itoa(identitymodel.CurrentSchemaVersion)),
		))
	}

	builder.AddService(
		"usermgmt",
		"cqrs-htmx identity",
		architectureCatalogVersion,
		"Event-sourced user management: registration, credentials, memberships, tenants, bots,"+
			" and the auth audit journal",
		messages...,
	)

	return builder.Build()
}
