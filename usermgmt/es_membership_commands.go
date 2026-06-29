package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// deriveMembershipID constructs a deterministic AggregateID for a membership.
// Same actor+tenant always produces the same ID (idempotent).
func deriveMembershipID(actorID ActorID, tenantID TenantID) id.AggregateID {
	return id.DeriveAggregateID("membership", actorID.PrefixedString(), tenantID.Get())
}

// AddMemberCmd adds an actor to a tenant with the given roles.
type AddMemberCmd struct {
	*command.BasicCommand
	actorID  ActorID
	tenantID TenantID
	roles    []Role
}

// NewAddMemberCmd constructs an AddMemberCmd. The aggregate ID is derived
// from the actor+tenant pair so each membership is a separate aggregate.
func NewAddMemberCmd(actorID ActorID, tenantID TenantID, roles []Role) *AddMemberCmd {
	return &AddMemberCmd{
		BasicCommand: mustCommand(cmdAddMember, deriveMembershipID(actorID, tenantID)),
		actorID:      actorID,
		tenantID:     tenantID,
		roles:        roles,
	}
}

// UpdateMemberRolesCmd changes the roles of an existing membership.
type UpdateMemberRolesCmd struct {
	*command.BasicCommand
	roles []Role
}

// NewUpdateMemberRolesCmd constructs an UpdateMemberRolesCmd.
func NewUpdateMemberRolesCmd(
	actorID ActorID, tenantID TenantID, roles []Role,
) *UpdateMemberRolesCmd {
	return &UpdateMemberRolesCmd{
		BasicCommand: mustCommand(cmdUpdateMemberRoles, deriveMembershipID(actorID, tenantID)),
		roles:        roles,
	}
}

// RemoveMemberCmd removes an actor from a tenant.
type RemoveMemberCmd struct {
	*command.BasicCommand
}

// NewRemoveMemberCmd constructs a RemoveMemberCmd.
func NewRemoveMemberCmd(actorID ActorID, tenantID TenantID) *RemoveMemberCmd {
	return &RemoveMemberCmd{
		BasicCommand: mustCommand(cmdRemoveMember, deriveMembershipID(actorID, tenantID)),
	}
}
