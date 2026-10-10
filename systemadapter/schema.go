package systemadapter

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	cqrsschema "github.com/larsartmann/go-cqrs-lite/schema/v4"
)

// envelopeSchemaVersion is the event-envelope schema version every
// identity-model event is emitted at: the deciders build events via
// event.New, whose default schema version is 1, and no envelope-level
// upcasting has ever shipped (the identity-model UpcasterRegistry versions
// the payload-embedded schema_version field, a separate homegrown layer
// that is not wired in production). Declaring the envelope truth here makes
// the system's coeffect gate see the full event universe and gives future
// envelope-level evolution a declaration to add ops to.
const envelopeSchemaVersion = 1

// EventSchemas declares all 21 identity-model event types for
// [system.DomainConfig.Schema] at their current envelope schema versions,
// with zero upcast ops — the declaration, not the wiring, is the point:
// every read path through system.New now runs the compiled (currently
// identity) upcast chain, so adding a schema op later is a one-line change
// with no new pipeline.
//
// The list is deliberately exhaustive over the identity-model constants —
// including EventRolesUpdated (legacy, no longer emitted but still decoded
// for backward compatibility): projections may still subscribe to it, and
// the coeffect gate counts exactly the declared types.
func EventSchemas() []cqrsschema.EventSchema {
	return []cqrsschema.EventSchema{
		// User events
		cqrsschema.Event(identitymodel.EventUserRegistered, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventRolesUpdated, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventEmailChanged, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventDisplayNameChanged, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventUserDeleted, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventCredentialAdded, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventCredentialRemoved, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventEmailVerified, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventTOTPEnabled, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventTOTPDisabled, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventExternalAccountLinked, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventExternalAccountUnlinked, envelopeSchemaVersion),

		// Membership events
		cqrsschema.Event(identitymodel.EventMemberAdded, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventMemberRolesChanged, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventMemberRemoved, envelopeSchemaVersion),

		// Tenant events
		cqrsschema.Event(identitymodel.EventTenantCreated, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventTenantSuspended, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventTenantReactivated, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventTenantDeleted, envelopeSchemaVersion),

		// Bot events
		cqrsschema.Event(identitymodel.EventBotRegistered, envelopeSchemaVersion),
		cqrsschema.Event(identitymodel.EventBotDeleted, envelopeSchemaVersion),
	}
}
