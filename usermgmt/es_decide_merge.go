package usermgmt

import (
	"bytes"
	"slices"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// mergeAdoption accumulates the survivor side of one MergeUsers decision:
// the assets adopted so far and the ordinary lifecycle events that record
// them, so every existing projection adopts a merged user's authenticators
// without merge-specific code.
type mergeAdoption struct {
	aggID           id.StreamID
	state           UserState
	version         event.Version
	events          []event.Event
	credentialIDs   [][]byte
	accounts        []externalAccountKey
	totpAdopted     bool
	verifiedAdopted bool
}

func newMergeAdoption(aggID id.StreamID, state UserState, version event.Version) *mergeAdoption {
	return &mergeAdoption{
		aggID:           aggID,
		state:           state,
		version:         version,
		events:          nil,
		credentialIDs:   nil,
		accounts:        nil,
		totpAdopted:     false,
		verifiedAdopted: false,
	}
}

func (a *mergeAdoption) emit(eventType event.Type, enc codec.Codec, payload any, what string) error {
	a.version = a.version.Increment()
	evt, err := event.New(
		eventType, a.aggID, aggregateTypeUser, a.version, payload,
		//cqrs-lint:ignore(A027) one codec per repository is intentional; global DefaultCodec would couple unrelated aggregates
		event.WithCodec(enc), event.WithSchemaVersion(1),
	)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "usermgmt.merge_users.event_failed", "create "+what+" event")
	}
	a.events = append(a.events, evt)
	return nil
}

func (a *mergeAdoption) holdsCredential(credentialID []byte) bool {
	return slices.ContainsFunc(a.credentialIDs, func(adopted []byte) bool {
		return bytes.Equal(adopted, credentialID)
	})
}

func (a *mergeAdoption) holdsAccount(key externalAccountKey) bool {
	return slices.Contains(a.accounts, key) || slices.ContainsFunc(a.state.ExternalAccounts,
		func(ea ExternalAccount) bool { return ea.Provider == key.provider && ea.Subject == key.subject })
}

func (a *mergeAdoption) adoptCredentials(creds []WebAuthnCredential) error {
	for _, cred := range creds {
		if len(cred.ID) == 0 || a.state.HasCredential(cred.ID) || a.holdsCredential(cred.ID) {
			continue
		}
		if err := a.emit(eventCredentialAdded, cborEventCodec, CredentialAddedPayload{
			SchemaVersion:   currentSchemaVersion,
			ID:              cred.ID,
			PublicKey:       cred.PublicKey,
			AttestationType: cred.AttestationType,
			Transports:      cred.Transports,
			AAGUID:          cred.AAGUID,
			SignCount:       cred.SignCount,
			UserPresent:     cred.UserPresent,
			UserVerified:    cred.UserVerified,
			BackupEligible:  cred.BackupEligible,
			BackupState:     cred.BackupState,
			Name:            cred.Name,
		}, "CredentialAdded"); err != nil {
			return err
		}
		a.credentialIDs = append(a.credentialIDs, cred.ID)
	}
	return nil
}

func (a *mergeAdoption) adoptExternalAccounts(accounts []ExternalAccount) error {
	for _, account := range accounts {
		key := externalAccountKey{provider: account.Provider, subject: account.Subject}
		if key.provider == "" || key.subject == "" || a.holdsAccount(key) {
			continue
		}
		if err := a.emit(eventExternalAccountLinked, codec.JSONCodec{}, ExternalAccountLinkedPayload{
			SchemaVersion: currentSchemaVersion,
			Provider:      account.Provider,
			Subject:       account.Subject,
			Email:         account.Email,
			DisplayName:   account.DisplayName,
		}, "ExternalAccountLinked"); err != nil {
			return err
		}
		a.accounts = append(a.accounts, key)
	}
	return nil
}

func (a *mergeAdoption) adoptTOTP(secret []byte) error {
	if a.state.TOTPEnabled || a.totpAdopted || len(secret) == 0 {
		return nil
	}
	if err := a.emit(eventTOTPEnabled, cborEventCodec, TOTPEnabledPayload{
		SchemaVersion: currentSchemaVersion,
		Secret:        secret,
	}, "TOTPEnabled"); err != nil {
		return err
	}
	a.totpAdopted = true
	return nil
}

func (a *mergeAdoption) adoptEmailVerified(contribution identitymodel.MergeContribution) error {
	if a.state.EmailVerified || a.verifiedAdopted || !contribution.EmailVerified ||
		contribution.Email != a.state.Email {
		return nil
	}
	if err := a.emit(eventEmailVerified, codec.JSONCodec{}, EmailVerifiedPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         a.state.Email,
	}, "EmailVerified"); err != nil {
		return err
	}
	a.verifiedAdopted = true
	return nil
}

func (a *mergeAdoption) adopt(contribution identitymodel.MergeContribution) error {
	if err := a.adoptCredentials(contribution.Credentials); err != nil {
		return err
	}
	if err := a.adoptExternalAccounts(contribution.ExternalAccounts); err != nil {
		return err
	}
	if err := a.adoptTOTP(contribution.TOTPSecret); err != nil {
		return err
	}
	return a.adoptEmailVerified(contribution)
}

