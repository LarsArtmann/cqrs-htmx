package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/scenario/v3"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Tenant scenario/v3 BDD tests. Exercises the four Tenant decide functions
// (create, suspend, reactivate, delete) through the scenario DSL.

func TestScenario_CreateTenant_HappyPath(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewCreateTenantCmd(aggID, "acme", "Acme Corp")

	decide := func(state TenantState, _ *CreateTenantCmd) ([]event.Event, error) {
		inner := decideCreateTenant(cmd.AggregateID(), cmd.name, cmd.displayName)
		return inner(state, 0)
	}

	scenario.Given[*CreateTenantCmd, TenantState](t, foldTenant, TenantState{}).
		When(cmd, decide).
		Then(eventTenantCreated)
}

func TestScenario_CreateTenant_AlreadyExists(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewCreateTenantCmd(aggID, "acme", "Acme Corp")

	existing, err := event.NewEvent(
		eventTenantCreated, aggID, aggregateTypeTenant, 1,
		mustMarshalPayload(t, TenantCreatedPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          "acme",
			DisplayName:   "Old Name",
		}),
	)
	if err != nil {
		t.Fatalf("create existing event: %v", err)
	}

	decide := func(state TenantState, _ *CreateTenantCmd) ([]event.Event, error) {
		inner := decideCreateTenant(cmd.AggregateID(), cmd.name, cmd.displayName)
		return inner(state, 1)
	}

	scenario.Given[*CreateTenantCmd, TenantState](t, foldTenant, TenantState{}, existing).
		When(cmd, decide).
		ThenError(errorfamily.NewConflict("usermgmt.tenant.already_exists", ""))
}

func TestScenario_CreateTenant_NameRequired(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewCreateTenantCmd(aggID, "", "Display")

	decide := func(state TenantState, _ *CreateTenantCmd) ([]event.Event, error) {
		inner := decideCreateTenant(cmd.AggregateID(), cmd.name, cmd.displayName)
		return inner(state, 0)
	}

	scenario.Given[*CreateTenantCmd, TenantState](t, foldTenant, TenantState{}).
		When(cmd, decide).
		ThenError(errorfamily.NewRejection("usermgmt.tenant.name_required", ""))
}

func TestScenario_SuspendTenant_HappyPath(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewSuspendTenantCmd(aggID, "policy violation")

	created, err := event.NewEvent(
		eventTenantCreated, aggID, aggregateTypeTenant, 1,
		mustMarshalPayload(t, TenantCreatedPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          "acme",
			DisplayName:   "Acme",
		}),
	)
	if err != nil {
		t.Fatalf("create tenant event: %v", err)
	}

	decide := func(state TenantState, _ *SuspendTenantCmd) ([]event.Event, error) {
		inner := decideSuspendTenant(cmd.AggregateID(), cmd.reason)
		return inner(state, 1)
	}

	scenario.Given[*SuspendTenantCmd, TenantState](t, foldTenant, TenantState{}, created).
		When(cmd, decide).
		Then(eventTenantSuspended)
}

func TestScenario_SuspendTenant_NotFound(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewSuspendTenantCmd(aggID, "reason")

	decide := func(state TenantState, _ *SuspendTenantCmd) ([]event.Event, error) {
		inner := decideSuspendTenant(cmd.AggregateID(), cmd.reason)
		return inner(state, 0)
	}

	scenario.Given[*SuspendTenantCmd, TenantState](t, foldTenant, TenantState{}).
		When(cmd, decide).
		ThenError(errorfamily.NewRejection("usermgmt.tenant_suspend.not_found", ""))
}

func TestScenario_ReactivateTenant_HappyPath(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewReactivateTenantCmd(aggID)

	created, err := event.NewEvent(
		eventTenantCreated, aggID, aggregateTypeTenant, 1,
		mustMarshalPayload(t, TenantCreatedPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          "acme",
			DisplayName:   "Acme",
		}),
	)
	if err != nil {
		t.Fatalf("create tenant event: %v", err)
	}
	suspended, err := event.NewEvent(
		eventTenantSuspended, aggID, aggregateTypeTenant, 2,
		mustMarshalPayload(t, TenantSuspendedPayload{
			SchemaVersion: currentSchemaVersion,
			Reason:        "temp",
		}),
	)
	if err != nil {
		t.Fatalf("create suspend event: %v", err)
	}

	decide := func(state TenantState, _ *ReactivateTenantCmd) ([]event.Event, error) {
		inner := decideReactivateTenant(cmd.AggregateID())
		return inner(state, 2)
	}

	scenario.Given[*ReactivateTenantCmd, TenantState](t, foldTenant, TenantState{}, created, suspended).
		When(cmd, decide).
		Then(eventTenantReactivated)
}

func TestScenario_DeleteTenant_HappyPath(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewDeleteTenantCmd(aggID, "shutdown")

	created, err := event.NewEvent(
		eventTenantCreated, aggID, aggregateTypeTenant, 1,
		mustMarshalPayload(t, TenantCreatedPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          "acme",
			DisplayName:   "Acme",
		}),
	)
	if err != nil {
		t.Fatalf("create tenant event: %v", err)
	}

	decide := func(state TenantState, _ *DeleteTenantCmd) ([]event.Event, error) {
		inner := decideDeleteTenant(cmd.AggregateID(), cmd.reason)
		return inner(state, 1)
	}

	scenario.Given[*DeleteTenantCmd, TenantState](t, foldTenant, TenantState{}, created).
		When(cmd, decide).
		Then(eventTenantDeleted)
}

func TestScenario_DeleteTenant_NotFound(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	cmd := NewDeleteTenantCmd(aggID, "reason")

	decide := func(state TenantState, _ *DeleteTenantCmd) ([]event.Event, error) {
		inner := decideDeleteTenant(cmd.AggregateID(), cmd.reason)
		return inner(state, 0)
	}

	scenario.Given[*DeleteTenantCmd, TenantState](t, foldTenant, TenantState{}).
		When(cmd, decide).
		ThenError(errorfamily.NewRejection("usermgmt.tenant_delete.not_found", ""))
}
