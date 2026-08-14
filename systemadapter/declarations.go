// cqrs-lint:ignore(E014) read-your-writes is owned by consumers via ProjectionLayer.WaitForDrain (projections.go); the
// adapter itself never serves commands
package systemadapter

import (
	"bytes"
	"encoding/hex"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
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
		system.RawQuery(externalAccountLinkScan()),
		// Authz
		system.RawQuery(authzPolicyLookup()),
		system.RawQuery(authzPolicyScan()),
		// AuditLog
		system.RawQuery(auditLogScan()),
	}
}

// -------------------------------------------------------------------------
// Tenant
// -------------------------------------------------------------------------

func tenantLookup() metaengine.QueryDecl[system.LookupInput[string], TenantView] {
	// cqrs-lint:ignore(F019) metaengine.Query has no volume-hint option (checked v4.10.0); planner falls back to its
	// default volume
	return metaengine.Query[system.LookupInput[string], TenantView]("tenant_by_id",
		insertTenant(),
		updateTenantSuspended(),
		updateTenantReactivated(),
		removeTenant(),
	)
}

func tenantScan() metaengine.QueryDecl[system.ScanInput, TenantView] {
	return metaengine.Query[system.ScanInput, TenantView]("tenants",
		insertTenant(),
		updateTenantSuspended(),
		updateTenantReactivated(),
		removeTenant(),
		metaengine.FilterOnField[TenantView]("Name", metaengine.FilterEq),
		metaengine.FilterOnField[TenantView]("Suspended", metaengine.FilterEq),
	)
}

func insertTenant() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventTenantCreated),
		projectionadapter.EventWithID[identitymodel.TenantCreatedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.TenantCreatedPayload]) (string, TenantView) {
			return e.ID, TenantView{
				ID:          e.ID,
				Name:        e.Payload.Name,
				DisplayName: e.Payload.DisplayName,
			}
		},
	)
}

func updateTenantSuspended() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventTenantSuspended),
		projectionadapter.EventWithID[identitymodel.TenantSuspendedPayload]{},
		func(
			_ record.Record,
			_ projectionadapter.EventWithID[identitymodel.TenantSuspendedPayload],
			prev TenantView,
		) TenantView {
			prev.Suspended = true
			return prev
		},
	)
}

func updateTenantReactivated() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventTenantReactivated),
		projectionadapter.EventWithID[identitymodel.TenantReactivatedPayload]{},
		func(
			_ record.Record,
			_ projectionadapter.EventWithID[identitymodel.TenantReactivatedPayload],
			prev TenantView,
		) TenantView {
			prev.Suspended = false
			return prev
		},
	)
}

func removeTenant() metaengine.Fold {
	return metaengine.OnRecordTyped(string(identitymodel.EventTenantDeleted),
		projectionadapter.EventWithID[identitymodel.TenantDeletedPayload]{},
		metaengine.Remove[TenantView]())
}

// -------------------------------------------------------------------------
// Bot
// -------------------------------------------------------------------------

func botLookup() metaengine.QueryDecl[system.LookupInput[string], BotView] {
	return metaengine.Query[system.LookupInput[string], BotView]("bot_by_id",
		insertBot(),
		removeBot(),
	)
}

func botScan() metaengine.QueryDecl[system.ScanInput, BotView] {
	return metaengine.Query[system.ScanInput, BotView]("bots",
		insertBot(),
		removeBot(),
		metaengine.FilterOnField[BotView]("OwnerID", metaengine.FilterEq),
	)
}

func insertBot() metaengine.Fold {
	return metaengine.OnRecordTyped(string(identitymodel.EventBotRegistered),
		projectionadapter.EventWithID[identitymodel.BotRegisteredPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.BotRegisteredPayload]) (string, BotView) {
			return e.ID, BotView{
				ID:        e.ID,
				Name:      e.Payload.Name,
				OwnerID:   e.Payload.OwnerID.String(),
				TokenHash: e.Payload.TokenHash,
				Scopes:    e.Payload.Scopes,
			}
		})
}

func removeBot() metaengine.Fold {
	return metaengine.OnRecordTyped(string(identitymodel.EventBotDeleted),
		projectionadapter.EventWithID[identitymodel.BotDeletedPayload]{},
		metaengine.Remove[BotView]())
}

