package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// CreateTenantCmd creates a new tenant.
type CreateTenantCmd struct {
	aggregateID id.AggregateID
	name        string
	displayName string
}

func NewCreateTenantCmd(aggID id.AggregateID, name, displayName string) *CreateTenantCmd {
	return &CreateTenantCmd{aggregateID: aggID, name: name, displayName: displayName}
}

func (c *CreateTenantCmd) Type() command.Type          { return cmdCreateTenant }
func (c *CreateTenantCmd) AggregateID() id.AggregateID { return c.aggregateID }

// SuspendTenantCmd temporarily suspends a tenant.
type SuspendTenantCmd struct {
	aggregateID id.AggregateID
	reason      string
}

func NewSuspendTenantCmd(aggID id.AggregateID, reason string) *SuspendTenantCmd {
	return &SuspendTenantCmd{aggregateID: aggID, reason: reason}
}

func (c *SuspendTenantCmd) Type() command.Type          { return cmdSuspendTenant }
func (c *SuspendTenantCmd) AggregateID() id.AggregateID { return c.aggregateID }

// ReactivateTenantCmd restores a suspended tenant.
type ReactivateTenantCmd struct {
	aggregateID id.AggregateID
}

func NewReactivateTenantCmd(aggID id.AggregateID) *ReactivateTenantCmd {
	return &ReactivateTenantCmd{aggregateID: aggID}
}

func (c *ReactivateTenantCmd) Type() command.Type          { return cmdReactivateTenant }
func (c *ReactivateTenantCmd) AggregateID() id.AggregateID { return c.aggregateID }

// DeleteTenantCmd permanently deletes a tenant.
type DeleteTenantCmd struct {
	aggregateID id.AggregateID
	reason      string
}

func NewDeleteTenantCmd(aggID id.AggregateID, reason string) *DeleteTenantCmd {
	return &DeleteTenantCmd{aggregateID: aggID, reason: reason}
}

func (c *DeleteTenantCmd) Type() command.Type          { return cmdDeleteTenant }
func (c *DeleteTenantCmd) AggregateID() id.AggregateID { return c.aggregateID }
