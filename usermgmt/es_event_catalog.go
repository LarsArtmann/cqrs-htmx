package usermgmt

import (
	"reflect"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// DefaultEventCatalog builds a cqrshtmx.EventCatalog populated with all 25
// event types across the five aggregates (User, Membership, Tenant, Bot,
// Auth). Payload field lists are reflection-derived from the payload structs
// (payloadFieldsFrom), so the served catalog is the wire truth — a renamed
// json tag or added field updates the catalog automatically, and
// TestEventCatalog_InventoryMatchesEventConstants fails when an event
// constant and the catalog drift apart. Descriptions stay curated here: they
// are prose, not derivable. Consumers can serve this catalog at an HTTP
// endpoint (e.g. GET /events/catalog) so that projection builders can
// discover the event schema without reading the source code.
func DefaultEventCatalog() *cqrshtmx.EventCatalog {
	catalog := cqrshtmx.NewEventCatalog()
	registerUserEvents(catalog)
	registerMembershipEvents(catalog)
	registerTenantEvents(catalog)
	registerBotEvents(catalog)
	registerAuthEvents(catalog)
	return catalog
}

// registerDerived registers one event whose PayloadFields are reflected from
// the payload struct T. Aggregate comes from the identity-model stream-type
// constants; description is the curated prose.
func registerDerived[T any](
	catalog *cqrshtmx.EventCatalog,
	eventType event.Type,
	aggregate id.StreamType,
	description string,
) {
	catalog.Register(cqrshtmx.EventMetadata{
		Type:          string(eventType),
		Aggregate:     string(aggregate),
		SchemaVersion: identitymodel.CurrentSchemaVersion,
		Description:   description,
		PayloadFields: payloadFieldsFrom[T](),
	})
}

// payloadFieldsFrom reflects T's JSON contract: field names come from the
// json tags, optionality from omitempty, and the type vocabulary mirrors the
// Go source ("int", "string", "[]byte", "[]Role", "AuthMethod") so the served
// catalog keeps its historical shape. Embedded structs (CredentialCore,
// ExternalAccountCore) are walked, so promoted fields appear exactly like
// declared ones — the drift class the hand-maintained lists missed.
func payloadFieldsFrom[T any]() []cqrshtmx.PayloadField {
	t := reflect.TypeOf(*new(T))
	fields := make([]cqrshtmx.PayloadField, 0, t.NumField())
	for i := range t.NumField() {
		fields = append(fields, fieldsFromStructField(t.Field(i))...)
	}
	return fields
}

func fieldsFromStructField(f reflect.StructField) []cqrshtmx.PayloadField {
	if f.Anonymous {
		t := f.Type
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return nil
		}
		embedded := make([]cqrshtmx.PayloadField, 0, t.NumField())
		for i := range t.NumField() {
			embedded = append(embedded, fieldsFromStructField(t.Field(i))...)
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

	return []cqrshtmx.PayloadField{{
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

func registerUserEvents(catalog *cqrshtmx.EventCatalog) {
	agg := identitymodel.AggregateTypeUser

	registerDerived[identitymodel.UserRegisteredPayload](catalog,
		identitymodel.EventUserRegistered, agg,
		"A user registered with an email address and display name")

	registerDerived[identitymodel.RolesUpdatedPayload](catalog,
		identitymodel.EventRolesUpdated, agg,
		"Legacy event: user roles updated. No longer emitted; decoded for backward compat")

	registerDerived[identitymodel.EmailChangedPayload](catalog,
		identitymodel.EventEmailChanged, agg,
		"User changed their email address")

	registerDerived[identitymodel.DisplayNameChangedPayload](catalog,
		identitymodel.EventDisplayNameChanged, agg,
		"User changed their display name")

	registerDerived[identitymodel.UserDeletedPayload](catalog,
		identitymodel.EventUserDeleted, agg,
		"User was deleted with an optional reason")

	registerDerived[identitymodel.CredentialAddedPayload](catalog,
		identitymodel.EventCredentialAdded, agg,
		"A WebAuthn credential was added to the user")

	registerDerived[identitymodel.CredentialRemovedPayload](catalog,
		identitymodel.EventCredentialRemoved, agg,
		"A WebAuthn credential was removed from the user")

	registerDerived[identitymodel.CredentialUsedPayload](catalog,
		identitymodel.EventCredentialUsed, agg,
		"A WebAuthn credential was used: the assertion's signature counter and flags were recorded")

	registerDerived[identitymodel.EmailVerifiedPayload](catalog,
		identitymodel.EventEmailVerified, agg,
		"User verified their email address")

	registerDerived[identitymodel.TOTPEnabledPayload](catalog,
		identitymodel.EventTOTPEnabled, agg,
		"TOTP two-factor authentication was enabled")

	registerDerived[identitymodel.TOTPDisabledPayload](catalog,
		identitymodel.EventTOTPDisabled, agg,
		"TOTP two-factor authentication was disabled")

	registerDerived[identitymodel.ExternalAccountLinkedPayload](catalog,
		identitymodel.EventExternalAccountLinked, agg,
		"An external account (OAuth2/OIDC) was linked to the user")

	registerDerived[identitymodel.ExternalAccountUnlinkedPayload](catalog,
		identitymodel.EventExternalAccountUnlinked, agg,
		"An external account was unlinked from the user")
}

func registerMembershipEvents(catalog *cqrshtmx.EventCatalog) {
	agg := identitymodel.AggregateTypeMembership

	registerDerived[identitymodel.MemberAddedPayload](catalog,
		identitymodel.EventMemberAdded, agg,
		"A member was added to a tenant with roles")

	registerDerived[identitymodel.MemberRolesChangedPayload](catalog,
		identitymodel.EventMemberRolesChanged, agg,
		"A member roles were changed within a tenant")

	registerDerived[identitymodel.MemberRemovedPayload](catalog,
		identitymodel.EventMemberRemoved, agg,
		"A member was removed from a tenant")
}

func registerTenantEvents(catalog *cqrshtmx.EventCatalog) {
	agg := identitymodel.AggregateTypeTenant

	registerDerived[identitymodel.TenantCreatedPayload](catalog,
		identitymodel.EventTenantCreated, agg,
		"A tenant was created")

	registerDerived[identitymodel.TenantSuspendedPayload](catalog,
		identitymodel.EventTenantSuspended, agg,
		"A tenant was suspended with an optional reason")

	registerDerived[identitymodel.TenantReactivatedPayload](catalog,
		identitymodel.EventTenantReactivated, agg,
		"A suspended tenant was reactivated")

	registerDerived[identitymodel.TenantDeletedPayload](catalog,
		identitymodel.EventTenantDeleted, agg,
		"A tenant was deleted with an optional reason")
}

func registerBotEvents(catalog *cqrshtmx.EventCatalog) {
	agg := identitymodel.AggregateTypeBot

	registerDerived[identitymodel.BotRegisteredPayload](catalog,
		identitymodel.EventBotRegistered, agg,
		"A bot was registered with a token hash and scopes")

	registerDerived[identitymodel.BotDeletedPayload](catalog,
		identitymodel.EventBotDeleted, agg,
		"A bot was deleted with an optional reason")
}

func registerAuthEvents(catalog *cqrshtmx.EventCatalog) {
	agg := identitymodel.AggregateTypeAuth

	registerDerived[identitymodel.LoginSucceededPayload](catalog,
		identitymodel.EventLoginSucceeded, agg,
		"A login succeeded (auth audit journal; one dedicated Auth stream per event)")

	registerDerived[identitymodel.LoginFailedPayload](catalog,
		identitymodel.EventLoginFailed, agg,
		"A known-user login failed with a bounded reason class (never raw error text or secrets)")

	registerDerived[identitymodel.RegistrationClosedPayload](catalog,
		identitymodel.EventRegistrationClosed, agg,
		"A registration attempt was rejected because registration is closed")
}
