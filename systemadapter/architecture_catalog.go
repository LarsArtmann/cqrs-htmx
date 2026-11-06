package systemadapter

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/systemcatalog/v4"
)

// ArchitectureCatalog derives the FULL message inventory of the identity
// domain — every registered command (with payload schema) and every declared
// event (with its envelope schema version) — from the declared truth in
// [DomainConfig] by booting an ephemeral in-memory system (the systemcatalog
// recipe: register, read the registration records, close; nothing starts,
// nothing persists).
//
// Use it in tests, CI drift gates, and catalog export commands. The curated
// baseline (title, service name, summary) matches the module's role; extra
// options (owners, curator hooks for flows/data stores, message overrides)
// are appended AFTER the baseline and may override it.
//
// For the events-only published-language surface (no ephemeral boot), prefer
// usermgmt.ArchitectureCatalog() or the served cqrshtmx event catalog.
func ArchitectureCatalog(
	ctx context.Context,
	opts ...systemcatalog.Option,
) (*catalog.Catalog, error) {
	return systemcatalog.FromDomainConfig(ctx, DomainConfig(), append([]systemcatalog.Option{
		systemcatalog.WithTitle("cqrs-htmx identity"),
		systemcatalog.WithServiceName("systemadapter (cqrs-htmx identity)"),
		systemcatalog.WithServiceSummary(
			"The identity domain composed through system.New: four deciders, the full command" +
				" surface, and the declared event universe",
		),
	}, opts...)...)
}
