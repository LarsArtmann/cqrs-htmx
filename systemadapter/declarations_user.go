package systemadapter

import (
	"bytes"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

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
		updateCredentialUsed(),
		updateEmailVerified(),
		updateTOTPEnabled(),
		updateTOTPDisabled(),
		updateExternalAccountLinked(),
		updateExternalAccountUnlinked(),
		removeUser(),
		metaengine.Volume(100_000),
	)
}

func userScan() metaengine.QueryDecl[system.ScanInput, UserView] {
	return metaengine.Query[system.ScanInput, UserView]("users",
		insertUser(),
		updateEmailChanged(),
		updateDisplayNameChanged(),
		updateCredentialAdded(),
		updateCredentialRemoved(),
		updateCredentialUsed(),
		updateEmailVerified(),
		updateTOTPEnabled(),
		updateTOTPDisabled(),
		updateExternalAccountLinked(),
		updateExternalAccountUnlinked(),
		removeUser(),
		metaengine.FilterOnField[UserView]("Email", metaengine.FilterEq),
		metaengine.FilterOnField[UserView]("EmailVerified", metaengine.FilterEq),
		metaengine.SortOnField[UserView]("CreatedAt", false),
		metaengine.Volume(100_000),
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

func updateCredentialUsed() metaengine.Fold {
	return metaengine.OnRecordTyped(
		string(identitymodel.EventCredentialUsed),
		projectionadapter.EventWithID[identitymodel.CredentialUsedPayload]{},
		func(_ record.Record, e projectionadapter.EventWithID[identitymodel.CredentialUsedPayload], prev UserView) UserView {
			for i := range prev.Credentials {
				if !bytes.Equal(prev.Credentials[i].ID, e.Payload.CredentialID) {
					continue
				}
				prev.Credentials[i].SignCount = e.Payload.SignCount
				prev.Credentials[i].UserPresent = e.Payload.UserPresent
				prev.Credentials[i].UserVerified = e.Payload.UserVerified
				prev.Credentials[i].BackupState = e.Payload.BackupState
				break
			}
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