// botTokenScan provides a secondary index for looking up bots by token hash.
// Keyed by bot stream ID (so removes work), with TokenHash as a filterable field.
func botTokenScan() metaengine.QueryDecl[system.ScanInput, BotTokenView] {
	return metaengine.Query[system.ScanInput, BotTokenView]("bot_tokens",
		metaengine.OnRecordTyped(
			string(identitymodel.EventBotRegistered),
			projectionadapter.EventWithID[identitymodel.BotRegisteredPayload]{},
			func(_ record.Record, e projectionadapter.EventWithID[identitymodel.BotRegisteredPayload]) (string, BotTokenView) {
				return e.ID, BotTokenView{
					TokenHash: hex.EncodeToString(e.Payload.TokenHash),
					BotID:     e.ID,
				}
			},
		),
		metaengine.OnRecordTyped(string(identitymodel.EventBotDeleted),
			projectionadapter.EventWithID[identitymodel.BotDeletedPayload]{},
			metaengine.Remove[BotTokenView]()),
		metaengine.FilterOnField[BotTokenView]("TokenHash", metaengine.FilterEq),
	)
}

// -------------------------------------------------------------------------
// Membership
// -------------------------------------------------------------------------

func membershipLookup() metaengine.QueryDecl[system.LookupInput[string], MembershipView] {
	return metaengine.Query[system.LookupInput[string], MembershipView]("membership_by_id",
		insertMembership(),
		updateMembershipRoles(),
		removeMembership(),
	)
}

func membershipScan() metaengine.QueryDecl[system.ScanInput, MembershipView] {
	return metaengine.Query[system.ScanInput, MembershipView]("memberships",
		insertMembership(),
		updateMembershipRoles(),
		removeMembership(),
		metaengine.FilterOnField[MembershipView]("ActorID", metaengine.FilterEq),
		metaengine.FilterOnField[MembershipView]("TenantID", metaengine.FilterEq),
	)
}

func insertMembership() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventMemberAdded),
		projectionadapter.EventWithID[identitymodel.MemberAddedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.MemberAddedPayload]) (string, MembershipView) {
			return e.ID, MembershipView{
				ID:       e.ID,
				ActorID:  e.Payload.ActorID,
				TenantID: e.Payload.TenantID,
				Roles:    rolesToStrings(e.Payload.Roles),
			}
		},
	)
}

func updateMembershipRoles() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventMemberRolesChanged),
		projectionadapter.EventWithID[identitymodel.MemberRolesChangedPayload]{},
		func(
			_ record.Record,
			e projectionadapter.EventWithID[identitymodel.MemberRolesChangedPayload],
			prev MembershipView,
		) MembershipView {
			prev.Roles = rolesToStrings(e.Payload.Roles)
			return prev
		},
	)
}

func removeMembership() metaengine.Fold {
	return metaengine.OnRecordTyped(string(identitymodel.EventMemberRemoved),
		projectionadapter.EventWithID[identitymodel.MemberRemovedPayload]{},
		metaengine.Remove[MembershipView]())
}

// -------------------------------------------------------------------------
// User
// -------------------------------------------------------------------------

func userLookup() metaengine.QueryDecl[system.LookupInput[string], UserView] {
	return metaengine.Query[system.LookupInput[string], UserView]("user_by_id",
		insertUser(),
		updateEmailChanged(),
		updateDisplayNameChanged(),
		updateCredentialAdded(),
		updateCredentialRemoved(),
		updateEmailVerified(),
		updateTOTPEnabled(),
		updateTOTPDisabled(),
		updateExternalAccountLinked(),
		updateExternalAccountUnlinked(),
		removeUser(),
	)
}

func userScan() metaengine.QueryDecl[system.ScanInput, UserView] {
	return metaengine.Query[system.ScanInput, UserView]("users",
		insertUser(),
		updateEmailChanged(),
		updateDisplayNameChanged(),
		updateCredentialAdded(),
		updateCredentialRemoved(),
		updateEmailVerified(),
		updateTOTPEnabled(),
		updateTOTPDisabled(),
		updateExternalAccountLinked(),
		updateExternalAccountUnlinked(),
		removeUser(),
		metaengine.FilterOnField[UserView]("Email", metaengine.FilterEq),
		metaengine.FilterOnField[UserView]("EmailVerified", metaengine.FilterEq),
		metaengine.SortOnField[UserView]("CreatedAt", false),
	)
}

