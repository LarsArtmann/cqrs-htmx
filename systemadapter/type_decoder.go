package systemadapter

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
)

// EventTypeDecoder returns a projectionadapter.TypeDecoder that maps all 21
// identity-model event types to their payload structs. This enables the
// system.New() projection host to decode events for metaengine fold handlers
// without manual switch/case boilerplate.
//
// The decoder wraps each payload in a projectionadapter.EventWithID[P] struct,
// giving fold handlers access to both the stream ID and the typed payload:
//
//	type UserView struct { Email string }
//	dec := systemadapter.EventTypeDecoder()
//	// Register with system.DomainConfig.ProjectionTypeDecoder
func EventTypeDecoder() *projectionadapter.TypeDecoder {
	return projectionadapter.NewTypeDecoder(
		// User events
		projectionadapter.Register[identitymodel.UserRegisteredPayload](
			identitymodel.EventUserRegistered, identitymodel.UserRegisteredPayload{},
		),
		projectionadapter.Register[identitymodel.RolesUpdatedPayload](
			identitymodel.EventRolesUpdated, identitymodel.RolesUpdatedPayload{},
		),
		projectionadapter.Register[identitymodel.EmailChangedPayload](
			identitymodel.EventEmailChanged, identitymodel.EmailChangedPayload{},
		),
		projectionadapter.Register[identitymodel.DisplayNameChangedPayload](
			identitymodel.EventDisplayNameChanged, identitymodel.DisplayNameChangedPayload{},
		),
		projectionadapter.Register[identitymodel.UserDeletedPayload](
			identitymodel.EventUserDeleted, identitymodel.UserDeletedPayload{},
		),
		projectionadapter.Register[identitymodel.CredentialAddedPayload](
			identitymodel.EventCredentialAdded, identitymodel.CredentialAddedPayload{},
		),
		projectionadapter.Register[identitymodel.CredentialRemovedPayload](
			identitymodel.EventCredentialRemoved, identitymodel.CredentialRemovedPayload{},
		),
		projectionadapter.Register[identitymodel.EmailVerifiedPayload](
			identitymodel.EventEmailVerified, identitymodel.EmailVerifiedPayload{},
		),
		projectionadapter.Register[identitymodel.TOTPEnabledPayload](
			identitymodel.EventTOTPEnabled, identitymodel.TOTPEnabledPayload{},
		),
		projectionadapter.Register[identitymodel.TOTPDisabledPayload](
			identitymodel.EventTOTPDisabled, identitymodel.TOTPDisabledPayload{},
		),
		projectionadapter.Register[identitymodel.ExternalAccountLinkedPayload](
			identitymodel.EventExternalAccountLinked, identitymodel.ExternalAccountLinkedPayload{},
		),
		projectionadapter.Register[identitymodel.ExternalAccountUnlinkedPayload](
			identitymodel.EventExternalAccountUnlinked, identitymodel.ExternalAccountUnlinkedPayload{},
		),

		// Membership events
		projectionadapter.Register[identitymodel.MemberAddedPayload](
			identitymodel.EventMemberAdded, identitymodel.MemberAddedPayload{},
		),
		projectionadapter.Register[identitymodel.MemberRolesChangedPayload](
			identitymodel.EventMemberRolesChanged, identitymodel.MemberRolesChangedPayload{},
		),
		projectionadapter.Register[identitymodel.MemberRemovedPayload](
			identitymodel.EventMemberRemoved, identitymodel.MemberRemovedPayload{},
		),

		// Tenant events
		projectionadapter.Register[identitymodel.TenantCreatedPayload](
			identitymodel.EventTenantCreated, identitymodel.TenantCreatedPayload{},
		),
		projectionadapter.Register[identitymodel.TenantSuspendedPayload](
			identitymodel.EventTenantSuspended, identitymodel.TenantSuspendedPayload{},
		),
		projectionadapter.Register[identitymodel.TenantReactivatedPayload](
			identitymodel.EventTenantReactivated, identitymodel.TenantReactivatedPayload{},
		),
		projectionadapter.Register[identitymodel.TenantDeletedPayload](
			identitymodel.EventTenantDeleted, identitymodel.TenantDeletedPayload{},
		),

		// Bot events
		projectionadapter.Register[identitymodel.BotRegisteredPayload](
			identitymodel.EventBotRegistered, identitymodel.BotRegisteredPayload{},
		),
		projectionadapter.Register[identitymodel.BotDeletedPayload](
			identitymodel.EventBotDeleted, identitymodel.BotDeletedPayload{},
		),
	)
}
