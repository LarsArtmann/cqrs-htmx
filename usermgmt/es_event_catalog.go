package usermgmt

import (
	"reflect"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/v4/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// DefaultEventCatalog builds a eventcatalog.Catalog populated with all 27
// event types across the five aggregates (User, Membership, Tenant, Bot,
// Auth). Both wire formats derive from ONE table (eventCatalogSpecs): the
// served catalog's PayloadFields are reflected from the payload structs
// (payloadFieldsFrom, Go-source vocabulary), and the architecture catalog
// ([ArchitectureCatalog]) derives its JSON schemas through the shared
// go-cqrs-lite catalog reflection. A renamed json tag or added field
// therefore updates both automatically, and
// TestEventCatalog_InventoryMatchesEventConstants fails when an event
// constant and the table drift apart. Descriptions stay curated in the
// table: they are prose, not derivable. Consumers can serve this catalog at
// an HTTP endpoint (e.g. GET /events/catalog) so that projection builders
// can discover the event schema without reading the source code.
func DefaultEventCatalog() *eventcatalog.Catalog {
	catalog := eventcatalog.New()
	for _, spec := range eventCatalogSpecs {
		catalog.Register(eventcatalog.EventMetadata{
			Type:          string(spec.eventType),
			Aggregate:     string(spec.aggregate),
			SchemaVersion: identitymodel.CurrentSchemaVersion,
			Description:   spec.description,
			PayloadFields: spec.fields(),
		})
	}
	return catalog
}

// eventCatalogSpec is one row of the identity domain's published event
// language: the single registration table feeding BOTH wire formats (the
// served eventcatalog.Catalog and the architecture catalog.Catalog). The
// two function fields are generic instantiations over the same payload
// struct, so a payload change cannot update one format and not the other.
type eventCatalogSpec struct {
	eventType   event.Type
	aggregate   id.StreamType
	description string
	schema      func() *catalog.Schema             // shared go-cqrs-lite JSON-schema derivation
	fields      func() []eventcatalog.PayloadField // Go-source vocabulary for the served catalog
}

var eventCatalogSpecs = []eventCatalogSpec{ //nolint:gochecknoglobals // single-table source of truth for the served + architecture catalogs (R2)
	// User events
	{
		eventType:   identitymodel.EventUserRegistered,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "A user registered with an email address and display name",
		schema:      catalog.SchemaFromType[identitymodel.UserRegisteredPayload],
		fields:      payloadFieldsFrom[identitymodel.UserRegisteredPayload],
	},
	{
		eventType:   identitymodel.EventRolesUpdated,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "Legacy event: user roles updated. No longer emitted; decoded for backward compat",
		schema:      catalog.SchemaFromType[identitymodel.RolesUpdatedPayload],
		fields:      payloadFieldsFrom[identitymodel.RolesUpdatedPayload],
	},
	{
		eventType:   identitymodel.EventEmailChanged,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "User changed their email address",
		schema:      catalog.SchemaFromType[identitymodel.EmailChangedPayload],
		fields:      payloadFieldsFrom[identitymodel.EmailChangedPayload],
	},
	{
		eventType:   identitymodel.EventDisplayNameChanged,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "User changed their display name",
		schema:      catalog.SchemaFromType[identitymodel.DisplayNameChangedPayload],
		fields:      payloadFieldsFrom[identitymodel.DisplayNameChangedPayload],
	},
	{
		eventType:   identitymodel.EventUserDeleted,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "User was deleted with an optional reason",
		schema:      catalog.SchemaFromType[identitymodel.UserDeletedPayload],
		fields:      payloadFieldsFrom[identitymodel.UserDeletedPayload],
	},
	{
		eventType:   identitymodel.EventCredentialAdded,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "A WebAuthn credential was added to the user",
		schema:      catalog.SchemaFromType[identitymodel.CredentialAddedPayload],
		fields:      payloadFieldsFrom[identitymodel.CredentialAddedPayload],
	},
	{
		eventType:   identitymodel.EventCredentialRemoved,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "A WebAuthn credential was removed from the user",
		schema:      catalog.SchemaFromType[identitymodel.CredentialRemovedPayload],
		fields:      payloadFieldsFrom[identitymodel.CredentialRemovedPayload],
	},
	{
		eventType:   identitymodel.EventCredentialUsed,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "A WebAuthn credential was used: the assertion's signature counter and flags were recorded",
		schema:      catalog.SchemaFromType[identitymodel.CredentialUsedPayload],
		fields:      payloadFieldsFrom[identitymodel.CredentialUsedPayload],
	},
	{
		eventType:   identitymodel.EventEmailVerified,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "User verified their email address",
		schema:      catalog.SchemaFromType[identitymodel.EmailVerifiedPayload],
		fields:      payloadFieldsFrom[identitymodel.EmailVerifiedPayload],
	},
	{
		eventType:   identitymodel.EventTOTPEnabled,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "TOTP two-factor authentication was enabled",
		schema:      catalog.SchemaFromType[identitymodel.TOTPEnabledPayload],
		fields:      payloadFieldsFrom[identitymodel.TOTPEnabledPayload],
	},
	{
		eventType:   identitymodel.EventTOTPDisabled,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "TOTP two-factor authentication was disabled",
		schema:      catalog.SchemaFromType[identitymodel.TOTPDisabledPayload],
		fields:      payloadFieldsFrom[identitymodel.TOTPDisabledPayload],
	},
	{
		eventType:   identitymodel.EventExternalAccountLinked,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "An external account (OAuth2/OIDC) was linked to the user",
		schema:      catalog.SchemaFromType[identitymodel.ExternalAccountLinkedPayload],
		fields:      payloadFieldsFrom[identitymodel.ExternalAccountLinkedPayload],
	},
	{
		eventType:   identitymodel.EventExternalAccountUnlinked,
		aggregate:   identitymodel.AggregateTypeUser,
		description: "An external account was unlinked from the user",
		schema:      catalog.SchemaFromType[identitymodel.ExternalAccountUnlinkedPayload],
		fields:      payloadFieldsFrom[identitymodel.ExternalAccountUnlinkedPayload],
	},

	// Membership events
	{
		eventType:   identitymodel.EventMemberAdded,
		aggregate:   identitymodel.AggregateTypeMembership,
		description: "A member was added to a tenant with roles",
		schema:      catalog.SchemaFromType[identitymodel.MemberAddedPayload],
		fields:      payloadFieldsFrom[identitymodel.MemberAddedPayload],
	},
	{
		eventType:   identitymodel.EventMemberRolesChanged,
		aggregate:   identitymodel.AggregateTypeMembership,
		description: "A member roles were changed within a tenant",
		schema:      catalog.SchemaFromType[identitymodel.MemberRolesChangedPayload],
		fields:      payloadFieldsFrom[identitymodel.MemberRolesChangedPayload],
	},
	{
		eventType:   identitymodel.EventMemberRemoved,
		aggregate:   identitymodel.AggregateTypeMembership,
		description: "A member was removed from a tenant",
		schema:      catalog.SchemaFromType[identitymodel.MemberRemovedPayload],
		fields:      payloadFieldsFrom[identitymodel.MemberRemovedPayload],
	},

	// Tenant events
	{
		eventType:   identitymodel.EventTenantCreated,
		aggregate:   identitymodel.AggregateTypeTenant,
		description: "A tenant was created",
		schema:      catalog.SchemaFromType[identitymodel.TenantCreatedPayload],
		fields:      payloadFieldsFrom[identitymodel.TenantCreatedPayload],
	},
	{
		eventType:   identitymodel.EventTenantSuspended,
		aggregate:   identitymodel.AggregateTypeTenant,
		description: "A tenant was suspended with an optional reason",
		schema:      catalog.SchemaFromType[identitymodel.TenantSuspendedPayload],
		fields:      payloadFieldsFrom[identitymodel.TenantSuspendedPayload],
	},
	{
		eventType:   identitymodel.EventTenantReactivated,
		aggregate:   identitymodel.AggregateTypeTenant,
		description: "A suspended tenant was reactivated",
		schema:      catalog.SchemaFromType[identitymodel.TenantReactivatedPayload],
		fields:      payloadFieldsFrom[identitymodel.TenantReactivatedPayload],
	},
	{
		eventType:   identitymodel.EventTenantDeleted,
		aggregate:   identitymodel.AggregateTypeTenant,
		description: "A tenant was deleted with an optional reason",
		schema:      catalog.SchemaFromType[identitymodel.TenantDeletedPayload],
		fields:      payloadFieldsFrom[identitymodel.TenantDeletedPayload],
	},

	// Bot events
	{
		eventType:   identitymodel.EventBotRegistered,
		aggregate:   identitymodel.AggregateTypeBot,
		description: "A bot was registered with a token hash and scopes",
		schema:      catalog.SchemaFromType[identitymodel.BotRegisteredPayload],
		fields:      payloadFieldsFrom[identitymodel.BotRegisteredPayload],
	},
	{
		eventType:   identitymodel.EventBotDeleted,
		aggregate:   identitymodel.AggregateTypeBot,
		description: "A bot was deleted with an optional reason",
		schema:      catalog.SchemaFromType[identitymodel.BotDeletedPayload],
		fields:      payloadFieldsFrom[identitymodel.BotDeletedPayload],
	},

	// Auth audit events (dedicated Auth streams; cqrs-htmx#23, #25)
	{
		eventType:   identitymodel.EventLoginSucceeded,
		aggregate:   identitymodel.AggregateTypeAuth,
		description: "A login succeeded (auth audit journal; one dedicated Auth stream per event)",
		schema:      catalog.SchemaFromType[identitymodel.LoginSucceededPayload],
		fields:      payloadFieldsFrom[identitymodel.LoginSucceededPayload],
	},
	{
		eventType:   identitymodel.EventLoginFailed,
		aggregate:   identitymodel.AggregateTypeAuth,
		description: "A known-user login failed with a bounded reason class (never raw error text or secrets)",
		schema:      catalog.SchemaFromType[identitymodel.LoginFailedPayload],
		fields:      payloadFieldsFrom[identitymodel.LoginFailedPayload],
	},
	{
		eventType:   identitymodel.EventRegistrationClosed,
		aggregate:   identitymodel.AggregateTypeAuth,
		description: "A registration attempt was rejected because registration is closed",
		schema:      catalog.SchemaFromType[identitymodel.RegistrationClosedPayload],
		fields:      payloadFieldsFrom[identitymodel.RegistrationClosedPayload],
	},
	{
		eventType: identitymodel.EventSessionStarted,
		aggregate: identitymodel.AggregateTypeAuth,
		description: "A session was established; carries the impersonation origin (by, reason) when an admin" +
			" acted on the user's behalf",
		schema: catalog.SchemaFromType[identitymodel.SessionStartedPayload],
		fields: payloadFieldsFrom[identitymodel.SessionStartedPayload],
	},
	{
		eventType:   identitymodel.EventSessionEnded,
		aggregate:   identitymodel.AggregateTypeAuth,
		description: "A session ended with a bounded cause (logout, expired, impersonation_ended, revoked, deleted)",
		schema:      catalog.SchemaFromType[identitymodel.SessionEndedPayload],
		fields:      payloadFieldsFrom[identitymodel.SessionEndedPayload],
	},
}

// payloadFieldsFrom reflects T's JSON contract: field names come from the
// json tags, optionality from omitempty, and the type vocabulary mirrors the
// Go source ("int", "string", "[]byte", "[]Role", "AuthMethod") so the served
// catalog keeps its historical shape. Embedded structs (CredentialCore,
// ExternalAccountCore) are walked, so promoted fields appear exactly like
// declared ones — the drift class the hand-maintained lists missed.
func payloadFieldsFrom[T any]() []eventcatalog.PayloadField {
	t := reflect.TypeOf(*new(T))
	fields := make([]eventcatalog.PayloadField, 0, t.NumField())
	for field := range t.Fields() {
		fields = append(fields, fieldsFromStructField(field)...)
	}
	return fields
}

func fieldsFromStructField(f reflect.StructField) []eventcatalog.PayloadField {
	if f.Anonymous {
		t := f.Type
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return nil
		}
		embedded := make([]eventcatalog.PayloadField, 0, t.NumField())
		for field := range t.Fields() {
			embedded = append(embedded, fieldsFromStructField(field)...)
		}
		return embedded
	}

	tag := f.Tag.Get("json")
	name, options, _ := strings.Cut(tag, ",")
	if name == "-" {
		return nil
	}
	if name == "" {
		name = f.Name
	}

	return []eventcatalog.PayloadField{{
		Name:     name,
		Type:     goTypeName(f.Type),
		Required: !strings.Contains(options, "omitempty"),
	}}
}

// goTypeName renders a reflection type in the catalog's Go-source vocabulary:
// named domain types by their Go name (Role, AuthMethod, UserID), builtins by
// kind, slices as "[]elem". Package qualifiers are dropped — the names are
// domain vocabulary, unambiguous within the identity model.
func goTypeName(t reflect.Type) string {
	switch t.Kind() { //nolint:exhaustive // partial-kind switch: unhandled kinds fall through to Name()/Kind().String() below
	case reflect.Slice, reflect.Array:
		return "[]" + goTypeName(t.Elem())
	case reflect.Pointer:
		return "*" + goTypeName(t.Elem())
	}
	if n := t.Name(); n != "" {
		if n == "uint8" && t.PkgPath() == "" {
			return "byte" // predeclared byte, not a named uint8 domain type
		}
		return n
	}
	return t.Kind().String()
}
