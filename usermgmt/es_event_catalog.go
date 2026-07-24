package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// DefaultEventCatalog builds a cqrshtmx.EventCatalog populated with all 21
// event types across the four aggregates (User, Membership, Tenant, Bot).
// Consumers can serve this catalog at an HTTP endpoint (e.g.
// GET /events/catalog) so that projection builders can discover the event
// schema without reading the source code.
func DefaultEventCatalog() *cqrshtmx.EventCatalog {
	catalog := cqrshtmx.NewEventCatalog()
	registerUserEvents(catalog)
	registerMembershipEvents(catalog)
	registerTenantEvents(catalog)
	registerBotEvents(catalog)
	return catalog
}

//nolint:exhaustruct // descriptions are intentional; field lists are illustrative
func registerUserEvents(catalog *cqrshtmx.EventCatalog) {
	sv := identitymodel.CurrentSchemaVersion

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventUserRegistered),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "A user registered with an email address and display name",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "email", Type: "string", Required: true},
			{Name: "display_name", Type: "string"},
			{Name: "roles", Type: "[]Role"},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventRolesUpdated),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "Legacy event: user roles updated. No longer emitted; decoded for backward compat",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "roles", Type: "[]Role"},
			{Name: "domain", Type: "string"},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventEmailChanged),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "User changed their email address",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "email", Type: "string", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventDisplayNameChanged),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "User changed their display name",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "display_name", Type: "string", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventUserDeleted),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "User was deleted with an optional reason",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "reason", Type: "string"},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventCredentialAdded),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "A WebAuthn credential was added to the user",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "id", Type: "[]byte", Required: true},
			{Name: "public_key", Type: "[]byte", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventCredentialRemoved),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "A WebAuthn credential was removed from the user",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "id", Type: "[]byte", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventEmailVerified),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "User verified their email address",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "email", Type: "string", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventTOTPEnabled),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "TOTP two-factor authentication was enabled",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "secret", Type: "[]byte", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventTOTPDisabled),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "TOTP two-factor authentication was disabled",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventExternalAccountLinked),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "An external account (OAuth2/OIDC) was linked to the user",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "provider", Type: "string", Required: true},
			{Name: "subject", Type: "string", Required: true},
			{Name: "email", Type: "string"},
			{Name: "display_name", Type: "string"},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventExternalAccountUnlinked),
		Aggregate:     string(identitymodel.AggregateTypeUser),
		SchemaVersion: sv,
		Description:   "An external account was unlinked from the user",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "provider", Type: "string", Required: true},
			{Name: "subject", Type: "string", Required: true},
		},
	})
}

func registerMembershipEvents(catalog *cqrshtmx.EventCatalog) {
	sv := identitymodel.CurrentSchemaVersion

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventMemberAdded),
		Aggregate:     string(identitymodel.AggregateTypeMembership),
		SchemaVersion: sv,
		Description:   "A member was added to a tenant with roles",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "actor_kind", Type: "string", Required: true},
			{Name: "actor_id", Type: "string", Required: true},
			{Name: "tenant_id", Type: "string", Required: true},
			{Name: "roles", Type: "[]Role", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventMemberRolesChanged),
		Aggregate:     string(identitymodel.AggregateTypeMembership),
		SchemaVersion: sv,
		Description:   "A member roles were changed within a tenant",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "actor_kind", Type: "string", Required: true},
			{Name: "actor_id", Type: "string", Required: true},
			{Name: "tenant_id", Type: "string", Required: true},
			{Name: "roles", Type: "[]Role", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventMemberRemoved),
		Aggregate:     string(identitymodel.AggregateTypeMembership),
		SchemaVersion: sv,
		Description:   "A member was removed from a tenant",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "actor_id", Type: "string", Required: true},
			{Name: "tenant_id", Type: "string", Required: true},
		},
	})
}

//nolint:exhaustruct // descriptions are intentional; field lists are illustrative
func registerTenantEvents(catalog *cqrshtmx.EventCatalog) {
	sv := identitymodel.CurrentSchemaVersion

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventTenantCreated),
		Aggregate:     string(identitymodel.AggregateTypeTenant),
		SchemaVersion: sv,
		Description:   "A tenant was created",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "name", Type: "string", Required: true},
			{Name: "display_name", Type: "string", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventTenantSuspended),
		Aggregate:     string(identitymodel.AggregateTypeTenant),
		SchemaVersion: sv,
		Description:   "A tenant was suspended with an optional reason",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "reason", Type: "string"},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventTenantReactivated),
		Aggregate:     string(identitymodel.AggregateTypeTenant),
		SchemaVersion: sv,
		Description:   "A suspended tenant was reactivated",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventTenantDeleted),
		Aggregate:     string(identitymodel.AggregateTypeTenant),
		SchemaVersion: sv,
		Description:   "A tenant was deleted with an optional reason",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "reason", Type: "string"},
		},
	})
}

//nolint:exhaustruct // descriptions are intentional; field lists are illustrative
func registerBotEvents(catalog *cqrshtmx.EventCatalog) {
	sv := identitymodel.CurrentSchemaVersion

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventBotRegistered),
		Aggregate:     string(identitymodel.AggregateTypeBot),
		SchemaVersion: sv,
		Description:   "A bot was registered with a token hash and scopes",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "name", Type: "string", Required: true},
			{Name: "owner_id", Type: "string", Required: true},
			{Name: "token_hash", Type: "[]byte", Required: true},
			{Name: "scopes", Type: "[]string"},
		},
	})

	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(identitymodel.EventBotDeleted),
		Aggregate:     string(identitymodel.AggregateTypeBot),
		SchemaVersion: sv,
		Description:   "A bot was deleted with an optional reason",
		PayloadFields: []cqrshtmx.PayloadField{
			{Name: "schema_version", Type: "int", Required: true},
			{Name: "reason", Type: "string"},
		},
	})
}