func insertUser() metaengine.Fold {
	return metaengine.OnRecordTyped(string(identitymodel.EventUserRegistered),
		projectionadapter.EventWithID[identitymodel.UserRegisteredPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.UserRegisteredPayload]) (string, UserView) {
			return e.ID, UserView{
				ID:          e.ID,
				Email:       e.Payload.Email,
				DisplayName: e.Payload.DisplayName,
				CreatedAt:   e.OccurredAt,
				UpdatedAt:   e.OccurredAt,
			}
		})
}

func updateEmailChanged() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventEmailChanged),
		projectionadapter.EventWithID[identitymodel.EmailChangedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.EmailChangedPayload], prev UserView) UserView {
			prev.Email = e.Payload.Email
			prev.EmailVerified = false
			prev.UpdatedAt = e.OccurredAt
			return prev
		},
	)
}

func updateDisplayNameChanged() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventDisplayNameChanged),
		projectionadapter.EventWithID[identitymodel.DisplayNameChangedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.DisplayNameChangedPayload], prev UserView) UserView {
			prev.DisplayName = e.Payload.DisplayName
			prev.UpdatedAt = e.OccurredAt
			return prev
		},
	)
}

func updateCredentialAdded() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventCredentialAdded),
		projectionadapter.EventWithID[identitymodel.CredentialAddedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.CredentialAddedPayload], prev UserView) UserView {
			prev.Credentials = append(prev.Credentials, CredentialView{
				ID:              e.Payload.ID,
				PublicKey:       e.Payload.PublicKey,
				AttestationType: e.Payload.AttestationType,
				Transports:      e.Payload.Transports,
				AAGUID:          e.Payload.AAGUID,
				SignCount:       e.Payload.SignCount,
				BackupEligible:  e.Payload.BackupEligible,
				BackupState:     e.Payload.BackupState,
				Name:            e.Payload.Name,
			})
			prev.UpdatedAt = e.OccurredAt
			return prev
		},
	)
}

func updateCredentialRemoved() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventCredentialRemoved),
		projectionadapter.EventWithID[identitymodel.CredentialRemovedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.CredentialRemovedPayload], prev UserView) UserView {
			filtered := prev.Credentials[:0]
			for _, c := range prev.Credentials {
				if !bytes.Equal(c.ID, e.Payload.ID) {
					filtered = append(filtered, c)
				}
			}
			prev.Credentials = filtered
			prev.UpdatedAt = e.OccurredAt
			return prev
		},
	)
}

func updateEmailVerified() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventEmailVerified),
		projectionadapter.EventWithID[identitymodel.EmailVerifiedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.EmailVerifiedPayload], prev UserView) UserView {
			prev.EmailVerified = true
			prev.UpdatedAt = e.OccurredAt
			return prev
		},
	)
}

func updateTOTPEnabled() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventTOTPEnabled),
		projectionadapter.EventWithID[identitymodel.TOTPEnabledPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.TOTPEnabledPayload], prev UserView) UserView {
			prev.TOTPEnabled = true
			prev.UpdatedAt = e.OccurredAt
			return prev
		},
	)
}

func updateTOTPDisabled() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventTOTPDisabled),
		projectionadapter.EventWithID[identitymodel.TOTPDisabledPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.TOTPDisabledPayload], prev UserView) UserView {
			prev.TOTPEnabled = false
			prev.UpdatedAt = e.OccurredAt
			return prev
		},
	)
}

func updateExternalAccountLinked() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventExternalAccountLinked),
		projectionadapter.EventWithID[identitymodel.ExternalAccountLinkedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.ExternalAccountLinkedPayload], prev UserView) UserView {
			prev.ExternalAccounts = append(prev.ExternalAccounts, ExternalAccountView{
				Provider:    e.Payload.Provider,
				Subject:     e.Payload.Subject,
				Email:       e.Payload.Email,
				DisplayName: e.Payload.DisplayName,
			})
			prev.UpdatedAt = e.OccurredAt
			return prev
		},
	)
}

