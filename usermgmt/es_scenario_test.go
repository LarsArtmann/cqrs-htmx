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

// NOTE: Error-path scenario testing requires either upstream support in
// go-error-family (an Is method that matches by Family) or a ThenErrorFamily
// method in scenario/v3. The current scenario.ThenError uses errors.Is,
// which doesn't work with go-error-family's code-based identity.
// Filed as a finding for future scenario/v3 adoption.
