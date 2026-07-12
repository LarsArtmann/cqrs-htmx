package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// CreateTenantCmd creates a new tenant.
type CreateTenantCmd struct {
	*command.BasicCommand
	name        string
	displayName string
}

func NewCreateTenantCmd(aggID id.AggregateID, name, displayName string) *CreateTenantCmd {
	return &CreateTenantCmd{
		BasicCommand: mustCommand(cmdCreateTenant, aggID),
		name:         name,
		displayName:  displayName,
	}
}

// SuspendTenantCmd temporarily suspends a tenant.
type SuspendTenantCmd struct {
	*command.BasicCommand
	reason string
}

func NewSuspendTenantCmd(aggID id.AggregateID, reason string) *SuspendTenantCmd {
	return &SuspendTenantCmd{
		BasicCommand: mustCommand(cmdSuspendTenant, aggID),
		reason:       reason,
	}
}

// ReactivateTenantCmd restores a suspended tenant.
type ReactivateTenantCmd struct {
	*command.BasicCommand
}

func NewReactivateTenantCmd(aggID id.AggregateID) *ReactivateTenantCmd {
	return &ReactivateTenantCmd{
		BasicCommand: mustCommand(cmdReactivateTenant, aggID),
	}
}

// DeleteTenantCmd permanently deletes a tenant.
type DeleteTenantCmd struct {
	*command.BasicCommand
	reason string
}

func NewDeleteTenantCmd(aggID id.AggregateID, reason string) *DeleteTenantCmd {
	return &DeleteTenantCmd{
		BasicCommand: mustCommand(cmdDeleteTenant, aggID),
		reason:       reason,
	}
}