func updateExternalAccountUnlinked() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventExternalAccountUnlinked),
		projectionadapter.EventWithID[identitymodel.ExternalAccountUnlinkedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.ExternalAccountUnlinkedPayload], prev UserView) UserView {
			filtered := prev.ExternalAccounts[:0]
			for _, ea := range prev.ExternalAccounts {
				if ea.Provider != e.Payload.Provider || ea.Subject != e.Payload.Subject {
					filtered = append(filtered, ea)
				}
			}
			prev.ExternalAccounts = filtered
			prev.UpdatedAt = e.OccurredAt
			return prev
		},
	)
}

func removeUser() metaengine.Fold {
	return metaengine.OnRecordTyped(string(identitymodel.EventUserDeleted),
		projectionadapter.EventWithID[identitymodel.UserDeletedPayload]{},
		metaengine.Remove[UserView]())
}

// -------------------------------------------------------------------------
// External Account Link (secondary index for User)
// -------------------------------------------------------------------------

func externalAccountLinkScan() metaengine.QueryDecl[system.ScanInput, ExternalAccountLink] {
	return metaengine.Query[system.ScanInput, ExternalAccountLink]("external_account_links",
		metaengine.OnRecordTyped(
			string(identitymodel.EventExternalAccountLinked),
			projectionadapter.EventWithID[identitymodel.ExternalAccountLinkedPayload]{},
			func(_ record.Record, e projectionadapter.EventWithID[identitymodel.ExternalAccountLinkedPayload]) (string, ExternalAccountLink) {
				return e.ID + ":" + e.Payload.Provider + ":" + e.Payload.Subject,
					ExternalAccountLink{
						ProviderSubject: e.Payload.Provider + ":" + e.Payload.Subject,
						UserID:          e.ID,
					}
			},
		),
		metaengine.OnRecordTyped(
			string(identitymodel.EventExternalAccountUnlinked),
			projectionadapter.EventWithID[identitymodel.ExternalAccountUnlinkedPayload]{},
			func(_ record.Record, e projectionadapter.EventWithID[identitymodel.ExternalAccountUnlinkedPayload], prev ExternalAccountLink) ExternalAccountLink {
				_ = prev
				return ExternalAccountLink{}
			},
		),
		metaengine.FilterOnField[ExternalAccountLink]("ProviderSubject", metaengine.FilterEq),
	)
}

// -------------------------------------------------------------------------
// Authz Policy (Casbin replacement)
// -------------------------------------------------------------------------

// PolicyEntry is keyed by the stream ID of the aggregate that produced it.
// For user roles: key = user stream ID, Subject = user ID, Domain = user ID.
// For membership roles: key = membership stream ID, Subject = actor ID, Domain = tenant ID.
// This makes removes work correctly: MemberRemoved's e.ID matches the insert key.

func authzPolicyLookup() metaengine.QueryDecl[system.LookupInput[string], PolicyEntry] {
	return metaengine.Query[system.LookupInput[string], PolicyEntry]("authz_policy_by_id",
		insertUserPolicy(),
		updateUserPolicy(),
		insertMemberPolicy(),
		updateMemberPolicy(),
		removeMemberPolicy(),
		removeUserPolicy(),
	)
}

func authzPolicyScan() metaengine.QueryDecl[system.ScanInput, PolicyEntry] {
	return metaengine.Query[system.ScanInput, PolicyEntry]("authz_policies",
		insertUserPolicy(),
		updateUserPolicy(),
		insertMemberPolicy(),
		updateMemberPolicy(),
		removeMemberPolicy(),
		removeUserPolicy(),
		metaengine.FilterOnField[PolicyEntry]("Subject", metaengine.FilterEq),
		metaengine.FilterOnField[PolicyEntry]("Domain", metaengine.FilterEq),
	)
}

func insertUserPolicy() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventUserRegistered),
		projectionadapter.EventWithID[identitymodel.UserRegisteredPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.UserRegisteredPayload]) (string, PolicyEntry) {
			return e.ID, PolicyEntry{
				Key:     e.ID,
				Subject: e.ID,
				Domain:  e.ID,
				Roles:   rolesToStrings(e.Payload.Roles),
			}
		},
	)
}

func updateUserPolicy() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventRolesUpdated),
		projectionadapter.EventWithID[identitymodel.RolesUpdatedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.RolesUpdatedPayload], prev PolicyEntry) PolicyEntry {
			prev.Roles = rolesToStrings(e.Payload.Roles)
			return prev
		},
	)
}

