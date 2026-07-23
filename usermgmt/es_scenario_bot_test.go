package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/scenario/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Bot scenario/v3 BDD tests. Exercises RegisterBot and DeleteBot decide functions.

func TestScenario_RegisterBot_HappyPath(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	ownerID := NewUserID("bot-owner")
	tokenHash := []byte{0x01, 0x02, 0x03}
	cmd := NewRegisterBotCmd(aggID, "ci-bot", ownerID, tokenHash, []string{"read", "write"})

	decide := func(state BotState, _ *RegisterBotCmd) ([]event.Event, error) {
		inner := decideRegisterBot(
			cmd.StreamID(), cmd.name, cmd.ownerID, cmd.tokenHash, cmd.scopes,
		)
		return inner(state, 0)
	}

	scenario.Given[*RegisterBotCmd, BotState](t, foldBot, BotState{}).
		When(cmd, decide).
		Then(eventBotRegistered)
}

func TestScenario_RegisterBot_AlreadyExists(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	ownerID := NewUserID("bot-owner")
	tokenHash := []byte{0x01, 0x02, 0x03}
	cmd := NewRegisterBotCmd(aggID, "ci-bot", ownerID, tokenHash, []string{"read"})

	existing, err := event.NewEvent(
		eventBotRegistered, aggID, aggregateTypeBot, 1,
		mustMarshalPayload(t, BotRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          "ci-bot",
			OwnerID:       ownerID,
			TokenHash:     tokenHash,
			Scopes:        []string{"read"},
		}),
	)
	if err != nil {
		t.Fatalf("create existing event: %v", err)
	}

	decide := func(state BotState, _ *RegisterBotCmd) ([]event.Event, error) {
		inner := decideRegisterBot(
			cmd.StreamID(), cmd.name, cmd.ownerID, cmd.tokenHash, cmd.scopes,
		)
		return inner(state, 1)
	}

	scenario.Given[*RegisterBotCmd, BotState](t, foldBot, BotState{}, existing).
		When(cmd, decide).
		ThenError(errorfamily.NewConflict("usermgmt.bot.already_exists", ""))
}

func TestScenario_RegisterBot_NameRequired(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	ownerID := NewUserID("bot-owner")
	tokenHash := []byte{0x01}
	cmd := NewRegisterBotCmd(aggID, "", ownerID, tokenHash, nil)

	decide := func(state BotState, _ *RegisterBotCmd) ([]event.Event, error) {
		inner := decideRegisterBot(
			cmd.StreamID(), cmd.name, cmd.ownerID, cmd.tokenHash, cmd.scopes,
		)
		return inner(state, 0)
	}

	scenario.Given[*RegisterBotCmd, BotState](t, foldBot, BotState{}).
		When(cmd, decide).
		ThenError(errorfamily.NewRejection("usermgmt.bot.name_required", ""))
}

func TestScenario_RegisterBot_OwnerRequired(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	tokenHash := []byte{0x01}
	cmd := NewRegisterBotCmd(aggID, "ci-bot", NewUserID(""), tokenHash, nil)

	decide := func(state BotState, _ *RegisterBotCmd) ([]event.Event, error) {
		inner := decideRegisterBot(
			cmd.StreamID(), cmd.name, cmd.ownerID, cmd.tokenHash, cmd.scopes,
		)
		return inner(state, 0)
	}

	scenario.Given[*RegisterBotCmd, BotState](t, foldBot, BotState{}).
		When(cmd, decide).
		ThenError(errorfamily.NewRejection("usermgmt.bot.owner_required", ""))
}

func TestScenario_DeleteBot_HappyPath(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	ownerID := NewUserID("bot-owner")
	tokenHash := []byte{0x01, 0x02, 0x03}
	cmd := NewDeleteBotCmd(aggID, "decommissioned")

	registered, err := event.NewEvent(
		eventBotRegistered, aggID, aggregateTypeBot, 1,
		mustMarshalPayload(t, BotRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          "ci-bot",
			OwnerID:       ownerID,
			TokenHash:     tokenHash,
			Scopes:        []string{"read"},
		}),
	)
	if err != nil {
		t.Fatalf("create bot event: %v", err)
	}

	decide := func(state BotState, _ *DeleteBotCmd) ([]event.Event, error) {
		inner := decideDeleteBot(cmd.StreamID(), cmd.reason)
		return inner(state, 1)
	}

	scenario.Given[*DeleteBotCmd, BotState](t, foldBot, BotState{}, registered).
		When(cmd, decide).
		Then(eventBotDeleted)
}

func TestScenario_DeleteBot_NotFound(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewDeleteBotCmd(aggID, "reason")

	decide := func(state BotState, _ *DeleteBotCmd) ([]event.Event, error) {
		inner := decideDeleteBot(cmd.StreamID(), cmd.reason)
		return inner(state, 0)
	}

	scenario.Given[*DeleteBotCmd, BotState](t, foldBot, BotState{}).
		When(cmd, decide).
		ThenError(errorfamily.NewRejection("usermgmt.bot_delete.not_found", ""))
}
