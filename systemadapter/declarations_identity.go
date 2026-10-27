package systemadapter

import (
	"encoding/hex"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// -------------------------------------------------------------------------
// Tenant
// -------------------------------------------------------------------------

func tenantLookup() metaengine.QueryDecl[system.LookupInput[string], TenantView] {
	return metaengine.Query[system.LookupInput[string], TenantView]("tenant_by_id",
		insertTenant(),
		updateTenantSuspended(),
		updateTenantReactivated(),
		removeTenant(),
		metaengine.Volume(100),
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
		metaengine.Volume(100),
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
		metaengine.Volume(1_000),
	)
}

func botScan() metaengine.QueryDecl[system.ScanInput, BotView] {
	return metaengine.Query[system.ScanInput, BotView]("bots",
		insertBot(),
		removeBot(),
		metaengine.FilterOnField[BotView]("OwnerID", metaengine.FilterEq),
		metaengine.Volume(1_000),
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
		metaengine.Volume(1_000),
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
		metaengine.Volume(10_000),
	)
}

func membershipScan() metaengine.QueryDecl[system.ScanInput, MembershipView] {
	return metaengine.Query[system.ScanInput, MembershipView]("memberships",
		insertMembership(),
		updateMembershipRoles(),
		removeMembership(),
		metaengine.FilterOnField[MembershipView]("ActorID", metaengine.FilterEq),
		metaengine.FilterOnField[MembershipView]("TenantID", metaengine.FilterEq),
		metaengine.Volume(10_000),
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