// unmergedLosers returns the contributing users the survivor has not
// recorded yet, in contribution order and without duplicates.
func unmergedLosers(state UserState, contributions []identitymodel.MergeContribution) []UserID {
	var losers []UserID
	for _, contribution := range contributions {
		if !slices.Contains(state.MergedUserIDs, contribution.UserID) &&
			!slices.Contains(losers, contribution.UserID) {
			losers = append(losers, contribution.UserID)
		}
	}
	return losers
}

// requireLiveSurvivor refuses a stream that cannot absorb another identity:
// one that was never registered, one that was merged away itself (it names
// the survivor it points at, so the caller sees the real identity), and one
// that was deleted.
func requireLiveSurvivor(state UserState) error {
	if err := requireExists(state, "merge_users"); err != nil {
		return err
	}
	if state.IsMerged() {
		return errorfamily.NewConflict("usermgmt.merge_users.survivor_merged",
			"a merged-away user cannot be the survivor of another merge").
			WithContext("merged_into", state.MergedInto.Get().String())
	}
	if state.Deleted {
		return errorfamily.NewRejection("usermgmt.merge_users.deleted",
			"cannot merge into a deleted user")
	}
	return nil
}

// requireOtherContributors refuses a contribution that names no user or names
// the survivor itself.
func requireOtherContributors(aggID id.StreamID, contributions []identitymodel.MergeContribution) error {
	for _, contribution := range contributions {
		switch {
		case contribution.UserID.IsZero():
			return errorfamily.NewRejection("usermgmt.merge_users.invalid_contribution",
				"contribution user ID is required")
		case contribution.UserID.Get().String() == aggID.Get():
			return errorfamily.NewRejection("usermgmt.merge_users.self_merge",
				"a user cannot be merged into itself")
		}
	}
	return nil
}

func decideMergeUsers(
	aggID id.StreamID,
	contributions []identitymodel.MergeContribution,
	reason string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireLiveSurvivor(state); err != nil {
			return nil, err
		}
		if len(contributions) == 0 {
			return nil, errorfamily.NewRejection("usermgmt.merge_users.no_contributions",
				"a merge needs at least one contribution")
		}
		if err := requireOtherContributors(aggID, contributions); err != nil {
			return nil, err
		}

		adoption := newMergeAdoption(aggID, state, version)
		for _, contribution := range contributions {
			if err := adoption.adopt(contribution); err != nil {
				return nil, err
			}
		}
		losers := unmergedLosers(state, contributions)
		if len(adoption.events) == 0 && len(losers) == 0 {
			return nil, nil
		}
		if len(losers) > 0 {
			if err := adoption.emit(eventUserMerged, codec.JSONCodec{}, identitymodel.UserMergedPayload{
				SchemaVersion: currentSchemaVersion,
				LoserIDs:      losers,
				Reason:        reason,
			}, "UserMerged"); err != nil {
				return nil, err
			}
		}
		return adoption.events, nil
	}
}

func decideMergeUserInto(
	aggID id.StreamID,
	survivorID UserID,
	reason string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "merge_user_into"); err != nil {
			return nil, err
		}
		switch {
		case survivorID.IsZero():
			return nil, errorfamily.NewRejection("usermgmt.merge_user_into.survivor_required",
				"survivor ID is required")
		case survivorID.Get().String() == aggID.Get():
			return nil, errorfamily.NewRejection("usermgmt.merge_user_into.self_merge",
				"a user cannot be merged into itself")
		case state.IsMerged() && state.MergedInto.Get() == survivorID.Get():
			return nil, nil
		case state.IsMerged():
			return nil, errorfamily.NewConflict("usermgmt.merge_user_into.already_merged",
				"user was already merged into another user").
				WithContext("merged_into", state.MergedInto.Get().String())
		case state.Deleted:
			return nil, errorfamily.NewConflict("usermgmt.merge_user_into.deleted",
				"only a live user can be merged away")
		}
		evt, err := event.New(
			eventUserMergedInto, aggID, aggregateTypeUser, version.Increment(),
			identitymodel.UserMergedIntoPayload{
				SchemaVersion: currentSchemaVersion,
				SurvivorID:    survivorID,
				Reason:        reason,
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.merge_user_into.event_failed",
				"create UserMergedInto event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideTransferBot(
	aggID id.StreamID,
	ownerID UserID,
	reason string,
) func(BotState, event.Version) ([]event.Event, error) {
	return func(state BotState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, errorfamily.NewRejection("usermgmt.bot_transfer.not_found", "bot does not exist")
		}
		switch {
		case ownerID.IsZero():
			return nil, errorfamily.NewRejection("usermgmt.bot_transfer.owner_required",
				"bot owner ID is required")
		case state.OwnerID.Get() == ownerID.Get():
			return nil, nil
		}
		evt, err := event.New(
			eventBotTransferred, aggID, aggregateTypeBot, version.Increment(),
			identitymodel.BotTransferredPayload{
				SchemaVersion: currentSchemaVersion,
				OwnerID:       ownerID,
				Reason:        reason,
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.bot_transfer.event_failed",
				"create BotTransferred event",
			)
		}
		return []event.Event{evt}, nil
	}
}
