package systemadapter

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// DeclarativeProjections returns all metaengine projection declarations for the
// identity-model domain. Wire these into system.DomainConfig.Projections so
// system.New() auto-creates the internal projection host — no separate
// ProjectionLayer needed.
//
// Each projection is a RawQuery wrapping a metaengine.Query built from typed
// fold handlers. Events are decoded by the EventTypeDecoder (set via
// ProjectionTypeDecoder on DomainConfig) into projectionadapter.EventWithID[P]
// structs, giving folds access to the stream ID, payload, and occurred-at time.
//
// Every declaration carries a Volume hint: the order-of-magnitude expected
// collection size for a mid-size deployment. The metaengine cost-based planner
// uses it to rank engines (memory for small collections, SQLite/DuckDB for
// large ones) — operator DeploymentConfig priorities can still override
// placement.
func DeclarativeProjections() []system.ProjectionDeclaration {
	return []system.ProjectionDeclaration{
		// Tenant
		system.RawQuery(tenantLookup()),
		system.RawQuery(tenantScan()),
		// Bot
		system.RawQuery(botLookup()),
		system.RawQuery(botScan()),
		system.RawQuery(botTokenScan()),
		// Membership
		system.RawQuery(membershipLookup()),
		system.RawQuery(membershipScan()),
		// User
		system.RawQuery(userLookup()),
		system.RawQuery(userScan()),
		// Authz
		system.RawQuery(authzPolicyLookup()),
		system.RawQuery(authzPolicyScan()),
		// AuditLog
		system.RawQuery(auditLogScan()),
	}
}

// -------------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------------

func rolesToStrings(roles []identitymodel.Role) []string {
	result := make([]string, len(roles))
	for i, r := range roles {
		result[i] = string(r)
	}
	return result
}
