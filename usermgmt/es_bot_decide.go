package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// BotDecider returns the Decider for the Bot aggregate.
func BotDecider() decider.Decider[BotState] {
	return decider.Decider[BotState]{
		Initial: BotState{}, //nolint:exhaustruct // zero-value is correct for aggregate initial state
		Apply:   foldBot,
	}
}

func decideRegisterBot(
	aggID id.AggregateID,
	name string,
	ownerID UserID,
	tokenHash []byte,
	scopes []string,
) func(BotState, event.Version) ([]event.Event, error) {
	return func(state BotState, version event.Version) ([]event.Event, error) {
		if state.Name != "" {
			return nil, errorfamily.NewConflict(
				"usermgmt.bot.already_exists",
				"bot with this ID already exists",
			)
		}
		if name == "" {
			return nil, errorfamily.NewRejection(
				"usermgmt.bot.name_required",
				"bot name is required",
			)
		}
		if ownerID.IsZero() {
			return nil, errorfamily.NewRejection(
				"usermgmt.bot.owner_required",
				"bot owner ID is required",
			)
		}
		if len(tokenHash) == 0 {
			return nil, errorfamily.NewRejection(
				"usermgmt.bot.token_hash_required",
				"bot token hash is required",
			)
		}
		scopesCopy := make([]string, len(scopes))
		copy(scopesCopy, scopes)
		payload, err := marshalPayload(BotRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          name,
			OwnerID:       ownerID,
			TokenHash:     tokenHash,
			Scopes:        scopesCopy,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.bot.marshal_failed",
				"marshal BotRegistered payload",
			)
		}
		evt, err := event.NewEvent(
			eventBotRegistered, aggID, aggregateTypeBot, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.bot.event_failed",
				"create BotRegistered event",
			)
		}
		return []event.Event{evt}, nil
	}
}

//nolint:dupl // mirrors decideDeleteTenant; tombstone deletion is structurally identical
func decideDeleteBot(
	aggID id.AggregateID,
	reason string,
) func(BotState, event.Version) ([]event.Event, error) {
	return func(state BotState, version event.Version) ([]event.Event, error) {
		if !state.Exists() {
			return nil, errorfamily.NewRejection(
				"usermgmt.bot_delete.not_found",
				"bot does not exist",
			)
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection(
				"usermgmt.bot_delete.already_deleted",
				"bot is already deleted",
			)
		}
		payload, err := marshalPayload(BotDeletedPayload{
			SchemaVersion: currentSchemaVersion,
			Reason:        reason,
		})
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.bot_delete.marshal_failed",
				"marshal BotDeleted payload",
			)
		}
		evt, err := event.NewEvent(
			eventBotDeleted, aggID, aggregateTypeBot, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.bot_delete.event_failed",
				"create BotDeleted event",
			)
		}
		marked, markErr := event.MarkTombstone(evt)
		if markErr != nil {
			return nil, errorfamily.WrapInfrastructure(
				markErr,
				"usermgmt.bot_delete.tombstone_failed",
				"mark bot tombstone",
			)
		}
		return []event.Event{marked}, nil
	}
}
