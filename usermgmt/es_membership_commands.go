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
	aggregateID id.AggregateID
	actorID     ActorID
	tenantID    TenantID
	roles       []Role
}

// NewAddMemberCmd constructs an AddMemberCmd. The aggregate ID is derived
// from the actor+tenant pair so each membership is a separate aggregate.
func NewAddMemberCmd(actorID ActorID, tenantID TenantID, roles []Role) *AddMemberCmd {
	return &AddMemberCmd{
		aggregateID: deriveMembershipID(actorID, tenantID),
		actorID:     actorID,
		tenantID:    tenantID,
		roles:       roles,
	}
}

func (c *AddMemberCmd) Type() command.Type { return cmdAddMember }

func (c *AddMemberCmd) AggregateID() id.AggregateID { return c.aggregateID }

// UpdateMemberRolesCmd changes the roles of an existing membership.
type UpdateMemberRolesCmd struct {
	aggregateID id.AggregateID
	roles       []Role
}

// NewUpdateMemberRolesCmd constructs an UpdateMemberRolesCmd.
func NewUpdateMemberRolesCmd(
	actorID ActorID, tenantID TenantID, roles []Role,
) *UpdateMemberRolesCmd {
	return &UpdateMemberRolesCmd{
		aggregateID: deriveMembershipID(actorID, tenantID),
		roles:       roles,
	}
}

func (c *UpdateMemberRolesCmd) Type() command.Type { return cmdUpdateMemberRoles }

func (c *UpdateMemberRolesCmd) AggregateID() id.AggregateID { return c.aggregateID }

// RemoveMemberCmd removes an actor from a tenant.
type RemoveMemberCmd struct {
	aggregateID id.AggregateID
}

// NewRemoveMemberCmd constructs a RemoveMemberCmd.
func NewRemoveMemberCmd(actorID ActorID, tenantID TenantID) *RemoveMemberCmd {
	return &RemoveMemberCmd{
		aggregateID: deriveMembershipID(actorID, tenantID),
	}
}

func (c *RemoveMemberCmd) Type() command.Type { return cmdRemoveMember }

func (c *RemoveMemberCmd) AggregateID() id.AggregateID { return c.aggregateID }
