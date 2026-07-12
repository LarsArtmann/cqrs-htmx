package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/scenario/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Membership scenario/v3 BDD tests. Exercises AddMember, UpdateMemberRoles,
// and RemoveMember decide functions. Membership aggregate IDs are derived
// from the actor+tenant pair, so we use the command's own AggregateID().

func TestScenario_AddMember_HappyPath(t *testing.T) {
	t.Parallel()

	actor := NewActorID(ActorUser, "01JXUSER001")
	tenant := NewTenantID("01JXTENANT01")
	cmd := NewAddMemberCmd(actor, tenant, []Role{"admin"})

	decide := func(state MembershipState, _ *AddMemberCmd) ([]event.Event, error) {
		inner := decideAddMember(cmd.AggregateID(), cmd.actorID, cmd.tenantID, cmd.roles)
		return inner(state, 0)
	}

	scenario.Given[*AddMemberCmd, MembershipState](t, foldMembership, MembershipState{}).
		When(cmd, decide).
		Then(eventMemberAdded)
}

func TestScenario_AddMember_AlreadyExists(t *testing.T) {
	t.Parallel()

	actor := NewActorID(ActorUser, "01JXUSER001")
	tenant := NewTenantID("01JXTENANT01")
	cmd := NewAddMemberCmd(actor, tenant, []Role{"admin"})

	existing, err := event.NewEvent(
		eventMemberAdded, cmd.AggregateID(), aggregateTypeMembership, 1,
		mustMarshalPayload(t, MemberAddedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorKind:     actor.Kind().String(),
			ActorID:       actor.String(),
			TenantID:      tenant.Get(),
			Roles:         []Role{"member"},
		}),
	)
	if err != nil {
		t.Fatalf("create existing event: %v", err)
	}

	decide := func(state MembershipState, _ *AddMemberCmd) ([]event.Event, error) {
		inner := decideAddMember(cmd.AggregateID(), cmd.actorID, cmd.tenantID, cmd.roles)
		return inner(state, 1)
	}

	scenario.Given[*AddMemberCmd, MembershipState](t, foldMembership, MembershipState{}, existing).
		When(cmd, decide).
		ThenError(errorfamily.NewConflict("usermgmt.membership.already_exists", ""))
}

func TestScenario_AddMember_ActorRequired(t *testing.T) {
	t.Parallel()

	tenant := NewTenantID("01JXTENANT01")
	cmd := NewAddMemberCmd(ActorID{}, tenant, []Role{"admin"})

	decide := func(state MembershipState, _ *AddMemberCmd) ([]event.Event, error) {
		inner := decideAddMember(cmd.AggregateID(), cmd.actorID, cmd.tenantID, cmd.roles)
		return inner(state, 0)
	}

	scenario.Given[*AddMemberCmd, MembershipState](t, foldMembership, MembershipState{}).
		When(cmd, decide).
		ThenError(errorfamily.NewRejection("usermgmt.membership.actor_required", ""))
}

func TestScenario_UpdateMemberRoles_HappyPath(t *testing.T) {
	t.Parallel()

	actor := NewActorID(ActorUser, "01JXUSER001")
	tenant := NewTenantID("01JXTENANT01")
	cmd := NewUpdateMemberRolesCmd(actor, tenant, []Role{"admin", "writer"})

	added, err := event.NewEvent(
		eventMemberAdded, cmd.AggregateID(), aggregateTypeMembership, 1,
		mustMarshalPayload(t, MemberAddedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorKind:     actor.Kind().String(),
			ActorID:       actor.String(),
			TenantID:      tenant.Get(),
			Roles:         []Role{"member"},
		}),
	)
	if err != nil {
		t.Fatalf("create member event: %v", err)
	}

	decide := func(state MembershipState, _ *UpdateMemberRolesCmd) ([]event.Event, error) {
		inner := decideUpdateMemberRoles(cmd.AggregateID(), cmd.roles)
		return inner(state, 1)
	}

	scenario.Given[*UpdateMemberRolesCmd, MembershipState](t, foldMembership, MembershipState{}, added).
		When(cmd, decide).
		Then(eventMemberRolesChanged)
}

func TestScenario_RemoveMember_HappyPath(t *testing.T) {
	t.Parallel()

	actor := NewActorID(ActorUser, "01JXUSER001")
	tenant := NewTenantID("01JXTENANT01")
	cmd := NewRemoveMemberCmd(actor, tenant)

	added, err := event.NewEvent(
		eventMemberAdded, cmd.AggregateID(), aggregateTypeMembership, 1,
		mustMarshalPayload(t, MemberAddedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorKind:     actor.Kind().String(),
			ActorID:       actor.String(),
			TenantID:      tenant.Get(),
			Roles:         []Role{"member"},
		}),
	)
	if err != nil {
		t.Fatalf("create member event: %v", err)
	}

	decide := func(state MembershipState, _ *RemoveMemberCmd) ([]event.Event, error) {
		inner := decideRemoveMember(cmd.AggregateID())
		return inner(state, 1)
	}

	scenario.Given[*RemoveMemberCmd, MembershipState](t, foldMembership, MembershipState{}, added).
		When(cmd, decide).
		Then(eventMemberRemoved)
}

func TestScenario_RemoveMember_NotFound(t *testing.T) {
	t.Parallel()

	actor := NewActorID(ActorUser, "01JXUSER001")
	tenant := NewTenantID("01JXTENANT01")
	cmd := NewRemoveMemberCmd(actor, tenant)

	decide := func(state MembershipState, _ *RemoveMemberCmd) ([]event.Event, error) {
		inner := decideRemoveMember(cmd.AggregateID())
		return inner(state, 0)
	}

	scenario.Given[*RemoveMemberCmd, MembershipState](t, foldMembership, MembershipState{}).
		When(cmd, decide).
		ThenError(errorfamily.NewRejection("usermgmt.membership_remove.not_found", ""))
}
