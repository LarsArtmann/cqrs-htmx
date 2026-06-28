package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// CreateTenantCmd creates a new tenant.
type CreateTenantCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
	name        string
	displayName string
}

func NewCreateTenantCmd(aggID id.AggregateID, name, displayName string) *CreateTenantCmd {
	return &CreateTenantCmd{aggregateID: aggID, cmdID: id.NewCommandID(), name: name, displayName: displayName}
}

func (c *CreateTenantCmd) Type() command.Type          { return cmdCreateTenant }
func (c *CreateTenantCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *CreateTenantCmd) ID() id.CommandID            { return c.cmdID }

// SuspendTenantCmd temporarily suspends a tenant.
type SuspendTenantCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
	reason      string
}

func NewSuspendTenantCmd(aggID id.AggregateID, reason string) *SuspendTenantCmd {
	return &SuspendTenantCmd{aggregateID: aggID, cmdID: id.NewCommandID(), reason: reason}
}

func (c *SuspendTenantCmd) Type() command.Type          { return cmdSuspendTenant }
func (c *SuspendTenantCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *SuspendTenantCmd) ID() id.CommandID            { return c.cmdID }

// ReactivateTenantCmd restores a suspended tenant.
type ReactivateTenantCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
}

func NewReactivateTenantCmd(aggID id.AggregateID) *ReactivateTenantCmd {
	return &ReactivateTenantCmd{aggregateID: aggID, cmdID: id.NewCommandID()}
}

func (c *ReactivateTenantCmd) Type() command.Type          { return cmdReactivateTenant }
func (c *ReactivateTenantCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *ReactivateTenantCmd) ID() id.CommandID            { return c.cmdID }

// DeleteTenantCmd permanently deletes a tenant.
type DeleteTenantCmd struct {
	aggregateID id.AggregateID
	cmdID       id.CommandID
	reason      string
}

func NewDeleteTenantCmd(aggID id.AggregateID, reason string) *DeleteTenantCmd {
	return &DeleteTenantCmd{aggregateID: aggID, cmdID: id.NewCommandID(), reason: reason}
}

func (c *DeleteTenantCmd) Type() command.Type          { return cmdDeleteTenant }
func (c *DeleteTenantCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *DeleteTenantCmd) ID() id.CommandID            { return c.cmdID }
