package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	CreateTenantCmd = identitymodel.CreateTenantCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	SuspendTenantCmd = identitymodel.SuspendTenantCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ReactivateTenantCmd = identitymodel.ReactivateTenantCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	DeleteTenantCmd = identitymodel.DeleteTenantCmd
)

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewCreateTenantCmd(aggID id.StreamID, name, displayName string) *CreateTenantCmd {
	return identitymodel.NewCreateTenantCmd(aggID, name, displayName)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewSuspendTenantCmd(aggID id.StreamID, reason string) *SuspendTenantCmd {
	return identitymodel.NewSuspendTenantCmd(aggID, reason)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewReactivateTenantCmd(aggID id.StreamID) *ReactivateTenantCmd {
	return identitymodel.NewReactivateTenantCmd(aggID)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewDeleteTenantCmd(aggID id.StreamID, reason string) *DeleteTenantCmd {
	return identitymodel.NewDeleteTenantCmd(aggID, reason)
}
