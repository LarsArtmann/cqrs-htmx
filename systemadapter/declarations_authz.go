package systemadapter

import (
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	usermgmt "github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

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
		metaengine.Volume(10_000),
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
		metaengine.Volume(10_000),
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

// Declarative-only audit vocabulary: the legacy AuditLog does not audit
// external-account events, so no usermgmt.AuditAction* constant exists. The
// labels follow the legacy past-tense convention and are pinned here so the
// vocabulary stays stable if legacy grows coverage later.
const (
	auditActionLinkExternalAccount   = "link_external_account"
	auditActionUnlinkExternalAccount = "unlink_external_account"
)

func auditLogScan() metaengine.QueryDecl[system.ScanInput, AuditEntryView] {
	return metaengine.Query[system.ScanInput, AuditEntryView]("audit_log",
		auditFold(string(identitymodel.EventUserRegistered),
			identitymodel.UserRegisteredPayload{}, usermgmt.AuditActionRegister),
		auditFold(string(identitymodel.EventEmailChanged),
			identitymodel.EmailChangedPayload{}, usermgmt.AuditActionEmailChanged),
		auditFold(string(identitymodel.EventDisplayNameChanged),
			identitymodel.DisplayNameChangedPayload{}, usermgmt.AuditActionDisplayNameChanged),
		auditFold(string(identitymodel.EventUserDeleted),
			identitymodel.UserDeletedPayload{}, usermgmt.AuditActionUserDeleted),
		auditFold(string(identitymodel.EventCredentialAdded),
			identitymodel.CredentialAddedPayload{}, usermgmt.AuditActionCredentialAdded),
		auditFold(string(identitymodel.EventCredentialRemoved),
			identitymodel.CredentialRemovedPayload{}, usermgmt.AuditActionCredentialRemoved),
		auditFold(string(identitymodel.EventCredentialUsed),
			identitymodel.CredentialUsedPayload{}, usermgmt.AuditActionCredentialUsed),
		auditFold(string(identitymodel.EventEmailVerified),
			identitymodel.EmailVerifiedPayload{}, usermgmt.AuditActionEmailVerified),
		auditFold(string(identitymodel.EventTOTPEnabled),
			identitymodel.TOTPEnabledPayload{}, usermgmt.AuditActionTOTPEnabled),
		auditFold(string(identitymodel.EventTOTPDisabled),
			identitymodel.TOTPDisabledPayload{}, usermgmt.AuditActionTOTPDisabled),
		auditFold(string(identitymodel.EventExternalAccountLinked),
			identitymodel.ExternalAccountLinkedPayload{}, auditActionLinkExternalAccount),
		auditFold(string(identitymodel.EventExternalAccountUnlinked),
			identitymodel.ExternalAccountUnlinkedPayload{}, auditActionUnlinkExternalAccount),
		auditFold(string(identitymodel.EventRolesUpdated),
			identitymodel.RolesUpdatedPayload{}, usermgmt.AuditActionRolesUpdated),
		auditFold(string(identitymodel.EventLoginSucceeded),
			identitymodel.LoginSucceededPayload{}, usermgmt.AuditActionLoginSucceeded),
		loginFailedAuditFold(),
		auditFold(string(identitymodel.EventRegistrationClosed),
			identitymodel.RegistrationClosedPayload{}, usermgmt.AuditActionRegistrationClosed),
		metaengine.FilterOnField[AuditEntryView]("AggregateID", metaengine.FilterEq),
		metaengine.SortOnField[AuditEntryView]("OccurredAt", true),
		metaengine.Volume(1_000_000),
	)
}

// auditFold creates an insert fold that records an AuditEntryView for a user event.
// The key is a composite of stream ID + event type + timestamp to ensure uniqueness.
// P is the event payload type; the sample is constructed as EventWithID[P].
func auditFold[P any](eventType string, payload P, action string) metaengine.Fold {
	return metaengine.OnRecordTyped(eventType,
		projectionadapter.EventWithID[P]{Payload: payload},
		func(rec record.Record, e projectionadapter.EventWithID[P]) (string, AuditEntryView) {
			return auditKey(e.ID, eventType, e.OccurredAt),
				newAuditEntryView(rec, eventType, e.ID, e.OccurredAt, action)
		})
}

// loginFailedAuditFold is the LoginFailed specialization: beyond the shared
// entry fields it carries the bounded failure-reason class as Detail, matching
// the legacy AuditLog contract (never raw provider error text).
func loginFailedAuditFold() metaengine.Fold {
	eventType := string(identitymodel.EventLoginFailed)

	return metaengine.OnRecordTyped(eventType,
		projectionadapter.EventWithID[identitymodel.LoginFailedPayload]{},
		func(
			rec record.Record, e projectionadapter.EventWithID[identitymodel.LoginFailedPayload],
		) (string, AuditEntryView) {
			view := newAuditEntryView(rec, eventType, e.ID, e.OccurredAt, usermgmt.AuditActionLoginFailed)
			view.Detail = string(e.Payload.Reason)

			return auditKey(e.ID, eventType, e.OccurredAt), view
		})
}

// auditKey builds the composite uniqueness key for an audit entry.
func auditKey(streamID, eventType string, occurredAt time.Time) string {
	return streamID + ":" + eventType + ":" + occurredAt.Format(time.RFC3339Nano)
}

// newAuditEntryView builds the shared audit-entry shape, carrying the actor
// from the record's metadata (S42c): the self-describing "kind:raw" wire form
// when one is recorded, empty otherwise. The declarative dispatch path itself
// attaches no actors (system.Execute drops the context) — records written by
// an enriching writer against the same store (the usermgmt service path wires
// event.ActorEnricher) replay through these folds with their actors intact.
func newAuditEntryView(
	rec record.Record, eventType, aggregateID string, occurredAt time.Time, action string,
) AuditEntryView {
	view := AuditEntryView{
		EventType:   eventType,
		AggregateID: aggregateID,
		OccurredAt:  occurredAt,
		Action:      action,
	}

	if !rec.MetaData.Actor.IsZero() {
		view.ActorID = rec.MetaData.Actor.String()
	}

	return view
}

