package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/scenario/v3"
)

// TestScenario_RegisterUser_New demonstrates the scenario/v3 BDD DSL
// for testing deciders. This is a spike to evaluate adoption — if the
// pattern proves clean, it can replace verbose setup-heavy tests.
func TestScenario_RegisterUser_New(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewRegisterUserCmd(aggID, "user@test.com", "Test User", []Role{"admin"})

	// Adapter: scenario.DecideFunc is (state, cmd) → events.
	// Our decider is (aggID, params) → (state, version) → events.
	// The adapter closes over the command to bridge the two shapes.
	decide := func(state UserState, _ *RegisterUserCmd) ([]event.Event, error) {
		inner := decideRegisterUser(cmd.AggregateID(), cmd.email, cmd.displayName, cmd.roles)
		return inner(state, 0)
	}

	scenario.Given[*RegisterUserCmd, UserState](t, foldUser, UserState{}).
		When(cmd, decide).
		Then(eventUserRegistered)
}

// TestScenario_RegisterUser_AlreadyExists demonstrates the ThenError path.
// go-error-family's *Error implements Is() matching by code+family, so we
// pass an event.NewConflict with the same code as the decider produces.
func TestScenario_RegisterUser_AlreadyExists(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewRegisterUserCmd(aggID, "user@test.com", "Test User", nil)

	existing, err := event.NewEvent(
		eventUserRegistered, aggID, aggregateTypeUser, 1,
		mustMarshalPayload(t, UserRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         "user@test.com",
			DisplayName:   "Old Name",
		}),
	)
	if err != nil {
		t.Fatalf("create existing event: %v", err)
	}

	decide := func(state UserState, _ *RegisterUserCmd) ([]event.Event, error) {
		inner := decideRegisterUser(cmd.AggregateID(), cmd.email, cmd.displayName, cmd.roles)
		return inner(state, 1)
	}

	scenario.Given[*RegisterUserCmd, UserState](t, foldUser, UserState{}, existing).
		When(cmd, decide).
		ThenError(event.NewConflict("usermgmt.user_already_exists", ""))
}
