package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/scenario/v4"
	errorfamily "github.com/larsartmann/go-error-family"
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
		inner := decideRegisterUser(cmd.StreamID(), cmd.email, cmd.displayName, cmd.roles)
		return inner(state, 0)
	}

	scenario.Given[*RegisterUserCmd, UserState](t, foldUser, UserState{}).
		When(cmd, decide).
		Then(eventUserRegistered)
}

// TestScenario_RegisterUser_AlreadyExists demonstrates the ThenError path.
// go-error-family's *Error implements Is() matching by code+family, so we
// pass an errorfamily.NewConflict with the same code as the decider produces.
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
		inner := decideRegisterUser(cmd.StreamID(), cmd.email, cmd.displayName, cmd.roles)
		return inner(state, 1)
	}

	scenario.Given[*RegisterUserCmd, UserState](t, foldUser, UserState{}, existing).
		When(cmd, decide).
		ThenError(errorfamily.NewConflict("usermgmt.user_already_exists", ""))
}

// TestScenario_ChangeEmail_HappyPath demonstrates the scenario/v3 DSL on a
// second decider. Given an existing registered user, changing email should
// emit exactly one EmailChanged event.
func TestScenario_ChangeEmail_HappyPath(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewChangeEmailCmd(aggID, "new@test.com")

	registered, err := event.NewEvent(
		eventUserRegistered, aggID, aggregateTypeUser, 1,
		mustMarshalPayload(t, UserRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         "old@test.com",
			DisplayName:   "Test User",
		}),
	)
	if err != nil {
		t.Fatalf("create registration event: %v", err)
	}

	decide := func(state UserState, _ *ChangeEmailCmd) ([]event.Event, error) {
		inner := decideChangeEmail(cmd.StreamID(), cmd.email)
		return inner(state, 1)
	}

	scenario.Given[*ChangeEmailCmd, UserState](t, foldUser, UserState{}, registered).
		When(cmd, decide).
		Then(eventEmailChanged)
}
