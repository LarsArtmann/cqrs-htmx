package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type (
	CreateTenantCmd     = identitymodel.CreateTenantCmd
	SuspendTenantCmd    = identitymodel.SuspendTenantCmd
	ReactivateTenantCmd = identitymodel.ReactivateTenantCmd
	DeleteTenantCmd     = identitymodel.DeleteTenantCmd
)

func NewCreateTenantCmd(aggID id.AggregateID, name, displayName string) *CreateTenantCmd {
	return identitymodel.NewCreateTenantCmd(aggID, name, displayName)
}
func NewSuspendTenantCmd(aggID id.AggregateID, reason string) *SuspendTenantCmd {
	return identitymodel.NewSuspendTenantCmd(aggID, reason)
}
func NewReactivateTenantCmd(aggID id.AggregateID) *ReactivateTenantCmd {
	return identitymodel.NewReactivateTenantCmd(aggID)
}
func NewDeleteTenantCmd(aggID id.AggregateID, reason string) *DeleteTenantCmd {
	return identitymodel.NewDeleteTenantCmd(aggID, reason)
}