func insertMemberPolicy() metaengine.Fold {
	return metaengine.OnRecordTyped(string(identitymodel.EventMemberAdded),
		projectionadapter.EventWithID[identitymodel.MemberAddedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.MemberAddedPayload]) (string, PolicyEntry) {
			return e.ID, PolicyEntry{
				Key:     e.ID,
				Subject: e.Payload.ActorID,
				Domain:  e.Payload.TenantID,
				Roles:   rolesToStrings(e.Payload.Roles),
			}
		})
}

func updateMemberPolicy() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventMemberRolesChanged),
		projectionadapter.EventWithID[identitymodel.MemberRolesChangedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.MemberRolesChangedPayload], prev PolicyEntry) PolicyEntry {
			prev.Roles = rolesToStrings(e.Payload.Roles)
			return prev
		},
	)
}

func removeMemberPolicy() metaengine.Fold {
	return metaengine.OnRecordTyped(string(identitymodel.EventMemberRemoved),
		projectionadapter.EventWithID[identitymodel.MemberRemovedPayload]{},
		metaengine.Remove[PolicyEntry]())
}

func removeUserPolicy() metaengine.Fold {
	return metaengine.OnRecordTyped(string(identitymodel.EventUserDeleted),
		projectionadapter.EventWithID[identitymodel.UserDeletedPayload]{},
		metaengine.Remove[PolicyEntry]())
}

// -------------------------------------------------------------------------
// AuditLog
// -------------------------------------------------------------------------

func auditLogScan() metaengine.QueryDecl[system.ScanInput, AuditEntryView] {
	return metaengine.Query[system.ScanInput, AuditEntryView]("audit_log",
		auditFold(string(identitymodel.EventUserRegistered),
			identitymodel.UserRegisteredPayload{}, "register"),
		auditFold(string(identitymodel.EventEmailChanged),
			identitymodel.EmailChangedPayload{}, "change_email"),
		auditFold(string(identitymodel.EventDisplayNameChanged),
			identitymodel.DisplayNameChangedPayload{}, "change_display_name"),
		auditFold(string(identitymodel.EventUserDeleted),
			identitymodel.UserDeletedPayload{}, "delete_user"),
		auditFold(string(identitymodel.EventCredentialAdded),
			identitymodel.CredentialAddedPayload{}, "add_credential"),
		auditFold(string(identitymodel.EventCredentialRemoved),
			identitymodel.CredentialRemovedPayload{}, "remove_credential"),
		auditFold(string(identitymodel.EventEmailVerified),
			identitymodel.EmailVerifiedPayload{}, "verify_email"),
		auditFold(string(identitymodel.EventTOTPEnabled),
			identitymodel.TOTPEnabledPayload{}, "enable_totp"),
		auditFold(string(identitymodel.EventTOTPDisabled),
			identitymodel.TOTPDisabledPayload{}, "disable_totp"),
		auditFold(string(identitymodel.EventExternalAccountLinked),
			identitymodel.ExternalAccountLinkedPayload{}, "link_external_account"),
		auditFold(string(identitymodel.EventExternalAccountUnlinked),
			identitymodel.ExternalAccountUnlinkedPayload{}, "unlink_external_account"),
		auditFold(string(identitymodel.EventRolesUpdated),
			identitymodel.RolesUpdatedPayload{}, "update_roles"),
		metaengine.FilterOnField[AuditEntryView]("AggregateID", metaengine.FilterEq),
		metaengine.SortOnField[AuditEntryView]("OccurredAt", true),
	)
}

// auditFold creates an insert fold that records an AuditEntryView for a user event.
// The key is a composite of stream ID + event type + timestamp to ensure uniqueness.
// P is the event payload type; the sample is constructed as EventWithID[P].
func auditFold[P any](eventType string, payload P, action string) metaengine.Fold {
	return metaengine.OnRecordTyped(eventType,
		projectionadapter.EventWithID[P]{Payload: payload},
		func(_ record.Record, e projectionadapter.EventWithID[P]) (string, AuditEntryView) {
			return e.ID + ":" + eventType + ":" + e.OccurredAt.Format(time.RFC3339Nano),
				AuditEntryView{
					EventType:   eventType,
					AggregateID: e.ID,
					OccurredAt:  e.OccurredAt,
					Action:      action,
				}
		})
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
